# Auth — no Meta developer app required

threadterm’s **primary** live path is the same idea as Twitter/X CLIs (bird, etc.):
use **your browser session cookies**. Official Graph API is optional.

## Modes

| Mode | Needs | Can do |
|------|-------|--------|
| **demo** | nothing | full TUI offline |
| **session** | browser cookies | read feed / profiles / threads |
| **session+write** | cookies + password (Bloks) | also post / like / reply |
| **token** | Meta app + Graph token | official API (optional) |

---

## 1. Cookie login (recommended)

1. Open [https://www.threads.com](https://www.threads.com) while logged in  
2. DevTools → **Application** → **Cookies** → `https://www.threads.com`  
3. Copy these values (paste as one Cookie string):

| Cookie | Required |
|--------|----------|
| `sessionid` | yes |
| `csrftoken` | yes |
| `ds_user_id` | yes |
| `mid` | recommended |
| `ig_did` | recommended |

### CLI

```bash
threadterm login --cookies "sessionid=...; csrftoken=...; ds_user_id=...; mid=...; ig_did=..."
```

Or flags:

```bash
threadterm login --session-id "..." --csrf "..." --ds-user-id "..." --mid "..." --ig-did "..."
```

Env vars also work:

```bash
export THREADS_SESSIONID=...
export THREADS_CSRFTOKEN=...
export THREADS_DS_USER_ID=...
export THREADS_MID=...
export THREADS_IG_DID=...
```

### TUI

Press **`a`** → **1 / c** → paste → enter.

---

## 2. Write login (post / like / reply)

Cookies alone are **read**. To publish:

```bash
threadterm login --user yourname --password 'yourpassword'
```

Or in TUI: **`a`** → **2 / w**.

This uses Instagram’s Bloks login (same private surface mobile apps use).  
It may hit **2FA / checkpoint**. If so, stay on cookies for reading and post from the app until we add challenge handling.

---

## 3. Official Graph API (optional)

Only if you already have a Meta Threads app:

```bash
export THREADTERM_CLIENT_ID=...
export THREADTERM_CLIENT_SECRET=...
threadterm login
# or
threadterm login --token "..." --user-id "..."
```

---

## Agent / automation

```bash
threadterm feed --json
threadterm post "shipped from agent" --json
threadterm whoami --json
```

Prefer env cookies in CI secrets — never commit `~/.threadterm/config.json`.

---

## Honesty / risk

- Cookie + private API paths can break when Meta changes endpoints.
- Against Meta ToS if abused; use your own account, pace requests, don’t scrape at scale.
- Sessions expire — re-paste cookies when `doctor` / feed fails.
- Official API is stabler but needs a developer app (which most people don’t have).

threadterm defaults to **demo** so anyone can try the TUI with zero setup.
