from fastapi import FastAPI, Request, HTTPException, Form
from fastapi.responses import HTMLResponse, RedirectResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from datetime import datetime
from pathlib import Path
import subprocess

app = FastAPI()

app.mount("/static", StaticFiles(directory="app/static"), name="static")
templates = Jinja2Templates(directory="app/templates")

NOTES_DIR = Path.home() / "notes" / "daily"
GIT_DIR = Path.home() / "notes"


def get_today_filename() -> str:
    return datetime.now().strftime("%Y-%m-%d.md")


def get_today_filepath() -> Path:
    return NOTES_DIR / get_today_filename()


def ensure_file_exists(filepath: Path) -> None:
    """Create file with header if it doesn't exist"""
    if not filepath.exists():
        NOTES_DIR.mkdir(parents=True, exist_ok=True)
        date_str = datetime.now().strftime("%Y-%m-%d")
        filepath.write_text(f"# daily {date_str}\n\n")


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

        # Push
        subprocess.run(["git", "push"], cwd=GIT_DIR, check=True, capture_output=True)

        return True, "Changes committed and pushed"

    except subprocess.CalledProcessError as e:
        error_msg = e.stderr if e.stderr else str(e)
        return False, f"Git error: {error_msg}"
    except Exception as e:
        return False, f"Error: {str(e)}"


@app.get("/", response_class=HTMLResponse)
async def root(request: Request):
    today = datetime.now().strftime("%Y-%m-%d")
    filepath = get_today_filepath()
    ensure_file_exists(filepath)
    content = filepath.read_text()
    return templates.TemplateResponse(
        "append.html",
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

    # Append content with format
    append_text = f"\n## {time_str}\n{content.strip()}\n\n"

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
