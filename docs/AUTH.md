# Auth

## Normal login (do this)

```bash
threadterm login
```

It asks for:

```
username:
password:
2FA secret (optional, Enter to skip):
```

That’s it. No Meta developer app. No DevTools. No cookie paste.

Then:

```bash
threadterm            # TUI — home feed
threadterm feed
threadterm post "hi"
```

Or non-interactive:

```bash
threadterm login --user you --password 'secret'
# with authenticator 2FA:
threadterm login --user you --password 'secret' --totp YOUR_APP_SECRET
```

In the TUI: press **`a`** → **1** → type username + password.

---

## What you get

| After password login | Works? |
|----------------------|--------|
| Home / For You feed | yes |
| Post | yes |
| Like / reply | yes |
| Search / other profiles | better with optional cookies |

---

## Optional: cookies

Only if you want richer profile search:

```bash
threadterm login --cookies "sessionid=…; csrftoken=…; ds_user_id=…"
```

## Optional: official Graph API

Only if you already have a Meta Threads app — see older docs / `--token`.

---

## If login fails

- **2FA** → pass `--totp` with your authenticator app secret  
- **Checkpoint / suspicious login** → Meta blocked the device; try again later, same network as your phone, or approve the login in the Instagram app  
- **Wrong password** → use your Instagram password (Threads shares it)

Sessions are saved in `~/.threadterm/config.json` (mode `0600`).
