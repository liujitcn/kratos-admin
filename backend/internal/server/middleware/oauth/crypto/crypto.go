package oauthcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/tjfoc/gmsm/sm4"
)

const (
	oauthCryptoAES        = "aes"
	oauthCryptoSM4        = "sm4"
	oauthCrypto3DES       = "3des"
	oauthCryptoAESKeySize = 32
	oauthCryptoSM4KeySize = 16
	oauthCryptoNonceSize  = 12
	oauthCryptoTagSize    = 16
)

// Crypto 定义开放授权协议的加解密能力。
type Crypto interface {
	Decrypt([]byte) ([]byte, error)
	Encrypt([]byte) ([]byte, error)
}

// New 根据客户端配置创建开放授权协议算法实例。
func New(cryptoType string, cryptoKey string) (Crypto, error) {
	return newCrypto(cryptoType, cryptoKey)
}

// newCrypto 根据客户端配置创建算法实例。
func newCrypto(cryptoType string, cryptoKey string) (Crypto, error) {
	switch strings.ToLower(cryptoType) {
	case oauthCryptoAES:
		key, err := decodeBase64Key(cryptoKey, oauthCryptoAESKeySize)
		if err != nil {
			return nil, err
		}
		return newGCMCrypto(func() (cipher.Block, error) { return aes.NewCipher(key) })
	case oauthCryptoSM4:
		key, err := decodeBase64Key(cryptoKey, oauthCryptoSM4KeySize)
		if err != nil {
			return nil, err
		}
		return newGCMCrypto(func() (cipher.Block, error) { return sm4.NewCipher(key) })
	case oauthCrypto3DES:
		return nil, errors.New("3des crypto is disabled; use sm4-gcm or aes-gcm")
	default:
		return nil, errors.New("unsupported crypto type")
	}
}

// KeyValid 判断已有客户端密钥是否符合当前算法格式。
func KeyValid(cryptoType string, cryptoKey string) bool {
	_, err := newCrypto(cryptoType, cryptoKey)
	return err == nil
}

// GenerateKey 生成与协议算法匹配的客户端密钥。
func GenerateKey(cryptoType string) (string, error) {
	switch strings.ToLower(cryptoType) {
	case oauthCryptoAES:
		return generateBase64CryptoKey(oauthCryptoAESKeySize)
	case oauthCryptoSM4:
		return generateBase64CryptoKey(oauthCryptoSM4KeySize)
	case oauthCrypto3DES:
		return "", errors.New("3des crypto is disabled; use sm4-gcm or aes-gcm")
	default:
		return "", errors.New("unsupported crypto type")
	}
}

// generateBase64CryptoKey 生成指定字节数的随机 Base64 密钥。
func generateBase64CryptoKey(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(value), nil
}

// decodeBase64Key 解码并校验 GCM 密钥。
func decodeBase64Key(value string, size int) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(key) != size {
		return nil, errors.New("invalid base64 crypto key")
	}
	return key, nil
}

// gcmCrypto 封装 AES-GCM 或 SM4-GCM 的统一加解密操作。
// gcm 保存已完成密钥校验的 AEAD 实例，调用方无需接触原始密钥。
type gcmCrypto struct {
	gcm cipher.AEAD // AEAD 加解密器。
}

// newGCMCrypto 使用已校验的分组密码创建 GCM 加解密器。
func newGCMCrypto(newBlock func() (cipher.Block, error)) (*gcmCrypto, error) {
	block, err := newBlock()
	if err != nil {
		return nil, err
	}
	var gcm cipher.AEAD
	gcm, err = cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &gcmCrypto{gcm: gcm}, nil
}

// Decrypt 校验并解密带 nonce 的 GCM 密文。
func (c *gcmCrypto) Decrypt(value []byte) ([]byte, error) {
	if len(value) < c.gcm.NonceSize()+c.gcm.Overhead() {
		return nil, errors.New("invalid gcm ciphertext")
	}
	return c.gcm.Open(nil, value[:c.gcm.NonceSize()], value[c.gcm.NonceSize():], nil)
}

// Encrypt 为明文生成随机 nonce 并返回带 nonce 的 GCM 密文。
func (c *gcmCrypto) Encrypt(value []byte) ([]byte, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return append(nonce, c.gcm.Seal(nil, nonce, value, nil)...), nil
}
