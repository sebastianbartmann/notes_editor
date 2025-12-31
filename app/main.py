from fastapi import FastAPI, Request, HTTPException, Form
from fastapi.responses import HTMLResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from datetime import datetime
from pathlib import Path
import re

from app.renderers.pinned import render_with_pinned_buttons
from app.services.git_sync import git_pull, git_commit_and_push
from app.services.vault_store import VAULT_ROOT, write_entry, append_entry, resolve_path

app = FastAPI()

app.mount("/static", StaticFiles(directory="app/static"), name="static")
templates = Jinja2Templates(directory="app/templates")

DAILY_DIR = VAULT_ROOT / "daily"


def get_today_filename() -> str:
    return datetime.now().strftime("%Y-%m-%d.md")


def get_today_filepath() -> Path:
    return DAILY_DIR / get_today_filename()


def get_most_recent_note() -> Path | None:
    """Find the most recent daily note file (excluding today's)"""
    if not DAILY_DIR.exists():
        return None

    # Get all markdown files that match the date pattern YYYY-MM-DD.md
    note_files = sorted(DAILY_DIR.glob("????-??-??.md"), reverse=True)

    today_file = get_today_filename()
    for note_file in note_files:
        if note_file.name != today_file:
            return note_file

    return None


def extract_section(filepath: Path, section_name: str) -> str:
    """Extract content from a ## section of a note file"""
    if not filepath.exists():
        return ""

    content = filepath.read_text()

    # Find the section
    section_match = re.search(rf'^## {section_name}\s*$', content, re.MULTILINE | re.IGNORECASE)
    if not section_match:
        return ""

    # Get content starting from section header
    start_pos = section_match.end()
    remaining_content = content[start_pos:]

    # Find the next ## header or end of file
    next_section_match = re.search(r'^## ', remaining_content, re.MULTILINE)
    if next_section_match:
        section_content = remaining_content[:next_section_match.start()]
    else:
        section_content = remaining_content

    return section_content.strip()


def extract_incomplete_todos(filepath: Path) -> str:
    """Extract incomplete todos from the ## todos section of a note file"""
    todos_content = extract_section(filepath, "todos")
    if not todos_content:
        return ""

    # Filter out completed todos (lines with - [x] or - [X], with any indentation)
    lines = todos_content.split('\n')
    incomplete_lines = []

    for line in lines:
        # Skip completed todo items (with any amount of leading whitespace)
        if re.match(r'^\s*-\s*\[x\]', line, re.IGNORECASE):
            continue
        incomplete_lines.append(line)

    # Join and clean up the result
    result = '\n'.join(incomplete_lines).strip()

    return result if result else ""


def extract_pinned_notes(filepath: Path) -> str:
    """Extract pinned notes (### entries with <pinned> marker) from custom notes section"""
    custom_notes = extract_section(filepath, "custom notes")
    if not custom_notes:
        return ""

    # Find all ### sections that have <pinned> in the header
    pinned_entries = []
    lines = custom_notes.split('\n')
    current_entry = []
    is_pinned = False

    for line in lines:
        if line.startswith('### '):
            # Save previous entry if it was pinned
            if is_pinned and current_entry:
                pinned_entries.append('\n'.join(current_entry))
            # Start new entry
            current_entry = [line]
            is_pinned = '<pinned>' in line.lower()
        elif current_entry:
            current_entry.append(line)

    # Don't forget last entry
    if is_pinned and current_entry:
        pinned_entries.append('\n'.join(current_entry))

    return '\n\n'.join(pinned_entries) if pinned_entries else ""


def ensure_file_exists(filepath: Path) -> None:
    """Create file with header if it doesn't exist"""
    if not filepath.exists():
        DAILY_DIR.mkdir(parents=True, exist_ok=True)
        date_str = datetime.now().strftime("%Y-%m-%d")

        # Start with header
        content = f"# daily {date_str}\n\n"

        # Try to get incomplete todos and pinned notes from previous note
        previous_note = get_most_recent_note()
        if previous_note:
            incomplete_todos = extract_incomplete_todos(previous_note)
            if incomplete_todos:
                content += f"## todos\n\n{incomplete_todos}\n\n"

            pinned_notes = extract_pinned_notes(previous_note)
            if pinned_notes:
                content += f"## custom notes\n\n{pinned_notes}\n\n"

        # Write the file
        filepath.write_text(content)

        # Commit and push the new file
        git_commit_and_push(f"Create daily note {date_str}")


@app.get("/", response_class=HTMLResponse)
async def root(request: Request):
    # Pull latest changes from remote
    git_pull()

    today = datetime.now().strftime("%Y-%m-%d")
    filepath = get_today_filepath()
    ensure_file_exists(filepath)
    content = filepath.read_text()
    relative_path = str(filepath.relative_to(VAULT_ROOT))
    note_html = render_with_pinned_buttons(content, relative_path)
    return templates.TemplateResponse(
        "editor.html",
        {
            "request": request,
            "today": today,
            "content": content,
            "note_html": note_html,
            "current_path": relative_path,
        }
    )


@app.post("/api/append")
async def append_note(content: str = Form(...), pinned: str = Form(None)):
    if not content.strip():
        raise HTTPException(status_code=400, detail="Content cannot be empty")

    filepath = get_today_filepath()
    ensure_file_exists(filepath)

    # Get current time
    time_str = datetime.now().strftime("%H:%M")

    # Read current content
    current_content = filepath.read_text()

    # Determine header based on pinned checkbox
    is_pinned = pinned == "on"
    header = f"### {time_str} <pinned>" if is_pinned else f"### {time_str}"

    # Check if custom notes section exists
    if re.search(r'^## custom notes\s*$', current_content, re.MULTILINE | re.IGNORECASE):
        # Append under existing section
        append_text = f"\n{header}\n\n{content.strip()}\n"
    else:
        # Create section and add first entry
        append_text = f"\n## custom notes\n\n{header}\n\n{content.strip()}\n"

    append_entry(str(filepath.relative_to(VAULT_ROOT)), append_text)

    # Git operations
    success, msg = git_commit_and_push(f"Append {'pinned ' if is_pinned else ''}note at {time_str}")

    return {
        "success": success,
        "message": "Content appended successfully" if success else msg
    }


@app.post("/api/clear-pinned")
async def clear_pinned():
    filepath = get_today_filepath()
    if not filepath.exists():
        return {"success": True, "message": "No notes to clear"}

    content = filepath.read_text()

    # Remove <pinned> markers from ### headers
    updated_content = re.sub(r'^(### \d{2}:\d{2})\s*<pinned>', r'\1', content, flags=re.MULTILINE | re.IGNORECASE)

    if content == updated_content:
        return {"success": True, "message": "No pinned notes to clear"}

    filepath.write_text(updated_content)

    success, msg = git_commit_and_push("Clear pinned markers")

    return {
        "success": success,
        "message": "Pinned markers cleared" if success else msg
    }


@app.post("/api/save")
async def save_note(content: str = Form(...)):
    filepath = get_today_filepath()
    ensure_file_exists(filepath)

    write_entry(str(filepath.relative_to(VAULT_ROOT)), content)

    # Git operations
    success, msg = git_commit_and_push("Update note")

    return {
        "success": success,
        "message": "Note saved successfully" if success else msg
    }


@app.post("/api/files/unpin")
async def unpin_entry(path: str = Form(...), line: int = Form(...)):
    if line < 1:
        raise HTTPException(status_code=400, detail="Invalid line number")

    filepath = resolve_path(path)
    if not filepath.exists():
        raise HTTPException(status_code=404, detail="File not found")

    content = filepath.read_text()
    lines = content.splitlines()
    if line > len(lines):
        raise HTTPException(status_code=400, detail="Line out of range")

    target = lines[line - 1]
    if not re.match(r"^###\s+.*<pinned>.*$", target, re.IGNORECASE):
        return {"success": True, "message": "Entry already unpinned"}

    updated_line = re.sub(r"\s*<pinned>\s*", "", target, flags=re.IGNORECASE).rstrip()
    lines[line - 1] = updated_line

    updated_content = "\n".join(lines)
    if content.endswith("\n"):
        updated_content += "\n"

    filepath.write_text(updated_content)
    success, msg = git_commit_and_push("Unpin entry")

    return {"success": success, "message": "Entry unpinned" if success else msg}


@app.get("/files", response_class=HTMLResponse)
async def files_page(request: Request):
    return templates.TemplateResponse("files.html", {"request": request})
