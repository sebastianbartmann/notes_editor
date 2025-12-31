from html import escape
import re


PINNED_HEADING = re.compile(r"^(###\s+.*<pinned>.*)$", re.IGNORECASE)


def render_with_pinned_buttons(content: str, file_path: str) -> str:
    lines = content.splitlines()
    html_lines: list[str] = []
    escaped_path = escape(file_path, quote=True)

    for index, line in enumerate(lines, start=1):
        if PINNED_HEADING.match(line):
            escaped_line = escape(line)
            html_lines.append(
                "<div class=\"note-line note-heading pinned\">"
                f"<span class=\"line-text\">{escaped_line}</span>"
                "<form class=\"pin-form\" "
                "hx-post=\"/api/files/unpin\" "
                "hx-target=\"#message\" "
                "hx-swap=\"innerHTML\">"
                f"<input type=\"hidden\" name=\"path\" value=\"{escaped_path}\">"
                f"<input type=\"hidden\" name=\"line\" value=\"{index}\">"
                "<button class=\"pin-action\" type=\"submit\">Unpin</button>"
                "</form>"
                "</div>"
            )
            continue

        if line == "":
            html_lines.append("<div class=\"note-line empty\">&nbsp;</div>")
        else:
            html_lines.append(f"<div class=\"note-line\">{escape(line)}</div>")

    return "\n".join(html_lines)
