# Auth

## Reality check (Jul 2026)

Meta often **blocks password login from new devices/CLIs** with:

> Unable to log in / An unexpected error occurred

That is **not** “wrong password”. It’s a risk/checkpoint response with no token.

So threadterm supports two paths:

| Path | Command | Reliability |
|------|---------|-------------|
| **Cookies (recommended when password fails)** | `threadterm login --cookies` | high |
| Password | `threadterm login` | works until Meta blocks the device |
| Official Graph | `--token` | needs Meta developer app |

---

## Easy login (do this now)

```powershell
D:\threadterm\threadterm.exe login --cookies
```

It opens Threads and asks you to paste:

1. Open https://www.threads.com (already logged in)
2. Press **F12** → **Application** → **Cookies** → `www.threads.com`
3. Copy **sessionid**, **csrftoken**, **ds_user_id** (mid + ig_did nice to have)
4. Paste when prompted

Then:

```powershell
D:\threadterm\threadterm.exe
D:\threadterm\threadterm.exe feed
```

---

## Password login

```powershell
D:\threadterm\threadterm.exe login
```

If it fails, it offers to switch to cookie login automatically.

2FA:

```powershell
threadterm login --user YOU --password '…' --totp YOUR_AUTH_SECRET
```

---

## Security

- Never paste passwords into chat.
- If you did, **change the Instagram/Threads password now**.
- `~/.threadterm/config.json` is mode `0600` — don’t commit it.
