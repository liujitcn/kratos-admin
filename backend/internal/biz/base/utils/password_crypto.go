package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	commonv1 "github.com/liujitcn/kratos-admin/backend/core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"

	"github.com/google/uuid"
	utilscrypto "github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/kratos-kit/sdk"
)

const (
	passwordCryptoAlgorithm = "RSA-OAEP-256+A256GCM"
	passwordCryptoKeyPrefix = "password_crypto:"
	passwordCryptoTTL       = 5 * time.Minute
)

var passwordCryptoSceneSet = map[basev1.PasswordCryptoScene]struct{}{
	basev1.PasswordCryptoScene_LOGIN:                    {},
	basev1.PasswordCryptoScene_CREATE_BASE_USER:         {},
	basev1.PasswordCryptoScene_RESET_BASE_USER_PASSWORD: {},
	basev1.PasswordCryptoScene_UPDATE_USER_PASSWORD:     {},
}

type passwordCryptoKeyRecord struct {
	PrivateKey string `json:"private_key"`
	Nonce      string `json:"nonce"`
	Scene      string `json:"scene"`
	Algorithm  string `json:"algorithm"`
}

// GeneratePasswordPublicKey 生成密码加密使用的临时公钥。
func GeneratePasswordPublicKey(scene basev1.PasswordCryptoScene) (*basev1.PasswordPublicKeyResponse, error) {
	if _, ok := passwordCryptoSceneSet[scene]; !ok {
		return nil, errorsx.InvalidArgument("密码加密场景不支持")
	}
	cache := sdk.Runtime.GetCache()
	if cache == nil {
		return nil, errorsx.Internal("密码加密缓存不可用")
	}

	rsaCrypto, err := utilscrypto.NewRSACrypto(2048)
	if err != nil {
		return nil, errorsx.Internal("生成密码临时密钥失败").WithCause(err)
	}
	var privateKeyPEM string
	privateKeyPEM, err = rsaCrypto.ExportPrivateKeyPKCS8()
	if err != nil {
		return nil, errorsx.Internal("生成密码临时密钥失败").WithCause(err)
	}
	var publicKeyPEM string
	publicKeyPEM, err = rsaCrypto.ExportPublicKeyPKIX()
	if err != nil {
		return nil, errorsx.Internal("生成密码临时密钥失败").WithCause(err)
	}

	keyID := uuid.NewString()
	var nonceBytes []byte
	nonceBytes, err = utilscrypto.GenerateAESKey(16)
	if err != nil {
		return nil, errorsx.Internal("生成密码临时密钥失败").WithCause(err)
	}
	nonce := base64.StdEncoding.EncodeToString(nonceBytes)
	record := passwordCryptoKeyRecord{
		PrivateKey: privateKeyPEM,
		Nonce:      nonce,
		Scene:      scene.String(),
		Algorithm:  passwordCryptoAlgorithm,
	}
	var recordBytes []byte
	recordBytes, err = json.Marshal(record)
	if err != nil {
		return nil, errorsx.Internal("生成密码临时密钥失败").WithCause(err)
	}
	err = cache.Set(makePasswordCryptoCacheKey(keyID), string(recordBytes), passwordCryptoTTL)
	if err != nil {
		return nil, errorsx.Internal("生成密码临时密钥失败").WithCause(err)
	}

	return &basev1.PasswordPublicKeyResponse{
		KeyId:     keyID,
		PublicKey: publicKeyPEM,
		Algorithm: passwordCryptoAlgorithm,
		Nonce:     nonce,
		ExpiresIn: int64(passwordCryptoTTL / time.Second),
	}, nil
}

// DecryptPassword 解密密码密文字段并返回原始密码。
func DecryptPassword(password *commonv1.PasswordCrypto, scene basev1.PasswordCryptoScene) (string, error) {
	if _, ok := passwordCryptoSceneSet[scene]; !ok {
		return "", errorsx.InvalidArgument("密码加密场景不支持")
	}
	if password == nil {
		return "", errorsx.InvalidArgument("密码不能为空")
	}
	if password.GetKeyId() == "" ||
		password.GetNonce() == "" ||
		password.GetEncryptedKey() == "" ||
		password.GetIv() == "" ||
		password.GetCiphertext() == "" {
		return "", errorsx.InvalidArgument("密码密文不能为空")
	}
	if password.GetAlgorithm() != passwordCryptoAlgorithm {
		return "", errorsx.InvalidArgument("密码加密算法不支持")
	}

	cache := sdk.Runtime.GetCache()
	if cache == nil {
		return "", errorsx.Internal("密码加密缓存不可用")
	}
	cacheKey := makePasswordCryptoCacheKey(password.GetKeyId())
	recordText, err := cache.Get(cacheKey)
	if err != nil || recordText == "" {
		return "", errorsx.InvalidArgument("密码密钥已过期，请重新提交")
	}
	// 临时密钥一次性使用，读取后立即删除，避免同一密文被重放。
	_ = cache.Del(cacheKey)

	var record passwordCryptoKeyRecord
	err = json.Unmarshal([]byte(recordText), &record)
	if err != nil {
		return "", errorsx.InvalidArgument("密码密钥无效").WithCause(err)
	}
	if record.Scene != scene.String() {
		return "", errorsx.InvalidArgument("密码密钥场景不匹配")
	}
	if record.Nonce != password.GetNonce() {
		return "", errorsx.InvalidArgument("密码随机值无效")
	}
	if record.Algorithm != passwordCryptoAlgorithm {
		return "", errorsx.InvalidArgument("密码密钥算法不支持")
	}

	var rsaCrypto *utilscrypto.RSACrypto
	rsaCrypto, err = utilscrypto.NewRSACryptoFromPrivateKeyPEM(record.PrivateKey)
	if err != nil {
		return "", errorsx.InvalidArgument("密码密钥无效").WithCause(err)
	}
	var aesKey []byte
	aesKey, err = rsaCrypto.DecryptBytes(password.GetEncryptedKey())
	if err != nil {
		return "", errorsx.InvalidArgument("密码密钥解密失败").WithCause(err)
	}
	var iv []byte
	iv, err = base64.StdEncoding.DecodeString(password.GetIv())
	if err != nil {
		return "", errorsx.InvalidArgument("密码初始化向量无效").WithCause(err)
	}
	var ciphertext []byte
	ciphertext, err = base64.StdEncoding.DecodeString(password.GetCiphertext())
	if err != nil {
		return "", errorsx.InvalidArgument("密码密文无效").WithCause(err)
	}
	var plaintext []byte
	plaintext, err = utilscrypto.AesGCMDecrypt(ciphertext, aesKey, iv)
	if err != nil {
		return "", errorsx.InvalidArgument("密码解密失败").WithCause(err)
	}
	return string(plaintext), nil
}

// GetDefaultPassword 生成默认密码
func GetDefaultPassword(userName, phone string) string {
	prefix := phone
	// 取手机号后4位
	if len(phone) > 4 {
		prefix = phone[len(phone)-4:]
	}
	// 不足4位左补0
	prefix = ("0000" + prefix)[len(prefix):]
	return fmt.Sprintf("%s@%s", userName, prefix)
}

// makePasswordCryptoCacheKey 生成临时密码密钥缓存键。
func makePasswordCryptoCacheKey(keyID string) string {
	return passwordCryptoKeyPrefix + keyID
}
