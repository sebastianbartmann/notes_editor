# Daily Notes Editor

A lightweight web app for quickly editing daily markdown notes with automatic git sync.

## Features

- **Daily Notes**: Automatically creates and edits date-based markdown files
- **Quick Append**: Add timestamped entries throughout the day
- **Git Sync**: Auto-commits and pushes changes, pulls latest on page load
- **Simple UI**: Clean, minimal interface using Pico CSS and HTMX

## Setup

### Requirements

- Python 3.12+
- Git repository for storing notes
- [uv](https://github.com/astral-sh/uv) (recommended) or pip

### Installation

1. Clone the repository:
```bash
git clone https://github.com/sebastianbartmann/notes_editor
cd notes_editor
```

2. Install dependencies:
```bash
uv sync
```

3. Configure git identity (required for auto-commits):
```bash
git config --global user.email "your@email.com"
git config --global user.name "Your Name"
```

4. Clone the notes repository:
```bash
git clone dev@dev:/home/dev/git/notes.git ~/notes
```

### Running

**Development:**
```bash
make run
```

**Production (systemd service):**

Create `/etc/systemd/system/notes-editor.service`:
```ini
[Unit]
Description=Daily Notes Editor
After=network.target

[Service]
Type=simple
User=YOUR_USERNAME
WorkingDirectory=/path/to/notes_editor
Environment="PATH=/home/YOUR_USERNAME/.local/bin:/usr/local/bin:/usr/bin:/bin"
ExecStart=/path/to/notes_editor/.venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable notes-editor
sudo systemctl start notes-editor
sudo systemctl status notes-editor
```

**Alternative (screen/tmux):**
```bash
screen -S notes
cd notes_editor
source .venv/bin/activate
uvicorn app.main:app --host 0.0.0.0 --port 8000
# Press Ctrl+A then D to detach
```

Service management:
- `sudo systemctl status notes-editor` - check status
- `sudo systemctl restart notes-editor` - restart after code changes
- `sudo systemctl stop notes-editor` - stop service

## License

MIT
