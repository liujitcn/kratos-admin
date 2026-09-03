package oauth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/oauthsecret"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	oauthcrypto "github.com/liujitcn/kratos-admin/backend/internal/server/middleware/oauth/crypto"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gorm"
)

// NewCryptoFilter 创建开放授权接口的数据加解密 HTTP Filter。
//
// Filter 必须包裹整个 Kratos HTTP 路由树，原因是 Proto HTTP 适配器会先绑定请求体，
// 只有在绑定前解密才能让业务收到正常 JSON。普通用户令牌、非开放授权路径和错误响应均不处理。
func NewCryptoFilter(clientRepo *data.OauthClientRepository, authenticator engine.TokenAuthenticator, protector *oauthsecret.Protector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if !isOauthDevelopmentAPI(request.URL.Path) {
				next.ServeHTTP(writer, request)
				return
			}

			client, ok, err := oauthCryptoClient(request.Context(), clientRepo, authenticator, protector, request)
			if err != nil {
				writeOauthCryptoError(writer, http.StatusUnauthorized, "客户端访问令牌无效")
				return
			}
			if !ok {
				next.ServeHTTP(writer, request)
				return
			}

			var crypto oauthcrypto.Crypto
			crypto, err = oauthcrypto.New(client.CryptoType, client.CryptoKey)
			if err != nil {
				writeOauthCryptoError(writer, http.StatusUnauthorized, "客户端加密配置无效")
				return
			}
			if err = decryptOauthRequest(request, crypto); err != nil {
				writeOauthCryptoError(writer, http.StatusBadRequest, "请求数据解密失败")
				return
			}

			buffered := newOauthCryptoResponseWriter(writer)
			next.ServeHTTP(buffered, request)
			var responseBody []byte
			responseBody, err = encryptOauthResponse(buffered.body.Bytes(), buffered.status, crypto)
			if err != nil {
				writeOauthCryptoError(writer, http.StatusInternalServerError, "响应数据加密失败")
				return
			}
			writer.Header().Del("Content-Length")
			writer.WriteHeader(buffered.status)
			if _, err = writer.Write(responseBody); err != nil {
				return
			}
		})
	}
}

// oauthCryptoClient 从已签发的客户端令牌解析加密配置。
func oauthCryptoClient(ctx context.Context, clientRepo *data.OauthClientRepository, authenticator engine.TokenAuthenticator, protector *oauthsecret.Protector, request *http.Request) (*models.OauthClient, bool, error) {
	if authenticator == nil {
		return nil, false, nil
	}
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return nil, false, nil
	}
	var err error
	var claims *engine.AuthClaims
	claims, err = authenticator.AuthenticateToken(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, false, err
	}
	var payload *authData.UserTokenPayload
	payload, err = authData.NewUserTokenPayloadWithClaims(claims)
	if err != nil || payload.UserId >= 0 || payload.RoleCode == "" {
		return nil, false, nil
	}
	query := clientRepo.Query(ctx).OauthClient
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ClientID.Eq(payload.RoleCode)))
	var client *models.OauthClient
	client, err = clientRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, true, err
		}
		return nil, true, err
	}
	if client.Status != _const.STATUS_STATUS_ENABLE {
		return nil, true, errors.New("oauth client disabled")
	}
	if protector == nil {
		return nil, true, errors.New("oauth credential protector unavailable")
	}
	client.CryptoKey, err = protector.Unprotect(client.CryptoKey)
	if err != nil {
		return nil, true, err
	}
	return client, true, nil
}

// decryptOauthRequest 解密 POST、PUT、PATCH 请求体；其他方法保留明文参数。
func decryptOauthRequest(request *http.Request, crypto oauthcrypto.Crypto) error {
	if request.Method != http.MethodPost && request.Method != http.MethodPut && request.Method != http.MethodPatch {
		return nil
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		request.Body = io.NopCloser(bytes.NewReader(nil))
		request.ContentLength = 0
		return nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(body)))
	if err != nil {
		return err
	}
	plaintext, err := crypto.Decrypt(ciphertext)
	if err != nil {
		return err
	}
	request.Body = io.NopCloser(bytes.NewReader(plaintext))
	request.ContentLength = int64(len(plaintext))
	request.Header.Set("Content-Length", strconv.FormatInt(int64(len(plaintext)), 10))
	return nil
}

// encryptOauthResponse 加密成功响应。兼容 code/data 包装响应，并对当前 ProtoJSON 直出响应整体加密。
func encryptOauthResponse(body []byte, status int, crypto oauthcrypto.Crypto) ([]byte, error) {
	if status < http.StatusOK || status >= http.StatusMultipleChoices || len(body) == 0 {
		return body, nil
	}
	var envelope map[string]json.RawMessage
	var err error
	err = json.Unmarshal(body, &envelope)
	if err == nil && string(envelope["code"]) == "200" && len(envelope["data"]) > 0 {
		var ciphertext []byte
		ciphertext, err = crypto.Encrypt(envelope["data"])
		if err != nil {
			return nil, err
		}
		envelope["data"] = json.RawMessage(strconv.Quote(base64.StdEncoding.EncodeToString(ciphertext)))
		return json.Marshal(envelope)
	}
	var ciphertext []byte
	ciphertext, err = crypto.Encrypt(body)
	if err != nil {
		return nil, err
	}
	return []byte(strconv.Quote(base64.StdEncoding.EncodeToString(ciphertext))), nil
}

// oauthCryptoResponseWriter 缓存开放授权接口的明文响应，供外层统一加密后再写回客户端。
type oauthCryptoResponseWriter struct {
	http.ResponseWriter              // 底层 HTTP 响应写入器。
	body                bytes.Buffer // 尚未加密的响应正文。
	status              int          // 被缓存的 HTTP 状态码。
	wroteHeader         bool         // 是否已经记录响应头，避免重复写状态。
}

// newOauthCryptoResponseWriter 创建缓存响应正文和状态码的写入器。
func newOauthCryptoResponseWriter(writer http.ResponseWriter) *oauthCryptoResponseWriter {
	return &oauthCryptoResponseWriter{ResponseWriter: writer, status: http.StatusOK}
}

// WriteHeader 记录首次写入的 HTTP 状态码。
func (w *oauthCryptoResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
}

// Write 缓存响应正文，延迟到底层写入器。
func (w *oauthCryptoResponseWriter) Write(value []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
	}
	return w.body.Write(value)
}

// writeOauthCryptoError 返回未加密错误，便于客户端定位协议或权限问题。
func writeOauthCryptoError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	if _, err := writer.Write([]byte(`{"code":` + strconv.Itoa(status) + `,"msg":` + strconv.Quote(message) + `}`)); err != nil {
		return
	}
}
