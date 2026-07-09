# Agent / automation

threadterm is meant to be called by agents the same way
[twitter-cli](https://github.com/public-clis/twitter-cli) is used under Agent-Reach.

## Setup (once)

```bash
# Be logged into https://www.threads.com in Chrome/Firefox
threadterm login
threadterm status --json
```

## Commands agents should use

```bash
threadterm feed --json -n 25
threadterm feed --discover --json -n 10
threadterm search "malaysia transit" --json -n 20
threadterm latest zuck --json -n 10
threadterm profile mosseri --json
threadterm thread <id> --json
threadterm whoami --json
threadterm status --json
```

## Watch pattern (like train scraping)

Poll a user or topic:

```bash
# latest posts from an account
threadterm latest ktmb_berhad --json -n 15

# topic discovery via user search + their recent posts
threadterm search "LRT" --json -n 30
```

Pipe into jq:

```bash
threadterm search "traffic KL" --json | jq -r '.posts[] | "@\(.username): \(.text)"'
```

## Auth notes

- Cookie session (default after `login`) → read feed / search / latest / profiles
- Password/Bearer (`--password-login`) → also post / like / reply (Meta often blocks)
- Never commit `~/.threadterm/config.json`
