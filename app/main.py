from fastapi import FastAPI, Request, HTTPException, Form
from fastapi.responses import HTMLResponse, RedirectResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from datetime import datetime
from pathlib import Path
import subprocess
import re

app = FastAPI()

app.mount("/static", StaticFiles(directory="app/static"), name="static")
templates = Jinja2Templates(directory="app/templates")

NOTES_DIR = Path.home() / "notes" / "daily"
GIT_DIR = Path.home() / "notes"


def get_today_filename() -> str:
    return datetime.now().strftime("%Y-%m-%d.md")


def get_today_filepath() -> Path:
    return NOTES_DIR / get_today_filename()


def get_most_recent_note() -> Path | None:
    """Find the most recent daily note file (excluding today's)"""
    if not NOTES_DIR.exists():
        return None

    # Get all markdown files that match the date pattern YYYY-MM-DD.md
    note_files = sorted(NOTES_DIR.glob("????-??-??.md"), reverse=True)

    today_file = get_today_filename()
    for note_file in note_files:
        if note_file.name != today_file:
            return note_file

    return None


def extract_incomplete_todos(filepath: Path) -> str:
    """Extract incomplete todos from the ## todos section of a note file"""
    if not filepath.exists():
        return ""

    content = filepath.read_text()

    # Find the ## todos section
    todos_match = re.search(r'^## todos\s*$', content, re.MULTILINE | re.IGNORECASE)
    if not todos_match:
        return ""

    # Get content starting from ## todos
    start_pos = todos_match.end()
    remaining_content = content[start_pos:]

    # Find the next ## header or end of file
    next_section_match = re.search(r'^## ', remaining_content, re.MULTILINE)
    if next_section_match:
        todos_content = remaining_content[:next_section_match.start()]
    else:
        todos_content = remaining_content

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


def ensure_file_exists(filepath: Path) -> None:
    """Create file with header if it doesn't exist"""
    if not filepath.exists():
        NOTES_DIR.mkdir(parents=True, exist_ok=True)
        date_str = datetime.now().strftime("%Y-%m-%d")

        # Start with header
        content = f"# daily {date_str}\n\n"

        # Try to get incomplete todos from previous note
        previous_note = get_most_recent_note()
        if previous_note:
            incomplete_todos = extract_incomplete_todos(previous_note)
            if incomplete_todos:
                content += f"## todos\n\n{incomplete_todos}\n\n"

        # Write the file
        filepath.write_text(content)

        # Commit and push the new file
        git_commit_and_push(f"Create daily note {date_str}")


def git_pull() -> tuple[bool, str]:
    """Pull latest changes from remote. Returns (success, message)"""
    # Check if git repo exists
    if not GIT_DIR.exists() or not (GIT_DIR / ".git").exists():
        return True, "No git repository"

    try:
        # Check for and abort any stuck rebase/merge states
        rebase_dir = GIT_DIR / ".git" / "rebase-merge"
        rebase_apply = GIT_DIR / ".git" / "rebase-apply"
        merge_head = GIT_DIR / ".git" / "MERGE_HEAD"

        if rebase_dir.exists() or rebase_apply.exists():
            subprocess.run(["git", "rebase", "--abort"], cwd=GIT_DIR, capture_output=True)

        if merge_head.exists():
            subprocess.run(["git", "merge", "--abort"], cwd=GIT_DIR, capture_output=True)

        # Pull with merge strategy (not rebase) and auto-resolve conflicts with remote version
        result = subprocess.run(
            ["git", "pull", "--no-rebase", "-X", "theirs"],
            cwd=GIT_DIR,
            capture_output=True,
            text=True
        )

        if result.returncode == 0:
            return True, "Pulled latest changes"

        # If pull failed, try to recover by resetting to remote
        subprocess.run(
            ["git", "fetch", "origin"],
            cwd=GIT_DIR,
            capture_output=True,
            text=True
        )

        # Get current branch name
        branch_result = subprocess.run(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            cwd=GIT_DIR,
            capture_output=True,
            text=True
        )

        if branch_result.returncode == 0:
            branch = branch_result.stdout.strip()
            subprocess.run(
                ["git", "reset", "--hard", f"origin/{branch}"],
                cwd=GIT_DIR,
                capture_output=True,
                text=True
            )
            return True, "Recovered by resetting to remote"

        return False, "Git pull failed"

    except Exception as e:
        return False, f"Error: {str(e)}"


def git_commit_and_push(message: str = "Update notes") -> tuple[bool, str]:
    """Perform git add, commit, and push. Returns (success, message)"""
    # Check if git repo exists
    if not GIT_DIR.exists() or not (GIT_DIR / ".git").exists():
        return True, "No git repository (notes saved locally)"

    try:
        # Check if there are changes
        result = subprocess.run(
            ["git", "status", "--porcelain"],
            cwd=GIT_DIR,
            capture_output=True,
            text=True,
            check=True
        )

        if not result.stdout.strip():
            return True, "No changes to commit"

        # Add all changes
        subprocess.run(["git", "add", "."], cwd=GIT_DIR, check=True, capture_output=True)

        # Commit
        subprocess.run(
            ["git", "commit", "-m", message],
            cwd=GIT_DIR,
            check=True,
            capture_output=True,
            text=True
        )

        # Push - try up to 2 times
        push_result = subprocess.run(
            ["git", "push"],
            cwd=GIT_DIR,
            capture_output=True,
            text=True
        )

        if push_result.returncode == 0:
            return True, "Changes committed and pushed"

        # If push failed (likely due to diverged history), pull and retry
        pull_success, _ = git_pull()

        if pull_success:
            # Retry push after successful pull
            retry_push = subprocess.run(
                ["git", "push"],
                cwd=GIT_DIR,
                capture_output=True,
                text=True
            )

            if retry_push.returncode == 0:
                return True, "Changes committed and pushed (after sync)"

        # If all retries failed, still return success since changes are committed locally
        return True, "Changes committed locally (push failed, will retry on next sync)"

    except subprocess.CalledProcessError as e:
        error_msg = e.stderr if e.stderr else str(e)
        # Don't fail the request - changes may still be saved locally
        return True, f"Changes saved locally: {error_msg}"
    except Exception as e:
        return True, f"Changes saved locally: {str(e)}"


@app.get("/", response_class=HTMLResponse)
async def root(request: Request):
    # Pull latest changes from remote
    git_pull()

    today = datetime.now().strftime("%Y-%m-%d")
    filepath = get_today_filepath()
    ensure_file_exists(filepath)
    content = filepath.read_text()
    return templates.TemplateResponse(
        "editor.html",
        {"request": request, "today": today, "content": content}
    )


@app.post("/api/append")
async def append_note(content: str = Form(...)):
    if not content.strip():
        raise HTTPException(status_code=400, detail="Content cannot be empty")

    filepath = get_today_filepath()
    ensure_file_exists(filepath)

    # Get current time
    time_str = datetime.now().strftime("%H:%M")

    # Read current content
    current_content = filepath.read_text()

    # Check if ## custom notes section exists
    if re.search(r'^## custom notes\s*$', current_content, re.MULTILINE | re.IGNORECASE):
        # Append under existing section
        append_text = f"\n### {time_str}\n\n{content.strip()}\n"
    else:
        # Create section and add first entry
        append_text = f"\n## custom notes\n\n### {time_str}\n\n{content.strip()}\n"

    with open(filepath, "a") as f:
        f.write(append_text)

    # Git operations
    success, msg = git_commit_and_push(f"Append note at {time_str}")

    return {
        "success": success,
        "message": "Content appended successfully" if success else msg
    }


@app.post("/api/save")
async def save_note(content: str = Form(...)):
    filepath = get_today_filepath()
    ensure_file_exists(filepath)

    filepath.write_text(content)

    # Git operations
    success, msg = git_commit_and_push("Update note")

    return {
        "success": success,
        "message": "Note saved successfully" if success else msg
    }
