# threadterm

**Threads in your terminal.**

A hybrid TUI + CLI for [Meta Threads](https://www.threads.net) — feed, thread view, compose, search, and `--json` for agents. Built with Go, Bubble Tea, and Lip Gloss.

```
threadterm                 # open the TUI
threadterm feed --json     # agent-friendly feed
threadterm post "ship it"  # publish from the shell
threadterm login           # Meta OAuth (localhost)
```

![demo placeholder](assets/demo.gif)

> **Not affiliated with Meta.** Uses the official Threads Graph API when authenticated. Ships with a rich **demo mode** so you can try the TUI offline instantly.

---

## Why

| Client | Stars vibe | Gap |
|--------|------------|-----|
| Mobile / web Threads | huge | not for terminal people |
| [ndl](https://github.com/pgray/ndl) | small | multi-network, thin Threads UX |
| assorted CLIs | tiny | no polished TUI |

**threadterm** aims to be what [tut](https://github.com/RasmusLindroth/tut) is for Mastodon: a keyboard-first, beautiful, single-binary Threads client — plus a first-class CLI for scripts and agents.

## Install

### From source

```bash
go install github.com/arifaqyl/threadterm/cmd/threadterm@latest
```

Or:

```bash
git clone https://github.com/arifaqyl/threadterm
cd threadterm
go build -o threadterm ./cmd/threadterm
```

### Requirements

- Go 1.22+
- A terminal with truecolor (Windows Terminal, iTerm2, Kitty, Alacritty, …)

## Quick start (demo)

No Meta app needed:

```bash
threadterm --demo          # TUI
threadterm --demo feed
threadterm --demo post "hello from the terminal"
threadterm --demo search golang
threadterm --demo doctor
```

## Live mode (official API)

1. Create a Meta app with **Threads API** access: [developers.facebook.com](https://developers.facebook.com/docs/threads)
2. Add redirect URI `http://127.0.0.1:8765/callback`
3. Export credentials (or put them in `~/.threadterm/config.json`):

```bash
export THREADTERM_CLIENT_ID=...
export THREADTERM_CLIENT_SECRET=...
threadterm login
```

Or paste a token directly:

```bash
threadterm login --token "$TOKEN" --user-id "$USER_ID"
```

See [docs/AUTH.md](docs/AUTH.md) for scopes, limits, and honesty about what the official API can and cannot do.

## TUI keys

| Key | Action |
|-----|--------|
| `j` / `k` | move |
| `enter` | open thread |
| `c` | compose |
| `R` | reply |
| `L` | like |
| `p` | profile |
| `r` | refresh |
| `?` | help |
| `q` | quit / back |

## CLI

```bash
threadterm feed [--json] [-n 25]
threadterm thread <id> [--json]
threadterm post "text" [--json]
threadterm search "query" [--json]
threadterm profile <username> [--json]
threadterm like <id> [--unlike]
threadterm whoami [--json]
threadterm doctor
threadterm login | logout
threadterm version
```

## Architecture

```
cmd/threadterm          entrypoint
internal/cli            cobra commands
internal/tui            Bubble Tea UI
internal/api            demo adapter + Graph API client
internal/auth           OAuth localhost + token save
internal/config         ~/.threadterm/config.json
internal/demo           offline viral dataset
internal/models         shared types
```

## Agent / automation

Every read/write command accepts `--json`. Example:

```bash
threadterm feed --json | jq '.posts[0].text'
threadterm post "automated ship note" --json
```

Config via env:

| Variable | Purpose |
|----------|---------|
| `THREADTERM_ACCESS_TOKEN` | Graph API token |
| `THREADTERM_USER_ID` | Threads user id |
| `THREADTERM_CLIENT_ID` | OAuth app id |
| `THREADTERM_CLIENT_SECRET` | OAuth secret |
| `THREADTERM_DEMO=1` | force demo |

## Roadmap

- [x] Demo mode TUI (feed / thread / compose / profile / help)
- [x] CLI + `--json`
- [x] Official Graph API client (feed, publish, reply, like, search)
- [x] OAuth localhost login
- [ ] VHS cassette → polished `assets/demo.gif`
- [ ] Homebrew / scoop formula
- [ ] Notifications pane
- [ ] Image attach (URL) from compose
- [ ] Config themes

## Disclaimer

- Respect [Meta Platform Terms](https://developers.facebook.com/terms/) and Threads policies.
- Official API coverage is **not** a full consumer clone (no arbitrary public firehose without permissions).
- Demo mode is synthetic data for UX — not live Threads.

## License

MIT © Arif Aqyl ([@mindofaqyl](https://www.threads.net/@mindofaqyl))
