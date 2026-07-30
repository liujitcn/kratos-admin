// Package health 提供存活检查和可扩展就绪检查。
package health

import (
	"context"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"sync"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
)

// Check 描述一项外部依赖就绪检查。
type Check struct {
	// Name 是检查项的稳定唯一名称。
	Name string
	// Run 执行检查。
	Run func(context.Context) error
}

// Registry 保存当前进程启用的就绪检查。
type Registry struct {
	mu     sync.RWMutex
	checks []Check
	names  map[string]struct{}
}

// NewRegistry 创建空的健康检查注册表。
func NewRegistry() *Registry {
	return &Registry{names: make(map[string]struct{})}
}

// Register 注册就绪检查，并拒绝空名称、空方法和重复名称。
func (r *Registry) Register(checks ...Check) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	registered := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		if check.Name == "" {
			return fmt.Errorf("健康检查名称不能为空")
		}
		if check.Run == nil {
			return fmt.Errorf("健康检查方法不能为空: %s", check.Name)
		}
		if _, exists := r.names[check.Name]; exists {
			return fmt.Errorf("健康检查名称重复: %s", check.Name)
		}
		if _, exists := registered[check.Name]; exists {
			return fmt.Errorf("健康检查名称重复: %s", check.Name)
		}
		registered[check.Name] = struct{}{}
	}
	for _, check := range checks {
		r.checks = append(r.checks, check)
		r.names[check.Name] = struct{}{}
	}
	return nil
}

// RegisterHTTP 挂载存活检查和聚合就绪检查路由。
func RegisterHTTP(server *kratosHTTP.Server, registry *Registry) {
	server.HandleFunc("/healthz", func(writer stdhttp.ResponseWriter, _ *stdhttp.Request) {
		writer.WriteHeader(stdhttp.StatusNoContent)
	})
	server.HandleFunc("/readyz", func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		var err error
		var name string
		name, err = registry.check(request.Context())
		if err == nil {
			writer.WriteHeader(stdhttp.StatusNoContent)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(stdhttp.StatusServiceUnavailable)
		_ = json.NewEncoder(writer).Encode(map[string]string{
			"check":   name,
			"message": err.Error(),
		})
	})
}

func (r *Registry) check(ctx context.Context) (string, error) {
	r.mu.RLock()
	checks := append([]Check(nil), r.checks...)
	r.mu.RUnlock()
	var err error
	for _, check := range checks {
		err = check.Run(ctx)
		if err != nil {
			return check.Name, err
		}
	}
	return "", nil
}
