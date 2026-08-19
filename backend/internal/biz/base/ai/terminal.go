package ai

import (
	"github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

// NormalizeTerminal 将 proto 终端枚举转换为数据库使用的整型值。
func NormalizeTerminal(terminal basev1.Terminal) int32 {
	// 当前 AI 助手默认服务管理端，只有明确传应用端时才写入 app 终端。
	switch terminal {
	case basev1.Terminal_TERMINAL_APP:
		return TerminalApp
	default:
		return TerminalAdmin
	}
}

// NormalizeTerminalString 将数据库终端整型值转换为模型提示词中的终端标识。
func NormalizeTerminalString(terminal int32) string {
	// 提示词中只暴露稳定文本标识，避免把数据库枚举值泄漏给模型。
	switch terminal {
	case TerminalApp:
		return "app"
	default:
		return "admin"
	}
}

// NormalizeTerminalEnum 将数据库终端整型值转换为 proto 终端枚举。
func NormalizeTerminalEnum(terminal int32) basev1.Terminal {
	// 数据库历史值异常时按管理端返回，和 NormalizeTerminal 的默认语义保持一致。
	switch terminal {
	case TerminalApp:
		return basev1.Terminal_TERMINAL_APP
	default:
		return basev1.Terminal_TERMINAL_ADMIN
	}
}
