package oauthcrypto

import (
	"bytes"
	"encoding/base64"
	"testing"
)

// TestNewAESGCM 验证 OAuth AES-GCM 使用 go-utils 底层实现并保持协议密文格式。
func TestNewAESGCM(t *testing.T) {
	testOauthCrypto(t, oauthCryptoAES, []byte("0123456789abcdef0123456789abcdef"))
}

// TestNewSM4GCM 验证 OAuth SM4-GCM 使用 go-utils 底层实现并保持协议密文格式。
func TestNewSM4GCM(t *testing.T) {
	testOauthCrypto(t, oauthCryptoSM4, []byte("0123456789abcdef"))
}

// testOauthCrypto 验证 OAuth 加解密的 nonce 拼接和 GCM 完整性校验。
func testOauthCrypto(t *testing.T, cryptoType string, key []byte) {
	t.Helper()
	encodedKey := base64.StdEncoding.EncodeToString(key)
	crypto, err := New(cryptoType, encodedKey)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	plainText := []byte(`{"message":"hello"}`)
	cipherText, err := crypto.Encrypt(plainText)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if len(cipherText) != oauthCryptoNonceSize+len(plainText)+oauthCryptoTagSize {
		t.Fatalf("Encrypt() length = %d, want %d", len(cipherText), oauthCryptoNonceSize+len(plainText)+oauthCryptoTagSize)
	}
	decrypted, err := crypto.Decrypt(cipherText)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if !bytes.Equal(decrypted, plainText) {
		t.Fatalf("Decrypt() = %q, want %q", decrypted, plainText)
	}

	cipherText[len(cipherText)-1] ^= 1
	if _, err = crypto.Decrypt(cipherText); err == nil {
		t.Fatal("Decrypt() expected tampered ciphertext error")
	}
}
