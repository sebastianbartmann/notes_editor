from html import escape
import re


PINNED_HEADING = re.compile(r"^(###\s+.*<pinned>.*)$", re.IGNORECASE)
HEADING_LINE = re.compile(r"^(#{1,6})\s+(.*)$")
TASK_LINE = re.compile(r"^\s*-\s*\[([ xX])\]\s*(.*)$")


def render_with_pinned_buttons(content: str, file_path: str) -> str:
    lines = content.splitlines()
    html_lines: list[str] = []
    escaped_path = escape(file_path, quote=True)

    for index, line in enumerate(lines, start=1):
        if PINNED_HEADING.match(line):
            escaped_line = escape(line)
            html_lines.append(
                "<div class=\"note-line note-heading pinned heading h3\">"
                f"<span class=\"line-text heading-text\">{escaped_line}</span>"
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

        heading_match = HEADING_LINE.match(line)
        if heading_match:
            level = len(heading_match.group(1))
            hashes = escape(heading_match.group(1))
            text = escape(heading_match.group(2))
            html_lines.append(
                f"<div class=\"note-line heading h{level}\">"
                f"<span class=\"heading-text\">{hashes} {text}</span>"
                "</div>"
            )
            continue

        task_match = TASK_LINE.match(line)
        if task_match:
            checked = task_match.group(1).lower() == "x"
            text = escape(task_match.group(2))
            class_name = "note-line task-line done" if checked else "note-line task-line"
            checkbox = "checked" if checked else ""
            html_lines.append(
                f"<div class=\"{class_name}\">"
                "<form class=\"inline-form\">"
                f"<input type=\"hidden\" name=\"line\" value=\"{index}\">"
                f"<input type=\"hidden\" name=\"path\" value=\"{escaped_path}\">"
                f"<input type=\"checkbox\" {checkbox} "
                "hx-post=\"/api/todos/toggle\" "
                "hx-trigger=\"change\" "
                "hx-target=\"#message\" "
                "hx-swap=\"innerHTML\" "
                "hx-include=\"closest form\">"
                "</form>"
                f"<span class=\"task-text\">{text}</span>"
                "</div>"
            )
            continue

        if line == "":
            html_lines.append("<div class=\"note-line empty\">&nbsp;</div>")
        else:
            html_lines.append(f"<div class=\"note-line\">{escape(line)}</div>")

    return "\n".join(html_lines)
