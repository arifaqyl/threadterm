# VHS demo cassette
# https://github.com/charmbracelet/vhs
# Run: vhs docs/demo.tape

Output assets/demo.gif

Set FontSize 15
Set Width 1280
Set Height 720
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 35ms

Hide
Type "clear"
Enter
Show

Type "# threadterm — Threads in your terminal (no Meta app)"
Enter
Sleep 800ms

Type "./threadterm --demo whoami"
Enter
Sleep 1s

Type "./threadterm --demo feed -n 4"
Enter
Sleep 2s

Type "./threadterm --demo post \"hello from threadterm\" --json"
Enter
Sleep 1.5s

Type "./threadterm theme ocean"
Enter
Sleep 1s

Type "# press a inside TUI to paste threads.com cookies"
Enter
Sleep 1s

Type "./threadterm --demo"
Enter
Sleep 3s
Type "1"
Sleep 2s
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "?"
Sleep 2s
Type "q"
Sleep 400ms
Type "t"
Sleep 1.5s
Type "q"
Sleep 400ms
Type "a"
Sleep 2s
Type "q"
Sleep 400ms
Type "q"
Sleep 1s
