package kit

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"github.com/liujitcn/kratos-kit/sdk"
)

const legacyFieldCipherKeyName = "kratos-kit:config"

type fieldCipherMigrationTestKey map[string][]byte

// Derive 返回字段加密迁移测试指定用途的派生密钥。
func (k fieldCipherMigrationTestKey) Derive(_ context.Context, purpose string) ([]byte, error) {
	key, exists := k[purpose]
	if !exists {
		return nil, fmt.Errorf("未知测试密钥用途: %s", purpose)
	}
	return append([]byte(nil), key...), nil
}

// TestFieldCipherReencryptsExistingPayload 验证存量AES和SM4密文可以解密后使用新密钥重加密。
func TestFieldCipherReencryptsExistingPayload(t *testing.T) {
	previousKey := sdk.Runtime.GetKey()
	t.Cleanup(func() { sdk.Runtime.SetKey(previousKey) })
	sdk.Runtime.SetKey(fieldCipherMigrationTestKey{
		legacyFieldCipherKeyName: []byte("abcdefghijklmnopqrstuvwxyz123456"),
		fieldCipherKeyName:       []byte("12345678901234567890123456789012"),
	})
	keyValue := sdk.Runtime.GetKey()
	legacyKey, err := keyValue.Derive(context.Background(), legacyFieldCipherKeyName)
	if err != nil {
		t.Fatal(err)
	}
	var newCipher *fieldCipher
	newCipher, err = newRuntimeFieldCipher()
	if err != nil {
		t.Fatal(err)
	}
	for _, algorithm := range []string{"AES_GCM", "SM4_GCM"} {
		t.Run(algorithm, func(t *testing.T) {
			nonce := []byte("123456789012")
			oldCiphertext, testErr := encryptFieldValue(algorithm, legacyKey, nonce, []byte("existing-secret"), legacyFieldCipherKeyName)
			if testErr != nil {
				t.Fatal(testErr)
			}
			oldPayload := make([]byte, 0, len(nonce)+len(oldCiphertext))
			oldPayload = append(oldPayload, nonce...)
			oldPayload = append(oldPayload, oldCiphertext...)
			var migrated string
			migrated, testErr = reencryptLegacyFieldCipherValue(algorithm, base64.RawURLEncoding.EncodeToString(oldPayload))
			if testErr != nil {
				t.Fatal(testErr)
			}
			if migrated == base64.RawURLEncoding.EncodeToString(oldPayload) {
				t.Fatal("迁移后密文不应保留旧载荷")
			}
			var plaintext string
			plaintext, testErr = newCipher.Decrypt(algorithm, migrated)
			if testErr != nil {
				t.Fatal(testErr)
			}
			if plaintext != "existing-secret" {
				t.Fatalf("新密钥解密结果错误: %q", plaintext)
			}
		})
	}
}

// reencryptLegacyFieldCipherValue 在测试迁移程序中将单个旧密文转换为新密钥密文。
func reencryptLegacyFieldCipherValue(algorithm, value string) (string, error) {
	keyValue := sdk.Runtime.GetKey()
	var err error
	var legacyKey []byte
	legacyKey, err = keyValue.Derive(context.Background(), legacyFieldCipherKeyName)
	if err != nil {
		return "", err
	}
	var payload []byte
	payload, err = base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("解析旧字段密文: %w", err)
	}
	if len(payload) <= fieldCipherNonceSize {
		return "", errors.New("旧字段密文长度无效")
	}
	var plaintext []byte
	plaintext, err = decryptFieldValue(algorithm, legacyKey, payload[:fieldCipherNonceSize], payload[fieldCipherNonceSize:], legacyFieldCipherKeyName)
	if err != nil {
		return "", fmt.Errorf("解密旧字段密文: %w", err)
	}
	var newCipher *fieldCipher
	newCipher, err = newRuntimeFieldCipher()
	if err != nil {
		return "", err
	}
	return newCipher.Encrypt(algorithm, string(plaintext))
}
