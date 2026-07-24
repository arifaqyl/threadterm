# Launch on X

---

**1/**

I built threadterm — Threads in your terminal.

Just:
```
threadterm login
```
reads your Threads session from Chrome/Edge/Brave/Firefox — just be logged
into threads.com in the browser first. No Meta app. No password.

TUI + CLI + --json for agents.
github.com/arifaqyl/threadterm

**2/**

```
go install github.com/arifaqyl/threadterm/cmd/threadterm@latest
threadterm --demo     # try offline
threadterm login      # then go live
threadterm
```

**3/**

Agents:
```
threadterm feed --json
threadterm post "shipped" --json
```

**4/**

Not affiliated with Meta. Browser-cookie login by default; password login can
hit 2FA/checkpoints — we document it.
Star if you want Homebrew next ⭐
