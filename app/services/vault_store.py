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


def delete_entry(relative_path: str) -> None:
    filepath = resolve_path(relative_path)
    if not filepath.exists():
        return
    if filepath.is_dir():
        raise IsADirectoryError("Path is a directory")
    filepath.unlink()


def list_dir(relative_path: str) -> list[dict]:
    directory = resolve_path(relative_path)
    if not directory.exists():
        raise FileNotFoundError("Directory does not exist")
    if not directory.is_dir():
        raise NotADirectoryError("Path is not a directory")

    entries = []
    for entry in directory.iterdir():
        name = entry.name
        if name.startswith("."):
            continue
        entries.append(
            {
                "name": name,
                "path": str(entry.relative_to(VAULT_ROOT)),
                "is_dir": entry.is_dir(),
            }
        )

    entries.sort(key=lambda item: (0 if not item["is_dir"] else 1, item["name"].lower()))
    return entries
