from contextvars import ContextVar, Token

from claude_agent_sdk import create_sdk_mcp_server, tool

from . import linkedin_service


CURRENT_PERSON: ContextVar[str | None] = ContextVar("linkedin_current_person", default=None)


def set_current_person(person: str) -> Token:
    return CURRENT_PERSON.set(person)


def reset_current_person(token: Token) -> None:
    CURRENT_PERSON.reset(token)


def _require_person() -> str:
    person = CURRENT_PERSON.get()
    if not person:
        raise RuntimeError("LinkedIn tools require a person context")
    return person


@tool("linkedin_post", "Post a text update to LinkedIn", {"text": str})
async def linkedin_post(args: dict) -> dict:
    text = (args.get("text") or "").strip()
    if not text:
        return {"content": [{"type": "text", "text": "Error: text is required"}], "is_error": True}

    try:
        response = linkedin_service.create_post(text)
        person = _require_person()
        linkedin_service.log_post(person, text, response)
        post_urn = response.get("id", "")
        return {"content": [{"type": "text", "text": f"Posted to LinkedIn: {post_urn}"}]}
    except Exception as exc:
        return {"content": [{"type": "text", "text": f"Error posting to LinkedIn: {exc}"}], "is_error": True}


@tool("linkedin_read_comments", "Read comments for a LinkedIn post", {"post_urn": str})
async def linkedin_read_comments(args: dict) -> dict:
    post_urn = (args.get("post_urn") or "").strip()
    if not post_urn:
        return {"content": [{"type": "text", "text": "Error: post_urn is required"}], "is_error": True}
    try:
        response = linkedin_service.read_comments(post_urn)
        return {"content": [{"type": "text", "text": str(response)}]}
    except Exception as exc:
        return {"content": [{"type": "text", "text": f"Error reading comments: {exc}"}], "is_error": True}


@tool("linkedin_post_comment", "Post a comment on a LinkedIn post", {"post_urn": str, "text": str})
async def linkedin_post_comment(args: dict) -> dict:
    post_urn = (args.get("post_urn") or "").strip()
    text = (args.get("text") or "").strip()
    if not post_urn or not text:
        return {"content": [{"type": "text", "text": "Error: post_urn and text are required"}], "is_error": True}

    try:
        response = linkedin_service.create_comment(post_urn, text)
        person = _require_person()
        linkedin_service.log_comment(person, text, response, post_urn=post_urn)
        comment_urn = response.get("id", "")
        return {"content": [{"type": "text", "text": f"Comment posted: {comment_urn}"}]}
    except Exception as exc:
        return {"content": [{"type": "text", "text": f"Error posting comment: {exc}"}], "is_error": True}


@tool("linkedin_reply_comment", "Reply to a LinkedIn comment", {"post_urn": str, "comment_urn": str, "text": str})
async def linkedin_reply_comment(args: dict) -> dict:
    post_urn = (args.get("post_urn") or "").strip()
    comment_urn = (args.get("comment_urn") or "").strip()
    text = (args.get("text") or "").strip()
    if not post_urn or not comment_urn or not text:
        return {
            "content": [{"type": "text", "text": "Error: post_urn, comment_urn, and text are required"}],
            "is_error": True,
        }

    try:
        response = linkedin_service.create_comment(post_urn, text, parent_comment_urn=comment_urn)
        person = _require_person()
        linkedin_service.log_comment(
            person,
            text,
            response,
            post_urn=post_urn,
            comment_urn=comment_urn,
            action="reply",
        )
        reply_urn = response.get("id", "")
        return {"content": [{"type": "text", "text": f"Reply posted: {reply_urn}"}]}
    except Exception as exc:
        return {"content": [{"type": "text", "text": f"Error replying to comment: {exc}"}], "is_error": True}


LINKEDIN_TOOLS = [
    linkedin_post,
    linkedin_read_comments,
    linkedin_post_comment,
    linkedin_reply_comment,
]

LINKEDIN_MCP_SERVER = create_sdk_mcp_server("linkedin", tools=LINKEDIN_TOOLS)


def get_mcp_server():
    return LINKEDIN_MCP_SERVER
