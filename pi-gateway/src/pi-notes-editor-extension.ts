import type { ExtensionAPI } from '@mariozechner/pi-coding-agent';
import { Type } from '@sinclair/typebox';

type ToolExecResponse = {
  ok: boolean;
  content?: string;
  error?: string;
};

const SYSTEM_PROMPT_MARKER_PREFIX = '[[notes_editor_system_prompt_base64:';

function decodeSystemPromptFromMarker(text: string): { prompt: string; stripped: string } | null {
  const trimmed = text.trimStart();
  if (!trimmed.startsWith(SYSTEM_PROMPT_MARKER_PREFIX)) return null;
  const end = trimmed.indexOf(']]');
  if (end < 0) return null;

  const marker = trimmed.slice(0, end + 2);
  const b64 = marker.slice(SYSTEM_PROMPT_MARKER_PREFIX.length, -2);
  try {
    const prompt = Buffer.from(b64, 'base64').toString('utf8');
    const stripped = trimmed.slice(end + 2).replace(/^\s+/, '');
    return { prompt, stripped };
  } catch {
    return null;
  }
}

async function callTool(tool: string, args: Record<string, unknown>): Promise<string> {
  const baseUrl = (process.env.NOTES_SERVER_URL || 'http://127.0.0.1:8080').replace(/\/+$/, '');
  const token = (process.env.NOTES_TOKEN || '').trim();
  const person = (process.env.NOTES_PERSON || '').trim();
  if (!person) {
    throw new Error('NOTES_PERSON is required for tool execution');
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
    'X-Notes-Person': person,
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const resp = await fetch(`${baseUrl}/api/agent/tools/execute`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ tool, args }),
  });
  const raw = await resp.text();
  if (!resp.ok) {
    throw new Error(`tool execute failed: HTTP ${resp.status}: ${raw}`);
  }

  const payload = JSON.parse(raw) as ToolExecResponse;
  if (!payload.ok) {
    throw new Error(payload.error || 'tool failed');
  }
  return payload.content || '';
}

export default function (pi: ExtensionAPI) {
  let lastSystemPrompt: string | null = null;

  // Allow the gateway to set/refresh the per-person system prompt without
  // command-line flags by injecting a hidden marker into the prompt.
  pi.on('input', async (event) => {
    if (event.source !== 'rpc') return { action: 'continue' };
    const parsed = decodeSystemPromptFromMarker(event.text);
    if (!parsed) return { action: 'continue' };
    lastSystemPrompt = parsed.prompt;
    return { action: 'transform', text: parsed.stripped, images: event.images };
  });

  pi.on('before_agent_start', async (event) => {
    if (!lastSystemPrompt) return;
    // Anthropic OAuth tokens in Pi run in "Claude Code identity" mode internally.
    // Custom tool names can cause Anthropic to reject the token ("only authorized for Claude Code").
    // Therefore we expose only Claude Code tool names (read/write/grep/glob/webfetch/websearch) and
    // map them to Notes Editor server tools.
    const toolHint = [
      'Notes Editor tools:',
      '- read: read a file from the person vault (path is vault-relative)',
      '- write: write a file in the person vault (path is vault-relative)',
      '- grep: search text in the person vault (path optional, vault-relative)',
      '- glob: find files by glob pattern in the person vault (path optional, vault-relative)',
      '- webfetch: fetch a URL (server-side)',
      '- websearch: search the web (server-side)',
    ].join('\n');
    return { systemPrompt: `${toolHint}\n\n${lastSystemPrompt}` };
  });

  // Tool names intentionally match Claude Code's canonical tools (case-insensitive).
  // Pi's Anthropic OAuth flow relies on this.
  pi.registerTool({
    name: 'read',
    label: 'read',
    description: 'Read a file from the person vault (path is vault-relative).',
    parameters: Type.Object({
      path: Type.String({ description: 'Vault-relative path to read' }),
      offset: Type.Optional(Type.Number({ description: 'Line number to start reading from (1-indexed)' })),
      limit: Type.Optional(Type.Number({ description: 'Maximum number of lines to read' })),
    }),
    async execute(_toolCallId, params) {
      const content = await callTool('read_file', params as unknown as Record<string, unknown>);
      return { content: [{ type: 'text', text: content }], details: {} };
    },
  });

  pi.registerTool({
    name: 'write',
    label: 'write',
    description: 'Write a file to the person vault (path is vault-relative).',
    parameters: Type.Object({
      path: Type.String({ description: 'Vault-relative path to write' }),
      content: Type.String({ description: 'Content to write' }),
    }),
    async execute(_toolCallId, params) {
      const content = await callTool('write_file', params as unknown as Record<string, unknown>);
      return { content: [{ type: 'text', text: content }], details: {} };
    },
  });

  pi.registerTool({
    name: 'grep',
    label: 'grep',
    description: 'Search files in the person vault for a pattern.',
    parameters: Type.Object({
      pattern: Type.String({ description: 'Search pattern (regex or literal string)' }),
      path: Type.Optional(Type.String({ description: 'Vault-relative directory or file to search' })),
    }),
    async execute(_toolCallId, params) {
      const { pattern, path } = params as any;
      const content = await callTool('search_files', { pattern, path } as Record<string, unknown>);
      return { content: [{ type: 'text', text: content }], details: {} };
    },
  });

  pi.registerTool({
    name: 'glob',
    label: 'glob',
    description: 'Find files in the person vault by glob pattern.',
    parameters: Type.Object({
      pattern: Type.String({ description: "Glob pattern, e.g. '*.md' or '**/*.prompt.md'" }),
      path: Type.Optional(Type.String({ description: 'Vault-relative directory to search in (default: vault root)' })),
      limit: Type.Optional(Type.Number({ description: 'Maximum number of results (default: 1000)' })),
    }),
    async execute(_toolCallId, params) {
      const { pattern, path, limit } = params as any;
      const content = await callTool('glob_files', { pattern, path, limit } as Record<string, unknown>);
      return { content: [{ type: 'text', text: content }], details: {} };
    },
  });

  pi.registerTool({
    name: 'websearch',
    label: 'websearch',
    description: 'Search the web (server-side).',
    parameters: Type.Object({ query: Type.String() }),
    async execute(_toolCallId, params) {
      const content = await callTool('web_search', params as unknown as Record<string, unknown>);
      return { content: [{ type: 'text', text: content }], details: {} };
    },
  });

  pi.registerTool({
    name: 'webfetch',
    label: 'webfetch',
    description: 'Fetch a URL (server-side).',
    parameters: Type.Object({ url: Type.String() }),
    async execute(_toolCallId, params) {
      const content = await callTool('web_fetch', params as unknown as Record<string, unknown>);
      return { content: [{ type: 'text', text: content }], details: {} };
    },
  });
}
