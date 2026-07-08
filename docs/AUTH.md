# Auth

## Default — like bird (Twitter CLI)

```powershell
D:\threadterm\threadterm.exe login
```

That’s it.

1. Be logged into https://www.threads.com in Chrome / Edge / Brave / Firefox  
2. Run `threadterm login`  
3. It **auto-reads** `sessionid` / `csrftoken` / `ds_user_id` from your browser  

No Meta developer app. No DevTools. No password paste.

Powered by the same approach as [`bird`](https://github.com/steipete/bird) → [`sweetcookie`](https://github.com/steipete/sweetcookie).

---

## Other options

| Command | When |
|---------|------|
| `threadterm login` | default — auto browser |
| `threadterm login --password-login` | try username+password (Meta often blocks) |
| `threadterm login --cookies` | guided manual paste |
| `threadterm login --user X --password Y` | non-interactive password |

In the TUI: press **`a`** → **1** auto from browser.

---

## Why password failed earlier

Meta returned *“Unable to log in / unexpected error”* with **no token**.  
That’s a device/risk block, not wrong password. Browser-session login sidesteps it — same reason Twitter CLIs never ask for your X password.

---

## Security

- Never paste passwords into chat.  
- Cookies stay in `~/.threadterm/config.json` (mode `0600`).  
- `threadterm logout` clears them.
