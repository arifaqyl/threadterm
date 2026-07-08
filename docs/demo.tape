# VHS demo cassette
# https://github.com/charmbracelet/vhs
# Run: vhs docs/demo.tape

Output assets/demo.gif

Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Hide
Type "clear"
Enter
Type "./threadterm --demo feed -n 5"
Enter
Sleep 1s
Show

Sleep 2s
Type "./threadterm --demo whoami"
Enter
Sleep 1s

Type "./threadterm --demo post \"hello from threadterm\" --json"
Enter
Sleep 1.5s

Type "# open the TUI — press ? for help, q to quit"
Enter
Sleep 1s
Type "./threadterm --demo"
Enter
Sleep 4s
Type "?"
Sleep 2s
Type "q"
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "enter"
Sleep 2s
Type "q"
Sleep 500ms
Type "q"
Sleep 1s
