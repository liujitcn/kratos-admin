package sessionregistry

import (
	"fmt"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/locker"
)

// NewLoginLocker 根据 Redis 配置创建登录签发锁；未配置时使用进程内锁。
func NewLoginLocker(cfg *configv1.Bootstrap) (locker.Locker, func(), error) {
	manager, err := locker.NewLocker(cfg.GetData().GetRedis())
	if err != nil {
		return nil, nil, fmt.Errorf("初始化登录签发锁失败: %w", err)
	}
	return manager, func() { _ = manager.Close() }, nil
}
