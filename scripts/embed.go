// Package scripts embeds the Python helper scripts so that features which
// depend on them keep working from `go install` and release archives, where
// the on-disk scripts/ tree is not present.
package scripts

import (
	_ "embed"
	"os"
	"sync"
)

//go:embed threads_search.py
var searchScriptBytes []byte

var (
	scriptOnce sync.Once
	scriptPath string
	scriptErr  error
)

// SearchScriptPath materializes the embedded threads_search.py to a temp file
// once (per process) and returns its path. Callers run it with the user's
// Python; the script itself still requires Python 3 + Playwright at runtime.
func SearchScriptPath() (string, error) {
	scriptOnce.Do(func() {
		f, err := os.CreateTemp("", "threadterm-threads-search-*.py")
		if err != nil {
			scriptErr = err
			return
		}
		if _, err := f.Write(searchScriptBytes); err != nil {
			scriptErr = err
			_ = f.Close()
			_ = os.Remove(f.Name())
			return
		}
		if err := f.Close(); err != nil {
			scriptErr = err
			_ = os.Remove(f.Name())
			return
		}
		scriptPath = f.Name()
	})
	return scriptPath, scriptErr
}
