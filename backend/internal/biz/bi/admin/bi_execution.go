package admin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/query"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth/authz/engine"
)

// compile 冻结当前源模型的执行定义，配置源始终保留项目和宿主连接边界。
func (c *BiSourceCase) compile(ctx context.Context, scope bi.Scope, spec bi.QuerySpec) (*dto.CardExecution, error) {
	var err error
	if c.registry.Has(spec.SourceCode) {
		var descriptors []bi.SourceDescriptor
		descriptors, err = c.registry.List(ctx, scope)
		if err != nil {
			return nil, err
		}
		for _, descriptor := range descriptors {
			if descriptor.Code == spec.SourceCode {
				return query.AdapterExecution(descriptor, spec)
			}
		}
		return nil, errorsx.ResourceNotFound("BI 数据源不存在")
	}
	if scope.ProjectID != 0 {
		return nil, errorsx.PermissionDenied("可配置数据源暂不支持项目范围查询")
	}
	var item *models.BiSource
	var model dto.ConfiguredSourceSchema
	item, model, err = c.findEnabledSource(ctx, scope.TenantID, spec.SourceCode)
	if err != nil {
		return nil, err
	}
	if err = c.requireHostClientAccess(ctx, item.Host); err != nil {
		return nil, err
	}
	var driver string
	driver, err = c.driverForSource(item)
	if err != nil {
		return nil, err
	}
	return query.Compile(driver, item.MaxRows, model, spec)
}

// driverForSource 使用当前连接的实际方言，宿主客户端不能由表单伪造驱动。
func (c *BiSourceCase) driverForSource(item *models.BiSource) (string, error) {
	if item.Host != "" {
		return item.Driver, nil
	}
	client := c.GormClients[item.SourceName]
	if client == nil {
		return "", errorsx.InvalidArgument("BI 数据源绑定的数据库客户端不存在")
	}
	return client.Driver(), nil
}

// checkExecution 在执行前后核对定义完整性和当前修订，不在每次取数时重新拼 SQL。
func (c *BiSourceCase) checkExecution(ctx context.Context, scope bi.Scope, spec bi.QuerySpec, execution *dto.CardExecution) error {
	if execution.Version != 1 || execution.QueryHash != query.SpecHash(spec) {
		return errorsx.Conflict("卡片执行定义已变化，请刷新")
	}
	copyExecution := *execution
	err := query.Seal(&copyExecution)
	if err != nil {
		return err
	}
	if copyExecution.Hash != execution.Hash {
		return errorsx.Conflict("卡片执行定义校验失败")
	}
	if execution.Kind == "adapter" {
		var current *dto.CardExecution
		current, err = c.compile(ctx, scope, spec)
		if err != nil {
			return err
		}
		if current.Hash != execution.Hash {
			return errorsx.Conflict("公共指标已更新，请重新发布或刷新")
		}
		return nil
	}
	if execution.Kind != "configured_sql" || scope.ProjectID != 0 || c.registry.Has(spec.SourceCode) {
		return errorsx.PermissionDenied("卡片执行范围无效")
	}
	var item *models.BiSource
	var model dto.ConfiguredSourceSchema
	item, model, err = c.findEnabledSource(ctx, scope.TenantID, spec.SourceCode)
	if err != nil {
		return err
	}
	if err = c.requireHostClientAccess(ctx, item.Host); err != nil {
		return err
	}
	var driver string
	driver, err = c.driverForSource(item)
	if err != nil {
		return err
	}
	if driver != execution.Driver || model.ModelRevision != execution.ModelRevision || len(execution.Metrics) != len(spec.Measures) {
		return errorsx.Conflict("公共指标或模型已更新，请重新发布或刷新")
	}
	for _, selected := range execution.Metrics {
		found := false
		for _, metric := range model.Measures {
			if metric.Code == selected.Code && metric.Revision == selected.Revision {
				found = true
				break
			}
		}
		if !found {
			return errorsx.Conflict("公共指标已更新，请重新发布或刷新")
		}
	}
	return nil
}

// executeDefinition 使用保存的 SQL 和执行时绑定参数，候选模型预览不走当前版本校验。
func (c *BiSourceCase) executeDefinition(ctx context.Context, scope bi.Scope, spec bi.QuerySpec, execution *dto.CardExecution, days int32, now time.Time) (bi.QueryResult, error) {
	var err error
	if execution.Kind == "adapter" {
		executionSpec := spec
		var descriptors []bi.SourceDescriptor
		descriptors, err = c.registry.List(ctx, scope)
		if err != nil {
			return bi.QueryResult{}, err
		}
		for _, descriptor := range descriptors {
			if descriptor.Code == spec.SourceCode {
				executionSpec = applyBiCardPeriod(spec, descriptor, days, now)
				break
			}
		}
		executionSpec.Limit = execution.Limit
		if executionSpec.TimeRange != nil {
			var start, end string
			start, end, err = executionTimeRange(executionSpec, 0, now)
			if err != nil {
				return bi.QueryResult{}, err
			}
			executionSpec.Filters = append(append([]bi.FilterSpec{}, executionSpec.Filters...), bi.FilterSpec{Field: executionSpec.TimeRange.Field, Operator: "between", Values: []string{start, end}})
			executionSpec.TimeRange = nil
		}
		return c.registry.Execute(ctx, scope, executionSpec)
	}
	var item *models.BiSource
	item, _, err = c.findEnabledSource(ctx, scope.TenantID, spec.SourceCode)
	if err != nil {
		return bi.QueryResult{}, err
	}
	var database *configuredDatabase
	database, err = c.openSource(ctx, item)
	if err != nil {
		return bi.QueryResult{}, err
	}
	defer database.Close()
	var args []any
	args, err = bindExecutionParameters(execution, spec, scope, days, now)
	if err != nil {
		return bi.QueryResult{}, err
	}
	return executeConfiguredDefinition(ctx, database, execution, args, execution.Limit)
}

// requireModelPermission 在业务层检查公共模型管理权，覆盖所有传输及进程内调用。
func (c *BiSourceCase) requireModelPermission(ctx context.Context) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if c.authorizer == nil {
		return errorsx.PermissionDenied("无权修改数据源公共指标")
	}
	tenant := engine.Tenant(authInfo.TenantCode)
	authzCtx := engine.ContextWithAuthClaims(ctx, &engine.AuthClaims{Tenant: &tenant})
	var allowed bool
	allowed, err = c.authorizer.IsAuthorized(authzCtx, engine.Subject(authInfo.RoleCode), engine.Action("PUT"), engine.Resource("/bi.admin.v1.BiSourceService/SaveBiSourceModel"), "")
	if err != nil {
		return errorsx.Internal("校验数据源模型权限失败").WithCause(err)
	}
	if !allowed {
		return errorsx.PermissionDenied("无权修改数据源公共指标")
	}
	return nil
}

// decodeExecution 读取当前闭合执行定义，不解释旧版本契约。
func decodeExecution(value string) (*dto.CardExecution, error) {
	var execution dto.CardExecution
	err := json.Unmarshal([]byte(value), &execution)
	if err != nil || execution.Version != 1 {
		return nil, errorsx.Conflict("卡片执行定义尚未生成，请重新问数保存")
	}
	return &execution, nil
}

// bindExecutionParameters 只从受信身份、保存的过滤配置和当前时间取得参数。
func bindExecutionParameters(execution *dto.CardExecution, spec bi.QuerySpec, scope bi.Scope, days int32, now time.Time) ([]any, error) {
	start, end, err := executionTimeRange(spec, days, now)
	if err != nil {
		return nil, err
	}
	args := make([]any, 0, len(execution.Parameters))
	for index, parameter := range execution.Parameters {
		if parameter.Position != index+1 {
			return nil, errorsx.Internal("执行参数顺序无效")
		}
		switch parameter.Kind {
		case "scope":
			if parameter.Field != "tenant_id" {
				return nil, errorsx.Internal("执行身份参数无效")
			}
			args = append(args, scope.TenantID)
		case "filter":
			if parameter.FilterIndex < 0 || parameter.FilterIndex >= len(spec.Filters) || parameter.ValueIndex < 0 || parameter.ValueIndex >= len(spec.Filters[parameter.FilterIndex].Values) {
				return nil, errorsx.Internal("执行过滤参数无效")
			}
			value := spec.Filters[parameter.FilterIndex].Values[parameter.ValueIndex]
			if parameter.Contains {
				value = "%" + value + "%"
			}
			args = append(args, value)
		case "time_enabled":
			value := 0
			if start != "" && end != "" {
				value = 1
			}
			args = append(args, value)
		case "time_start", "time_end":
			var value any
			if start != "" && end != "" {
				value = start
				if parameter.Kind == "time_end" {
					value = end
				}
			}
			args = append(args, value)
		case "time_override":
			value := 0
			if days != 0 {
				value = 1
			}
			args = append(args, value)
		default:
			return nil, errorsx.Internal("执行参数来源无效")
		}
	}
	return args, nil
}

// executionTimeRange 在执行时计算卡片原始时间配置或受控周期覆盖。
func executionTimeRange(spec bi.QuerySpec, days int32, now time.Time) (string, string, error) {
	if days != 0 {
		if days != 7 && days != 30 {
			return "", "", errorsx.InvalidArgument("仅支持卡片配置、近7日或近30日")
		}
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return today.AddDate(0, 0, 1-int(days)).Format(biTimeLayout), today.AddDate(0, 0, 1).Add(-time.Millisecond).Format("2006-01-02 15:04:05.999"), nil
	}
	if spec.TimeRange == nil {
		return "", "", nil
	}
	start, err := resolveRelativeTime(spec.TimeRange.Start, now)
	if err != nil {
		return "", "", err
	}
	var end string
	end, err = resolveRelativeTime(spec.TimeRange.End, now)
	return start, end, err
}
