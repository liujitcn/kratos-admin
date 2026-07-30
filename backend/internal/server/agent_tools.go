package server

import (
	einoTool "github.com/liujitcn/kratos-admin/backend/internal/agent/tool"
)

// AdminAgentToolsProvider 表示可提供管理端 AI 助手工具的业务模块。
type AdminAgentToolsProvider interface {
	AdminAgentTools() ([]einoTool.Invokable, error)
}

// AppAgentToolsProvider 表示可提供应用端 AI 助手工具的业务模块。
type AppAgentToolsProvider interface {
	AppAgentTools() ([]einoTool.Invokable, error)
}

// TerminalToolSetter 表示可接收不同终端 AI 助手工具的运行时。
type TerminalToolSetter interface {
	SetTerminalTools([]einoTool.Invokable, []einoTool.Invokable)
}

// AgentToolsReady 表示 AI 助手工具已完成注册。
type AgentToolsReady struct{}

// NewAgentToolsReady 汇总全部业务模块的 AI 助手工具并写入运行时。
func NewAgentToolsReady(setter TerminalToolSetter, modules Modules) (AgentToolsReady, error) {
	var err error
	var adminTools []einoTool.Invokable
	adminTools, err = modules.adminAgentTools()
	if err != nil {
		return AgentToolsReady{}, err
	}
	var appTools []einoTool.Invokable
	appTools, err = modules.appAgentTools()
	if err != nil {
		return AgentToolsReady{}, err
	}
	setter.SetTerminalTools(adminTools, appTools)
	return AgentToolsReady{}, nil
}

// adminAgentTools 汇总管理端 AI 助手工具。
func (modules Modules) adminAgentTools() ([]einoTool.Invokable, error) {
	var tools []einoTool.Invokable
	for _, module := range modules {
		provider, ok := module.(AdminAgentToolsProvider)
		if !ok {
			continue
		}
		values, err := provider.AdminAgentTools()
		if err != nil {
			return nil, err
		}
		tools = append(tools, values...)
	}
	return tools, nil
}

// appAgentTools 汇总应用端 AI 助手工具。
func (modules Modules) appAgentTools() ([]einoTool.Invokable, error) {
	var tools []einoTool.Invokable
	for _, module := range modules {
		provider, ok := module.(AppAgentToolsProvider)
		if !ok {
			continue
		}
		values, err := provider.AppAgentTools()
		if err != nil {
			return nil, err
		}
		tools = append(tools, values...)
	}
	return tools, nil
}
