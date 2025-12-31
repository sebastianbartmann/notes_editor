from pathlib import Path
import subprocess


GIT_DIR = Path.home() / "notes"


def git_pull() -> tuple[bool, str]:
    """Pull latest changes from remote. Returns (success, message)."""
    if not GIT_DIR.exists() or not (GIT_DIR / ".git").exists():
        return True, "No git repository"

    try:
        rebase_dir = GIT_DIR / ".git" / "rebase-merge"
        rebase_apply = GIT_DIR / ".git" / "rebase-apply"
        merge_head = GIT_DIR / ".git" / "MERGE_HEAD"

        if rebase_dir.exists() or rebase_apply.exists():
            subprocess.run(["git", "rebase", "--abort"], cwd=GIT_DIR, capture_output=True)

        if merge_head.exists():
            subprocess.run(["git", "merge", "--abort"], cwd=GIT_DIR, capture_output=True)

        result = subprocess.run(
            ["git", "pull", "--no-rebase", "-X", "theirs"],
            cwd=GIT_DIR,
            capture_output=True,
            text=True
        )

        if result.returncode == 0:
            return True, "Pulled latest changes"

        subprocess.run(
            ["git", "fetch", "origin"],
            cwd=GIT_DIR,
            capture_output=True,
            text=True
        )

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

    except Exception as exc:
        return False, f"Error: {str(exc)}"


def git_commit_and_push(message: str = "Update notes") -> tuple[bool, str]:
    """Perform git add, commit, and push. Returns (success, message)."""
    if not GIT_DIR.exists() or not (GIT_DIR / ".git").exists():
        return True, "No git repository (notes saved locally)"

    try:
        result = subprocess.run(
            ["git", "status", "--porcelain"],
            cwd=GIT_DIR,
            capture_output=True,
            text=True,
            check=True
        )

        if not result.stdout.strip():
            return True, "No changes to commit"

        subprocess.run(["git", "add", "."], cwd=GIT_DIR, check=True, capture_output=True)
        subprocess.run(
            ["git", "commit", "-m", message],
            cwd=GIT_DIR,
            check=True,
            capture_output=True,
            text=True
        )

        push_result = subprocess.run(
            ["git", "push"],
            cwd=GIT_DIR,
            capture_output=True,
            text=True
        )

        if push_result.returncode == 0:
            return True, "Changes committed and pushed"

        pull_success, _ = git_pull()

        if pull_success:
            retry_push = subprocess.run(
                ["git", "push"],
                cwd=GIT_DIR,
                capture_output=True,
                text=True
            )

            if retry_push.returncode == 0:
                return True, "Changes committed and pushed (after sync)"

        return True, "Changes committed locally (push failed, will retry on next sync)"

    except subprocess.CalledProcessError as exc:
        error_msg = exc.stderr if exc.stderr else str(exc)
        return True, f"Changes saved locally: {error_msg}"
    except Exception as exc:
        return True, f"Changes saved locally: {str(exc)}"
