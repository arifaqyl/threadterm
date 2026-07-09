package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/arifaqyl/threadterm/internal/config"
	"github.com/arifaqyl/threadterm/internal/models"
)

type searchRequest struct {
	Session config.SessionCookies `json:"session"`
	Query   string                `json:"query"`
	Limit   int                   `json:"limit"`
}

type searchResponse struct {
	Posts  []searchPost `json:"posts"`
	Source string       `json:"source"`
	Error  string       `json:"error"`
}

type searchPost struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	URL       string `json:"url"`
	Timestamp string `json:"timestamp"`
}

// SearchPosts renders threads.com/search via Playwright (requires python + playwright).
func SearchPosts(ctx context.Context, cfg *config.Config, query string, limit int) ([]models.Post, error) {
	if cfg == nil || !cfg.HasSession() {
		return nil, fmt.Errorf("browser search needs session cookies")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}
	if limit <= 0 {
		limit = 25
	}

	script, err := searchScriptPath()
	if err != nil {
		return nil, err
	}
	python, err := pythonBin()
	if err != nil {
		return nil, err
	}

	reqBody, err := json.Marshal(searchRequest{
		Session: cfg.Session,
		Query:   query,
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, python, script)
	cmd.Stdin = bytes.NewReader(reqBody)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("browser search: %s", msg)
	}

	var resp searchResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("browser search json: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("browser search: %s", resp.Error)
	}

	out := make([]models.Post, 0, len(resp.Posts))
	for _, p := range resp.Posts {
		if p.ID == "" || p.Username == "" || strings.TrimSpace(p.Text) == "" {
			continue
		}
		ts := time.Now().UTC()
		if p.Timestamp != "" {
			if parsed, err := time.Parse(time.RFC3339, p.Timestamp); err == nil {
				ts = parsed.UTC()
			}
		}
		out = append(out, models.Post{
			ID:        p.ID,
			Username:  p.Username,
			Text:      p.Text,
			Timestamp: ts,
		})
	}
	return out, nil
}

func pythonBin() (string, error) {
	if p := strings.TrimSpace(os.Getenv("THREADTERM_PYTHON")); p != "" {
		return p, nil
	}
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("python not found (install Python 3 + playwright: pip install playwright && playwright install chromium)")
}

func searchScriptPath() (string, error) {
	candidates := []string{
		"scripts/threads_search.py",
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "scripts", "threads_search.py"))
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
		candidates = append(candidates, filepath.Join(root, "scripts", "threads_search.py"))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "scripts", "threads_search.py"))
	}

	seen := map[string]bool{}
	for _, c := range candidates {
		c = filepath.Clean(c)
		if seen[c] {
			continue
		}
		seen[c] = true
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("scripts/threads_search.py not found")
}
