package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
)

// ErrReauthorizationRequired 表示账号需由用户在连接页重新授权，执行中不能自动弹窗或重放。
var ErrReauthorizationRequired = errors.New("连接授权已失效，请在连接页重新授权")

type savedTokenHandler struct{ token string }

var _ auth.OAuthHandler = (*savedTokenHandler)(nil)

// Transport 使用官方SDK的OAuth边界发送已保存令牌，并关闭自动重连和请求重放。
func Transport(endpoint, token string, config Config) *mcp.StreamableClientTransport {
	transport := &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: config.Client(""), MaxRetries: -1, DisableStandaloneSSE: true}
	if token != "" {
		transport.OAuthHandler = &savedTokenHandler{token: token}
	}
	return transport
}

// TokenSource 只返回本次已读取的令牌；续期在业务层账号行锁内完成。
func (h *savedTokenHandler) TokenSource(context.Context) (oauth2.TokenSource, error) {
	return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: h.token, TokenType: "Bearer"}), nil
}

// Authorize 拒绝SDK对401/403进行交互授权和重试，结果由连接页处理。
func (h *savedTokenHandler) Authorize(_ context.Context, _ *http.Request, response *http.Response) error {
	err := response.Body.Close()
	if err != nil {
		return errors.Join(ErrReauthorizationRequired, err)
	}
	return ErrReauthorizationRequired
}

// RevokeOAuth 在本地撤销完成后尽力按RFC7009撤销刷新令牌和访问令牌，不重试远端请求。
func RevokeOAuth(ctx context.Context, token *oauth2.Token, config Config) error {
	if config.OAuth == nil || config.OAuth.RevocationURL == "" {
		return fmt.Errorf("供应商尚未配置令牌撤销端点")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client := config.Client("")
	var err error
	for _, item := range []struct{ value, hint string }{{token.RefreshToken, "refresh_token"}, {token.AccessToken, "access_token"}} {
		if item.value == "" {
			continue
		}
		values := url.Values{"token": {item.value}, "token_type_hint": {item.hint}, "client_id": {config.OAuth.ClientID}}
		var request *http.Request
		request, err = http.NewRequestWithContext(ctx, http.MethodPost, config.OAuth.RevocationURL, strings.NewReader(values.Encode()))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		var response *http.Response
		response, err = client.Do(request)
		if err != nil {
			return fmt.Errorf("供应商令牌撤销请求失败")
		}
		_, err = io.Copy(io.Discard, response.Body)
		closeErr := response.Body.Close()
		if err != nil || closeErr != nil || response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("供应商未确认令牌撤销")
		}
	}
	return nil
}
