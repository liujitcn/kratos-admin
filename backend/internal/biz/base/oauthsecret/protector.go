// Package oauthsecret 提供开放授权客户端敏感凭据的服务端加密存储。
package oauthsecret

import (
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

const protectedPrefix = "enc:v1:"

// Protector 使用独立的服务端密钥保护 OAuth 客户端凭据。
// keyID 写入密文头部用于识别密钥版本；密钥材料只从 Secret Manager 注入的环境变量读取。
type Protector struct {
	key   []byte // 由独立环境密钥派生的 AES-256 密钥。
	keyID string // 当前密钥版本标识，写入 enc:v1 密文前缀。
}

// NewProtector 根据独立环境密钥创建凭据保护器。
func NewProtector(config *configv1.Bootstrap) (*Protector, error) {
	if config == nil {
		return nil, errors.New("应用配置未初始化")
	}
	secret := os.Getenv("OAUTH_CREDENTIAL_ENCRYPTION_KEY")
	if len(secret) < 16 {
		// OAuth 客户端属于可选能力；未配置独立密钥时保留主服务启动，具体操作返回未初始化错误。
		_, _ = fmt.Fprintln(os.Stderr, "OAUTH_CREDENTIAL_ENCRYPTION_KEY 未配置，OAuth 客户端功能已禁用")
		return nil, nil
	}
	keyID := os.Getenv("OAUTH_CREDENTIAL_ENCRYPTION_KEY_ID")
	if keyID == "" {
		keyID = "primary"
	}
	if strings.ContainsAny(keyID, ":\r\n") {
		return nil, errors.New("OAUTH_CREDENTIAL_ENCRYPTION_KEY_ID 格式无效")
	}
	digest := sha256.Sum256([]byte("kratos-admin/oauth-client-credentials/" + keyID + "/" + secret))
	return &Protector{key: digest[:], keyID: keyID}, nil
}

// Protect 使用 AES-GCM 加密单个 OAuth 凭据。
func (p *Protector) Protect(value string) (string, error) {
	if p == nil || len(p.key) == 0 {
		return "", errors.New("OAuth 凭据保护器未初始化")
	}
	if value == "" {
		return "", errors.New("OAuth 凭据不能为空")
	}
	var block cipher.Block
	var err error
	block, err = aes.NewCipher(p.key)
	if err != nil {
		return "", fmt.Errorf("创建凭据加密器失败: %w", err)
	}
	var aead cipher.AEAD
	aead, err = cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建凭据 AEAD 失败: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = cryptorand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成凭据随机数失败: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, []byte(value), nil)
	encoded := append(nonce, ciphertext...)
	return protectedPrefix + p.keyID + ":" + base64.RawStdEncoding.EncodeToString(encoded), nil
}

// Unprotect 解密 OAuth 凭据；未使用服务端保护格式的值直接拒绝。
func (p *Protector) Unprotect(value string) (string, error) {
	if p == nil || len(p.key) == 0 {
		return "", errors.New("OAuth 凭据保护器未初始化")
	}
	if !strings.HasPrefix(value, protectedPrefix) {
		return "", errors.New("OAuth 凭据未使用服务端加密格式")
	}
	var err error
	var encoded []byte
	protectedValue := strings.TrimPrefix(value, protectedPrefix)
	parts := strings.SplitN(protectedValue, ":", 2)
	if len(parts) != 2 || parts[0] != p.keyID {
		return "", errors.New("OAuth 凭据密钥版本不匹配")
	}
	encoded, err = base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("解析 OAuth 凭据失败: %w", err)
	}
	var block cipher.Block
	block, err = aes.NewCipher(p.key)
	if err != nil {
		return "", fmt.Errorf("创建凭据解密器失败: %w", err)
	}
	var aead cipher.AEAD
	aead, err = cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建凭据 AEAD 失败: %w", err)
	}
	if len(encoded) < aead.NonceSize() {
		return "", errors.New("OAuth 凭据密文长度无效")
	}
	var plaintext []byte
	plaintext, err = aead.Open(nil, encoded[:aead.NonceSize()], encoded[aead.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("解密 OAuth 凭据失败: %w", err)
	}
	return string(plaintext), nil
}
