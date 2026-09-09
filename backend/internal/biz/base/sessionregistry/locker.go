package sessionregistry

import (
	"fmt"

	"github.com/liujitcn/kratos-core/job"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// LoginLocker 复用自动续租能力，为账号登录签发提供跨实例互斥。
type LoginLocker struct {
	*job.ExecutionLocker
}

// NewLoginLocker 根据缓存配置创建账号锁；配置 Redis 时禁止降级为单机锁。
func NewLoginLocker(cfg *configv1.Bootstrap) (*LoginLocker, func(), error) {
	redisConfig := cfg.GetData().GetRedis()
	if redisConfig == nil {
		manager := job.NewMemoryExecutionLocker()
		return &LoginLocker{ExecutionLocker: manager}, manager.Close, nil
	}
	manager := job.NewExecutionLocker(redisConfig)
	if manager.Mode() != "redis" {
		manager.Close()
		return nil, nil, fmt.Errorf("账号分布式锁初始化失败，禁止降级为单机锁")
	}
	return &LoginLocker{ExecutionLocker: manager}, manager.Close, nil
}
