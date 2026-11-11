package plugin

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"
)

// NetworkPolicy 是平台审核的远程网络范围；普通用户不能扩大此范围。
type NetworkPolicy struct {
	AllowedHosts []string `json:"allowed_hosts"`
	AllowPrivate bool     `json:"allow_private"`
}

// ValidateURL 检查端点和 OAuth 端点属于明确批准主机；显式允许私网时才放行 HTTP 或非 443 端口。
func (p NetworkPolicy) ValidateURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("端点格式错误: %w", err)
	}
	if u.Hostname() == "" || u.User != nil || u.Fragment != "" || !slices.Contains(p.AllowedHosts, strings.ToLower(u.Hostname())) || (u.Scheme != "https" && !(p.AllowPrivate && u.Scheme == "http")) || (u.Port() != "" && u.Port() != "443" && !p.AllowPrivate) {
		return nil, fmt.Errorf("端点未被平台批准")
	}
	return u, nil
}

// AllowsIP 拒绝本地、链路、多播、运营商共享及私网地址；内网只允许平台显式配置。
func (p NetworkPolicy) AllowsIP(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsUnspecified() {
		return false
	}
	if address.IsPrivate() || netip.MustParsePrefix("100.64.0.0/10").Contains(address) || netip.MustParsePrefix("198.18.0.0/15").Contains(address) {
		return p.AllowPrivate
	}
	return true
}

// Client 对每次请求执行同一 URL 和 DNS 策略，直接连接已验证 IP，禁止代理和重定向。
func (p NetworkPolicy) Client(token string) *http.Client {
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			var addresses []net.IPAddr
			addresses, err = net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(addresses) == 0 {
				return nil, fmt.Errorf("端点DNS没有可用地址")
			}
			for _, item := range addresses {
				if !p.AllowsIP(item.IP) {
					return nil, fmt.Errorf("端点解析到禁止访问的地址")
				}
			}
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
		},
	}
	return &http.Client{Transport: &boundedTransport{policy: p, base: transport, token: token}, Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("远程端点禁止重定向") }}
}

type boundedTransport struct {
	policy NetworkPolicy
	base   http.RoundTripper
	token  string
}

// RoundTrip 禁止向未批准端点发送凭据，并限制远端响应大小。
func (t *boundedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	_, err := t.policy.ValidateURL(request.URL.String())
	if err != nil {
		return nil, err
	}
	request = request.Clone(request.Context())
	request.Header = request.Header.Clone()
	if t.token != "" {
		request.Header.Set("Authorization", "Bearer "+t.token)
	}
	var response *http.Response
	response, err = t.base.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	response.Body = &limitedBody{ReadCloser: response.Body, remaining: 4 << 20}
	return response, nil
}

type limitedBody struct {
	io.ReadCloser
	remaining int64
}

// Read 在大小超限时返回错误，禁止将截断响应当作成功结果。
func (b *limitedBody) Read(data []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, fmt.Errorf("远程响应超过4MiB限制")
	}
	if int64(len(data)) > b.remaining {
		data = data[:b.remaining]
	}
	n, err := b.ReadCloser.Read(data)
	b.remaining -= int64(n)
	return n, err
}
