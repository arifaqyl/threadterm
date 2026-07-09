# threadterm

[![CI](https://github.com/arifaqyl/threadterm/actions/workflows/ci.yml/badge.svg)](https://github.com/arifaqyl/threadterm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/arifaqyl/threadterm)](https://github.com/arifaqyl/threadterm/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Threads in your terminal.** Login like bird — auto from Chrome.

```bash
threadterm login          # reads Threads session from your browser
threadterm                # TUI
threadterm feed --json    # agents
threadterm --demo         # try offline first
```

> **Not affiliated with Meta.** Be logged into threads.com in Chrome/Firefox first. Official Graph API is optional.

---

## Why threadterm

- Browser-cookie login by default (no Meta developer app required)
- TUI + scriptable JSON CLI in one tool
- Search, latest, profile, thread views for automation workflows
- Themeable terminal UX with keyboard-first navigation

---

## How to use (TUI)

| Key | Action |
|-----|--------|
| `j` / `k` | move |
| `/` | **search** users |
| `y` | copy link/text to clipboard |
| `f` / `r` | your following feed |
| `d` | public discover (not your feed) |
| `enter` | open thread |
| `c` | compose |
| `R` / `L` | reply / like |
| `a` | **login** (cookies / write / demo) |
| `t` / `T` | theme picker / cycle |
| `?` | help |
| `q` | quit |

Mouse select/copy works in the terminal (mouse capture is off). Use `y` to yank the selected post link.

### Login

```bash
# 1. Log into https://www.threads.com in Chrome
# 2. Then:
threadterm login
```

Same UX as [bird](https://github.com/steipete/bird) for Twitter — auto cookie extract, no DevTools.

Details: [docs/AUTH.md](docs/AUTH.md)

### Themes

`jade` · `ocean` · `ember` · `mono` · `orchid`

```bash
threadterm theme ocean
```

---

## Install

```bash
go install github.com/arifaqyl/threadterm/cmd/threadterm@latest
```

Prebuilt binaries (Windows/macOS/Linux) are published on [GitHub Releases](https://github.com/arifaqyl/threadterm/releases).

Windows (from source):

```powershell
git clone https://github.com/arifaqyl/threadterm.git
cd threadterm
go build -o threadterm.exe ./cmd/threadterm
.\threadterm.exe --demo
```

---

## CLI (agent-friendly)

```bash
threadterm login                 # auto from Chrome/Firefox
threadterm status --json
threadterm feed --json -n 25     # your following feed
threadterm feed --discover       # public sample (opt-in)
threadterm search "LRT KL" --json        # post search (needs Python + Playwright)
threadterm search-users "myrapidkl"      # account search
threadterm latest zuck --json -n 10
threadterm profile mosseri --json
threadterm whoami --json
threadterm doctor
```

Automation guide: [docs/AGENTS.md](docs/AGENTS.md)

Env: `THREADS_SESSIONID`, `THREADS_CSRFTOKEN`, `THREADS_DS_USER_ID`, `THREADS_MID`, `THREADS_IG_DID`, `THREADTERM_THEME`, `THREADTERM_DEMO=1`

**Post search** uses headless Chromium via Playwright (same approach as TrafficMY):

```bash
pip install playwright
playwright install chromium
```

---

## Architecture

```
demo          → offline synthetic feed
session       → browser cookies (primary live)
session+write → cookies + Bloks bearer (post/like)
token         → official Graph API (optional)
```

Powered under the hood by [threads-go](https://github.com/teslashibe/threads-go) for the private session surface.

---

## Disclaimer

Respect Meta’s terms. Cookie/private APIs can break. Don’t abuse rate limits. Use your own account.

## License

MIT © Arif Aqyl
