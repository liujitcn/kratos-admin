package admin

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestArchiveRowsRetryDoesNotDuplicate 验证删除失败后的同批次重试不会重复追加归档记录。
func TestArchiveRowsRetryDoesNotDuplicate(t *testing.T) {
	t.Setenv("AUDIT_ARCHIVE_INTEGRITY_KEY", "test-integrity-key")
	directory := t.TempDir()
	pathPrefix := filepath.Join(directory, "base_login_log")
	rows := []any{map[string]any{"id": 1}, map[string]any{"id": 2}}
	ids := []int64{1, 2}
	total := 0
	task := &AuditRetentionTask{}
	deleteFailed := func([]int64) error { return errors.New("delete failed") }
	err := task.archiveRows(rows, ids, deleteFailed, pathPrefix, &total)
	if err == nil {
		t.Fatal("first archive should report delete failure")
	}
	if total != 0 {
		t.Fatalf("failed deletion changed archived total to %d", total)
	}
	deleteSucceeded := func([]int64) error { return nil }
	if err = task.archiveRows(rows, ids, deleteSucceeded, pathPrefix, &total); err != nil {
		t.Fatal(err)
	}
	var paths []string
	paths, err = filepath.Glob(pathPrefix + "-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected one deterministic archive file, got %d", len(paths))
	}
	var content []byte
	content, err = os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(content), "\n"); lines != len(rows) {
		t.Fatalf("expected %d archived rows, got %d", len(rows), lines)
	}
	if total != len(rows) {
		t.Fatalf("expected archived total %d, got %d", len(rows), total)
	}
	if _, err = os.Stat(paths[0] + ".sha256"); err != nil {
		t.Fatalf("expected SHA-256 sidecar: %v", err)
	}
	if _, err = os.Stat(paths[0] + ".hmac"); err != nil {
		t.Fatalf("expected HMAC sidecar: %v", err)
	}
}
