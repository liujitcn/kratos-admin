package biz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

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
	Cache          cache.Cache // Cache 提供后端业务缓存能力。
	queue          queue.Queue
	casbinRuleCase *CasbinRuleCase
	baseAPICase    *BaseAPICase
	baseTenantCase *BaseTenantCase
	quitChan       chan struct{} //退出Chan
	closeOnce      sync.Once
	taskTimer      *time.Timer
	rwLock         sync.RWMutex //异步数据锁
}

// NewBaseCase 创建基础业务实例。
func NewBaseCase(
	ctx *bootstrap.Context,
	cache cache.Cache,
	queue queue.Queue,
	gorm *gorm.Client,
	_ gormmigration.Ready,
	pprof pprof.Pprof,
	casbinRuleCase *CasbinRuleCase,
	baseAPICase *BaseAPICase,
	baseTenantCase *BaseTenantCase,
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
		Context:        ctx,
		Cache:          cache,
		queue:          queue,
		casbinRuleCase: casbinRuleCase,
		baseAPICase:    baseAPICase,
		baseTenantCase: baseTenantCase,
		quitChan:       make(chan struct{}),
		closeOnce:      sync.Once{},
		taskTimer:      nil,
		rwLock:         sync.RWMutex{},
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

// InitializeOpenAPI 根据全部具名 OpenAPI 文档同步接口元数据与权限策略。
func (c *BaseCase) InitializeOpenAPI(ctx context.Context, documents []coreOpenAPI.Document) error {
	baseAPIList := make([]*models.BaseAPI, 0)
	var err error
	for _, document := range documents {
		var documentBaseAPIs []*models.BaseAPI
		documentBaseAPIs, err = c.baseAPICase.openAPIDataToBaseAPI(document.Data)
		if err != nil {
			return fmt.Errorf("解析 OpenAPI 文档 %q: %w", document.Key, err)
		}
		baseAPIList = append(baseAPIList, documentBaseAPIs...)
	}

	// 所有模块文档收集完成后一次重建接口数据，避免扩展模块接口缺失或重复同步。
	err = c.baseAPICase.batchCreateBaseAPI(ctx, baseAPIList)
	if err != nil {
		return err
	}
	// API 数据就绪后，先将默认租户管理员角色菜单同步到普通租户副本。
	err = c.baseTenantCase.SyncTenantRoleMenus(ctx)
	if err != nil {
		return err
	}
	// 菜单和 API 数据均已就绪后，全量重建数据库规则并加载 Casbin 内存策略。
	err = c.casbinRuleCase.RebuildAllCasbinRules(ctx)
	if err != nil {
		return err
	}
	return nil
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

// RebuildPolicyRule 重建内存权限策略。
func (c *BaseCase) RebuildPolicyRule(ctx context.Context) error {
	return c.casbinRuleCase.rebuildPolicyRule(ctx)
}
