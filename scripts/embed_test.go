package scripts

import (
	"os"
	"testing"
)

func TestSearchScriptPathMaterializesEmbeddedScript(t *testing.T) {
	path, err := SearchScriptPath()
	if err != nil {
		t.Fatalf("SearchScriptPath: %v", err)
	}
	if path == "" {
		t.Fatal("empty script path")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("materialized script not found on disk: %v", err)
	}
}
