package backup

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/liujitcn/kratos-kit/sdk"
)

const (
	// IntegrityKeyName 是数据库备份完整性密钥的固定名称。
	IntegrityKeyName = "kratos-admin:backup/integrity"
	// EncryptionKeyName 是数据库备份加密密钥的固定名称。
	EncryptionKeyName = "kratos-admin:backup/encryption"
)

// Config 保存数据库备份执行所需的部署级参数。
type Config struct {
	// IntegrityKey 用于校验备份对象完整性。
	IntegrityKey string
	// EncryptionKey 用于加密和解密备份对象。
	EncryptionKey string
}

// FromRuntime 从运行时密钥服务读取数据库备份部署配置。
func FromRuntime() (Config, error) {
	keyValue := sdk.Runtime.GetKey()
	if keyValue == nil {
		return Config{}, errors.New("数据库备份密钥为空且运行时密钥未初始化")
	}
	var err error
	var integrityKey []byte
	integrityKey, err = keyValue.Derive(context.Background(), IntegrityKeyName)
	if err != nil {
		return Config{}, fmt.Errorf("派生数据库备份完整性密钥失败: %w", err)
	}
	var encryptionKey []byte
	encryptionKey, err = keyValue.Derive(context.Background(), EncryptionKeyName)
	if err != nil {
		return Config{}, fmt.Errorf("派生数据库备份加密密钥失败: %w", err)
	}
	return Config{
		IntegrityKey:  base64.RawStdEncoding.EncodeToString(integrityKey),
		EncryptionKey: base64.RawStdEncoding.EncodeToString(encryptionKey),
	}, nil
}

// WriteMySQLDefaultsFile 将数据库密码写入临时客户端配置文件。
func WriteMySQLDefaultsFile(password string) (string, error) {
	password = strings.NewReplacer("\\", "\\\\", "\r", "\\r", "\n", "\\n").Replace(password)
	return WriteSecretFile("kratos-mysql-", "[client]\npassword="+password+"\n")
}

// WriteSecretFile 将敏感内容写入权限受限的临时文件并返回路径。
func WriteSecretFile(prefix, value string) (string, error) {
	file, err := os.CreateTemp("", prefix)
	if err != nil {
		return "", fmt.Errorf("创建临时密钥文件失败: %w", err)
	}
	path := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(path)
	}
	if err = file.Chmod(0o600); err != nil {
		cleanup()
		return "", fmt.Errorf("设置临时密钥文件权限失败: %w", err)
	}
	if _, err = file.WriteString(value); err != nil {
		cleanup()
		return "", fmt.Errorf("写入临时密钥文件失败: %w", err)
	}
	if err = file.Sync(); err != nil {
		cleanup()
		return "", fmt.Errorf("同步临时密钥文件失败: %w", err)
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("关闭临时密钥文件失败: %w", err)
	}
	return path, nil
}
