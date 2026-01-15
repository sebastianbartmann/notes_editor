import csv
import json
from datetime import datetime
from pathlib import Path
from urllib.parse import quote

import os
import requests
from dotenv import load_dotenv

from .vault_store import VAULT_ROOT

BASE_DIR = Path(__file__).resolve().parent.parent
ENV_PATH = BASE_DIR / ".env"

load_dotenv(ENV_PATH)

LINKEDIN_API_BASE = "https://api.linkedin.com"
LINKEDIN_VERSION = ""

CSV_HEADERS = [
    "timestamp",
    "action",
    "post_urn",
    "comment_urn",
    "text",
    "response",
]


def _get_env(name: str) -> str:
    return os.getenv(name, "").strip()


def _require_env(name: str) -> str:
    value = _get_env(name)
    if not value:
        raise ValueError(f"Missing {name} in environment")
    return value


def update_env_value(env_path: Path, key: str, value: str) -> None:
    lines: list[str] = []
    found = False

    if env_path.exists():
        for line in env_path.read_text().splitlines():
            if not line or line.lstrip().startswith("#") or "=" not in line:
                lines.append(line)
                continue
            current_key, _ = line.split("=", 1)
            if current_key.strip() == key:
                lines.append(f"{key}={value}")
                found = True
            else:
                lines.append(line)

    if not found:
        lines.append(f"{key}={value}")

    env_path.parent.mkdir(parents=True, exist_ok=True)
    env_path.write_text("\n".join(lines) + "\n")


def exchange_code_for_token(auth_code: str) -> dict:
    client_id = _require_env("LINKEDIN_CLIENT_ID")
    client_secret = _require_env("LINKEDIN_CLIENT_SECRET")
    redirect_uri = _require_env("LINKEDIN_REDIRECT_URI")

    response = requests.post(
        "https://www.linkedin.com/oauth/v2/accessToken",
        data={
            "grant_type": "authorization_code",
            "code": auth_code,
            "redirect_uri": redirect_uri,
            "client_id": client_id,
            "client_secret": client_secret,
        },
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        timeout=30,
    )
    if not response.ok:
        raise RuntimeError(f"LinkedIn token exchange failed: {response.status_code} {response.text}")
    return response.json()


def persist_access_token(access_token: str) -> None:
    update_env_value(ENV_PATH, "LINKEDIN_ACCESS_TOKEN", access_token)
    os.environ["LINKEDIN_ACCESS_TOKEN"] = access_token


def get_access_token() -> str:
    return _require_env("LINKEDIN_ACCESS_TOKEN")


def get_headers(token: str) -> dict:
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json",
    }
    return headers


def _request(method: str, url: str, token: str, **kwargs) -> requests.Response:
    response = requests.request(
        method,
        url,
        headers=get_headers(token),
        timeout=30,
        **kwargs,
    )
    if not response.ok:
        print(f"LinkedIn API error {response.status_code}: {response.text}")
        raise RuntimeError(f"LinkedIn API error: {response.status_code} {response.text}")
    return response


def get_person_urn(token: str) -> str:
    response = _request("GET", f"{LINKEDIN_API_BASE}/v2/userinfo", token)
    data = response.json()
    person_id = data.get("sub", "")
    if not person_id:
        raise RuntimeError("LinkedIn userinfo response missing 'sub'")
    return f"urn:li:person:{person_id}"


def create_post(text: str) -> dict:
    token = get_access_token()
    person_urn = get_person_urn(token)

    payload = {
        "author": person_urn,
        "lifecycleState": "PUBLISHED",
        "specificContent": {
            "com.linkedin.ugc.ShareContent": {
                "shareCommentary": {"text": text},
                "shareMediaCategory": "NONE",
            }
        },
        "visibility": {"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC"},
    }

    response = _request(
        "POST",
        f"{LINKEDIN_API_BASE}/v2/ugcPosts",
        token,
        json=payload,
    )
    return response.json()


def read_comments(post_urn: str) -> dict:
    token = get_access_token()
    encoded_urn = quote(post_urn, safe="")
    response = _request(
        "GET",
        f"{LINKEDIN_API_BASE}/v2/socialActions/{encoded_urn}/comments",
        token,
    )
    return response.json()


def create_comment(post_urn: str, text: str, parent_comment_urn: str | None = None) -> dict:
    token = get_access_token()
    person_urn = get_person_urn(token)
    encoded_urn = quote(post_urn, safe="")

    payload = {
        "actor": person_urn,
        "message": {"text": text},
    }
    if parent_comment_urn:
        payload["parentComment"] = parent_comment_urn

    response = _request(
        "POST",
        f"{LINKEDIN_API_BASE}/v2/socialActions/{encoded_urn}/comments",
        token,
        json=payload,
    )
    return response.json()


def _append_csv_log(
    person: str,
    action: str,
    text: str,
    response: dict,
    post_urn: str | None = None,
    comment_urn: str | None = None,
) -> None:
    log_path = VAULT_ROOT / person / "linkedin" / "posts.csv"
    log_path.parent.mkdir(parents=True, exist_ok=True)
    is_new = not log_path.exists()

    row = {
        "timestamp": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
        "action": action,
        "post_urn": post_urn or "",
        "comment_urn": comment_urn or "",
        "text": text,
        "response": json.dumps(response, ensure_ascii=True),
    }

    with open(log_path, "a", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=CSV_HEADERS)
        if is_new:
            writer.writeheader()
        writer.writerow(row)


def log_post(person: str, text: str, response: dict) -> None:
    _append_csv_log(
        person=person,
        action="post",
        text=text,
        response=response,
        post_urn=response.get("id"),
    )


def log_comment(
    person: str,
    text: str,
    response: dict,
    post_urn: str,
    comment_urn: str | None = None,
    action: str = "comment",
) -> None:
    _append_csv_log(
        person=person,
        action=action,
        text=text,
        response=response,
        post_urn=post_urn,
        comment_urn=comment_urn or response.get("id"),
    )
