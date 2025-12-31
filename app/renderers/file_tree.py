from html import escape
from urllib.parse import quote
import re


def _safe_id(value: str) -> str:
    return re.sub(r"[^a-zA-Z0-9_-]", "-", value)


def render_tree(entries: list[dict]) -> str:
    html_lines: list[str] = []

    for entry in entries:
        name = escape(entry["name"])
        path = entry["path"]
        encoded_path = quote(path)
        item_id = _safe_id(path)

        if entry["is_dir"]:
            html_lines.append(
                "<div class=\"tree-item dir\">"
                f"<button class=\"tree-toggle\" "
                f"data-target=\"children-{item_id}\" "
                f"hx-get=\"/api/files/tree?path={encoded_path}\" "
                f"hx-target=\"#children-{item_id}\" "
                f"hx-swap=\"innerHTML\">+</button>"
                f"<span class=\"tree-name\">{name}/</span>"
                "</div>"
                f"<div id=\"children-{item_id}\" class=\"tree-children\"></div>"
            )
        else:
            html_lines.append(
                "<div class=\"tree-item file\" "
                f"hx-get=\"/api/files/open?path={encoded_path}\" "
                "hx-target=\"#file-editor\" "
                "hx-swap=\"innerHTML\">"
                f"<span class=\"tree-name\">{name}</span>"
                "</div>"
            )

    return "\n".join(html_lines)
