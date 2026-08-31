package admin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotateBackups(t *testing.T) {
	t.Setenv("BACKUP_RETENTION_COUNT", "2")
	directory := t.TempDir()
	for _, name := range []string{"kratos-admin-20260101.sql.gz", "kratos-admin-20260102.sql.gz", "kratos-admin-20260103.sql.gz"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("backup"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := rotateBackups(directory); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two retained backups, got %d", len(entries))
	}
}

func TestWriteBackupChecksum(t *testing.T) {
	t.Setenv("BACKUP_INTEGRITY_KEY", "test-integrity-key")
	directory := t.TempDir()
	path := filepath.Join(directory, "kratos-admin-test.sql.gz")
	if err := os.WriteFile(path, []byte("backup"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeBackupChecksum(path); err != nil {
		t.Fatal(err)
	}
	checksum, err := os.ReadFile(path + ".sha256")
	if err != nil {
		t.Fatal(err)
	}
	if len(checksum) < 64 {
		t.Fatalf("expected SHA-256 checksum, got %q", checksum)
	}
	if _, err = os.Stat(path + ".hmac"); err != nil {
		t.Fatalf("expected HMAC sidecar: %v", err)
	}
}
