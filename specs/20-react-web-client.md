# React Web Client

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-27
> Supersedes: [07-web-interface](./07-web-interface.md)

## Overview

The React web client provides a browser-based interface for the Notes Editor application. It replaces the previous server-rendered HTMX/Jinja2 implementation with a modern single-page application.

## Technology Stack

- **Build Tool:** Vite
- **Language:** TypeScript
- **Framework:** React 18+
- **Styling:** CSS modules (or Tailwind CSS)
- **HTTP Client:** Fetch API
- **State:** React Context + hooks

## Project Structure

```
clients/web/
├── public/
│   └── favicon.ico
│
├── src/
│   ├── api/
│   │   ├── client.ts            # HTTP client with auth
│   │   ├── daily.ts             # Daily note API calls
│   │   ├── files.ts             # File operations API
│   │   ├── todos.ts             # Todo API calls
│   │   ├── sleep.ts             # Sleep tracking API
│   │   ├── claude.ts            # Claude chat API + streaming
│   │   └── types.ts             # API response types
│   │
│   ├── components/
│   │   ├── NoteView/
│   │   │   ├── NoteView.tsx     # Markdown note renderer
│   │   │   ├── NoteView.module.css
│   │   │   └── TaskLine.tsx     # Interactive task checkbox
│   │   ├── FileTree/
│   │   │   ├── FileTree.tsx     # Directory tree browser
│   │   │   └── FileTree.module.css
│   │   ├── Editor/
│   │   │   ├── Editor.tsx       # Markdown text editor
│   │   │   └── Editor.module.css
│   │   ├── Chat/
│   │   │   ├── ChatWindow.tsx   # Claude chat interface
│   │   │   ├── ChatMessage.tsx  # Message bubble
│   │   │   └── StreamingText.tsx
│   │   ├── SleepForm/
│   │   │   └── SleepForm.tsx    # Sleep entry form
│   │   ├── NoisePlayer/
│   │   │   └── NoisePlayer.tsx  # Web Audio noise generator
│   │   └── Layout/
│   │       ├── Header.tsx
│   │       ├── Navigation.tsx
│   │       └── Layout.tsx
│   │
│   ├── pages/
│   │   ├── DailyPage.tsx        # Today's note view/edit
│   │   ├── FilesPage.tsx        # File browser
│   │   ├── FilePage.tsx         # Single file view/edit
│   │   ├── SleepPage.tsx        # Sleep tracking
│   │   ├── ClaudePage.tsx       # AI chat
│   │   ├── NoisePage.tsx        # Noise generator
│   │   └── SettingsPage.tsx     # Settings management
│   │
│   ├── hooks/
│   │   ├── useAuth.ts           # Authentication state
│   │   ├── usePerson.ts         # Person context
│   │   ├── useTheme.ts          # Theme (dark/light)
│   │   ├── useApi.ts            # API call wrapper
│   │   └── useClaudeStream.ts   # Streaming chat hook
│   │
│   ├── context/
│   │   ├── AuthContext.tsx      # Auth provider
│   │   ├── PersonContext.tsx    # Person provider
│   │   └── ThemeContext.tsx     # Theme provider
│   │
│   ├── App.tsx                  # Root component with routing
│   ├── main.tsx                 # Entry point
│   └── index.css                # Global styles, CSS variables
│
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## API Client

Centralized HTTP client with authentication:

```typescript
// src/api/client.ts
const API_BASE = import.meta.env.VITE_API_URL || '';

interface RequestOptions {
  method?: 'GET' | 'POST';
  body?: Record<string, string>;
}

export async function apiRequest<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const token = localStorage.getItem('notes_token');
  const person = localStorage.getItem('notes_person');

  const headers: Record<string, string> = {
    'Authorization': `Bearer ${token}`,
    'X-Notes-Person': person || '',
  };

  let fetchOptions: RequestInit = {
    method: options.method || 'GET',
    headers,
  };

  if (options.body) {
    headers['Content-Type'] = 'application/json';
    fetchOptions.body = JSON.stringify(options.body);
  }

  const response = await fetch(`${API_BASE}${endpoint}`, fetchOptions);

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail || 'Request failed');
  }

  return response.json();
}
```

## Pages

### DailyPage

Displays today's daily note with:
- Rendered markdown view (NoteView component)
- Edit mode toggle (Editor component)
- Task checkboxes (interactive toggle via API)
- Append note form

```typescript
// src/pages/DailyPage.tsx
export function DailyPage() {
  const [note, setNote] = useState<DailyNote | null>(null);
  const [editing, setEditing] = useState(false);

  useEffect(() => {
    fetchDaily().then(setNote);
  }, []);

  if (!note) return <Loading />;

  return (
    <div className={styles.daily}>
      <Header title={`Daily ${note.date}`} />

      {editing ? (
        <Editor
          content={note.content}
          onSave={async (content) => {
            await saveDaily(content);
            setNote({ ...note, content });
            setEditing(false);
          }}
          onCancel={() => setEditing(false)}
        />
      ) : (
        <>
          <NoteView
            content={note.content}
            path={note.path}
            onTaskToggle={handleTaskToggle}
          />
          <button onClick={() => setEditing(true)}>Edit</button>
        </>
      )}

      <AppendForm onAppend={handleAppend} />
    </div>
  );
}
```

### FilesPage

File browser with:
- Directory tree (lazy-loaded)
- File selection
- Create/delete operations

### ClaudePage

AI chat with:
- Message history display
- Streaming response rendering
- Session management
- Tool use visualization

```typescript
// src/hooks/useClaudeStream.ts
export function useClaudeStream() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [streaming, setStreaming] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);

  const sendMessage = async (text: string) => {
    setStreaming(true);
    setMessages(prev => [...prev, { role: 'user', content: text }]);

    const response = await fetch('/api/claude/chat-stream', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'X-Notes-Person': person,
        'Content-Type': 'application/json',
        'Accept': 'application/x-ndjson',
      },
      body: JSON.stringify({ message: text, session_id: sessionId }),
    });

    const reader = response.body?.getReader();
    const decoder = new TextDecoder();
    let assistantMessage = '';

    while (reader) {
      const { done, value } = await reader.read();
      if (done) break;

      const lines = decoder.decode(value).split('\n');
      for (const line of lines) {
        if (!line.trim()) continue;
        const event = JSON.parse(line);

        switch (event.type) {
          case 'text':
            assistantMessage += event.delta;
            updateLastMessage(assistantMessage);
            break;
          case 'session':
            setSessionId(event.session_id);
            break;
          case 'error':
            showError(event.message);
            break;
        }
      }
    }

    setStreaming(false);
  };

  return { messages, streaming, sendMessage, clearSession };
}
```

## Components

### NoteView

Line-by-line markdown renderer matching spec 11:

```typescript
// src/components/NoteView/NoteView.tsx
interface NoteViewProps {
  content: string;
  path: string;
  onTaskToggle: (line: number) => void;
  onUnpin?: (line: number) => void;
}

export function NoteView({ content, path, onTaskToggle, onUnpin }: NoteViewProps) {
  const lines = content.split('\n');

  return (
    <div className={styles.noteView}>
      {lines.map((line, index) => (
        <NoteLine
          key={index}
          line={line}
          lineNumber={index + 1}
          onTaskToggle={onTaskToggle}
          onUnpin={onUnpin}
        />
      ))}
    </div>
  );
}

function NoteLine({ line, lineNumber, onTaskToggle, onUnpin }) {
  // Heading detection
  if (line.startsWith('#### ')) return <h4>{line.slice(5)}</h4>;
  if (line.startsWith('### ')) {
    const isPinned = line.includes('<pinned>');
    return (
      <h3>
        {line.slice(4).replace('<pinned>', '')}
        {isPinned && onUnpin && (
          <button onClick={() => onUnpin(lineNumber)}>Unpin</button>
        )}
      </h3>
    );
  }
  if (line.startsWith('## ')) return <h2>{line.slice(3)}</h2>;
  if (line.startsWith('# ')) return <h1>{line.slice(2)}</h1>;

  // Task detection
  const taskMatch = line.match(/^- \[([ x])\] (.*)$/);
  if (taskMatch) {
    const checked = taskMatch[1] === 'x';
    const text = taskMatch[2];
    return (
      <div className={styles.task}>
        <input
          type="checkbox"
          checked={checked}
          onChange={() => onTaskToggle(lineNumber)}
        />
        <span className={checked ? styles.completed : ''}>{text}</span>
      </div>
    );
  }

  // Regular line
  return <p>{line || '\u00A0'}</p>;
}
```

### NoisePlayer

Web Audio noise generator (from spec 08):

```typescript
// src/components/NoisePlayer/NoisePlayer.tsx
export function NoisePlayer() {
  const [playing, setPlaying] = useState(false);
  const audioContextRef = useRef<AudioContext | null>(null);

  const start = () => {
    const ctx = new AudioContext();
    audioContextRef.current = ctx;

    // White noise buffer
    const bufferSize = ctx.sampleRate * 2;
    const buffer = ctx.createBuffer(1, bufferSize, ctx.sampleRate);
    const data = buffer.getChannelData(0);
    for (let i = 0; i < bufferSize; i++) {
      data[i] = Math.random() * 2 - 1;
    }

    const source = ctx.createBufferSource();
    source.buffer = buffer;
    source.loop = true;
    source.connect(ctx.destination);
    source.start();

    setPlaying(true);
  };

  const stop = () => {
    audioContextRef.current?.close();
    setPlaying(false);
  };

  return (
    <div className={styles.noisePlayer}>
      <button onClick={playing ? stop : start}>
        {playing ? 'Stop' : 'Play'} White Noise
      </button>
    </div>
  );
}
```

## Context Providers

### AuthContext

```typescript
// src/context/AuthContext.tsx
interface AuthState {
  token: string | null;
  isAuthenticated: boolean;
  login: (token: string) => void;
  logout: () => void;
}

export const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState(() => localStorage.getItem('notes_token'));

  const login = (newToken: string) => {
    localStorage.setItem('notes_token', newToken);
    setToken(newToken);
  };

  const logout = () => {
    localStorage.removeItem('notes_token');
    setToken(null);
  };

  return (
    <AuthContext.Provider value={{
      token,
      isAuthenticated: !!token,
      login,
      logout,
    }}>
      {children}
    </AuthContext.Provider>
  );
}
```

### ThemeContext

```typescript
// src/context/ThemeContext.tsx
type Theme = 'dark' | 'light';

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    const stored = localStorage.getItem('notes_theme');
    return (stored as Theme) || 'dark';
  });

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('notes_theme', theme);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}
```

## Routing

```typescript
// src/App.tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';

export function App() {
  return (
    <AuthProvider>
      <PersonProvider>
        <ThemeProvider>
          <BrowserRouter>
            <Layout>
              <Routes>
                <Route path="/" element={<Navigate to="/daily" />} />
                <Route path="/daily" element={<DailyPage />} />
                <Route path="/files" element={<FilesPage />} />
                <Route path="/files/:path" element={<FilePage />} />
                <Route path="/sleep" element={<SleepPage />} />
                <Route path="/claude" element={<ClaudePage />} />
                <Route path="/noise" element={<NoisePage />} />
                <Route path="/settings" element={<SettingsPage />} />
              </Routes>
            </Layout>
          </BrowserRouter>
        </ThemeProvider>
      </PersonProvider>
    </AuthProvider>
  );
}
```

## Styling

CSS variables for theming (see spec 12):

```css
/* src/index.css */
:root {
  /* Dark theme (default) */
  --bg-primary: #0F1012;
  --bg-secondary: #1A1B1E;
  --text-primary: #E8E9EA;
  --text-secondary: #9CA3AF;
  --accent: #D9832B;
  --accent-muted: #B36D24;
}

[data-theme="light"] {
  --bg-primary: #E9F7F7;
  --bg-secondary: #FFFFFF;
  --text-primary: #1A1B1E;
  --text-secondary: #4B5563;
  --accent: #3AA7A3;
  --accent-muted: #2D8B87;
}

body {
  background-color: var(--bg-primary);
  color: var(--text-primary);
  font-family: system-ui, sans-serif;
}
```

## Build Configuration

```typescript
// vite.config.ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    outDir: 'dist',
  },
});
```

## Development

```bash
# Install dependencies
npm install

# Development server (with API proxy)
npm run dev

# Production build
npm run build

# Preview production build
npm run preview
```

## Related Specifications

- [00-project-overview](./00-project-overview.md) - Overall architecture
- [01-rest-api-contract](./01-rest-api-contract.md) - API endpoints
- [11-note-rendering-markdown](./11-note-rendering-markdown.md) - Markdown parsing rules
- [12-theming-styling-system](./12-theming-styling-system.md) - Color and spacing system
- [13-claude-streaming-client](./13-claude-streaming-client.md) - Streaming chat protocol
