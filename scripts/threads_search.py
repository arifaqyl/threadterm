#!/usr/bin/env python3
"""Scrape threads.com/search via Playwright (same lane as aduanmy)."""
from __future__ import annotations

import json
import os
import re
import sys
from datetime import datetime, timezone
from urllib.parse import quote_plus, unquote

try:
    from playwright.sync_api import sync_playwright
except ImportError:
    sync_playwright = None

SEARCH_POST_SELECTOR = "a[href*='/post/'], a[href*='/video/']"
SEARCH_POST_JS = """
els => els.map(a => {
  const href = a.href || '';
  if (!href || href.endsWith('/media')) return null;
  let node = a;
  let bestText = '';
  let bestLen = 0;
  for (let i = 0; i < 12 && node; i++, node = node.parentElement) {
    const text = (node.innerText || '').trim();
    const len = text.length;
    if (len >= 40 && len <= 400 && len > bestLen) {
      bestText = text;
      bestLen = len;
    }
  }
  const timeEl = a.querySelector('time') || a.closest('div')?.querySelector('time');
  return {
    href,
    preview_text: bestText,
    created_at: timeEl ? (timeEl.getAttribute('datetime') || '') : ''
  };
}).filter(Boolean)
"""

POST_URL_RE = re.compile(
    r"https?://(?:www\.)?threads\.com/@([^/]+)/(?:post|video)/([^/?#]+)"
)


def _cookies_from_session(session: dict) -> list[dict]:
    out = []
    mapping = [
        ("sessionid", True),
        ("csrftoken", False),
        ("ds_user_id", False),
        ("mid", False),
        ("ig_did", False),
    ]
    for name, http_only in mapping:
        value = session.get(name, "")
        if not value:
            continue
        out.append(
            {
                "name": name,
                "value": value,
                "domain": ".threads.com",
                "path": "/",
                "httpOnly": http_only,
                "secure": True,
                "sameSite": "Lax",
            }
        )
    return out


def _parse_post_url(href: str) -> tuple[str, str]:
    m = POST_URL_RE.search(href or "")
    if not m:
        return "", ""
    return m.group(2), m.group(1)


def _parse_ts(raw: str) -> str:
    raw = (raw or "").strip()
    if not raw:
        return ""
    try:
        if raw.endswith("Z"):
            raw = raw[:-1] + "+00:00"
        dt = datetime.fromisoformat(raw)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")
    except ValueError:
        return ""


def search_posts(session: dict, query: str, limit: int = 25) -> list[dict]:
    if sync_playwright is None:
        raise RuntimeError("playwright not installed — pip install playwright && playwright install chromium")
    cookies = _cookies_from_session(session)
    if not any(c["name"] == "sessionid" for c in cookies):
        raise RuntimeError("missing sessionid cookie")

    rows: list[dict] = []
    seen: set[str] = set()
    search_url = f"https://www.threads.com/search?q={quote_plus(query)}&filter=recent"

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(viewport={"width": 1280, "height": 2200})
        context.add_cookies(cookies)
        page = context.new_page()
        page.goto(search_url, wait_until="domcontentloaded", timeout=35000)
        page.wait_for_timeout(1800)
        for _ in range(3):
            batch = page.locator(SEARCH_POST_SELECTOR).evaluate_all(SEARCH_POST_JS)
            for item in batch:
                href = item.get("href", "")
                if not href or href in seen:
                    continue
                seen.add(href)
                post_id, username = _parse_post_url(href)
                text = (item.get("preview_text") or "").strip()
                if not post_id or not username or len(text) < 8:
                    continue
                rows.append(
                    {
                        "id": post_id,
                        "username": username,
                        "text": text,
                        "url": href,
                        "timestamp": _parse_ts(item.get("created_at", "")),
                    }
                )
            if len(rows) >= limit:
                break
            page.mouse.wheel(0, 2400)
            page.wait_for_timeout(600)
        context.close()
        browser.close()

    return rows[:limit]


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        print(json.dumps({"error": f"invalid stdin json: {exc}"}))
        return 2

    session = payload.get("session") or {}
    query = (payload.get("query") or "").strip()
    limit = int(payload.get("limit") or 25)
    if not query:
        print(json.dumps({"error": "empty query"}))
        return 2

    try:
        posts = search_posts(session, query, limit=limit)
        print(json.dumps({"posts": posts, "source": "search-browser"}))
        return 0
    except Exception as exc:
        print(json.dumps({"error": str(exc)}))
        return 1


if __name__ == "__main__":
    if os.name == "nt":
        try:
            sys.stdout.reconfigure(encoding="utf-8", errors="replace")
            sys.stderr.reconfigure(encoding="utf-8", errors="replace")
        except Exception:
            pass
    raise SystemExit(main())
