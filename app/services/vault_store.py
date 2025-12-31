from pathlib import Path


VAULT_ROOT = Path.home() / "notes"


def resolve_path(relative_path: str) -> Path:
    if not relative_path:
        raise ValueError("Path is required")
    if Path(relative_path).is_absolute():
        raise ValueError("Path must be vault-relative")

    full_path = (VAULT_ROOT / relative_path).resolve()
    if VAULT_ROOT.resolve() not in full_path.parents and full_path != VAULT_ROOT.resolve():
        raise ValueError("Path escapes vault root")

    return full_path


def read_entry(relative_path: str) -> str:
    filepath = resolve_path(relative_path)
    return filepath.read_text()


def write_entry(relative_path: str, content: str) -> None:
    filepath = resolve_path(relative_path)
    filepath.parent.mkdir(parents=True, exist_ok=True)
    filepath.write_text(content)


def append_entry(relative_path: str, content: str) -> None:
    filepath = resolve_path(relative_path)
    filepath.parent.mkdir(parents=True, exist_ok=True)
    with open(filepath, "a") as handle:
        handle.write(content)
