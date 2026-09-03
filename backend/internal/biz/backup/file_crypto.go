package backup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	utilscrypto "github.com/liujitcn/go-utils/crypto"
)

// EncryptFile 使用 OpenSSL 命令优先、Go 实现兜底的方式加密文件。
func EncryptFile(ctx context.Context, encryptionKey, source, target string) error {
	if CommandAvailable(OpensslCommand) {
		return encryptFileByCommand(ctx, encryptionKey, source, target)
	}
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("打开待加密备份失败: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建加密备份文件失败: %w", err)
	}
	fileCrypto := utilscrypto.NewOpenSSLFileCrypto()
	if err = fileCrypto.EncryptReader(encryptionKey, input, output); err != nil {
		_ = output.Close()
		return fmt.Errorf("Go 加密备份失败: %w", err)
	}
	if err = output.Sync(); err != nil {
		_ = output.Close()
		return fmt.Errorf("同步加密备份失败: %w", err)
	}
	if err = output.Close(); err != nil {
		return fmt.Errorf("关闭加密备份文件失败: %w", err)
	}
	return nil
}

// DecryptFile 使用 OpenSSL 命令优先、Go 实现兜底的方式解密文件。
func DecryptFile(ctx context.Context, encryptionKey, source, target string) error {
	if CommandAvailable(OpensslCommand) {
		return decryptFileByCommand(ctx, encryptionKey, source, target)
	}
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("打开待解密备份失败: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建解密备份文件失败: %w", err)
	}
	fileCrypto := utilscrypto.NewOpenSSLFileCrypto()
	if err = fileCrypto.DecryptReader(encryptionKey, input, output); err != nil {
		_ = output.Close()
		return fmt.Errorf("Go 解密备份失败: %w", err)
	}
	if err = output.Sync(); err != nil {
		_ = output.Close()
		return fmt.Errorf("同步解密备份失败: %w", err)
	}
	if err = output.Close(); err != nil {
		return fmt.Errorf("关闭解密备份文件失败: %w", err)
	}
	return nil
}

// encryptFileByCommand 使用固定协议参数调用 OpenSSL 加密文件。
func encryptFileByCommand(ctx context.Context, encryptionKey, source, target string) error {
	passwordFile, err := WriteSecretFile("kratos-openssl-", encryptionKey+"\n")
	if err != nil {
		return err
	}
	defer os.Remove(passwordFile)
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建加密备份文件失败: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("关闭加密备份文件失败: %w", err)
	}
	args := opensslFileCommandArgs(false, source, target, passwordFile)
	command := exec.CommandContext(ctx, OpensslCommand, args...)
	command.Stderr = os.Stderr
	if err = command.Run(); err != nil {
		return fmt.Errorf("执行 OpenSSL 加密失败: %w", err)
	}
	return nil
}

// decryptFileByCommand 使用固定协议参数调用 OpenSSL 解密文件。
func decryptFileByCommand(ctx context.Context, encryptionKey, source, target string) error {
	passwordFile, err := WriteSecretFile("kratos-openssl-", encryptionKey+"\n")
	if err != nil {
		return err
	}
	defer os.Remove(passwordFile)
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建解密备份文件失败: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("关闭解密备份文件失败: %w", err)
	}
	args := opensslFileCommandArgs(true, source, target, passwordFile)
	command := exec.CommandContext(ctx, OpensslCommand, args...)
	command.Stderr = os.Stderr
	if err = command.Run(); err != nil {
		return fmt.Errorf("执行 OpenSSL 解密失败: %w", err)
	}
	return nil
}

// opensslFileCommandArgs 构造固定协议的 OpenSSL enc 参数。
func opensslFileCommandArgs(decrypt bool, source, target, passwordFile string) []string {
	args := []string{"enc"}
	if decrypt {
		args = append(args, "-d")
	}
	args = append(args, "-aes-256-cbc", "-pbkdf2", "-iter", "100000", "-md", "sha256", "-salt")
	if opensslSupportsSaltLength() {
		args = append(args, "-saltlen", "8")
	}
	return append(args, "-in", source, "-out", target, "-pass", "file:"+passwordFile)
}

// opensslSupportsSaltLength 判断当前 OpenSSL 是否支持显式指定 PBKDF2 盐值长度。
func opensslSupportsSaltLength() bool {
	output, err := exec.Command(OpensslCommand, "enc", "-help").CombinedOutput()
	return err == nil && bytes.Contains(output, []byte("-saltlen"))
}
