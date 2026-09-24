package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	utilscrypto "github.com/liujitcn/go-utils/crypto"
)

const fieldCipherNonceSize = 12

// FieldCipher 提供存储脱敏 ENCRYPT 规则使用的可逆字段加密能力。
type FieldCipher struct {
	key []byte
}

// NewFieldCipher 根据32字节派生密钥创建字段加密器。
func NewFieldCipher(key []byte) (*FieldCipher, error) {
	if len(key) != 32 {
		return nil, errors.New("字段加密密钥必须是32字节")
	}
	return &FieldCipher{key: append([]byte(nil), key...)}, nil
}

// Encrypt 使用指定算法加密字段原文并返回Base64URL密文载荷。
func (c *FieldCipher) Encrypt(algorithm, value string) (string, error) {
	if value == "" {
		return value, nil
	}
	nonce := make([]byte, fieldCipherNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成字段加密随机数: %w", err)
	}
	var ciphertext []byte
	var err error
	switch algorithm {
	case "AES_GCM":
		ciphertext, err = utilscrypto.AesGCMEncryptWithAAD([]byte(value), c.key, nonce, []byte(configSecretKeyName))
	case "SM4_GCM":
		ciphertext, err = utilscrypto.Sm4GCMEncrypt([]byte(value), c.key[:16], nonce)
	default:
		return "", fmt.Errorf("不支持的字段加密算法: %s", algorithm)
	}
	if err != nil {
		return "", err
	}
	payload := append(append(make([]byte, 0, len(nonce)+len(ciphertext)), nonce...), ciphertext...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// Decrypt 使用指定算法解密Base64URL字段密文载荷。
func (c *FieldCipher) Decrypt(algorithm, value string) (string, error) {
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
	var plaintext []byte
	switch algorithm {
	case "AES_GCM":
		plaintext, err = utilscrypto.AesGCMDecryptWithAAD(ciphertext, c.key, nonce, []byte(configSecretKeyName))
	case "SM4_GCM":
		plaintext, err = utilscrypto.Sm4GCMDecrypt(ciphertext, c.key[:16], nonce)
	default:
		return "", fmt.Errorf("不支持的字段加密算法: %s", algorithm)
	}
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
