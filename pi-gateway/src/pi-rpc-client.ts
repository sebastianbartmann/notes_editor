import { spawn } from 'node:child_process';
import * as readline from 'node:readline';

export type PiRpcEvent = Record<string, any>;

type RpcResponse = {
  type: 'response';
  id?: string;
  command: string;
  success: boolean;
  data?: any;
  error?: string;
};

export type PiRpcClientOptions = {
  // Node binary used to run pi in RPC mode. Default: current Node (process.execPath).
  nodeBin?: string;
  cliPath: string;
  provider?: string;
  model?: string;
  args?: string[];
  env?: Record<string, string>;
  cwd?: string;
};

export class PiRpcClient {
  private options: PiRpcClientOptions;
  private proc: ReturnType<typeof spawn> | null = null;
  private rl: readline.Interface | null = null;
  private stderr = '';
  private listeners: Array<(event: PiRpcEvent) => void> = [];
  private pending = new Map<string, { resolve: (r: RpcResponse) => void; reject: (e: Error) => void }>();
  private idCounter = 0;

  constructor(options: PiRpcClientOptions) {
    this.options = options;
  }

  async start(): Promise<void> {
    if (this.proc) throw new Error('PiRpcClient already started');

    const args: string[] = [this.options.cliPath, '--mode', 'rpc'];
    if (this.options.provider) args.push('--provider', this.options.provider);
    if (this.options.model) args.push('--model', this.options.model);
    if (this.options.args) args.push(...this.options.args);

    const nodeBin = this.options.nodeBin || process.execPath;
    this.proc = spawn(nodeBin, args, {
      cwd: this.options.cwd,
      env: { ...process.env, ...this.options.env },
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    this.proc.stderr?.on('data', (d) => {
      this.stderr += d.toString();
    });

    this.rl = readline.createInterface({ input: this.proc.stdout!, terminal: false });
    this.rl.on('line', (line) => this.handleLine(line));

    await new Promise((resolve) => setTimeout(resolve, 100));
    if (this.proc.exitCode !== null) {
      throw new Error(`pi rpc process exited immediately with code ${this.proc.exitCode}. stderr=${this.stderr}`);
    }
  }

  async stop(): Promise<void> {
    if (!this.proc) return;
    this.rl?.close();
    this.proc.kill('SIGTERM');
    await new Promise<void>((resolve) => {
      const t = setTimeout(() => {
        this.proc?.kill('SIGKILL');
        resolve();
      }, 1000);
      this.proc?.on('exit', () => {
        clearTimeout(t);
        resolve();
      });
    });
    this.proc = null;
    this.rl = null;
    this.pending.clear();
  }

  onEvent(listener: (event: PiRpcEvent) => void): () => void {
    this.listeners.push(listener);
    return () => {
      const idx = this.listeners.indexOf(listener);
      if (idx >= 0) this.listeners.splice(idx, 1);
    };
  }

  getStderr(): string {
    return this.stderr;
  }

  async prompt(message: string): Promise<void> {
    await this.send({ type: 'prompt', message });
  }

  async abort(): Promise<void> {
    await this.send({ type: 'abort' });
  }

  async switchSession(sessionPath: string): Promise<void> {
    await this.send({ type: 'switch_session', sessionPath });
  }

  async getState(): Promise<any> {
    const resp = await this.send({ type: 'get_state' });
    return resp.data;
  }

  waitForIdle(timeoutMs: number): Promise<void> {
    return new Promise((resolve, reject) => {
      const t = setTimeout(() => {
        unsub();
        reject(new Error(`timeout waiting for agent_end. stderr=${this.stderr}`));
      }, timeoutMs);
      const unsub = this.onEvent((event) => {
        if (event?.type === 'agent_end') {
          clearTimeout(t);
          unsub();
          resolve();
        }
      });
    });
  }

  private nextId(): string {
    this.idCounter++;
    return `req-${this.idCounter}`;
  }

  private async send(command: Record<string, any>): Promise<RpcResponse> {
    if (!this.proc || !this.proc.stdin) throw new Error('PiRpcClient not started');
    const id = command.id || this.nextId();
    command.id = id;

    const raw = JSON.stringify(command);
    this.proc.stdin.write(raw + '\n');

    return await new Promise<RpcResponse>((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      // Responses are expected quickly; the agent streaming continues via events.
      setTimeout(() => {
        if (this.pending.has(id)) {
          this.pending.delete(id);
          reject(new Error(`rpc response timeout for ${command.type}. stderr=${this.stderr}`));
        }
      }, 5000);
    });
  }

  private handleLine(line: string): void {
    let data: any;
    try {
      data = JSON.parse(line);
    } catch {
      return;
    }

    if (data?.type === 'response' && typeof data.id === 'string' && this.pending.has(data.id)) {
      const pending = this.pending.get(data.id)!;
      this.pending.delete(data.id);
      if (data.success) {
        pending.resolve(data as RpcResponse);
      } else {
        pending.reject(new Error(data.error || 'rpc command failed'));
      }
      return;
    }

    for (const l of this.listeners) {
      try {
        l(data);
      } catch {
        // ignore listener errors
      }
    }
  }
}
