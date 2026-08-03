package script

import (
	"context"
	"fmt"
)

// Script 描述一项具名启动脚本及其依赖。
type Script struct {
	// Name 是脚本的稳定唯一名称。
	Name string
	// Dependencies 是必须先执行的脚本名称。
	Dependencies []string
	// Run 执行脚本，数据库迁移等具体实现由外部适配器提供。
	Run func(context.Context) error
}

// Registry 保存启动脚本并负责依赖排序。
type Registry struct {
	scripts map[string]Script
	order   []string
}

// NewRegistry 创建空的脚本注册表。
func NewRegistry() *Registry {
	return &Registry{scripts: make(map[string]Script)}
}

// Register 注册启动脚本，并拒绝空名称、空执行方法和重复名称。
func (r *Registry) Register(scripts ...Script) error {
	registered := make(map[string]struct{}, len(scripts))
	for _, script := range scripts {
		if script.Name == "" {
			return fmt.Errorf("启动脚本名称不能为空")
		}
		if script.Run == nil {
			return fmt.Errorf("启动脚本执行方法不能为空: %s", script.Name)
		}
		if _, exists := r.scripts[script.Name]; exists {
			return fmt.Errorf("启动脚本名称重复: %s", script.Name)
		}
		if _, exists := registered[script.Name]; exists {
			return fmt.Errorf("启动脚本名称重复: %s", script.Name)
		}
		registered[script.Name] = struct{}{}
	}
	for _, script := range scripts {
		script.Dependencies = append([]string(nil), script.Dependencies...)
		r.scripts[script.Name] = script
		r.order = append(r.order, script.Name)
	}
	return nil
}

// Run 按依赖顺序执行全部脚本，任一脚本失败时立即停止。
func (r *Registry) Run(ctx context.Context) error {
	var err error
	var ordered []Script
	ordered, err = r.ordered()
	if err != nil {
		return err
	}
	for _, current := range ordered {
		err = current.Run(ctx)
		if err != nil {
			return fmt.Errorf("执行启动脚本 %q: %w", current.Name, err)
		}
	}
	return nil
}

func (r *Registry) ordered() ([]Script, error) {
	const (
		unvisited = iota
		visiting
		visited
	)
	states := make(map[string]int, len(r.scripts))
	ordered := make([]Script, 0, len(r.scripts))
	var err error
	var visit func(string) error
	visit = func(name string) error {
		script, exists := r.scripts[name]
		if !exists {
			return fmt.Errorf("启动脚本依赖不存在: %s", name)
		}
		switch states[name] {
		case visiting:
			return fmt.Errorf("启动脚本存在循环依赖: %s", name)
		case visited:
			return nil
		}
		states[name] = visiting
		for _, dependency := range script.Dependencies {
			err = visit(dependency)
			if err != nil {
				return err
			}
		}
		states[name] = visited
		ordered = append(ordered, script)
		return nil
	}
	for _, name := range r.order {
		err = visit(name)
		if err != nil {
			return nil, err
		}
	}
	return ordered, nil
}
