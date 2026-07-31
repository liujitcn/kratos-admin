package biz

import (
	"context"
	"sync"
	"time"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	"github.com/liujitcn/kratos-kit/auth"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/database/gorm"
	gormmigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
	"github.com/liujitcn/kratos-kit/pprof"
	"github.com/liujitcn/kratos-kit/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

// BaseCase 承载后端通用业务上下文与基础能力。
type BaseCase struct {
	*bootstrap.Context
	Cache     cache.Cache // Cache 提供后端业务缓存能力。
	queue     queue.Queue
	quitChan  chan struct{} //退出Chan
	closeOnce sync.Once
	taskTimer *time.Timer
	rwLock    sync.RWMutex //异步数据锁
}

// NewBaseCase 创建基础业务实例。
func NewBaseCase(
	ctx *bootstrap.Context,
	cache cache.Cache,
	queue queue.Queue,
	gorm *gorm.Client,
	_ gormmigration.Ready,
	pprof pprof.Pprof,
) (*BaseCase, func(), error) {

	// 设置全局变量
	sdk.Runtime.SetGormClient(gorm)
	sdk.Runtime.SetCache(cache)
	sdk.Runtime.SetQueue(queue)

	// 启动服务监控
	// 配置了 pprof 时，启动运行时性能分析服务。
	if pprof != nil {
		pprof.Start()
	}

	s := BaseCase{
		Context:   ctx,
		Cache:     cache,
		queue:     queue,
		quitChan:  make(chan struct{}),
		closeOnce: sync.Once{},
		taskTimer: nil,
		rwLock:    sync.RWMutex{},
	}
	// 启动后台队列消费线程，并等待清理信号退出。
	go func() {
		s.queue.Run()
		<-s.quitChan
	}()

	cleanup := func() {
		s.closeOnce.Do(func() {
			// 已创建后台定时器时，先停止避免关闭后继续触发任务。
			if s.taskTimer != nil {
				s.taskTimer.Stop()
			}
			close(s.quitChan)
		})
		// 启用了 pprof 时，清理阶段同步停止性能分析服务。
		if pprof != nil {
			pprof.Stop()
		}
	}

	return &s, cleanup, nil
}

// RegisterQueueConsumer 注册异步队列消费者。
func (c *BaseCase) RegisterQueueConsumer(queueName _const.Queue, fn func(message queueData.Message) error) {
	c.queue.Register(string(queueName), fn)
}

// GetAuthInfo 获取当前登录用户认证信息
func (c *BaseCase) GetAuthInfo(ctx context.Context) (*authData.UserTokenPayload, error) {
	authInfo, err := auth.FromContext(ctx)
	if err != nil {
		return nil, errorsx.Unauthenticated("用户认证失败").WithCause(err)
	}
	return authInfo, nil
}
