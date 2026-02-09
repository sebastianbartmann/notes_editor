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
    return { systemPrompt: lastSystemPrompt };
  });

  const toolDefs = [
    {
      name: 'read_file',
      label: 'Read file',
      description: 'Read a file from the notes vault (person-scoped).',
      parameters: Type.Object({ path: Type.String() }),
    },
    {
      name: 'write_file',
      label: 'Write file',
      description: 'Write a file to the notes vault (person-scoped).',
      parameters: Type.Object({ path: Type.String(), content: Type.String() }),
    },
    {
      name: 'list_directory',
      label: 'List directory',
      description: 'List directory entries in the notes vault (person-scoped).',
      parameters: Type.Object({ path: Type.Optional(Type.String()) }),
    },
    {
      name: 'search_files',
      label: 'Search files',
      description: 'Search files in the notes vault (person-scoped).',
      parameters: Type.Object({ pattern: Type.String(), path: Type.Optional(Type.String()) }),
    },
    {
      name: 'web_search',
      label: 'Web search',
      description: 'Search the web (server-side).',
      parameters: Type.Object({ query: Type.String() }),
    },
    {
      name: 'web_fetch',
      label: 'Web fetch',
      description: 'Fetch a URL (server-side).',
      parameters: Type.Object({ url: Type.String() }),
    },
    {
      name: 'linkedin_post',
      label: 'LinkedIn post',
      description: 'Create a LinkedIn post (server-side).',
      parameters: Type.Object({ text: Type.String() }),
    },
    {
      name: 'linkedin_read_comments',
      label: 'LinkedIn read comments',
      description: 'Read comments for a LinkedIn post (server-side).',
      parameters: Type.Object({ post_urn: Type.String() }),
    },
    {
      name: 'linkedin_post_comment',
      label: 'LinkedIn post comment',
      description: 'Post a comment on a LinkedIn post (server-side).',
      parameters: Type.Object({ post_urn: Type.String(), text: Type.String() }),
    },
    {
      name: 'linkedin_reply_comment',
      label: 'LinkedIn reply comment',
      description: 'Reply to a LinkedIn comment (server-side).',
      parameters: Type.Object({ post_urn: Type.String(), parent_comment_urn: Type.String(), text: Type.String() }),
    },
  ] as const;

  for (const def of toolDefs) {
    pi.registerTool({
      name: def.name,
      label: def.label,
      description: def.description,
      parameters: def.parameters,
      async execute(_toolCallId, params, _signal, _onUpdate, _ctx) {
        const content = await callTool(def.name, params as unknown as Record<string, unknown>);
        return { content: [{ type: 'text', text: content }], details: {} };
      },
    });
  }
}

