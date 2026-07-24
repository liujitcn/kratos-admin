package codegen

import (
	"os"
	"regexp"
	"time"
)

const (
	// PageTypeTree 表示树形列表页面。
	PageTypeTree = "tree"
	// PageTypeTreeLazy 表示懒加载树形列表页面。
	PageTypeTreeLazy = "tree_lazy"
	// PageTypeLeftTree 表示左树右表页面。
	PageTypeLeftTree = "left_tree"

	// TriggerCRUD 表示由标准增删改查触发生成。
	TriggerCRUD = "crud"
	// TriggerPageTree 表示由树形页面触发生成。
	TriggerPageTree = "page_tree"
	// TriggerEntityOption 表示由实体选项能力触发生成。
	TriggerEntityOption = "entity_option"
	// TriggerLeftTree 表示由左树配置触发生成。
	TriggerLeftTree = "left_tree"
	// TriggerFieldOption 表示由字段选项配置触发生成。
	TriggerFieldOption = "field_option"
	// TriggerFieldStatus 表示由字段状态配置触发生成。
	TriggerFieldStatus = "field_status"

	// APIKindCRUD 表示标准增删改查接口。
	APIKindCRUD = "crud"
	// APIKindList 表示列表接口。
	APIKindList = "list"
	// APIKindOption 表示普通选项接口。
	APIKindOption = "option"
	// APIKindTree 表示树形接口。
	APIKindTree = "tree"
	// APIKindStatus 表示状态设置接口。
	APIKindStatus = "status"

	// OptionSourceStatic 表示静态选项数据源。
	OptionSourceStatic = "static"
	// OptionSourceDict 表示字典选项数据源。
	OptionSourceDict = "dict"
	// OptionSourceTable 表示数据表选项数据源。
	OptionSourceTable = "table"

	// StatusDraft 表示生成对象尚未生成。
	StatusDraft = 1
	// StatusGenerated 表示生成对象已经生成。
	StatusGenerated = 2
	// StatusDisabled 表示生成对象已停用。
	StatusDisabled = 3

	// CommandOutputMaxRunes 限制命令输出保存的最大字符数。
	CommandOutputMaxRunes = 4000
	// RemarkMaxRunes 限制生成备注保存的最大字符数。
	RemarkMaxRunes = 500
	// WorkflowTimeout 限制单次生成工作流的执行时长。
	WorkflowTimeout = 5 * time.Minute
	// FormatTimeout 限制格式化命令的执行时长。
	FormatTimeout = 2 * time.Minute

	// MenuStepID 标识菜单同步步骤。
	MenuStepID = "menu:sync"
	// CommandStepPrefix 标识命令执行步骤前缀。
	CommandStepPrefix = "command:"
)

var (
	// protoRPCPattern 匹配 Proto RPC 方法声明及方法名。
	protoRPCPattern = regexp.MustCompile(`rpc\s+([A-Za-z0-9_]+)\s*\(`)
	// protoMessagePattern 匹配 Proto message 声明及消息名。
	protoMessagePattern = regexp.MustCompile(`message\s+([A-Za-z0-9_]+)\s*\{`)
	// protoMessageFieldPattern 匹配生成器维护的 Proto 字段声明、字段名和编号，兼容多行 OpenAPI 注解。
	protoMessageFieldPattern = regexp.MustCompile(`(?ms)^[\t ]*(?:repeated[\t ]+|optional[\t ]+)?[A-Za-z_][A-Za-z0-9_.]*[\t ]+([A-Za-z_][A-Za-z0-9_]*)[\t ]*=[\t ]*([0-9]+).*?;[^\r\n]*$`)
	// commandSourcePattern 匹配生成命令中的数据源参数。
	commandSourcePattern = regexp.MustCompile(`(?i)(-source=)(?:'[^']*'|"[^"]*"|\S+)`)
	// commandDSNPattern 匹配命令输出中的数据库连接信息。
	commandDSNPattern = regexp.MustCompile(`(?i)[^\s'"]+:[^\s'"]+@tcp\([^)]+\)/[^\s'"]+`)
	// commandSecretPattern 匹配命令输出中的密码参数。
	commandSecretPattern = regexp.MustCompile(`(?i)((?:password|passwd|pwd)=)[^&\s'"]+`)
	// tsClassMethodPattern 匹配 TypeScript 类方法声明及方法名。
	tsClassMethodPattern = regexp.MustCompile(`(?m)^  ([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`)
	// redundantTimePattern 匹配生成代码中重复的时间字段转换赋值。
	redundantTimePattern = regexp.MustCompile(`(?m)^[\t ]*[A-Za-z_][A-Za-z0-9_]*\.(?:CreatedAt = _time\.TimeToTimeString\(item\.CreatedAt\)|UpdatedAt = _time\.TimeToTimeString\(item\.UpdatedAt\))\r?\n`)
)

// renderer 保存单次静态生成所需的非数据库上下文。
type renderer struct {
	tableComment string // 数据库表注释，由业务层查询后传入
	readFile     func(string) ([]byte, error)
}

// repoFileExists 判断当前渲染上下文中是否存在目标仓库文件。
func (c *renderer) repoFileExists(path string) (bool, error) {
	_, err := c.readRepoFile(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// protoMethodExists 判断当前渲染上下文中的指定 Proto service 是否已定义目标方法。
func (c *renderer) protoMethodExists(protoPath string, targetEntity string, methodName string) (bool, string) {
	content, err := c.readRepoFile(protoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "Proto文件不存在"
		}
		return false, err.Error()
	}
	if protoServiceMethodExists(string(content), targetEntity+"Service", methodName) {
		return true, "已存在"
	}
	return false, "缺少，可选择生成"
}

// readRepoFile 读取当前渲染上下文中的仓库文件。
func (c *renderer) readRepoFile(path string) ([]byte, error) {
	if c.readFile != nil {
		return c.readFile(path)
	}
	fullPath, err := SafeRepoFilePath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fullPath)
}
