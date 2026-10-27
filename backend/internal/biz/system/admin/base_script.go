package biz

import (
	"context"
	"time"

	stringutil "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/scripting"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	corebiz "github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseScriptCase 脚本管理业务实例。
type BaseScriptCase struct {
	*corebiz.BaseCase
	*data.BaseScriptRepository
	tx            data.Transaction
	baseScriptLog *BaseScriptLogCase
	runtime       *scripting.Runtime
}

// NewBaseScriptCase 创建脚本管理业务实例。
func NewBaseScriptCase(
	baseCase *corebiz.BaseCase,
	tx data.Transaction,
	repo *data.BaseScriptRepository,
	baseScriptLog *BaseScriptLogCase,
	runtime *scripting.Runtime,
) *BaseScriptCase {
	return &BaseScriptCase{
		BaseCase:             baseCase,
		BaseScriptRepository: repo,
		tx:                   tx,
		baseScriptLog:        baseScriptLog,
		runtime:              runtime,
	}
}

// PageBaseScript 分页查询脚本列表。
func (c *BaseScriptCase) PageBaseScript(ctx context.Context, req *adminv1.PageBaseScriptRequest) (*adminv1.PageBaseScriptResponse, error) {
	query := c.Query(ctx).BaseScript
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.GetHookPoint() != "" {
		opts = append(opts, repository.Where(query.HookPoint.Like("%"+req.GetHookPoint()+"%")))
	}
	if req.Language != nil && req.GetLanguage() != adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_UNSPECIFIED {
		language := scriptLanguageString(req.GetLanguage())
		if language != "" {
			opts = append(opts, repository.Where(query.Language.Eq(language)))
		}
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	items, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseScript, 0, len(items))
	for _, item := range items {
		result = append(result, toBaseScript(item))
	}
	return &adminv1.PageBaseScriptResponse{BaseScripts: result, Total: int32(total)}, nil
}

// GetBaseScript 获取脚本详情。
func (c *BaseScriptCase) GetBaseScript(ctx context.Context, id int64) (*adminv1.BaseScriptForm, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBaseScriptForm(item), nil
}

// CreateBaseScript 创建脚本。
func (c *BaseScriptCase) CreateBaseScript(ctx context.Context, req *adminv1.BaseScriptForm) error {
	if err := validateScriptForm(req); err != nil {
		return err
	}
	entity := baseScriptEntity(req)
	entity.Version = 1
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err := c.Create(txCtx, entity); err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("脚本名称重复", "base_script", "name", "unique_base_script_name").WithCause(err)
			}
			return err
		}
		return nil
	})
}

// UpdateBaseScript 更新脚本并递增版本号。
func (c *BaseScriptCase) UpdateBaseScript(ctx context.Context, req *adminv1.BaseScriptForm) error {
	if err := validateScriptForm(req); err != nil {
		return err
	}
	previous, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	entity := baseScriptEntity(req)
	entity.ID = previous.ID
	entity.Version = previous.Version + 1
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err := c.UpdateByID(txCtx, entity); err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("脚本名称重复", "base_script", "name", "unique_base_script_name").WithCause(err)
			}
			return err
		}
		return nil
	})
}

// DeleteBaseScript 删除脚本，执行日志保留用于审计。
func (c *BaseScriptCase) DeleteBaseScript(ctx context.Context, id string) error {
	ids := stringutil.ConvertStringToInt64Array(id)
	if len(ids) == 0 {
		return errorsx.InvalidArgument("脚本ID列表不能为空")
	}
	return c.DeleteByIDs(ctx, ids)
}

// SetBaseScriptStatus 设置脚本启用或停用状态。
func (c *BaseScriptCase) SetBaseScriptStatus(ctx context.Context, req *adminv1.SetBaseScriptStatusRequest) error {
	if !scriptStatusValid(int32(req.GetStatus())) {
		return errorsx.InvalidArgument("脚本状态无效")
	}
	if _, err := c.FindByID(ctx, req.GetId()); err != nil {
		return err
	}
	return c.UpdateByID(ctx, &models.BaseScript{ID: req.GetId(), Status: int32(req.GetStatus())})
}

// TestRunBaseScript 在隔离运行时执行已保存脚本或草稿源码。
func (c *BaseScriptCase) TestRunBaseScript(ctx context.Context, req *adminv1.TestRunBaseScriptRequest) (*adminv1.TestRunBaseScriptResponse, error) {
	return c.testRunBaseScript(ctx, req, false)
}

// testRunBaseScript 复用保存脚本读取与审计，流程调用选择无宿主能力的专用执行器。
func (c *BaseScriptCase) testRunBaseScript(ctx context.Context, req *adminv1.TestRunBaseScriptRequest, workflowOnly bool) (*adminv1.TestRunBaseScriptResponse, error) {
	var scriptID int64
	var scriptName string
	var language string
	var source string
	var version int32
	if req.GetId() > 0 {
		item, err := c.FindByID(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		scriptID = item.ID
		scriptName = item.Name
		language = item.Language
		source = item.Source
		version = item.Version
	} else {
		scriptName = "draft"
		language = scriptLanguageString(req.GetLanguage())
		source = req.GetSource()
	}
	if language == "" {
		return nil, errorsx.InvalidArgument("脚本语言无效")
	}
	if source == "" {
		return nil, errorsx.InvalidArgument("脚本源码不能为空")
	}
	input, err := scripting.ParseContext(req.GetContextJson())
	if err != nil {
		return nil, errorsx.InvalidArgument(err.Error())
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	var output any
	if workflowOnly {
		output, err = c.runtime.ExecuteWorkflow(ctx, language, source, input)
	} else {
		output, err = c.runtime.Execute(ctx, language, source, input)
	}
	response := &adminv1.TestRunBaseScriptResponse{Success: err == nil}
	if err != nil {
		response.Error = err.Error()
	} else {
		response.OutputJson, err = scripting.MarshalOutput(output)
		if err != nil {
			response.Success = false
			response.Error = err.Error()
		}
	}
	response.DurationMs = time.Since(started).Milliseconds()
	logItem := &models.BaseScriptLog{
		ScriptID:    scriptID,
		ScriptName:  scriptName,
		Language:    language,
		TriggerType: "test_run",
		Version:     version,
		Success:     boolToInt32(response.Success),
		DurationMs:  response.DurationMs,
		Error:       response.Error,
		TenantID:    authInfo.TenantId,
		UserID:      authInfo.UserId,
	}
	if logErr := c.baseScriptLog.Record(ctx, logItem); logErr != nil {
		return nil, errorsx.WrapInternal(logErr, "记录脚本执行日志失败")
	}
	return response, nil
}

// validateScriptForm 校验脚本表单的业务边界。
func validateScriptForm(req *adminv1.BaseScriptForm) error {
	if req.GetId() < 0 {
		return errorsx.InvalidArgument("脚本ID无效")
	}
	if scriptLanguageString(req.GetLanguage()) == "" {
		return errorsx.InvalidArgument("脚本语言无效")
	}
	if !scriptStatusValid(int32(req.GetStatus())) {
		return errorsx.InvalidArgument("脚本状态无效")
	}
	return nil
}

// baseScriptEntity 将脚本表单转换为数据库模型。
func baseScriptEntity(req *adminv1.BaseScriptForm) *models.BaseScript {
	return &models.BaseScript{
		ID:          req.GetId(),
		Name:        req.GetName(),
		Language:    scriptLanguageString(req.GetLanguage()),
		HookPoint:   req.GetHookPoint(),
		Source:      req.GetSource(),
		Priority:    req.GetPriority(),
		Description: req.GetDescription(),
		Critical:    boolToInt32(req.GetCritical()),
		Status:      scriptStatusValue(int32(req.GetStatus())),
	}
}

// toBaseScript 将脚本模型转换为列表对象。
func toBaseScript(item *models.BaseScript) *adminv1.BaseScript {
	if item == nil {
		return nil
	}
	return &adminv1.BaseScript{
		Id:          item.ID,
		Name:        item.Name,
		Language:    scriptLanguageEnum(item.Language),
		HookPoint:   item.HookPoint,
		Priority:    item.Priority,
		Description: item.Description,
		Critical:    item.Critical == 1,
		Version:     item.Version,
		Status:      statusEnum(item.Status),
		CreatedAt:   item.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// toBaseScriptForm 将脚本模型转换为编辑表单。
func toBaseScriptForm(item *models.BaseScript) *adminv1.BaseScriptForm {
	if item == nil {
		return nil
	}
	return &adminv1.BaseScriptForm{
		Id:          item.ID,
		Name:        item.Name,
		Language:    scriptLanguageEnum(item.Language),
		HookPoint:   item.HookPoint,
		Source:      item.Source,
		Priority:    item.Priority,
		Description: item.Description,
		Critical:    item.Critical == 1,
		Status:      statusEnum(item.Status),
	}
}

// scriptLanguageString 将协议语言转换为数据库语言标识。
func scriptLanguageString(language adminv1.BaseScriptLanguage) string {
	switch language {
	case adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_JAVASCRIPT:
		return scripting.JavaScriptLanguage
	case adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_LUA:
		return scripting.LuaLanguage
	case adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_PYTHON:
		return scripting.PythonLanguage
	case adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_GO:
		return scripting.GoLanguage
	case adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_WASM:
		return scripting.WasmLanguage
	case adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_CEL:
		return scripting.CELLanguage
	case adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_EXPR:
		return scripting.ExprLanguage
	case adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_STARLARK:
		return scripting.StarlarkLanguage
	case adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_TCL:
		return scripting.TclLanguage
	}
	return ""
}

// scriptLanguageEnum 将数据库语言标识转换为协议枚举。
func scriptLanguageEnum(language string) adminv1.BaseScriptLanguage {
	switch language {
	case scripting.JavaScriptLanguage:
		return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_JAVASCRIPT
	case scripting.LuaLanguage:
		return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_LUA
	case scripting.PythonLanguage:
		return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_PYTHON
	case scripting.GoLanguage:
		return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_GO
	case scripting.WasmLanguage:
		return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_WASM
	case scripting.CELLanguage:
		return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_CEL
	case scripting.ExprLanguage:
		return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_EXPR
	case scripting.StarlarkLanguage:
		return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_STARLARK
	case scripting.TclLanguage:
		return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_TCL
	}
	return adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_UNSPECIFIED
}

// statusEnum 将数据库状态转换为公共协议枚举。
func statusEnum(status int32) commonv1.Status {
	return commonv1.Status(status)
}

// boolToInt32 将布尔值转换为数据库布尔值。
func boolToInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}
