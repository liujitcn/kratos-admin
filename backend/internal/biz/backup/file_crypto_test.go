package backup

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestFileCryptoGoFallback 验证外部命令不存在时使用 Go 完成文件往返。
func TestFileCryptoGoFallback(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source.bin")
	encrypted := filepath.Join(directory, "encrypted.bin")
	decrypted := filepath.Join(directory, "decrypted.bin")
	content := []byte("kratos-admin backup fallback\x00\n")
	if err := os.WriteFile(source, content, 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := withCommandUnavailable(t, func() error {
		return EncryptFile(context.Background(), "backup-password", source, encrypted)
	}); err != nil {
		t.Fatalf("EncryptFile() error = %v", err)
	}
	if err := withCommandUnavailable(t, func() error {
		return DecryptFile(context.Background(), "backup-password", encrypted, decrypted)
	}); err != nil {
		t.Fatalf("DecryptFile() error = %v", err)
	}
	assertFileContent(t, decrypted, content)
}

// TestFileCryptoCommandToGoFallback 验证 OpenSSL 加密文件可以由 Go fallback 解密。
func TestFileCryptoCommandToGoFallback(t *testing.T) {
	opensslPath, err := exec.LookPath("openssl")
	if err != nil {
		t.Skipf("openssl is unavailable: %v", err)
	}
	directory := t.TempDir()
	source := filepath.Join(directory, "source.bin")
	encrypted := filepath.Join(directory, "encrypted.bin")
	decrypted := filepath.Join(directory, "decrypted.bin")
	content := []byte("OpenSSL to Go backup fallback")
	if err = os.WriteFile(source, content, 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	command := exec.Command(opensslPath, "enc", "-aes-256-cbc", "-pbkdf2", "-iter", "100000", "-md", "sha256", "-salt", "-saltlen", "8", "-in", source, "-out", encrypted, "-pass", "pass:backup-password")
	if output, runErr := command.CombinedOutput(); runErr != nil {
		t.Fatalf("openssl encryption: %v: %s", runErr, output)
	}
	if err = withCommandUnavailable(t, func() error {
		return DecryptFile(context.Background(), "backup-password", encrypted, decrypted)
	}); err != nil {
		t.Fatalf("DecryptFile() error = %v", err)
	}
	assertFileContent(t, decrypted, content)
}

// withCommandUnavailable 在测试中临时隐藏 openssl 命令以验证 Go fallback。
func withCommandUnavailable(t *testing.T, action func() error) error {
	t.Helper()
	path := os.Getenv("PATH")
	t.Setenv("PATH", t.TempDir())
	defer func() {
		_ = os.Setenv("PATH", path)
	}()
	return action()
}

// assertFileContent 验证指定文件内容。
func assertFileContent(t *testing.T, path string, expected []byte) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(actual) != string(expected) {
		t.Fatalf("file %s does not match expected content", path)
	}
}
