# threadterm

**Threads in your terminal.** Just `threadterm login` — username + password.

Hybrid TUI + CLI. No Meta developer app. Built with Go + Bubble Tea.

```bash
threadterm login          # username + password (normal)
threadterm                # TUI
threadterm feed --json    # agents
threadterm --demo         # try offline first
```

> **Not affiliated with Meta.** Demo works offline. Live mode uses your session cookies (optional Bloks password for posting). Official Graph API is optional.

---

## Why this isn’t lame

| Project | Stars | Gap |
|---------|-------|-----|
| [ndl](https://github.com/pgray/ndl) | ~9 | thin multi-network TUI |
| [yarn-threads-cli](https://github.com/jeizzon/yarn-threads-cli) | ~25 | CLI only, no polished TUI |
| Official Graph API apps | — | need Meta developer approval |

**threadterm** = tut-style TUI + agent CLI + cookie auth + themes + in-app login.

---

## How to use (TUI)

| Key | Action |
|-----|--------|
| `j` / `k` | move |
| `enter` | open thread |
| `c` | compose |
| `R` / `L` | reply / like |
| `a` | **login** (cookies / write / demo) |
| `t` / `T` | theme picker / cycle |
| `?` | help |
| `q` | quit |

### Login

```bash
threadterm login
# username:
# password:
```

Or in TUI press **`a`** → username + password.

Optional cookies / official Graph API: [docs/AUTH.md](docs/AUTH.md)

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

Windows (this repo):

```powershell
cd D:\threadterm
go build -o threadterm.exe ./cmd/threadterm
D:\threadterm\threadterm.exe --demo
```

---

## CLI

```bash
threadterm login --cookies "sessionid=…; csrftoken=…; ds_user_id=…"
threadterm feed --json
threadterm post "hello"
threadterm search golang
threadterm profile zuck
threadterm whoami
threadterm doctor
threadterm theme orchid
```

Env: `THREADS_SESSIONID`, `THREADS_CSRFTOKEN`, `THREADS_DS_USER_ID`, `THREADS_MID`, `THREADS_IG_DID`, `THREADTERM_THEME`, `THREADTERM_DEMO=1`

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
