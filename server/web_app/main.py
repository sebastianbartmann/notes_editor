from fastapi import FastAPI, Request, HTTPException, Form, Depends
from fastapi.responses import HTMLResponse, RedirectResponse, StreamingResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from datetime import datetime
import asyncio
from pathlib import Path
from dotenv import load_dotenv
import json
import os
import re
import secrets
import subprocess

from .renderers.pinned import render_with_pinned_buttons
from .renderers.file_tree import render_tree
from .services.git_sync import git_pull, git_commit_and_push
from .services.vault_store import VAULT_ROOT, write_entry, append_entry, resolve_path, read_entry, list_dir, delete_entry
from .services import claude_service
from .services import linkedin_service

BASE_DIR = Path(__file__).resolve().parent
load_dotenv(BASE_DIR / ".env")
NOTES_TOKEN = os.environ.get("NOTES_TOKEN", "VJY9EoAf1xx1bO-LaduCmItwRitCFm9BPuQZ8jd0tcg")
security = HTTPBearer(auto_error=False)


def require_auth(
    request: Request,
    credentials: HTTPAuthorizationCredentials = Depends(security),
) -> None:
    if request.url.path.startswith("/login"):
        return
    if request.url.path.startswith("/api/linkedin/oauth/callback"):
        return
    if credentials is not None and credentials.scheme.lower() == "bearer":
        if secrets.compare_digest(credentials.credentials, NOTES_TOKEN):
            return
    cookie_token = request.cookies.get("notes_token")
    if cookie_token and secrets.compare_digest(cookie_token, NOTES_TOKEN):
        return
    raise HTTPException(status_code=401, detail="Unauthorized")


app = FastAPI(dependencies=[Depends(require_auth)])

app.mount("/static", StaticFiles(directory=str(BASE_DIR / "static")), name="static")
templates = Jinja2Templates(directory=str(BASE_DIR / "templates"))

PERSONS = {"sebastian", "petra"}
PERSON_COOKIE = "notes_person"
PERSON_HEADER = "X-Notes-Person"
THEME_COOKIE = "notes_theme"
THEMES = {"dark", "light"}


def get_person(request: Request) -> str | None:
    candidate = (
        request.headers.get(PERSON_HEADER)
        or request.cookies.get(PERSON_COOKIE)
        or ""
    ).strip().lower()
    return candidate if candidate in PERSONS else None


def ensure_person_or_redirect(request: Request):
    person = get_person(request)
    if person:
        return person
    return RedirectResponse("/settings")


def ensure_person(request: Request) -> str:
    person = get_person(request)
    if not person:
        raise HTTPException(status_code=400, detail="Person not selected")
    return person


def get_theme(request: Request) -> str:
    theme = (request.cookies.get(THEME_COOKIE) or "dark").strip().lower()
    return theme if theme in THEMES else "dark"


def person_root_path(person: str) -> Path:
    return VAULT_ROOT / person


def person_relative_path(person: str, relative_path: str) -> str:
    normalized = relative_path.strip().lstrip("/")
    if normalized in {"", "."}:
        return person
    if normalized.startswith(f"{person}/") or normalized == person:
        return normalized
    return f"{person}/{normalized}"


def strip_person_path(person: str, relative_path: str) -> str:
    if relative_path == person:
        return "."
    if relative_path.startswith(f"{person}/"):
        return relative_path[len(person) + 1 :]
    return relative_path


def get_today_filename() -> str:
    return datetime.now().strftime("%Y-%m-%d.md")


def get_today_filepath(person: str) -> Path:
    return person_root_path(person) / "daily" / get_today_filename()


def get_most_recent_note(person: str) -> Path | None:
    """Find the most recent daily note file (excluding today's)"""
    daily_dir = person_root_path(person) / "daily"
    if not daily_dir.exists():
        return None

    # Get all markdown files that match the date pattern YYYY-MM-DD.md
    note_files = sorted(daily_dir.glob("????-??-??.md"), reverse=True)

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
    if section_name.lower() == "todos":
        section_pattern = r'^##\s+todos\s*$'
    else:
        section_pattern = rf'^## {section_name}\s*$'
    section_match = re.search(section_pattern, content, re.MULTILINE | re.IGNORECASE)
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


def ensure_file_exists(person: str, filepath: Path) -> None:
    """Create file with header if it doesn't exist"""
    if not filepath.exists():
        filepath.parent.mkdir(parents=True, exist_ok=True)
        date_str = datetime.now().strftime("%Y-%m-%d")

        # Start with header
        content = f"# daily {date_str}\n\n"

        # Try to get incomplete todos and pinned notes from previous note
        previous_note = get_most_recent_note(person)
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


def add_task_to_todos(filepath: Path, category: str) -> None:
    content = filepath.read_text() if filepath.exists() else ""
    lines = content.splitlines()
    had_trailing = content.endswith("\n")

    todo_index = None
    for idx, line in enumerate(lines):
        if re.match(r"^##\s+todo(?:s)?\s*$", line, re.IGNORECASE):
            todo_index = idx
            break

    if todo_index is None:
        if lines and lines[-1].strip():
            lines.append("")
        lines.append("## todos")
        lines.append("")
        lines.append(f"### {category}")
        lines.append("")
        lines.append("- [ ]")
        lines.append("")
    else:
        next_section = None
        for idx in range(todo_index + 1, len(lines)):
            if re.match(r"^##\s+", lines[idx]):
                next_section = idx
                break
        section_end = next_section if next_section is not None else len(lines)

        sub_index = None
        for idx in range(todo_index + 1, section_end):
            if re.match(rf"^###\s+{re.escape(category)}\s*$", lines[idx], re.IGNORECASE):
                sub_index = idx
                break

        if sub_index is None:
            insert_at = section_end
            if insert_at > 0 and lines[insert_at - 1].strip():
                lines.insert(insert_at, "")
                insert_at += 1
            lines.insert(insert_at, f"### {category}")
            lines.insert(insert_at + 1, "")
            lines.insert(insert_at + 2, "- [ ]")
            lines.insert(insert_at + 3, "")
        else:
            insert_at = sub_index + 1
            if insert_at < len(lines) and lines[insert_at].strip() == "":
                insert_at += 1
            lines.insert(insert_at, "- [ ]")

    updated = "\n".join(lines)
    if had_trailing:
        updated += "\n"
    filepath.write_text(updated)


def get_sleep_times_filepath() -> Path:
    return VAULT_ROOT / "sleep_times.md"


def ensure_sleep_times_file_exists(filepath: Path) -> None:
    if not filepath.exists():
        content = (
            "# Sleep Times\n\n"
            "Log entries in the format:\n"
            "- YYYY-MM-DD | Name | 19:30-06:10 | night\n\n"
        )
        filepath.write_text(content)
        git_commit_and_push("Create sleep times log")


def get_recent_sleep_entries(filepath: Path, limit: int = 20) -> list[dict]:
    if not filepath.exists():
        return []

    content = filepath.read_text()
    lines = content.splitlines()
    entries: list[dict] = []
    for index, line in enumerate(lines, start=1):
        if not line.strip():
            continue
        if line.startswith("#") or line.startswith("- "):
            continue
        entries.append({"line_no": index, "text": line})

    recent = entries[-limit:]
    recent.reverse()
    return recent


@app.get("/", response_class=HTMLResponse)
async def root(request: Request):
    person = ensure_person_or_redirect(request)
    if isinstance(person, RedirectResponse):
        return person
    # Pull latest changes from remote
    git_pull()

    today = datetime.now().strftime("%Y-%m-%d")
    filepath = get_today_filepath(person)
    ensure_file_exists(person, filepath)
    content = filepath.read_text()
    relative_path = strip_person_path(person, str(filepath.relative_to(VAULT_ROOT)))
    note_html = render_with_pinned_buttons(content, relative_path)
    theme_class = f"theme-{get_theme(request)}"
    return templates.TemplateResponse(
        "editor.html",
        {
            "request": request,
            "today": today,
            "content": content,
            "note_html": note_html,
            "current_path": relative_path,
            "person": person,
            "theme_class": theme_class,
        }
    )


@app.get("/login", response_class=HTMLResponse)
async def login_form():
    return HTMLResponse(
        """
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="utf-8">
            <meta name="viewport" content="width=device-width, initial-scale=1">
            <title>Notes Editor Login</title>
        </head>
        <body>
            <main style="max-width: 420px; margin: 80px auto; font-family: sans-serif;">
                <h1>Notes Editor</h1>
                <form method="post" action="/login">
                    <label for="token">Access token</label>
                    <input id="token" name="token" type="password" required style="width: 100%; margin-top: 8px;" />
                    <button type="submit" style="margin-top: 16px;">Continue</button>
                </form>
            </main>
        </body>
        </html>
        """,
        status_code=200,
    )


@app.post("/login")
async def login(token: str = Form(...)):
    if not secrets.compare_digest(token, NOTES_TOKEN):
        return HTMLResponse("Invalid token", status_code=401)
    response = RedirectResponse("/", status_code=302)
    response.set_cookie("notes_token", token, httponly=True, samesite="lax")
    return response


@app.get("/settings", response_class=HTMLResponse)
async def settings_page(request: Request):
    current_person = get_person(request)
    current_theme = get_theme(request)
    env_path = BASE_DIR / ".env"
    env_content = env_path.read_text() if env_path.exists() else ""
    return templates.TemplateResponse(
        "settings.html",
        {
            "request": request,
            "current_person": current_person,
            "current_theme": current_theme,
            "people": sorted(PERSONS),
            "themes": sorted(THEMES),
            "theme_class": f"theme-{current_theme}",
            "env_content": env_content,
        },
    )


@app.post("/settings")
async def save_settings(
    request: Request,
    person: str | None = Form(None),
    theme: str | None = Form(None),
):
    response = RedirectResponse("/settings", status_code=302)
    if person:
        normalized = person.strip().lower()
        if normalized not in PERSONS:
            return HTMLResponse("Invalid person", status_code=400)
        response.set_cookie(PERSON_COOKIE, normalized, max_age=60 * 60 * 24 * 365, samesite="lax")
    if theme:
        normalized = theme.strip().lower()
        if normalized not in THEMES:
            return HTMLResponse("Invalid theme", status_code=400)
        response.set_cookie(THEME_COOKIE, normalized, max_age=60 * 60 * 24 * 365, samesite="lax")
    if not person and not theme:
        return HTMLResponse("Nothing to update", status_code=400)
    return response


@app.post("/settings/env")
async def save_env_settings(env_content: str = Form(...)):
    env_path = BASE_DIR / ".env"
    normalized = env_content.replace("\r\n", "\n")
    if normalized and not normalized.endswith("\n"):
        normalized += "\n"
    env_path.parent.mkdir(parents=True, exist_ok=True)
    env_path.write_text(normalized)
    load_dotenv(env_path, override=True)
    return RedirectResponse("/settings", status_code=302)


@app.get("/api/daily")
async def get_daily_note(request: Request):
    person = ensure_person(request)
    git_pull()
    today = datetime.now().strftime("%Y-%m-%d")
    filepath = get_today_filepath(person)
    ensure_file_exists(person, filepath)
    content = filepath.read_text()
    relative_path = strip_person_path(person, str(filepath.relative_to(VAULT_ROOT)))
    return {
        "date": today,
        "path": relative_path,
        "content": content,
    }


@app.post("/api/append")
async def append_note(request: Request, content: str = Form(...), pinned: str = Form(None)):
    if not content.strip():
        raise HTTPException(status_code=400, detail="Content cannot be empty")

    person = ensure_person(request)
    filepath = get_today_filepath(person)
    ensure_file_exists(person, filepath)

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


@app.post("/api/todos/add")
async def add_todo(request: Request, category: str = Form(...)):
    category = category.strip().lower()
    if category not in {"work", "priv"}:
        raise HTTPException(status_code=400, detail="Invalid category")

    person = ensure_person(request)
    filepath = get_today_filepath(person)
    ensure_file_exists(person, filepath)
    add_task_to_todos(filepath, category)
    success, msg = git_commit_and_push(f"Add {category} task")

    return {
        "success": success,
        "message": "Task added" if success else msg
    }


@app.post("/api/todos/toggle")
async def toggle_todo(request: Request, path: str = Form(...), line: int = Form(...)):
    if line < 1:
        raise HTTPException(status_code=400, detail="Invalid line number")

    person = ensure_person(request)
    filepath = resolve_path(person_relative_path(person, path))
    if not filepath.exists():
        raise HTTPException(status_code=404, detail="File not found")

    content = filepath.read_text()
    lines = content.splitlines()
    if line > len(lines):
        raise HTTPException(status_code=400, detail="Line out of range")

    target = lines[line - 1]
    if re.match(r"^\s*-\s*\[\s*\]\s*", target):
        lines[line - 1] = re.sub(r"\[\s*\]", "[x]", target, count=1)
    elif re.match(r"^\s*-\s*\[x\]\s*", target, re.IGNORECASE):
        lines[line - 1] = re.sub(r"\[\s*x\s*\]", "[ ]", target, count=1, flags=re.IGNORECASE)
    else:
        return {"success": True, "message": "Not a task line"}

    updated_content = "\n".join(lines)
    if content.endswith("\n"):
        updated_content += "\n"

    filepath.write_text(updated_content)
    success, msg = git_commit_and_push("Toggle todo")

    return {
        "success": success,
        "message": "Task updated" if success else msg
    }




@app.post("/api/clear-pinned")
async def clear_pinned(request: Request):
    person = ensure_person(request)
    filepath = get_today_filepath(person)
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
async def save_note(request: Request, content: str = Form(...)):
    person = ensure_person(request)
    filepath = get_today_filepath(person)
    ensure_file_exists(person, filepath)

    write_entry(str(filepath.relative_to(VAULT_ROOT)), content)

    # Git operations
    success, msg = git_commit_and_push("Update note")

    return {
        "success": success,
        "message": "Note saved successfully" if success else msg
    }




@app.post("/api/files/unpin")
async def unpin_entry(request: Request, path: str = Form(...), line: int = Form(...)):
    if line < 1:
        raise HTTPException(status_code=400, detail="Invalid line number")

    person = ensure_person(request)
    filepath = resolve_path(person_relative_path(person, path))
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
    person = ensure_person_or_redirect(request)
    if isinstance(person, RedirectResponse):
        return person
    git_pull()
    entries = list_dir(person)
    for entry in entries:
        entry["path"] = strip_person_path(person, entry["path"])
    tree_html = render_tree(entries)
    theme_class = f"theme-{get_theme(request)}"
    return templates.TemplateResponse(
        "files.html",
        {
            "request": request,
            "tree_html": tree_html,
            "person": person,
            "theme_class": theme_class,
        },
    )


@app.get("/file", response_class=HTMLResponse)
async def file_page(request: Request, path: str):
    person = ensure_person_or_redirect(request)
    if isinstance(person, RedirectResponse):
        return person
    git_pull()
    filepath = resolve_path(person_relative_path(person, path))
    if not filepath.exists():
        raise HTTPException(status_code=404, detail="File not found")
    if filepath.is_dir():
        raise HTTPException(status_code=400, detail="Path is a directory")

    content = read_entry(str(filepath.relative_to(VAULT_ROOT)))
    note_html = render_with_pinned_buttons(content, strip_person_path(person, str(filepath.relative_to(VAULT_ROOT))))
    theme_class = f"theme-{get_theme(request)}"
    return templates.TemplateResponse(
        "file_page.html",
        {
            "request": request,
            "file_path": strip_person_path(person, str(filepath.relative_to(VAULT_ROOT))),
            "content": content,
            "note_html": note_html,
            "message": "",
            "person": person,
            "theme_class": theme_class,
        },
    )


@app.get("/tools", response_class=HTMLResponse)
async def tools_page(request: Request):
    person = ensure_person_or_redirect(request)
    if isinstance(person, RedirectResponse):
        return person
    return templates.TemplateResponse(
        "tools.html",
        {"request": request, "person": person, "theme_class": f"theme-{get_theme(request)}"},
    )


@app.get("/tools/llm", response_class=HTMLResponse)
async def llm_page(request: Request):
    person = ensure_person_or_redirect(request)
    if isinstance(person, RedirectResponse):
        return person
    return templates.TemplateResponse(
        "llm.html",
        {"request": request, "person": person, "theme_class": f"theme-{get_theme(request)}"},
    )


@app.get("/tools/noise", response_class=HTMLResponse)
async def noise_page(request: Request):
    person = ensure_person_or_redirect(request)
    if isinstance(person, RedirectResponse):
        return person
    return templates.TemplateResponse(
        "noise.html",
        {"request": request, "person": person, "theme_class": f"theme-{get_theme(request)}"},
    )


@app.get("/sleep-times", response_class=HTMLResponse)
async def sleep_times_page(request: Request):
    person = ensure_person_or_redirect(request)
    if isinstance(person, RedirectResponse):
        return person
    git_pull()
    filepath = get_sleep_times_filepath()
    ensure_sleep_times_file_exists(filepath)
    entries = get_recent_sleep_entries(filepath)
    return templates.TemplateResponse(
        "sleep_times.html",
        {
            "request": request,
            "entries": entries,
            "person": person,
            "theme_class": f"theme-{get_theme(request)}",
        },
    )


@app.get("/api/sleep-times")
async def list_sleep_times(request: Request):
    ensure_person(request)
    git_pull()
    filepath = get_sleep_times_filepath()
    ensure_sleep_times_file_exists(filepath)
    entries = get_recent_sleep_entries(filepath)
    return {"entries": entries}


@app.post("/api/sleep-times/append")
async def append_sleep_times(
    request: Request,
    child: str = Form(...),
    entry: str = Form(...),
    asleep: str = Form(None),
    woke: str = Form(None),
):
    ensure_person(request)
    if not entry.strip():
        raise HTTPException(status_code=400, detail="Entry cannot be empty")

    filepath = get_sleep_times_filepath()
    ensure_sleep_times_file_exists(filepath)

    date_str = datetime.now().strftime("%Y-%m-%d")
    normalized_child = child.strip().capitalize()
    suffix = ""
    if asleep == "on":
        suffix = " | eingeschlafen"
    elif woke == "on":
        suffix = " | aufgewacht"
    line = f"{date_str} | {normalized_child} | {entry.strip()}{suffix}\n"
    append_entry("sleep_times.md", line)

    success, msg = git_commit_and_push("Append sleep times")
    return {
        "success": success,
        "message": "Entry added" if success else msg,
    }


@app.post("/api/sleep-times/delete")
async def delete_sleep_entry(request: Request, line: int = Form(...)):
    ensure_person(request)
    if line < 1:
        raise HTTPException(status_code=400, detail="Invalid line number")

    filepath = get_sleep_times_filepath()
    ensure_sleep_times_file_exists(filepath)
    content = filepath.read_text()
    lines = content.splitlines()
    if line > len(lines):
        raise HTTPException(status_code=400, detail="Line out of range")

    lines.pop(line - 1)
    updated_content = "\n".join(lines)
    if content.endswith("\n"):
        updated_content += "\n"
    filepath.write_text(updated_content)

    success, msg = git_commit_and_push("Delete sleep times entry")
    return {
        "success": success,
        "message": "Entry deleted" if success else msg,
    }


@app.get("/api/files/tree", response_class=HTMLResponse)
async def files_tree(request: Request, path: str = "."):
    person = ensure_person(request)
    entries = list_dir(person_relative_path(person, path))
    for entry in entries:
        entry["path"] = strip_person_path(person, entry["path"])
    tree_html = render_tree(entries)
    return HTMLResponse(tree_html)


@app.get("/api/files/list")
async def list_files(request: Request, path: str = "."):
    person = ensure_person(request)
    git_pull()
    entries = list_dir(person_relative_path(person, path))
    for entry in entries:
        entry["path"] = strip_person_path(person, entry["path"])
    return {"entries": entries}


@app.get("/api/files/read")
async def read_file(request: Request, path: str):
    person = ensure_person(request)
    git_pull()
    filepath = resolve_path(person_relative_path(person, path))
    if not filepath.exists():
        raise HTTPException(status_code=404, detail="File not found")
    if filepath.is_dir():
        raise HTTPException(status_code=400, detail="Path is a directory")
    return {
        "path": strip_person_path(person, str(filepath.relative_to(VAULT_ROOT))),
        "content": read_entry(str(filepath.relative_to(VAULT_ROOT)))
    }


@app.get("/api/files/open", response_class=HTMLResponse)
async def open_file(request: Request, path: str):
    person = ensure_person(request)
    filepath = resolve_path(person_relative_path(person, path))
    if not filepath.exists():
        raise HTTPException(status_code=404, detail="File not found")
    if filepath.is_dir():
        raise HTTPException(status_code=400, detail="Path is a directory")

    content = read_entry(str(filepath.relative_to(VAULT_ROOT)))
    note_html = render_with_pinned_buttons(content, strip_person_path(person, str(filepath.relative_to(VAULT_ROOT))))
    return templates.TemplateResponse(
        "file_editor.html",
        {
            "request": request,
            "file_path": strip_person_path(person, str(filepath.relative_to(VAULT_ROOT))),
            "content": content,
            "note_html": note_html,
            "message": "",
        },
    )


@app.post("/api/files/create")
async def create_file(request: Request, path: str = Form(...)):
    if not path.strip():
        raise HTTPException(status_code=400, detail="Path required")

    person = ensure_person(request)
    normalized = person_relative_path(person, path.strip())
    filepath = resolve_path(normalized)
    if filepath.exists():
        return {"success": True, "message": "File already exists"}

    write_entry(normalized, "")
    success, msg = git_commit_and_push("Create file")
    return {
        "success": success,
        "message": "File created" if success else msg,
    }


@app.post("/api/files/save", response_class=HTMLResponse)
async def save_file(request: Request, path: str = Form(...), content: str = Form(...)):
    person = ensure_person(request)
    write_entry(person_relative_path(person, path), content)
    success, msg = git_commit_and_push("Update file")
    note_html = render_with_pinned_buttons(content, path)
    return templates.TemplateResponse(
        "file_editor.html",
        {
            "request": request,
            "file_path": path,
            "content": content,
            "note_html": note_html,
            "message": "File saved" if success else msg,
        },
    )


@app.post("/api/files/save-json")
async def save_file_json(request: Request, path: str = Form(...), content: str = Form(...)):
    person = ensure_person(request)
    write_entry(person_relative_path(person, path), content)
    success, msg = git_commit_and_push("Update file")
    return {"success": success, "message": "File saved" if success else msg}


@app.post("/api/files/delete", response_class=HTMLResponse)
async def delete_file(request: Request, path: str = Form(...)):
    person = ensure_person(request)
    try:
        delete_entry(person_relative_path(person, path))
    except IsADirectoryError:
        return templates.TemplateResponse(
            "file_editor.html",
            {
                "request": request,
                "file_path": path,
                "content": "",
                "note_html": "",
                "message": "Cannot delete a directory",
            },
        )

    success, msg = git_commit_and_push("Delete file")
    response = templates.TemplateResponse(
        "file_editor.html",
        {
            "request": request,
            "file_path": None,
            "content": "",
            "note_html": "",
            "message": "File deleted" if success else msg,
        },
    )
    response.headers["HX-Trigger"] = "fileDeleted"
    return response


@app.post("/api/files/delete-json")
async def delete_file_json(request: Request, path: str = Form(...)):
    person = ensure_person(request)
    try:
        delete_entry(person_relative_path(person, path))
    except IsADirectoryError:
        return {"success": False, "message": "Cannot delete a directory"}
    success, msg = git_commit_and_push("Delete file")
    return {"success": success, "message": "File deleted" if success else msg}


@app.post("/api/claude/chat")
async def claude_chat(
    request: Request,
    message: str = Form(...),
    session_id: str = Form(None),
):
    person = ensure_person(request)
    try:
        session, response = await claude_service.chat(session_id, message, person)
        return {
            "success": True,
            "session_id": session.session_id,
            "response": response,
            "history": [{"role": m.role, "content": m.content} for m in session.messages],
        }
    except Exception as e:
        import traceback
        traceback.print_exc()
        return {"success": False, "message": str(e), "response": "", "history": []}


@app.get("/api/linkedin/oauth/callback")
async def linkedin_oauth_callback(code: str):
    try:
        token_data = linkedin_service.exchange_code_for_token(code)
        access_token = token_data.get("access_token", "")
        if not access_token:
            raise RuntimeError("LinkedIn response missing access_token")
        linkedin_service.persist_access_token(access_token)
        return {"success": True, "expires_in": token_data.get("expires_in")}
    except Exception as exc:
        raise HTTPException(status_code=400, detail=str(exc))


@app.post("/api/claude/chat-stream")
async def claude_chat_stream(
    request: Request,
    message: str = Form(...),
    session_id: str = Form(None),
):
    person = ensure_person(request)

    async def event_stream():
        queue: asyncio.Queue[dict] = asyncio.Queue()
        done_marker = {"type": "_stream_done"}

        async def producer():
            try:
                async for event in claude_service.chat_stream(session_id, message, person):
                    await queue.put(event)
            except Exception as e:
                await queue.put({"type": "error", "message": str(e)})
            finally:
                await queue.put(done_marker)

        task = asyncio.create_task(producer())
        try:
            while True:
                try:
                    event = await asyncio.wait_for(queue.get(), timeout=5)
                except asyncio.TimeoutError:
                    yield json.dumps({"type": "ping"}, default=str) + "\n"
                    continue
                if event is done_marker:
                    break
                yield json.dumps(event, default=str) + "\n"
        finally:
            task.cancel()

    return StreamingResponse(event_stream(), media_type="application/x-ndjson")


@app.get("/api/claude/history")
async def claude_history(request: Request, session_id: str):
    ensure_person(request)
    history = claude_service.get_session_history(session_id)
    if history is None:
        return {"success": False, "message": "Session not found", "history": []}
    return {
        "success": True,
        "history": [{"role": m.role, "content": m.content} for m in history],
    }


@app.post("/api/claude/clear")
async def claude_clear(request: Request, session_id: str = Form(...)):
    ensure_person(request)
    cleared = claude_service.clear_session(session_id)
    return {"success": cleared, "message": "Session cleared" if cleared else "Session not found"}
