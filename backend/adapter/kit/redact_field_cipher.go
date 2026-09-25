package kit

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/kratos-kit/sdk"
)

const (
	fieldCipherNonceSize = 12
	fieldCipherKeyName   = "kratos-admin:redact/field"
)

type fieldCipher struct {
	key []byte
}

// newRuntimeFieldCipher 使用运行时主密钥创建脱敏字段加密器。
func newRuntimeFieldCipher() (*fieldCipher, error) {
	keyValue := sdk.Runtime.GetKey()
	if keyValue == nil {
		return nil, errors.New("脱敏字段加密密钥为空且运行时密钥未初始化")
	}
	key, err := keyValue.Derive(context.Background(), fieldCipherKeyName)
	if err != nil {
		return nil, err
	}
	return newFieldCipher(key)
}

// newFieldCipher 创建使用指定密钥的字段加密器。
func newFieldCipher(key []byte) (*fieldCipher, error) {
	if len(key) != 32 {
		return nil, errors.New("字段加密密钥必须是32字节")
	}
	return &fieldCipher{key: append([]byte(nil), key...)}, nil
}

// Encrypt 使用脱敏专用密钥加密字段原文并返回Base64URL密文载荷。
func (c *fieldCipher) Encrypt(algorithm, value string) (string, error) {
	if value == "" {
		return value, nil
	}
	nonce := make([]byte, fieldCipherNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成字段加密随机数: %w", err)
	}
	ciphertext, err := encryptFieldValue(algorithm, c.key, nonce, []byte(value), fieldCipherKeyName)
	if err != nil {
		return "", err
	}
	payload := make([]byte, 0, len(nonce)+len(ciphertext))
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// Decrypt 使用脱敏专用密钥解密Base64URL字段密文。
func (c *fieldCipher) Decrypt(algorithm, value string) (string, error) {
	if value == "" {
		return value, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("解析字段密文: %w", err)
	}
	if len(payload) <= fieldCipherNonceSize {
		return "", errors.New("字段密文长度无效")
	}
	nonce := payload[:fieldCipherNonceSize]
	ciphertext := payload[fieldCipherNonceSize:]
	plaintext, err := decryptFieldValue(algorithm, c.key, nonce, ciphertext, fieldCipherKeyName)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// encryptFieldValue 按指定密钥、算法和关联数据生成密文字节。
func encryptFieldValue(algorithm string, key, nonce, plaintext []byte, associatedData string) ([]byte, error) {
	switch algorithm {
	case "AES_GCM":
		return crypto.AesGCMEncryptWithAAD(plaintext, key, nonce, []byte(associatedData))
	case "SM4_GCM":
		return crypto.Sm4GCMEncrypt(plaintext, key[:16], nonce)
	default:
		return nil, fmt.Errorf("不支持的字段加密算法: %s", algorithm)
	}
}

// decryptFieldValue 按指定密钥、算法和关联数据恢复明文字节。
func decryptFieldValue(algorithm string, key, nonce, ciphertext []byte, associatedData string) ([]byte, error) {
	switch algorithm {
	case "AES_GCM":
		return crypto.AesGCMDecryptWithAAD(ciphertext, key, nonce, []byte(associatedData))
	case "SM4_GCM":
		return crypto.Sm4GCMDecrypt(ciphertext, key[:16], nonce)
	default:
		return nil, fmt.Errorf("不支持的字段加密算法: %s", algorithm)
	}
}
