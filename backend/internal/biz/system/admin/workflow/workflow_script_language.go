package workflow

// ScriptLanguageSupported 判断引擎是否支持流程执行所需的受限环境与超时取消。
// 共享脚本管理保留其他语言；工作流不执行有系统导入能力或缺乏取消机制的引擎。
func ScriptLanguageSupported(language string) bool {
	return language == "JAVASCRIPT"
}
