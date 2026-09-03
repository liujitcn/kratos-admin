package codegen

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/go-utils/stringcase"
)

const (
	// ProtoRootPath 表示仓库内 Proto 根目录。
	ProtoRootPath = "backend/api/proto"
)

// ProtoTarget 描述代码生成器支持的 Proto 与后端服务分组。
type ProtoTarget struct {
	Directory               string // Proto 根目录下的相对目录
	PackageName             string // Proto package 名称
	GoAlias                 string // Go 协议包别名
	GoImportPath            string // Go 协议包导入路径
	ServiceImportAlias      string // 后端服务包别名
	BackendBizDirectory     string // 后端 Biz 模块目录
	BackendModuleDirectory  string // 后端 Service 模块目录
	ModuleRegisterPath      string // 业务模块注册文件路径
	FrontendPackageName     string // 前端业务模块包名
	FrontendAPIDirectory    string // 前端 API 目录
	FrontendPageDirectory   string // 前端页面目录
	FrontendLocaleDirectory string // 前端语言包目录
}

// ProtoTargetForBusinessModule 根据业务模块值推导代码生成目标。
func ProtoTargetForBusinessModule(module string) (ProtoTarget, bool) {
	module = filepath.ToSlash(filepath.Clean(module))
	if module == "." || module == "" || strings.Contains(module, "/") || strings.Contains(module, "\\") {
		return ProtoTarget{}, false
	}
	return ProtoTarget{
		Directory:               module + "/admin/v1",
		PackageName:             module + ".admin.v1",
		GoAlias:                 strings.ReplaceAll(module, "_", "") + "adminv1",
		GoImportPath:            "github.com/liujitcn/kratos-admin/backend/api/gen/go/" + module + "/admin/v1",
		ServiceImportAlias:      strings.ReplaceAll(module, "_", "") + "admin",
		BackendBizDirectory:     "backend/internal/biz/" + module + "/admin",
		BackendModuleDirectory:  "backend/internal/service/" + module + "/admin/v1",
		ModuleRegisterPath:      "backend/internal/server/" + module + "/admin/v1/register.go",
		FrontendPackageName:     "@liujitcn/kratos-admin-" + module,
		FrontendAPIDirectory:    "frontend/admin/packages/modules/" + module + "/src/api/" + module,
		FrontendPageDirectory:   "frontend/admin/packages/modules/" + module + "/src/views",
		FrontendLocaleDirectory: "frontend/admin/packages/modules/" + module + "/src/locales",
	}, true
}

// ProtoTargetForTable 返回表配置对应的动态代码生成目标。
func ProtoTargetForTable(table *Table) ProtoTarget {
	if table == nil {
		return ProtoTarget{}
	}
	target, _ := ProtoTargetForBusinessModule(table.BusinessModule)
	return target
}

// ProtoTargetForProtoPath 根据 Proto 文件路径推导代码生成目标。
func ProtoTargetForProtoPath(protoPath string) (ProtoTarget, bool) {
	relativePath := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(protoPath)), ProtoRootPath+"/")
	if relativePath == protoPath || filepath.Base(relativePath) == relativePath {
		return ProtoTarget{}, false
	}
	parts := strings.Split(relativePath, "/")
	if len(parts) != 4 || parts[1] != "admin" || parts[2] != "v1" || filepath.Ext(parts[3]) != ".proto" {
		return ProtoTarget{}, false
	}
	return ProtoTargetForBusinessModule(parts[0])
}

// ProtoFilePath 根据 Proto 目录和实体名称生成仓库相对 Proto 文件路径。
func ProtoFilePath(directory string, entityName string) string {
	return filepath.ToSlash(filepath.Join(ProtoRootPath, directory, stringcase.ToSnakeCase(entityName)+".proto"))
}

// BackendBizFilePath 返回目标分组内的 Biz 文件路径。
func (t ProtoTarget) BackendBizFilePath(entityName string) string {
	return filepath.ToSlash(filepath.Join(t.BackendBizDirectory, stringcase.ToSnakeCase(entityName)+".go"))
}

// BackendServiceFilePath 返回目标分组内的 Service 文件路径。
func (t ProtoTarget) BackendServiceFilePath(entityName string) string {
	return filepath.ToSlash(filepath.Join(t.BackendModuleDirectory, stringcase.ToSnakeCase(entityName)+"_service.go"))
}

// FrontendAPIFilePath 返回目标分组内的前端 API 文件路径。
func (t ProtoTarget) FrontendAPIFilePath(entityName string) string {
	return filepath.ToSlash(filepath.Join(t.FrontendAPIDirectory, stringcase.ToSnakeCase(entityName)+".ts"))
}

// FrontendLocaleFilePath 返回目标分组内的前端语言包文件路径。
func (t ProtoTarget) FrontendLocaleFilePath(localeValue string) string {
	return filepath.ToSlash(filepath.Join(t.FrontendLocaleDirectory, localeValue+".json"))
}

// BackendBizImportPath 返回目标分组 Biz 包的 Go 导入路径。
func (t ProtoTarget) BackendBizImportPath() string {
	return "github.com/liujitcn/kratos-admin/backend/" + strings.TrimPrefix(filepath.ToSlash(t.BackendBizDirectory), "backend/")
}

// Table 描述一次代码生成所需的表配置快照。
type Table struct {
	ID               int64                   // 代码生成表配置 ID
	SourceName       string                  // 数据源名称
	TableName_       string                  // 业务表名
	TableComment     string                  // 业务表描述
	BusinessModule   string                  // 业务模块
	BusinessName     string                  // 业务名称
	EntityName       string                  // 实体名称
	ModulePath       string                  // 模块路径
	APIPath          string                  // Proto 目录
	ProtoFilePath    string                  // 本次生成覆盖的 Proto 文件路径
	PermissionPrefix string                  // 权限标识前缀
	ParentMenuID     int64                   // 父级菜单 ID
	PageType         string                  // 页面类型
	ParentColumn     string                  // 树形页面父节点字段
	TreeLabelColumn  string                  // 树形页面显示字段
	LeftTreeConfig   string                  // 左树右表配置 JSON
	I18NConfig       map[string]LocaleConfig // 表级国际化配置
	GenBackend       int32                   // 是否生成后端
	GenFrontend      int32                   // 是否生成前端
	GenSql           int32                   // 是否同步菜单权限
	Status           int32                   // 配置状态
	CreatedAt        time.Time               // 配置创建时间
	UpdatedAt        time.Time               // 配置更新时间
}

// Proto 描述一次代码生成所需的 Proto 接口配置快照。
type Proto struct {
	ID                       int64  // Proto 配置 ID
	TableID                  int64  // 代码生成表配置 ID
	Name                     string // 触发字段名
	TriggerType              string // 触发来源
	APIKind                  string // 接口类型
	TargetEntityName         string // 目标实体名
	TargetBusinessName       string // 目标数据库表描述
	MethodName               string // RPC 方法名
	ProtoFilePath            string // Proto 文件路径
	ParentColumn             string // 树接口父节点字段
	LabelColumn              string // 选项显示字段
	ValueColumn              string // 选项取值字段
	OptionStatusColumn       string // Option 禁用状态字段
	OptionStatusEnabledValue string // Option 状态启用值
	Lazy                     bool   // 树形 Option 接口是否懒加载
	GenerateWhenMissing      int32  // 缺失时是否生成
	Sort                     int32  // 排序
}

// ProtoCheck 描述渲染阶段推导出的 Proto 接口检查项。
type ProtoCheck struct {
	TableID                  int64  // 代码生成表配置 ID
	Name                     string // 触发字段名
	TriggerType              string // 触发来源
	APIKind                  string // 接口类型
	TargetEntityName         string // 目标实体名
	TargetBusinessName       string // 目标数据库表描述
	MethodName               string // RPC 方法名
	ProtoFilePath            string // Proto 文件路径
	Exists                   bool   // RPC 是否已经存在
	GenerateWhenMissing      bool   // 缺失时是否生成
	ParentColumn             string // 树接口父节点字段
	LabelColumn              string // 选项显示字段
	ValueColumn              string // 选项取值字段
	OptionStatusColumn       string // Option 禁用状态字段
	OptionStatusEnabledValue string // Option 状态启用值
	Lazy                     bool   // 树形 Option 接口是否懒加载
	Message                  string // 检查说明
}

// CodeGenProtoPatch 描述向现有 Proto 文件追加的内容。
type CodeGenProtoPatch struct {
	// ServiceNames 需要追加 RPC 的服务名称。
	ServiceNames []string
	// RPCs 按服务名称分组的 RPC 定义。
	RPCs map[string][]string
	// Messages 需要补齐的消息定义。
	Messages []string
}

// CommonImportRequired 判断追加内容是否依赖 common 响应类型。
func (p CodeGenProtoPatch) CommonImportRequired() bool {
	for _, serviceName := range p.ServiceNames {
		for _, rpc := range p.RPCs[serviceName] {
			if strings.Contains(rpc, ".common.v1.") {
				return true
			}
		}
	}
	return false
}

// CodeGenProtoRPCBlock 描述 Proto service 中可重排的单个 RPC 块。
type CodeGenProtoRPCBlock struct {
	// Name RPC 方法名。
	Name string
	// Content 包含相邻注释的完整 RPC 内容。
	Content string
	// OriginalIndex 原始位置，用于稳定保留未知方法顺序。
	OriginalIndex int
}

// CodeGenSourceMethodBlock 描述 Go 接收者或 TypeScript 类中的可重排方法块。
type CodeGenSourceMethodBlock struct {
	// Name 方法名。
	Name string
	// Content 包含方法注释的完整源码。
	Content string
	// Start 在当前源码中的起始偏移。
	Start int
	// End 在当前源码中的结束偏移。
	End int
	// OriginalIndex 原始位置，用于稳定保留扩展方法顺序。
	OriginalIndex int
}

// CodeGenExternalTarget 描述生成流程依赖的外部实体及其方法。
type CodeGenExternalTarget struct {
	// Table 外部实体对应的生成对象。
	Table *Table
	// Methods 外部实体需要补齐的方法。
	Methods []*Proto
}

// CodeGenMenuSpec 描述待同步的生成菜单。
type CodeGenMenuSpec struct {
	// Menu 待创建或更新的菜单。
	Menu *models.BaseMenu
	// SourceTitle 菜单中文权威标题。
	SourceTitle string
	// I18ns 按语言区域索引的菜单标题。
	I18ns map[string]string
}

// TableInfo 数据库表元数据查询结果。
type TableInfo struct {
	// TableName 数据库表名。
	TableName string `gorm:"column:table_name"`
	// TableComment 数据库表注释。
	TableComment string `gorm:"column:table_comment"`
}

// CodeGenColumn 汇总数据库字段与用户保存的生成配置。
type CodeGenColumn struct {
	// ID 字段配置 ID。
	ID int64
	// TableID 生成对象 ID。
	TableID int64
	// Name 字段名称。
	Name string
	// Comment 字段注释。
	Comment string
	// I18NConfig 字段国际化配置。
	I18NConfig map[string]LocaleConfig
	// DbType 数据库基础类型。
	DbType string
	// ColumnType 数据库完整类型。
	ColumnType string
	// DbLength 字段长度。
	DbLength int32
	// DbScale 小数位数。
	DbScale int32
	// DefaultValue 默认值。
	DefaultValue string
	// HasDefault 是否声明默认值。
	HasDefault bool
	// Extra 数据库附加属性。
	Extra string
	// IsPrimary 是否为主键。
	IsPrimary int32
	// IsAutoIncrement 是否自增。
	IsAutoIncrement int32
	// IsNullable 是否允许为空。
	IsNullable int32
	// GoType Go 字段类型。
	GoType string
	// ProtoType Proto 字段类型。
	ProtoType string
	// TsType TypeScript 字段类型。
	TsType string
	// IsQuery 是否作为查询条件。
	IsQuery int32
	// QueryOperator 查询操作符。
	QueryOperator string
	// QueryComponent 查询组件。
	QueryComponent string
	// IsList 是否在列表展示。
	IsList int32
	// ListComponent 列表组件。
	ListComponent string
	// IsForm 是否在表单展示。
	IsForm int32
	// FormComponent 表单组件。
	FormComponent string
	// IsRequired 表单是否必填。
	IsRequired int32
	// FormMultiple 表单树形选择是否多选。
	FormMultiple bool
	// OptionKind 选项展示类型。
	OptionKind string
	// OptionSourceType 选项数据源类型。
	OptionSourceType string
	// OptionSourceValue 选项数据源值。
	OptionSourceValue string
	// OptionLabelField 选项标签字段。
	OptionLabelField string
	// OptionValueField 选项取值字段。
	OptionValueField string
	// OptionParentField 树形选项父级字段。
	OptionParentField string
	// QueryOption 查询条件独立使用的选项配置。
	QueryOption CodeGenColumnOptionConfig
	// ListOption 列表展示独立使用的选项配置。
	ListOption CodeGenColumnOptionConfig
	// FormOption 表单录入独立使用的选项配置。
	FormOption CodeGenColumnOptionConfig
	// IsStatusField 是否为状态字段。
	IsStatusField int32
	// StatusDataType 状态数据类型。
	StatusDataType string
	// StatusDictCode 状态字典编码。
	StatusDictCode string
	// StatusEnumName 状态枚举名称。
	StatusEnumName string
	// StatusEnabledValue 启用状态值。
	StatusEnabledValue string
	// StatusDisabledValue 禁用状态值。
	StatusDisabledValue string
	// StatusDefaultValue 状态默认值。
	StatusDefaultValue string
	// StatusGenerateAPI 是否生成状态接口。
	StatusGenerateAPI int32
	// StatusTableColumn 是否作为状态列表列。
	StatusTableColumn int32
	// StatusSearch 是否支持状态查询。
	StatusSearch int32
	// StatusSwitch 是否使用状态开关。
	StatusSwitch int32
	// StatusForm 是否在表单配置状态。
	StatusForm int32
	// Sort 字段排序。
	Sort int32
}

// LocaleConfig 描述单语言的代码生成展示配置。
type LocaleConfig struct {
	// Comment 业务或字段描述。
	Comment string `json:"comment"`
	// LeftTreeComment 左树描述。
	LeftTreeComment string `json:"left_tree_comment"`
}

// CodeGenColumnQueryConfig 描述字段查询配置。
type CodeGenColumnQueryConfig struct {
	// Enabled 是否启用查询。
	Enabled bool `json:"enabled"`
	// Operator 查询操作符。
	Operator string `json:"operator"`
	// Component 查询组件。
	Component string `json:"component"`
}

// CodeGenColumnListConfig 描述字段列表配置。
type CodeGenColumnListConfig struct {
	// Enabled 是否在列表展示。
	Enabled bool `json:"enabled"`
	// Component 列表组件。
	Component string `json:"component"`
}

// CodeGenColumnFormConfig 描述字段表单配置。
type CodeGenColumnFormConfig struct {
	// Enabled 是否在表单展示。
	Enabled bool `json:"enabled"`
	// Component 表单组件。
	Component string `json:"component"`
	// Required 是否必填。
	Required bool `json:"required"`
	// Multiple 树形选择是否多选。
	Multiple bool `json:"multiple"`
}

// CodeGenColumnOptionConfig 描述字段选项配置。
type CodeGenColumnOptionConfig struct {
	// Kind 选项展示类型。
	Kind string `json:"kind"`
	// SourceType 数据源类型。
	SourceType string `json:"source_type"`
	// SourceValue 数据源值。
	SourceValue string `json:"source_value"`
	// LabelField 标签字段。
	LabelField string `json:"label_field"`
	// ValueField 取值字段。
	ValueField string `json:"value_field"`
	// ParentField 树形父级字段。
	ParentField string `json:"parent_field"`
	// ActiveValue 开关开启值。
	ActiveValue string `json:"active_value"`
	// InactiveValue 开关关闭值。
	InactiveValue string `json:"inactive_value"`
	// Lazy 树形选项是否懒加载。
	Lazy bool `json:"lazy"`
}

// CodeGenStaticOption 描述静态选择项。
type CodeGenStaticOption struct {
	// Label 选择项显示文案。
	Label string `json:"label"`
	// Value 选择项提交值，兼容历史数字和布尔类型。
	Value any `json:"value"`
}

// CodeGenColumnStatusConfig 描述字段状态能力配置。
type CodeGenColumnStatusConfig struct {
	// Enabled 是否启用状态能力。
	Enabled bool `json:"enabled"`
	// DataType 状态数据类型。
	DataType string `json:"data_type"`
	// DictCode 状态字典编码。
	DictCode string `json:"dict_code"`
	// EnumName 状态枚举名称。
	EnumName string `json:"enum_name"`
	// EnabledValue 启用状态值。
	EnabledValue string `json:"enabled_value"`
	// DisabledValue 禁用状态值。
	DisabledValue string `json:"disabled_value"`
	// DefaultValue 默认状态值。
	DefaultValue string `json:"default_value"`
	// GenerateAPI 是否生成状态接口。
	GenerateAPI bool `json:"generate_api"`
	// TableColumn 是否作为列表列。
	TableColumn bool `json:"table_column"`
	// Search 是否支持查询。
	Search bool `json:"search"`
	// Switch 是否使用开关组件。
	Switch bool `json:"switch"`
	// Form 是否在表单展示。
	Form bool `json:"form"`
}

// CodeGenColumnExtraConfig 汇总字段的扩展配置。
type CodeGenColumnExtraConfig struct {
	// Option 选项配置。
	Option CodeGenColumnOptionConfig `json:"option"`
	// Status 状态配置。
	Status CodeGenColumnStatusConfig `json:"status"`
}

// CodeGenLeftTreeConfig 描述左树右表页面配置。
type CodeGenLeftTreeConfig struct {
	// Enabled 是否启用左树布局。
	Enabled bool `json:"enabled"`
	// SourceType 左树数据源类型。
	SourceType string `json:"source_type"`
	// SourceValue 左树数据源值。
	SourceValue string `json:"source_value"`
	// Comment 左树数据表描述。
	Comment string `json:"comment"`
	// FilterColumn 列表关联筛选字段。
	FilterColumn string `json:"filter_column"`
	// ParentColumn 树节点父级字段。
	ParentColumn string `json:"parent_column"`
	// LabelColumn 树节点标签字段。
	LabelColumn string `json:"label_column"`
	// ValueColumn 树节点取值字段。
	ValueColumn string `json:"value_column"`
	// Lazy 是否按节点懒加载子节点。
	Lazy bool `json:"lazy"`
}
