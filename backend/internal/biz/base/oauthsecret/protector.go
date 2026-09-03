package oauthsecret

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/sdk"
)

const (
	protectedPrefix        = "enc:v1:"
	oauthCredentialKeyName = "kratos-admin:oauth/credential-protection"
	oauthCredentialKeyID   = "oauth-credential-protection"
)

// Protector 使用独立的服务端密钥保护 OAuth 客户端凭据。
// keyID 写入密文头部用于识别密钥版本；密钥材料由运行时密钥服务按固定用途派生。
type Protector struct {
	key   []byte // 由运行时密钥服务派生的 AES-256 密钥。
	keyID string // 当前密钥版本标识，写入 enc:v1 密文前缀。
}

// NewProtector 根据运行时密钥服务创建 OAuth 凭据保护器。
func NewProtector(config *configv1.Bootstrap) (*Protector, error) {
	if config == nil {
		return nil, errors.New("应用配置未初始化")
	}
	keyValue := sdk.Runtime.GetKey()
	if keyValue == nil {
		return nil, errors.New("OAuth 凭据密钥为空且运行时密钥未初始化")
	}
	derived, err := keyValue.Derive(context.Background(), oauthCredentialKeyName)
	if err != nil {
		return nil, fmt.Errorf("派生 OAuth 凭据密钥失败: %w", err)
	}
	return &Protector{key: append([]byte(nil), derived...), keyID: oauthCredentialKeyID}, nil
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
