package sync_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"td-file/parser"
	"td-file/sync"
)

func TestFileSynchronizer_StartStop(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "todos.md")
	if err := os.WriteFile(file, []byte(":td\n- [ ] Test\n:td\n"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	fs := sync.NewFileSynchronizer(file)
	if err := fs.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	fs.Stop()
}

func TestFileSynchronizer_SaveChWrites(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "todos.md")
	if err := os.WriteFile(file, []byte(":td\n- [ ] Old\n:td\n"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	fs := sync.NewFileSynchronizer(file)
	if err := fs.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer fs.Stop()
	todo := parser.Todo{ID: 1, Text: "New todo", State: parser.Incomplete, IndentLevel: 0, LineNumber: 1}
	fs.SaveCh <- []parser.Todo{todo}
	// Wait for goroutine to write
	time.Sleep(100 * time.Millisecond)
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) == ":td\n- [ ] Old\n:td\n" {
		t.Errorf("file was not updated by SaveCh")
	}
	if !contains(string(content), "New todo") {
		t.Errorf("file does not contain new todo: %q", string(content))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > 0 && (contains(s[1:], substr) || contains(s[:len(s)-1], substr)))) || (len(substr) == 0)
}
