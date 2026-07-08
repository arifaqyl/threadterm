# Auth

threadterm supports two modes.

## Demo (default)

No credentials. Offline synthetic feed for trying the TUI and CLI.

```bash
threadterm --demo
# or
export THREADTERM_DEMO=1
```

## Live — Meta Threads Graph API

Uses the **official** Threads API (`graph.threads.net`). This is the supported path for publishing and reading **your** media.

### 1. Create a Meta app

1. Open [Meta Developer Console](https://developers.facebook.com/)
2. Create an app and add the **Threads** product
3. Note **Threads App ID** and **Threads App Secret**
4. Under Threads → settings, add OAuth redirect:
   - `http://127.0.0.1:8765/callback`

### 2. Scopes threadterm requests

```
threads_basic
threads_content_publish
threads_read_replies
threads_manage_replies
threads_manage_insights
threads_keyword_search
```

Keyword search and some reply features require app review / advanced access depending on Meta’s current policy. Without them, `search` falls back to filtering your own threads.

### 3. Login

```bash
export THREADTERM_CLIENT_ID=your_threads_app_id
export THREADTERM_CLIENT_SECRET=your_threads_app_secret
threadterm login
```

Browser opens → approve → token saved to `~/.threadterm/config.json` (mode `0600`).

### Manual token

```bash
threadterm login --token "THQV..." --user-id "1234567890"
```

Or env vars (override file):

```bash
export THREADTERM_ACCESS_TOKEN=...
export THREADTERM_USER_ID=...
```

### Check

```bash
threadterm whoami
threadterm doctor
```

### Logout

```bash
threadterm logout
```

## What the official API cannot do

Be honest in launch posts:

| Want | Official API |
|------|----------------|
| Your posts / publish | ✅ |
| Replies to your threads | ✅ (with scopes) |
| Keyword search | ✅ (with `threads_keyword_search`) |
| Full “For You” firehose | ❌ not exposed like the app |
| Arbitrary public scrape | ❌ / against ToS |

threadterm will **not** ship cookie-scraping as the default path. If an experimental public-browse backend is added later, it will be opt-in, documented, and clearly marked as unofficial / ToS-risk.

## Security

- Never commit `~/.threadterm/config.json` or `.env`
- Prefer long-lived tokens from the OAuth exchange threadterm performs
- Rotate secrets if leaked
