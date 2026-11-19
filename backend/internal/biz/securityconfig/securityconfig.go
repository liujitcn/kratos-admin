package securityconfig

import (
	"maps"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/liujitcn/kratos-kit/transport/safehttp"
)

const defaultAIHTTPTimeout = 30 * time.Second

// Config 汇总服务运行期间使用的安全配置。
type Config struct {
	// ProxyTrust 表示是否信任反向代理上报的 X-Forwarded-Proto。
	ProxyTrust bool
	// TrustedProxyCIDRs 表示允许转发来源地址的代理网段。
	TrustedProxyCIDRs []string
	// ScriptReadPaths 表示脚本沙箱的只读路径。
	ScriptReadPaths []string
	// ScriptWritePaths 表示脚本沙箱的可写路径。
	ScriptWritePaths []string
	// ScriptCommands 表示脚本沙箱允许执行的绝对命令路径。
	ScriptCommands []string
	// ScriptAllowedHosts 表示脚本出站允许访问的主机。
	ScriptAllowedHosts []string
	// ScriptAllowedCIDRs 表示脚本出站允许访问的网段。
	ScriptAllowedCIDRs []string
	// OutboundAllowedHosts 表示所有受控出站请求共享的主机白名单。
	OutboundAllowedHosts []string
	// OutboundAllowedCIDRs 表示所有受控出站请求共享的网段白名单。
	OutboundAllowedCIDRs []string
	// AIAllowedHosts 表示 AI 模型出站允许访问的主机。
	AIAllowedHosts []string
	// AIAllowedCIDRs 表示 AI 模型出站允许访问的网段。
	AIAllowedCIDRs []string
	// ReservedUsernames 表示系统保留用户名。
	ReservedUsernames []string
	// AccountRenames 表示一次性存量账号更名映射。
	AccountRenames map[string]string
	// BackfillSensitiveData 表示是否执行一次性敏感字段回填。
	BackfillSensitiveData bool
	// LegacyPlaintextManifest 表示经核实的历史明文记录清单。
	LegacyPlaintextManifest string
}

// snapshot 将配置与其 AI 出站连接池一起发布。
type snapshot struct {
	config      Config
	aiClient    *http.Client
	aiClientErr error
}

var current atomic.Pointer[snapshot]

// init 在进程启动时发布默认安全配置。
func init() {
	Set(Default())
}

// Default 返回安全配置默认值，默认关闭代理信任并拒绝私网出站访问。
func Default() Config {
	return Config{}
}

// Current 返回当前进程使用的安全配置快照。
func Current() Config {
	return clone(current.Load().config)
}

// Set 替换当前进程使用的安全配置快照，并释放旧策略下的空闲连接。
func Set(value Config) {
	value = clone(value)
	client, err := value.AIHTTPClient(defaultAIHTTPTimeout)
	previous := current.Swap(&snapshot{config: value, aiClient: client, aiClientErr: err})
	if previous != nil && previous.aiClient != nil {
		previous.aiClient.CloseIdleConnections()
	}
}

// NewLiveAIHTTPClient 创建长生命周期客户端，每次发送和重定向都使用当前进程的安全策略。
func NewLiveAIHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultAIHTTPTimeout
	}
	return &http.Client{Timeout: timeout, Transport: liveAITransport{}, CheckRedirect: checkAIRedirect}
}

// liveAITransport 让模型客户端复用当前策略的连接池，配置更新后不再复用旧连接池。
type liveAITransport struct{}

// RoundTrip 按当前策略验证并发送请求，策略无效时拒绝出站。
func (liveAITransport) RoundTrip(request *http.Request) (*http.Response, error) {
	value := current.Load()
	if value.aiClientErr != nil {
		return nil, value.aiClientErr
	}
	return value.aiClient.Transport.RoundTrip(request)
}

// ScriptPolicy 根据公共和脚本专属白名单创建脚本出站策略。
func (c Config) ScriptPolicy() (*safehttp.Policy, error) {
	hosts := append(slices.Clone(c.OutboundAllowedHosts), c.ScriptAllowedHosts...)
	cidrs := append(slices.Clone(c.OutboundAllowedCIDRs), c.ScriptAllowedCIDRs...)
	return safehttp.NewPolicy(false, hosts, cidrs)
}

// AIPolicy 根据公共和 AI 专属白名单创建模型出站策略。
func (c Config) AIPolicy() (*safehttp.Policy, error) {
	hosts := append(slices.Clone(c.OutboundAllowedHosts), c.AIAllowedHosts...)
	cidrs := append(slices.Clone(c.OutboundAllowedCIDRs), c.AIAllowedCIDRs...)
	return safehttp.NewPolicy(true, hosts, cidrs)
}

// AIHTTPClient 创建使用当前 AI 出站策略的 HTTP 客户端。
func (c Config) AIHTTPClient(timeout time.Duration) (*http.Client, error) {
	policy, err := c.AIPolicy()
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = defaultAIHTTPTimeout
	}
	return policy.Client(timeout), nil
}

// IsSecureRequest 判断请求是否为直接 HTTPS 或受配置允许的代理 HTTPS。
func (c Config) IsSecureRequest(request *http.Request) bool {
	if request == nil {
		return false
	}
	if request.TLS != nil {
		return true
	}
	return c.ProxyTrust && strings.EqualFold(request.Header.Get("X-Forwarded-Proto"), "https")
}

// ClientIP 仅在 TCP 对端属于受信代理时解析转发链。
func (c Config) ClientIP(request *http.Request) string {
	if request == nil {
		return ""
	}
	address, err := netip.ParseAddrPort(request.RemoteAddr)
	var peer netip.Addr
	if err == nil {
		peer = address.Addr().Unmap()
	} else {
		peer, err = netip.ParseAddr(request.RemoteAddr)
		if err != nil {
			return request.RemoteAddr
		}
		peer = peer.Unmap()
	}
	trusted := make([]netip.Prefix, 0, len(c.TrustedProxyCIDRs))
	for _, network := range c.TrustedProxyCIDRs {
		network = strings.TrimSpace(network)
		if network == "" {
			continue
		}
		var prefix netip.Prefix
		prefix, err = netip.ParsePrefix(network)
		if err != nil {
			return peer.String()
		}
		trusted = append(trusted, prefix.Masked())
	}
	forwarded := request.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		forwarded = request.Header.Get("X-Real-IP")
	}
	for _, hop := range slices.Backward(strings.Split(forwarded, ",")) {
		if !slices.ContainsFunc(trusted, func(prefix netip.Prefix) bool { return prefix.Contains(peer) }) {
			break
		}
		var candidate netip.Addr
		candidate, err = netip.ParseAddr(strings.TrimSpace(hop))
		if err != nil {
			break
		}
		peer = candidate.Unmap()
	}
	return peer.String()
}

// checkAIRedirect 沿用当前策略的同源和目标校验。
func checkAIRedirect(request *http.Request, via []*http.Request) error {
	value := current.Load()
	if value.aiClientErr != nil {
		return value.aiClientErr
	}
	return value.aiClient.CheckRedirect(request, via)
}

// clone 深拷贝配置中的切片和映射，确保全局快照与调用方数据不共享底层存储，避免并发读写相互影响。
func clone(value Config) Config {
	value.TrustedProxyCIDRs = slices.Clone(value.TrustedProxyCIDRs)
	value.ScriptReadPaths = slices.Clone(value.ScriptReadPaths)
	value.ScriptWritePaths = slices.Clone(value.ScriptWritePaths)
	value.ScriptCommands = slices.Clone(value.ScriptCommands)
	value.ScriptAllowedHosts = slices.Clone(value.ScriptAllowedHosts)
	value.ScriptAllowedCIDRs = slices.Clone(value.ScriptAllowedCIDRs)
	value.OutboundAllowedHosts = slices.Clone(value.OutboundAllowedHosts)
	value.OutboundAllowedCIDRs = slices.Clone(value.OutboundAllowedCIDRs)
	value.AIAllowedHosts = slices.Clone(value.AIAllowedHosts)
	value.AIAllowedCIDRs = slices.Clone(value.AIAllowedCIDRs)
	value.ReservedUsernames = slices.Clone(value.ReservedUsernames)
	if value.AccountRenames != nil {
		value.AccountRenames = maps.Clone(value.AccountRenames)
	}
	return value
}
