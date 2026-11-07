package biz

import (
	"context"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/scripting"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestRunBaseScriptRejectsDrafts 验证试运行在查询保存脚本前拒绝草稿源码和语言。
func TestRunBaseScriptRejectsDrafts(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{"missing_id", `{"language":"BASE_SCRIPT_LANGUAGE_JAVASCRIPT","source":"1+1"}`},
		{"negative_id", `{"id":"-1","language":"BASE_SCRIPT_LANGUAGE_JAVASCRIPT","source":"1+1"}`},
		{"saved_with_language", `{"id":"1","language":"BASE_SCRIPT_LANGUAGE_JAVASCRIPT"}`},
		{"saved_with_source", `{"id":"1","source":"1+1"}`},
		{"saved_with_zero_language", `{"id":"1","language":0}`},
		{"saved_with_empty_source", `{"id":"1","source":""}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if failure := recover(); failure != nil {
					t.Fatalf("非法试运行请求进入了数据库或认证依赖：%v", failure)
				}
			}()
			req := &adminv1.TestRunBaseScriptRequest{}
			err := protojson.Unmarshal([]byte(test.json), req)
			if err != nil {
				t.Fatal(err)
			}
			c := &BaseScriptCase{}
			_, err = c.TestRunBaseScript(context.Background(), req)
			if errors.Reason(err) != errorsx.ReasonInvalidArgument {
				t.Fatalf("非法试运行请求应返回参数错误，实际：%v", err)
			}
		})
	}
}

// TestRunBaseScriptProtocolRequiresSavedID 验证传输契约拒绝草稿字段，包括显式空值。
func TestRunBaseScriptProtocolRequiresSavedID(t *testing.T) {
	cases := []struct {
		name  string
		json  string
		valid bool
	}{
		{"saved_id", `{"id":"1","context_json":"{}"}`, true},
		{"missing_id", `{}`, false},
		{"negative_id", `{"id":"-1"}`, false},
		{"language", `{"id":"1","language":"BASE_SCRIPT_LANGUAGE_JAVASCRIPT"}`, false},
		{"source", `{"id":"1","source":"1+1"}`, false},
		{"zero_language", `{"id":"1","language":0}`, false},
		{"empty_source", `{"id":"1","source":""}`, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			req := &adminv1.TestRunBaseScriptRequest{}
			err := protojson.Unmarshal([]byte(test.json), req)
			if err != nil {
				t.Fatal(err)
			}
			err = protovalidate.Validate(req)
			if (err == nil) != test.valid {
				t.Fatalf("试运行契约校验错误：valid=%t，实际错误：%v", test.valid, err)
			}
		})
	}
}

// TestRunSavedBaseScriptUsesRestrictedRuntime 验证已保存脚本同样受限，执行结果仍记录审计。
func TestRunSavedBaseScriptUsesRestrictedRuntime(t *testing.T) {
	cases := []struct {
		language string
		source   string
		success  int32
		output   string
	}{
		{scripting.LuaLanguage, `return io.popen("printf script-process-marker")`, 0, ""},
		{scripting.TclLanguage, "exec echo script-process-marker", 0, ""},
		{scripting.JavaScriptLanguage, "({total: ctx.value + 1})", 1, `{"total":3}`},
	}
	for _, test := range cases {
		t.Run(test.language, func(t *testing.T) {
			stored := &models.BaseScript{ID: 7, Name: "saved_script", Language: test.language, Source: test.source, Version: 3, Status: 1}
			logs := &baseScriptSecurityLogRepository{}
			c := &BaseScriptCase{
				BaseCase:             &biz.BaseCase{},
				BaseScriptRepository: &data.BaseScriptRepository{BaseRepository: &baseScriptSecurityRepository{item: stored}},
				baseScriptLog:        &BaseScriptLogCase{BaseScriptLogRepository: &data.BaseScriptLogRepository{BaseRepository: logs}},
				runtime:              scripting.NewRuntime(),
			}
			identity := &authdata.UserTokenPayload{TenantId: 1, TenantCode: "default", UserId: 2, RoleCode: "admin"}
			ctx := engine.ContextWithAuthClaims(context.Background(), identity.MakeAuthClaims())
			response, err := c.TestRunBaseScript(ctx, &adminv1.TestRunBaseScriptRequest{Id: 7, ContextJson: `{"value":2}`})
			if err != nil {
				t.Fatal(err)
			}
			if response.Success != (test.success == 1) || response.OutputJson != test.output {
				t.Fatalf("已保存脚本执行结果错误：%+v", response)
			}
			if test.success == 0 && response.Error == "" {
				t.Fatal("拒绝高风险脚本时必须返回错误原因")
			}
			if logs.item == nil || logs.item.ScriptID != 7 || logs.item.Version != 3 || logs.item.Success != test.success || logs.item.UserID != 2 {
				t.Fatalf("未正确记录已保存脚本执行结果：%+v", logs.item)
			}
		})
	}
}

// TestValidateScriptFormRejectsUnsafeLanguages 验证创建和更新无法重新开放高风险引擎。
func TestValidateScriptFormRejectsUnsafeLanguages(t *testing.T) {
	for _, language := range []adminv1.BaseScriptLanguage{
		adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_LUA,
		adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_PYTHON,
		adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_GO,
		adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_WASM,
		adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_STARLARK,
		adminv1.BaseScriptLanguage_BASE_SCRIPT_LANGUAGE_TCL,
	} {
		t.Run(language.String(), func(t *testing.T) {
			err := validateScriptForm(&adminv1.BaseScriptForm{Language: language, Status: 1})
			if errors.Reason(err) != errorsx.ReasonInvalidArgument {
				t.Fatalf("不受限语言应被拒绝，实际：%v", err)
			}
		})
	}
}

// baseScriptSecurityRepository 提供安全回归测试所需的已保存脚本读取边界。
type baseScriptSecurityRepository struct {
	repository.BaseRepository[models.BaseScript]
	item *models.BaseScript
}

// FindByID 返回测试中保存的脚本，仅支持约定的脚本编号。
func (r *baseScriptSecurityRepository) FindByID(_ context.Context, id int64) (*models.BaseScript, error) {
	if id != r.item.ID {
		return nil, errorsx.ResourceNotFound("脚本不存在")
	}
	return r.item, nil
}

// baseScriptSecurityLogRepository 接收安全回归测试产生的执行审计。
type baseScriptSecurityLogRepository struct {
	repository.BaseRepository[models.BaseScriptLog]
	item *models.BaseScriptLog
}

// Create 保存实际业务生成的脚本执行审计。
func (r *baseScriptSecurityLogRepository) Create(_ context.Context, item *models.BaseScriptLog) error {
	r.item = item
	return nil
}
