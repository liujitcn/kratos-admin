package module

import (
	"fmt"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/runtimeconfig"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/securityconfig"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
)

// loadSecurityConfig 从分组的运行配置表单加载安全配置快照。
func loadSecurityConfig(baseConfigCase *biz.BaseConfigCase) (securityconfig.Config, error) {
	config := securityconfig.Default()
	var proxy runtimeconfig.SecurityProxyConfig
	err := runtimeconfig.LoadJSON(baseConfigCase.Cache, runtimeconfig.SecurityProxyKey, &proxy)
	if err != nil {
		return config, fmt.Errorf("读取反向代理安全配置失败: %w", err)
	}
	config.ProxyTrust = proxy.GetTrustProxy()
	config.TrustedProxyCIDRs = splitSecurityValues(proxy.GetTrustedProxyCidrs()...)

	var script runtimeconfig.SecurityScriptConfig
	err = runtimeconfig.LoadJSON(baseConfigCase.Cache, runtimeconfig.SecurityScriptKey, &script)
	if err != nil {
		return config, fmt.Errorf("读取脚本安全配置失败: %w", err)
	}
	config.ScriptReadPaths = splitSecurityValues(script.GetReadOnlyPaths()...)
	config.ScriptWritePaths = splitSecurityValues(script.GetWritablePaths()...)
	config.ScriptCommands = splitSecurityValues(script.GetCommands()...)
	config.ScriptAllowedHosts = splitSecurityValues(script.GetAllowedHosts()...)
	config.ScriptAllowedCIDRs = splitSecurityValues(script.GetAllowedCidrs()...)

	var outbound runtimeconfig.SecurityOutboundConfig
	err = runtimeconfig.LoadJSON(baseConfigCase.Cache, runtimeconfig.SecurityOutboundKey, &outbound)
	if err != nil {
		return config, fmt.Errorf("读取出站安全配置失败: %w", err)
	}
	config.OutboundAllowedHosts = splitSecurityValues(outbound.GetAllowedHosts()...)
	config.OutboundAllowedCIDRs = splitSecurityValues(outbound.GetAllowedCidrs()...)
	config.AIAllowedHosts = splitSecurityValues(outbound.GetAiAllowedHosts()...)
	config.AIAllowedCIDRs = splitSecurityValues(outbound.GetAiAllowedCidrs()...)

	var identity runtimeconfig.SecurityIdentityConfig
	err = runtimeconfig.LoadJSON(baseConfigCase.Cache, runtimeconfig.SecurityIdentityKey, &identity)
	if err != nil {
		return config, fmt.Errorf("读取账号安全配置失败: %w", err)
	}
	config.ReservedUsernames = splitSecurityValues(identity.GetReservedUsernames()...)
	config.AccountRenames = make(map[string]string, len(identity.GetAccountRenames()))
	for _, item := range identity.GetAccountRenames() {
		if strings.TrimSpace(item.GetKey()) != "" {
			config.AccountRenames[item.GetKey()] = item.GetValue()
		}
	}

	var migration runtimeconfig.SecurityMigrationConfig
	err = runtimeconfig.LoadJSON(baseConfigCase.Cache, runtimeconfig.SecurityMigrationKey, &migration)
	if err != nil {
		return config, fmt.Errorf("读取安全迁移配置失败: %w", err)
	}
	config.BackfillSensitiveData = migration.GetBackfillSensitiveData()
	config.LegacyPlaintextManifest = migration.GetLegacyPlaintextManifest()
	return config, nil
}

func splitSecurityValues(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		for item := range strings.SplitSeq(value, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				result = append(result, item)
			}
		}
	}
	return result
}
