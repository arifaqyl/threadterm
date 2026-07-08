# threadterm

**Threads in your terminal.**

A hybrid TUI + CLI for [Meta Threads](https://www.threads.net) — feed, thread view, compose, themes, in-app login, and `--json` for agents. Built with Go, Bubble Tea, and Lip Gloss.

```bash
D:\threadterm\threadterm.exe --demo   # Windows
threadterm --demo                     # after install
```

> **Not affiliated with Meta.** Demo mode works offline. Live mode uses the official Threads Graph API.

---

## Is this already done by someone else?

Short answer: **the niche is still open.**

| Project | What it is | Stars (approx) |
|---------|------------|----------------|
| [ndl](https://github.com/pgray/ndl) | Threads + Bluesky TUI | ~9 |
| [yarn-threads-cli](https://github.com/jeizzon/yarn-threads-cli) | Cookie CLI for agents | ~25 |
| assorted CLIs | thin / early | ~0–25 |
| [tut](https://github.com/RasmusLindroth/tut) | Mastodon TUI (inspiration) | ~500 |

Nobody has shipped the “tut for Threads” yet — polished single-binary TUI **plus** agent CLI, with honest official-API auth and a zero-setup demo. That’s the bet.

---

## How to use (TUI)

First launch shows a **welcome screen** with the basics. Then:

| Key | Action |
|-----|--------|
| `j` / `k` | move |
| `enter` | open thread |
| `c` | compose |
| `R` | reply |
| `L` | like |
| `p` | profile |
| `a` | **login** (token / OAuth / demo) |
| `t` | **theme picker** |
| `T` | cycle theme |
| `r` | refresh |
| `?` | help |
| `q` | quit / back |

Sidebar (wide terminals): nav + status + selected post.

### Themes

`jade` · `ocean` · `ember` · `mono` · `orchid`

```bash
threadterm theme ocean
# or press t inside the TUI
```

### Login (inside the TUI)

Press **`a`**:

1. **Paste token** — Graph API access token + user id  
2. **OAuth** — browser callback (needs Meta app credentials)  
3. **Demo** — stay offline  

Full auth guide: [docs/AUTH.md](docs/AUTH.md)

---

## Install

```bash
go install github.com/arifaqyl/threadterm/cmd/threadterm@latest
```

Or from this repo:

```bash
cd D:\threadterm
go build -o threadterm.exe ./cmd/threadterm
.\threadterm.exe --demo
```

---

## CLI

```bash
threadterm                  # TUI
threadterm feed --json
threadterm post "ship it"
threadterm search golang
threadterm profile arifaqyl
threadterm like <id>
threadterm login            # or press a in TUI
threadterm theme ocean
threadterm whoami
threadterm doctor
```

Env vars: `THREADTERM_ACCESS_TOKEN`, `THREADTERM_USER_ID`, `THREADTERM_CLIENT_ID`, `THREADTERM_CLIENT_SECRET`, `THREADTERM_THEME`, `THREADTERM_DEMO=1`

Config file: `~/.threadterm/config.json`

---

## Architecture

```
cmd/threadterm       entrypoint
internal/cli         cobra (feed, post, login, theme, …)
internal/tui         Bubble Tea (welcome, feed, login, themes)
internal/api         demo adapter + Graph API
internal/auth        OAuth localhost + token save
internal/config      theme, welcome flag, credentials
internal/demo        offline dataset
```

---

## Roadmap

- [x] Demo TUI + CLI + `--json`
- [x] Welcome / how-to onboarding
- [x] Themes + in-TUI login
- [x] Official Graph API + OAuth
- [ ] VHS `assets/demo.gif`
- [ ] Homebrew / scoop
- [ ] Notifications pane
- [ ] Image URL attach

---

## License

MIT © Arif Aqyl ([@mindofaqyl](https://www.threads.net/@mindofaqyl))
