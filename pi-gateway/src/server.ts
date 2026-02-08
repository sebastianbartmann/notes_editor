import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { createServer, IncomingMessage, ServerResponse } from 'node:http';

type ChatRequest = {
  person?: string;
  session_id?: string;
  message?: string;
  max_tool_calls?: number;
};

type ToolResultPayload = {
  id?: string;
  ok?: boolean;
  content?: string;
  error?: string;
};

type PendingToolCall = {
  resolve: (payload: ToolResultPayload) => void;
  reject: (err: Error) => void;
  timer: NodeJS.Timeout;
};

type ClaudeDecision = {
  kind: 'final' | 'tool_call';
  text?: string;
  tool?: string;
  args?: Record<string, unknown>;
};

const PORT = Number(process.env.PI_GATEWAY_PORT || '4317');
const MODE = (process.env.PI_GATEWAY_MODE || 'claude_cli').trim();
const TOOL_RESULT_TIMEOUT_MS = Number(process.env.PI_GATEWAY_TOOL_RESULT_TIMEOUT_MS || '20000');
const CLAUDE_BIN = (process.env.PI_GATEWAY_CLAUDE_BIN || 'claude').trim();
const CLAUDE_MODEL = (process.env.PI_GATEWAY_CLAUDE_MODEL || '').trim();
const CLAUDE_DISABLE_TOOLS = (process.env.PI_GATEWAY_CLAUDE_DISABLE_TOOLS || 'true').trim().toLowerCase() !== 'false';
const DEFAULT_MAX_TOOL_CALLS = Number(process.env.PI_GATEWAY_DEFAULT_MAX_TOOL_CALLS || '20');

const pendingToolCalls = new Map<string, PendingToolCall>();
const CANONICAL_TOOLS = [
  'read_file',
  'write_file',
  'list_directory',
  'search_files',
  'web_search',
  'web_fetch',
  'linkedin_post',
  'linkedin_read_comments',
  'linkedin_post_comment',
  'linkedin_reply_comment',
];

const DECISION_SCHEMA = {
  type: 'object',
  properties: {
    kind: { type: 'string', enum: ['final', 'tool_call'] },
    text: { type: 'string' },
    tool: { type: 'string' },
    args: { type: 'object' },
  },
  required: ['kind'],
  additionalProperties: false,
};

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
  if (!raw) {
    return {} as T;
  }
  return JSON.parse(raw) as T;
}

function parseToolDirective(message: string): { name: string; args: Record<string, unknown> } | null {
  const marker = '[[tool:';
  const start = message.indexOf(marker);
  if (start < 0) {
    return null;
  }
  const end = message.indexOf(']]', start);
  if (end < 0) {
    return null;
  }

  const inner = message.slice(start + marker.length, end).trim();
  const firstSpace = inner.indexOf(' ');
  if (firstSpace < 0) {
    return { name: inner, args: {} };
  }

  const name = inner.slice(0, firstSpace).trim();
  const rawArgs = inner.slice(firstSpace + 1).trim();
  if (!name) {
    return null;
  }
  if (!rawArgs) {
    return { name, args: {} };
  }

  try {
    const parsed = JSON.parse(rawArgs);
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return { name, args: parsed as Record<string, unknown> };
    }
  } catch {
    return { name, args: { raw: rawArgs } };
  }

  return { name, args: {} };
}

function buildDecisionPrompt(userMessage: string, toolHistory: string[]): string {
  const intro = [
    'You are an agent runtime behind a tool bridge.',
    'Decide the NEXT step only, based on conversation and tool history.',
    'Return strict JSON matching schema (no markdown).',
    'If you need a tool call, return kind="tool_call" with tool and args.',
    'If you can answer, return kind="final" with text.',
    `Only use these tool names: ${CANONICAL_TOOLS.join(', ')}.`,
    'Do not use Claude Code native tools like Read, Bash, Edit, Write.',
    'Required args:',
    '- read_file: {"path":"relative/path.md"}',
    '- write_file: {"path":"relative/path.md","content":"..."}',
    '- list_directory: {"path":"."}',
    '- search_files: {"pattern":"text","path":"."}',
    '- web_search: {"query":"..."}',
    '- web_fetch: {"url":"https://..."}',
    '- linkedin_post: {"text":"..."}',
    '- linkedin_read_comments: {"post_urn":"..."}',
    '- linkedin_post_comment: {"post_urn":"...","text":"..."}',
    '- linkedin_reply_comment: {"post_urn":"...","parent_comment_urn":"...","text":"..."}',
    '',
    `User message: ${userMessage}`,
  ];

  if (toolHistory.length > 0) {
    intro.push('', 'Tool history:');
    for (const item of toolHistory) {
      intro.push(item);
    }
  }

  return intro.join('\n');
}

function normalizePathCandidate(pathValue: string): string {
  const p = pathValue.replaceAll('\\', '/').trim();
  const markers = ['/notes/', '/daily/', '/agent/', '/sleep_times.md'];
  for (const marker of markers) {
    const idx = p.indexOf(marker);
    if (idx >= 0) {
      return p.slice(idx + 1);
    }
  }
  if (p.startsWith('/')) {
    return p.slice(1);
  }
  return p;
}

function normalizeToolCall(tool: string, args: Record<string, unknown>): { tool: string; args: Record<string, unknown> } {
  if (CANONICAL_TOOLS.includes(tool)) {
    return { tool, args };
  }

  if (tool === 'Read') {
    const filePath = typeof args.file_path === 'string' ? args.file_path : '';
    if (filePath) {
      return {
        tool: 'read_file',
        args: { path: normalizePathCandidate(filePath) },
      };
    }
  }

  if (tool === 'Write') {
    const filePath = typeof args.file_path === 'string' ? args.file_path : '';
    const content = typeof args.content === 'string' ? args.content : '';
    if (filePath) {
      return {
        tool: 'write_file',
        args: { path: normalizePathCandidate(filePath), content },
      };
    }
  }

  if (tool === 'Bash') {
    const command = typeof args.command === 'string' ? args.command.trim() : '';
    const catMatch = command.match(/^cat\s+['"]?([^'"\s]+)['"]?$/);
    if (catMatch) {
      return {
        tool: 'read_file',
        args: { path: normalizePathCandidate(catMatch[1]) },
      };
    }
  }

  return { tool, args };
}

function parseClaudeJsonLines(chunk: string, lineBuffer: { value: string }, onEvent: (event: any) => void): void {
  lineBuffer.value += chunk;
  const lines = lineBuffer.value.split('\n');
  lineBuffer.value = lines.pop() ?? '';

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    try {
      onEvent(JSON.parse(trimmed));
    } catch {
      // ignore non-json noise
    }
  }
}

async function runClaudeDecision(_sessionId: string, prompt: string): Promise<ClaudeDecision> {
  const args = ['-p', prompt, '--output-format', 'stream-json', '--verbose', '--json-schema', JSON.stringify(DECISION_SCHEMA)];
  if (CLAUDE_MODEL) {
    args.push('--model', CLAUDE_MODEL);
  }
  if (CLAUDE_DISABLE_TOOLS) {
    args.push('--tools', '');
  }

  return await new Promise<ClaudeDecision>((resolve, reject) => {
    const child = spawn(CLAUDE_BIN, args, {
      env: process.env,
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    const lineBuffer = { value: '' };
    let resultText = '';
    let structuredOutput: ClaudeDecision | null = null;
    let stderr = '';
    let authError: string | null = null;
    const decisionTimeoutMs = Number(process.env.PI_GATEWAY_CLAUDE_DECISION_TIMEOUT_MS || '15000');
    const timeout = setTimeout(() => {
      child.kill('SIGTERM');
      reject(new Error(`Claude CLI decision timed out after ${decisionTimeoutMs}ms`));
    }, decisionTimeoutMs);

    child.stdout.setEncoding('utf8');
    child.stdout.on('data', (chunk: string) => {
      parseClaudeJsonLines(chunk, lineBuffer, (event) => {
        // Claude CLI can loop indefinitely when not authenticated; fail fast.
        if (typeof event?.error === 'string' && event.error === 'authentication_failed') {
          authError = 'Claude CLI not authenticated. Run: claude setup-token (or /login).';
          child.kill('SIGTERM');
          return;
        }
        if (event?.type === 'assistant' && event?.message?.content && Array.isArray(event.message.content)) {
          for (const block of event.message.content) {
            if (block?.type === 'text' && typeof block.text === 'string') {
              if (block.text.includes('Invalid API key') || block.text.includes('Please run /login')) {
                authError = 'Claude CLI not authenticated. Run: claude setup-token (or /login).';
                child.kill('SIGTERM');
                return;
              }
            }
          }
        }
        if (event?.type === 'result' && typeof event?.result === 'string') {
          resultText = event.result;
          if (event?.structured_output && typeof event.structured_output === 'object') {
            structuredOutput = event.structured_output as ClaudeDecision;
          }
        }
      });
    });

    child.stderr.setEncoding('utf8');
    child.stderr.on('data', (chunk: string) => {
      stderr += chunk;
    });

    child.on('error', (err) => reject(new Error(`failed to start Claude CLI: ${err.message}`)));

    child.on('close', (code) => {
      clearTimeout(timeout);
      if (authError) {
        reject(new Error(authError));
        return;
      }
      if (code !== 0 && !resultText) {
        reject(new Error(stderr.trim() || `Claude CLI exited with code ${code}`));
        return;
      }
      if (structuredOutput && (structuredOutput.kind === 'final' || structuredOutput.kind === 'tool_call')) {
        resolve(structuredOutput);
        return;
      }
      try {
        const parsed = JSON.parse(resultText) as ClaudeDecision;
        if (!parsed || (parsed.kind !== 'final' && parsed.kind !== 'tool_call')) {
          reject(new Error(`invalid Claude decision payload: ${resultText}`));
          return;
        }
        resolve(parsed);
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'invalid JSON';
        reject(new Error(`failed to parse Claude decision: ${msg}; raw=${resultText}`));
      }
    });
  });
}

async function waitForToolResult(runId: string, callId: string): Promise<ToolResultPayload> {
  const key = `${runId}:${callId}`;
  return await new Promise<ToolResultPayload>((resolve, reject) => {
    const timer = setTimeout(() => {
      pendingToolCalls.delete(key);
      reject(new Error('tool result timeout'));
    }, TOOL_RESULT_TIMEOUT_MS);

    pendingToolCalls.set(key, {
      resolve: (value) => {
        clearTimeout(timer);
        pendingToolCalls.delete(key);
        resolve(value);
      },
      reject: (err) => {
        clearTimeout(timer);
        pendingToolCalls.delete(key);
        reject(err);
      },
      timer,
    });
  });
}

async function streamClaudeCliWithToolBridge(
  req: IncomingMessage,
  res: ServerResponse,
  runtimeSessionId: string,
  runId: string,
  message: string,
  requestedMaxToolCalls?: number,
): Promise<void> {
  writeEvent(res, { type: 'status', message: `gateway mode=claude_cli bin=${CLAUDE_BIN}` });

  const toolHistory: string[] = [];
  const maxToolCalls = requestedMaxToolCalls && requestedMaxToolCalls > 0
    ? requestedMaxToolCalls
    : DEFAULT_MAX_TOOL_CALLS;

  let cancelled = false;
  req.on('close', () => {
    cancelled = true;
  });

  let toolCalls = 0;
  while (!cancelled) {
    if (toolCalls >= maxToolCalls) {
      writeEvent(res, { type: 'error', run_id: runId, message: `tool call limit exceeded (${maxToolCalls})` });
      writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
      res.end();
      return;
    }

    const prompt = buildDecisionPrompt(message, toolHistory);

    let decision: ClaudeDecision;
    try {
      decision = await runClaudeDecision(runtimeSessionId, prompt);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Claude CLI decision failed';
      writeEvent(res, { type: 'error', run_id: runId, message: msg });
      writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
      res.end();
      return;
    }

    if (decision.kind === 'final') {
      const finalText = (decision.text || '').trim();
      if (finalText) {
        writeEvent(res, { type: 'text', run_id: runId, delta: finalText });
      }
      writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
      res.end();
      return;
    }

    const rawTool = (decision.tool || '').trim();
    const rawArgs = decision.args && typeof decision.args === 'object' ? decision.args : {};
    if (!rawTool) {
      writeEvent(res, { type: 'error', run_id: runId, message: 'tool_call decision missing tool name' });
      writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
      res.end();
      return;
    }
    const normalized = normalizeToolCall(rawTool, rawArgs);
    const tool = normalized.tool;
    const args = normalized.args;
    if (!CANONICAL_TOOLS.includes(tool)) {
      toolCalls += 1;
      const unsupported = `unsupported tool ${tool}; use one of: ${CANONICAL_TOOLS.join(', ')}`;
      writeEvent(res, { type: 'status', run_id: runId, message: unsupported });
      toolHistory.push(`- ${tool} args=${JSON.stringify(args)} -> ERROR: ${unsupported}`);
      continue;
    }

    toolCalls += 1;
    const callId = randomUUID();
    writeEvent(res, { type: 'tool_call', run_id: runId, id: callId, tool, args });

    let result: ToolResultPayload;
    try {
      result = await waitForToolResult(runId, callId);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'tool result failure';
      writeEvent(res, { type: 'error', run_id: runId, message: msg });
      writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
      res.end();
      return;
    }

    if (result.ok) {
      writeEvent(res, { type: 'tool_result', run_id: runId, tool, ok: true, summary: `Tool ${tool} executed` });
      toolHistory.push(`- ${tool} args=${JSON.stringify(args)} -> OK: ${String(result.content || '').slice(0, 4000)}`);
    } else {
      writeEvent(res, { type: 'tool_result', run_id: runId, tool, ok: false, summary: String(result.error || 'tool failed') });
      toolHistory.push(`- ${tool} args=${JSON.stringify(args)} -> ERROR: ${String(result.error || 'tool failed')}`);
    }
  }

  writeEvent(res, { type: 'error', run_id: runId, message: 'stream cancelled' });
  writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
  res.end();
}

async function handleMockChatStream(message: string, runtimeSessionId: string, runId: string, res: ServerResponse): Promise<void> {
  writeEvent(res, { type: 'status', message: 'gateway mode=mock' });

  const tool = parseToolDirective(message);
  if (tool) {
    const callId = randomUUID();
    writeEvent(res, { type: 'tool_call', run_id: runId, id: callId, tool: tool.name, args: tool.args });

    try {
      const result = await waitForToolResult(runId, callId);
      if (result.ok) {
        writeEvent(res, { type: 'text', delta: `Tool ${tool.name} succeeded. ` });
        if (result.content) {
          writeEvent(res, { type: 'text', delta: String(result.content).slice(0, 4000) });
        }
      } else {
        writeEvent(res, { type: 'error', message: `Tool ${tool.name} failed: ${result.error || 'unknown error'}` });
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'tool result failure';
      writeEvent(res, { type: 'error', message: msg });
    }
  } else {
    writeEvent(res, { type: 'text', delta: `Gateway mock response: ${message}` });
  }

  writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
  res.end();
}

async function handleChatStream(req: IncomingMessage, res: ServerResponse): Promise<void> {
  let payload: ChatRequest;
  try {
    payload = await readJson<ChatRequest>(req);
  } catch {
    sendJson(res, 400, { error: 'invalid_json' });
    return;
  }

  const message = (payload.message || '').trim();
  if (!message) {
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

  if (MODE === 'mock') {
    await handleMockChatStream(message, runtimeSessionId, runId, res);
    return;
  }
  if (MODE === 'claude_cli') {
    await streamClaudeCliWithToolBridge(req, res, runtimeSessionId, runId, message, payload.max_tool_calls);
    return;
  }

  writeEvent(res, { type: 'error', message: `unsupported PI_GATEWAY_MODE: ${MODE}` });
  writeEvent(res, { type: 'done', session_id: runtimeSessionId, run_id: runId });
  res.end();
}

async function handleToolResult(req: IncomingMessage, res: ServerResponse, runId: string): Promise<void> {
  let payload: ToolResultPayload;
  try {
    payload = await readJson<ToolResultPayload>(req);
  } catch {
    sendJson(res, 400, { error: 'invalid_json' });
    return;
  }

  const callId = (payload.id || '').trim();
  if (!callId) {
    sendJson(res, 400, { error: 'id is required' });
    return;
  }

  const key = `${runId}:${callId}`;
  const pending = pendingToolCalls.get(key);
  if (!pending) {
    sendJson(res, 404, { error: 'pending tool call not found' });
    return;
  }

  pending.resolve(payload);
  res.statusCode = 204;
  res.end();
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
        claude_bin: CLAUDE_BIN,
        pending_tool_calls: pendingToolCalls.size,
      });
      return;
    }

    if (req.method === 'POST' && req.url === '/v1/chat-stream') {
      await handleChatStream(req, res);
      return;
    }

    const toolResultMatch = req.url.match(/^\/v1\/runs\/([^/]+)\/tool-result$/);
    if (req.method === 'POST' && toolResultMatch) {
      await handleToolResult(req, res, toolResultMatch[1]);
      return;
    }

    sendJson(res, 404, { error: 'not_found' });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'internal_error';
    sendJson(res, 500, { error: message });
  }
});

server.listen(PORT, () => {
  // eslint-disable-next-line no-console
  console.log(`pi-gateway listening on :${PORT} (mode=${MODE})`);
});
