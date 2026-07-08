# Demo walkthrough (until VHS GIF is recorded)

```text
$ threadterm --demo
┌─────────────────────────────────────┐
│ threadterm          demo · ocean    │
│ Threads in your terminal            │
│                                     │
│ How to use                          │
│   j/k move · enter thread · c post  │
│   a login (cookies — no Meta app)   │
│                                     │
│ 1 enter  browse demo                │
│ 2 l      login with cookies         │
│ 3 t      theme                      │
└─────────────────────────────────────┘

$ threadterm login --cookies "sessionid=…; csrftoken=…; ds_user_id=…"
session saved · @you · mode=session

$ threadterm feed --json | head
{ "posts": [ { "username": "…", "text": "…" } ] }

$ threadterm post "shipped from the terminal" --json
{ "id": "…", "permalink": "https://www.threads.net/…" }
```

Record a real GIF later:

```bash
# needs https://github.com/charmbracelet/vhs
vhs docs/demo.tape
```
