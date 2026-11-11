package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// Config 是平台审核并持久化的远端接入配置；缺失配置时连接明确不可用。
type Config struct {
	NetworkPolicy
	OAuth *OAuthConfig `json:"oauth,omitempty"`
}

// OAuthConfig 是专用 MCP 客户端注册，不复用 Admin 登录 OAuth 凭据。
type OAuthConfig struct {
	ClientID         string   `json:"client_id"`
	AuthorizationURL string   `json:"authorization_url"`
	TokenURL         string   `json:"token_url"`
	RevocationURL    string   `json:"revocation_url,omitempty"`
	RedirectURL      string   `json:"redirect_url"`
	Issuer           string   `json:"issuer"`
	Scopes           []string `json:"scopes"`
}

// ParseConfig 验证持久化的网络和公开 OAuth 配置。
func ParseConfig(raw string) (Config, error) {
	var config Config
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&config)
	if err != nil {
		return config, fmt.Errorf("接入配置JSON不合法: %w", err)
	}
	var extra any
	err = decoder.Decode(&extra)
	if err != io.EOF {
		return config, fmt.Errorf("接入配置只允许一个JSON对象")
	}
	if len(config.AllowedHosts) == 0 {
		return config, fmt.Errorf("平台尚未批准网络端点")
	}
	if config.OAuth != nil {
		var redirect *url.URL
		redirect, err = url.Parse(config.OAuth.RedirectURL)
		if err != nil || redirect.Scheme != "https" || redirect.Host == "" || redirect.User != nil || redirect.Fragment != "" {
			return config, fmt.Errorf("OAuth回调必须为平台HTTPS地址")
		}
		if config.OAuth.ClientID == "" || config.OAuth.RedirectURL == "" || config.OAuth.Issuer == "" {
			return config, fmt.Errorf("尚未配置OAuth客户端注册")
		}
		for _, endpoint := range []string{config.OAuth.AuthorizationURL, config.OAuth.TokenURL, config.OAuth.Issuer} {
			_, err = config.ValidateURL(endpoint)
			if err != nil {
				return config, err
			}
		}
		if config.OAuth.RevocationURL != "" {
			_, err = config.ValidateURL(config.OAuth.RevocationURL)
			if err != nil {
				return config, err
			}
		}
	}
	return config, nil
}
