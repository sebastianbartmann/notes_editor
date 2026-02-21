import { createServer, IncomingMessage, ServerResponse } from 'node:http';
import { mkdirSync, writeFileSync, existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { randomUUID } from 'node:crypto';
import { fileURLToPath } from 'node:url';

import { PiRpcClient } from './pi-rpc-client.js';

function maybeLoadServerDotEnv(): void {
  // In production, systemd injects env vars via EnvironmentFile.
  // For local runs (e.g. `npm start`), load ../server/.env so tools can authenticate back to the Go server.
  if ((process.env.NOTES_TOKEN || '').trim() && (process.env.NOTES_ROOT || '').trim()) return;

  try {
    const selfDir = path.dirname(fileURLToPath(import.meta.url));
    const candidate = path.resolve(selfDir, '..', '..', 'server', '.env');
    if (!existsSync(candidate)) return;

    const raw = readFileSync(candidate, 'utf8');
    for (const line of raw.split('\n')) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) continue;
      const eq = trimmed.indexOf('=');
      if (eq <= 0) continue;
      const key = trimmed.slice(0, eq).trim();
      let val = trimmed.slice(eq + 1).trim();
      if (!key) continue;
      if (val.startsWith('"') && val.endsWith('"') && val.length >= 2) {
        val = val.slice(1, -1);
      }
      if (val.startsWith("'") && val.endsWith("'") && val.length >= 2) {
        val = val.slice(1, -1);
      }

      // Do not overwrite explicit env vars.
      if (process.env[key] === undefined) {
        process.env[key] = val;
      }
    }
  } catch {
    // Best-effort only.
  }
}

function requireNode20(): void {
  const major = Number((process.versions.node || '').split('.')[0] || '0');
  if (Number.isNaN(major) || major < 20) {
    // Pi dependencies (pi-tui) require Node >= 20 (e.g. RegExp 'v' flag).
    // Fail fast with a clear message so systemd/journal shows the reason.
    // eslint-disable-next-line no-console
    console.error(`Node.js ${process.versions.node} is too old. pi-gateway requires Node >= 20.`);
    process.exit(1);
  }
}

requireNode20();
maybeLoadServerDotEnv();

type ChatRequest = {
  person?: string;
  session_id?: string;
  message?: string;
  // Optional: system prompt content for this run (will be applied by our Pi extension).
  system_prompt?: string;
};

const PORT = Number(process.env.PI_GATEWAY_PORT || '4317');
const MODE = (process.env.PI_GATEWAY_MODE || 'pi_rpc').trim();

const DEFAULT_PROVIDER = (process.env.PI_GATEWAY_PI_PROVIDER || 'anthropic').trim();
const DEFAULT_MODEL = (process.env.PI_GATEWAY_PI_MODEL || '').trim();
const SESSION_DIR = (process.env.PI_GATEWAY_PI_SESSION_DIR || path.join(process.env.HOME || '/tmp', '.pi', 'notes-editor-sessions')).trim();
const PI_TIMEOUT_MS = Number(process.env.PI_GATEWAY_PI_TIMEOUT_MS || '1800000');
const SESSION_IDLE_TTL_MS = Number(process.env.PI_GATEWAY_SESSION_IDLE_TTL_MS || String(10 * 60 * 1000));

// Our Pi extension registers tools and applies system prompt updates.
const SELF_DIR = path.dirname(fileURLToPath(import.meta.url));
function resolveExtensionPath(): string {
  const override = (process.env.PI_GATEWAY_PI_EXTENSION_PATH || '').trim();
  if (override) return override;

  const candidates = [
    path.join(SELF_DIR, 'pi-notes-editor-extension.js'), // production build output
    path.join(SELF_DIR, 'pi-notes-editor-extension.ts'), // dev (ts-node)
    path.join(SELF_DIR, '..', 'src', 'pi-notes-editor-extension.ts'), // running dist/ from project root
  ];
  for (const candidate of candidates) {
    if (existsSync(candidate)) return candidate;
  }
  // Fallback to the most-likely location even if missing (we'll surface error via stderr).
  return candidates[0];
}

const EXTENSION_PATH = resolveExtensionPath();

type SessionClient = {
  key: string;
  person: string;
  runtimeSessionId: string;
  client: PiRpcClient;
  provider: string;
  model: string;
  lastUsedAt: number;
  activeRequests: number;
  idleTimer: ReturnType<typeof setTimeout> | null;
};

const sessionClients = new Map<string, SessionClient>();

function sendJson(res: ServerResponse, status: number, payload: unknown): void {
  res.statusCode = status;
  res.setHeader('Content-Type', 'application/json');
  res.end(JSON.stringify(payload));
}

function writeEvent(res: ServerResponse, event: Record<string, unknown>): void {
  res.write(`${JSON.stringify(event)}\n`);
}

async function readJson<T>(req: IncomingMessage): Promise<T> {
  const chunks: Buffer[] = [];
  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  const raw = Buffer.concat(chunks).toString('utf8').trim();
  if (!raw) return {} as T;
  return JSON.parse(raw) as T;
}

function safeSessionFilename(person: string, sessionId: string): string {
  const raw = `${person}--${sessionId}`;
  // Keep this filesystem-friendly and stable.
  return raw.replace(/[^a-zA-Z0-9._-]+/g, '_') + '.jsonl';
}

function ensureSessionPath(person: string, sessionId: string): string {
  mkdirSync(SESSION_DIR, { recursive: true });
  const filePath = path.join(SESSION_DIR, safeSessionFilename(person, sessionId));
  if (!existsSync(filePath)) {
    writeFileSync(filePath, '', 'utf8');
  }
  return filePath;
}

function sessionClientKey(person: string, runtimeSessionId: string): string {
  return `${person}::${runtimeSessionId}`;
}

async function stopAndDeleteSessionClient(sc: SessionClient): Promise<void> {
  const current = sessionClients.get(sc.key);
  if (!current || current !== sc) return;
  if (sc.activeRequests > 0) return;
  sessionClients.delete(sc.key);
  if (sc.idleTimer) {
    clearTimeout(sc.idleTimer);
    sc.idleTimer = null;
  }
  try {
    await sc.client.stop();
  } catch {
    // Best-effort only.
  }
}

function scheduleSessionIdleCleanup(sc: SessionClient): void {
  if (sc.idleTimer) {
    clearTimeout(sc.idleTimer);
    sc.idleTimer = null;
  }
  sc.idleTimer = setTimeout(() => {
    if (sc.activeRequests > 0) return;
    const idleFor = Date.now() - sc.lastUsedAt;
    if (idleFor < SESSION_IDLE_TTL_MS) {
      scheduleSessionIdleCleanup(sc);
      return;
    }
    void stopAndDeleteSessionClient(sc);
  }, SESSION_IDLE_TTL_MS);
}

async function getOrStartSessionClient(person: string, runtimeSessionId: string): Promise<SessionClient> {
  const key = sessionClientKey(person, runtimeSessionId);
  const existing = sessionClients.get(key);
  if (existing) return existing;

  const provider = DEFAULT_PROVIDER;
  const model = DEFAULT_MODEL;
  const sessionPath = ensureSessionPath(person, runtimeSessionId);

  // Run one Pi process per runtime session to avoid cross-session interleaving.
  // Tools and system prompt injection are handled by our extension.
  const rpc = new PiRpcClient({
    nodeBin: process.execPath,
    cliPath: path.join(process.cwd(), 'node_modules', '@mariozechner', 'pi-coding-agent', 'dist', 'cli.js'),
    provider,
    model: model || undefined,
    args: [
      '--session', sessionPath,
      '--no-tools',
      '--extension', EXTENSION_PATH,
    ],
    env: {
      // Tool extension will use these.
      NOTES_SERVER_URL: process.env.NOTES_SERVER_URL || 'http://127.0.0.1:8080',
      NOTES_TOKEN: process.env.NOTES_TOKEN || '',
      NOTES_PERSON: person,
    },
  });

  await rpc.start();

  const sc: SessionClient = {
    key,
    person,
    runtimeSessionId,
    client: rpc,
    provider,
    model: model || '',
    lastUsedAt: Date.now(),
    activeRequests: 0,
    idleTimer: null,
  };
  sessionClients.set(key, sc);
  scheduleSessionIdleCleanup(sc);
  return sc;
}

function markSessionClientInUse(sc: SessionClient): void {
  sc.activeRequests += 1;
  sc.lastUsedAt = Date.now();
  if (sc.idleTimer) {
    clearTimeout(sc.idleTimer);
    sc.idleTimer = null;
  }
}

function releaseSessionClient(sc: SessionClient): void {
  if (sc.activeRequests > 0) sc.activeRequests -= 1;
  sc.lastUsedAt = Date.now();
  if (sc.activeRequests === 0) {
    scheduleSessionIdleCleanup(sc);
  }
}

function buildSystemPromptMarker(systemPrompt: string): string {
  const b64 = Buffer.from(systemPrompt, 'utf8').toString('base64');
  return `[[notes_editor_system_prompt_base64:${b64}]]`;
}

async function handleMockChatStream(message: string, sessionId: string, runId: string, res: ServerResponse): Promise<void> {
  writeEvent(res, { type: 'status', message: 'gateway mode=mock' });
  writeEvent(res, { type: 'text', delta: `Gateway mock response: ${message}` });
  writeEvent(res, { type: 'done', session_id: sessionId, run_id: runId });
  res.end();
}

async function handlePiRpcChatStream(req: IncomingMessage, res: ServerResponse, payload: ChatRequest): Promise<void> {
  const person = (payload.person || '').trim();
  const userMessage = (payload.message || '').trim();
  if (!person) {
    sendJson(res, 400, { error: 'person is required' });
    return;
  }
  if (!userMessage) {
    sendJson(res, 400, { error: 'message is required' });
    return;
  }

  const runtimeSessionId = (payload.session_id || randomUUID()).trim() || randomUUID();
  const runId = randomUUID();

  res.statusCode = 200;
  res.setHeader('Content-Type', 'application/x-ndjson');
  res.setHeader('Cache-Control', 'no-cache');
  res.setHeader('Connection', 'keep-alive');

  writeEvent(res, { type: 'start', session_id: runtimeSessionId, run_id: runId });

  let sc: SessionClient;
  try {
    sc = await getOrStartSessionClient(person, runtimeSessionId);
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'failed to start pi runtime';
    writeEvent(res, { type: 'error', run_id: runId, message: msg });
    writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
    res.end();
    return;
  }

  markSessionClientInUse(sc);
  let contextWindow: number | undefined;
  try {
    const state = await sc.client.getState();
    const cw = Number(state?.model?.contextWindow || 0);
    if (Number.isFinite(cw) && cw > 0) {
      contextWindow = cw;
    }
  } catch {
    // Best-effort only.
  }

  writeEvent(res, {
    type: 'status',
    message: `gateway mode=pi_rpc provider=${sc.provider} model=${sc.model || 'default'}`,
    run_id: runId,
  });

  let cancelled = false;
  req.on('close', () => {
    cancelled = true;
    // Best effort abort only for this session-bound process.
    void sc.client.abort().catch(() => {});
  });

  const systemPrompt = (payload.system_prompt || '').trim();
  const promptText = systemPrompt
    ? `${buildSystemPromptMarker(systemPrompt)}\n${userMessage}`
    : userMessage;

  // Track whether we streamed any assistant text and capture runtime errors.
  // These variables must be in the same scope as the event handler.
  let sawAnyText = false;
  let lastRunError = '';
  let sawRunError = false;
  let lastEmittedError = '';

  const emitRunError = (message: string): void => {
    const normalized = String(message || '').trim();
    if (!normalized || normalized === lastEmittedError) return;
    lastEmittedError = normalized;
    sawRunError = true;
    writeEvent(res, { type: 'error', run_id: runId, message: normalized });
  };

  const unsubscribe = sc.client.onEvent((event: any) => {
    if (cancelled) return;

    switch (event?.type) {
      case 'message_update': {
        const ev = event.assistantMessageEvent;
        if (ev?.type === 'text_delta' && typeof ev.delta === 'string' && ev.delta.length > 0) {
          // Pi models sometimes start with a newline. Strip exactly one leading newline on the first text chunk
          // so UIs don't render an empty first line.
          let delta = ev.delta;
          if (!sawAnyText) {
            delta = delta.replace(/^\r?\n/, '');
          }
          if (!delta) break;
          sawAnyText = true;
          writeEvent(res, { type: 'text', run_id: runId, delta });
        }
        break;
      }
      case 'message_end': {
        const msg = event?.message;
        const usage = msg?.usage;
        if (usage && typeof usage === 'object') {
          const input = Number(usage.input || 0);
          const output = Number(usage.output || 0);
          const cacheRead = Number(usage.cacheRead || 0);
          const cacheWrite = Number(usage.cacheWrite || 0);
          const total = Number(usage.totalTokens || input + output + cacheRead + cacheWrite);
          const usagePayload: Record<string, unknown> = {
            input_tokens: input,
            output_tokens: output,
            cache_read_tokens: cacheRead,
            cache_write_tokens: cacheWrite,
            total_tokens: total,
          };
          if (contextWindow && contextWindow > 0) {
            usagePayload.context_window = contextWindow;
            usagePayload.remaining_tokens = Math.max(0, contextWindow - total);
          }
          writeEvent(res, { type: 'usage', run_id: runId, usage: usagePayload });
        }
        if (msg?.role === 'assistant' && typeof msg?.errorMessage === 'string' && msg.errorMessage.trim()) {
          lastRunError = msg.errorMessage.trim();
          emitRunError(lastRunError);
        }
        break;
      }
      case 'tool_execution_start': {
        writeEvent(res, { type: 'tool_call', run_id: runId, tool: event.toolName, args: event.args || {} });
        break;
      }
      case 'tool_execution_end': {
        const ok = !event.isError;
        const summary = ok ? `Tool ${event.toolName} executed` : 'Tool failed';
        writeEvent(res, { type: 'tool_result', run_id: runId, tool: event.toolName, ok, summary });
        break;
      }
      case 'extension_error': {
        lastRunError = String(event.error || 'extension error');
        emitRunError(lastRunError);
        break;
      }
      case 'auto_retry_start': {
        writeEvent(res, { type: 'status', run_id: runId, message: `Retrying after error: ${String(event.errorMessage || '').slice(0, 200)}` });
        break;
      }
      case 'auto_compaction_start': {
        writeEvent(res, { type: 'status', run_id: runId, message: `Compacting context (${event.reason || 'unknown'})...` });
        break;
      }
    }
  });

  try {
    await sc.client.prompt(promptText);
    await sc.client.waitForIdle(PI_TIMEOUT_MS);

    // If Pi ended with an error but didn't stream it (or we missed it), surface it.
    if (!cancelled && !sawAnyText && lastRunError && !sawRunError) {
      emitRunError(lastRunError);
    }

    writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
    res.end();
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'pi rpc failed';
    emitRunError(msg);
    writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
    res.end();
  } finally {
    unsubscribe();
    releaseSessionClient(sc);
  }
}

async function handleChatStream(req: IncomingMessage, res: ServerResponse): Promise<void> {
  let payload: ChatRequest;
  try {
    payload = await readJson<ChatRequest>(req);
  } catch {
    sendJson(res, 400, { error: 'invalid_json' });
    return;
  }

  const runtimeSessionId = (payload.session_id || randomUUID()).trim() || randomUUID();
  const runId = randomUUID();

  if (MODE === 'mock') {
    const message = (payload.message || '').trim();
    if (!message) {
      sendJson(res, 400, { error: 'message is required' });
      return;
    }
    res.statusCode = 200;
    res.setHeader('Content-Type', 'application/x-ndjson');
    res.setHeader('Cache-Control', 'no-cache');
    res.setHeader('Connection', 'keep-alive');
    writeEvent(res, { type: 'start', session_id: runtimeSessionId, run_id: runId });
    await handleMockChatStream(message, runtimeSessionId, runId, res);
    return;
  }

  if (MODE === 'pi_rpc') {
    await handlePiRpcChatStream(req, res, payload);
    return;
  }

  sendJson(res, 400, { error: `unsupported PI_GATEWAY_MODE: ${MODE}` });
}

const server = createServer(async (req, res) => {
  try {
    if (!req.url || !req.method) {
      sendJson(res, 400, { error: 'bad_request' });
      return;
    }

    if (req.method === 'GET' && req.url === '/health') {
      sendJson(res, 200, {
        ok: true,
        mode: MODE,
        pi: {
          provider: DEFAULT_PROVIDER,
          model: DEFAULT_MODEL || 'default',
          session_dir: SESSION_DIR,
          extension: EXTENSION_PATH,
          active_sessions: [...sessionClients.values()].map((sc) => ({
            key: sc.key,
            person: sc.person,
            session_id: sc.runtimeSessionId,
            provider: sc.provider,
            model: sc.model || 'default',
            active_requests: sc.activeRequests,
            idle_ms: Math.max(0, Date.now() - sc.lastUsedAt),
          })),
          session_idle_ttl_ms: SESSION_IDLE_TTL_MS,
        },
      });
      return;
    }

    if (req.method === 'POST' && req.url === '/v1/chat-stream') {
      await handleChatStream(req, res);
      return;
    }

    sendJson(res, 404, { error: 'not_found' });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'internal_error';
    if (res.headersSent) {
      // Best-effort: we may already be streaming NDJSON.
      try {
        res.write(`${JSON.stringify({ type: 'error', message })}\n`);
      } catch {
        // ignore
      }
      try {
        res.end();
      } catch {
        // ignore
      }
      return;
    }
    sendJson(res, 500, { error: message });
  }
});

server.listen(PORT, () => {
  // eslint-disable-next-line no-console
  console.log(`pi-gateway listening on :${PORT} (mode=${MODE})`);
});
