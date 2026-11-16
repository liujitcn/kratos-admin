package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/query"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/mysql"
)

// TestCloudSCPMySQLJoins 用真实只读业务连接及隔离元数据库验证五表查询和公共指标闭环。
func TestCloudSCPMySQLJoins(t *testing.T) {
	cards, dashboards, ctx, source, database := newCloudSCPTestCases(t, "cloud-scp/default")
	sourceCase := cards.biQueryCase.biSourceCase
	metadata, err := sourceCase.ListBiSourceTables(ctx, &adminv1.ListBiSourceTablesRequest{Id: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("实际发现 %d 张表，五表事实数据量较小，不能代表大数据量性能", len(metadata.Tables))
	model := cloudSCPUserModel(source.DatabaseName, "left")
	request := &adminv1.SaveBiSourceModelRequest{Id: source.ID, ExpectedSourceVersion: 1, Model: model}
	if _, err = sourceCase.PreviewBiSourceModel(ctx, request); err != nil {
		t.Fatal(err)
	}
	var savedSource *adminv1.BiSource
	savedSource, err = sourceCase.SaveBiSourceModel(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if savedSource.Version != 2 {
		t.Fatal("保存模型未更新版本")
	}
	from := " FROM base_user u LEFT JOIN base_dept d ON u.dept_id=d.id AND d.tenant_id=? LEFT JOIN base_role r ON u.role_id=r.id AND r.tenant_id=? LEFT JOIN base_post p ON u.post_id=p.id AND p.tenant_id=? LEFT JOIN base_tenant n ON u.tenant_id=n.id AND n.id=? WHERE u.tenant_id=?"
	measures := []bi.MeasureSpec{{Code: "users"}, {Code: "status_sum"}, {Code: "status_avg"}, {Code: "status_ratio"}, {Code: "identified"}}
	scope := bi.Scope{TenantID: 1, UserID: 10, Terminal: "admin"}
	for _, test := range []struct {
		name       string
		dimensions []string
		measures   []bi.MeasureSpec
		filters    []bi.FilterSpec
		orders     []bi.OrderSpec
		timeRange  *bi.TimeRangeSpec
		limit      int32
		selectSQL  string
		tail       string
		args       []any
	}{
		{name: "五表五指标", measures: measures, selectSQL: "COUNT(*),SUM(u.status),AVG(u.status),SUM(u.status)*1.0/NULLIF(COUNT(*),0),COUNT(u.id)"},
		{name: "双维度分组排序", dimensions: []string{"dept", "role"}, selectSQL: "d.name,r.name,COUNT(*)", tail: " GROUP BY d.name,r.name ORDER BY COUNT(*) DESC,d.name ASC", orders: []bi.OrderSpec{{Field: "users", Direction: "desc"}, {Field: "dept", Direction: "asc"}}},
		{name: "岗位维度分组", dimensions: []string{"post"}, selectSQL: "p.name,COUNT(*)", tail: " GROUP BY p.name"},
		{name: "租户维度", dimensions: []string{"tenant"}, selectSQL: "n.code,COUNT(*)", tail: " GROUP BY n.code"},
		{name: "等值筛选", filters: []bi.FilterSpec{{Field: "status", Operator: "eq", Values: []string{"1"}}}, selectSQL: "COUNT(*)", tail: " AND u.status=?", args: []any{"1"}},
		{name: "IN筛选", filters: []bi.FilterSpec{{Field: "status", Operator: "in", Values: []string{"1", "2"}}}, selectSQL: "COUNT(*)", tail: " AND u.status IN (?,?)", args: []any{"1", "2"}},
		{name: "区间筛选", filters: []bi.FilterSpec{{Field: "status", Operator: "between", Values: []string{"0", "2"}}}, selectSQL: "COUNT(*)", tail: " AND u.status BETWEEN ? AND ?", args: []any{"0", "2"}},
		{name: "维表包含筛选", filters: []bi.FilterSpec{{Field: "role", Operator: "contains", Values: []string{"管理"}}}, selectSQL: "COUNT(*)", tail: " AND r.name LIKE ?", args: []any{"%管理%"}},
		{name: "组合筛选", filters: []bi.FilterSpec{{Field: "status", Operator: "eq", Values: []string{"1"}}, {Field: "dept", Operator: "contains", Values: []string{"部"}}}, selectSQL: "COUNT(*)", tail: " AND u.status=? AND d.name LIKE ?", args: []any{"1", "%部%"}},
		{name: "注入值参数化", filters: []bi.FilterSpec{{Field: "role", Operator: "eq", Values: []string{"x' OR 1=1 --"}}}, selectSQL: "COUNT(*)", tail: " AND r.name=?", args: []any{"x' OR 1=1 --"}},
		{name: "空聚合与NULL", measures: measures, filters: []bi.FilterSpec{{Field: "status", Operator: "eq", Values: []string{"-999"}}}, selectSQL: "COUNT(*),SUM(u.status),AVG(u.status),SUM(u.status)*1.0/NULLIF(COUNT(*),0),COUNT(u.id)", tail: " AND u.status=?", args: []any{"-999"}},
		{name: "空分组", dimensions: []string{"dept"}, filters: []bi.FilterSpec{{Field: "status", Operator: "eq", Values: []string{"-999"}}}, selectSQL: "d.name,COUNT(*)", tail: " AND u.status=? GROUP BY d.name", args: []any{"-999"}},
		{name: "时间范围", timeRange: &bi.TimeRangeSpec{Field: "created", Start: "2000-01-01 00:00:00", End: "2100-01-01 00:00:00"}, selectSQL: "COUNT(*)", tail: " AND u.created_at BETWEEN ? AND ?", args: []any{"2000-01-01 00:00:00", "2100-01-01 00:00:00"}},
		{name: "Top1", dimensions: []string{"role"}, orders: []bi.OrderSpec{{Field: "role", Direction: "asc"}}, limit: 1, selectSQL: "r.name,COUNT(*)", tail: " GROUP BY r.name ORDER BY r.name ASC LIMIT 1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			selected := test.measures
			if selected == nil {
				selected = []bi.MeasureSpec{{Code: "users"}}
			}
			spec := bi.QuerySpec{SourceCode: source.Code, Dimensions: test.dimensions, Measures: selected, Filters: test.filters, Orders: test.orders, TimeRange: test.timeRange, Limit: test.limit}
			var result bi.QueryResult
			result, err = cards.biQueryCase.execute(ctx, scope, spec, nil)
			if err != nil {
				t.Fatal(err)
			}
			args := append([]any{1, 1, 1, 1, 1}, test.args...)
			checkCloudSCPRows(t, database.db, result, "SELECT "+test.selectSQL+from+test.tail, args, len(test.orders) > 0)
			t.Logf("结果 %d 行，BI 耗时 %dms", len(result.Rows), result.ElapsedMS)
		})
	}

	t.Run("乱序链式关联", func(t *testing.T) {
		candidate := proto.Clone(model).(*adminv1.BiSourceModel)
		candidate.Relations[3].LeftTable = sourceTableKey(source.DatabaseName, "base_dept")
		slices.Reverse(candidate.Relations)
		request.Model, request.ExpectedSourceVersion = candidate, 2
		if _, err = sourceCase.SaveBiSourceModel(ctx, request); err != nil {
			t.Fatal(err)
		}
		var result bi.QueryResult
		result, err = sourceCase.Execute(ctx, scope, bi.QuerySpec{SourceCode: source.Code, Dimensions: []string{"tenant"}, Measures: []bi.MeasureSpec{{Code: "users"}}})
		if err != nil {
			t.Fatal(err)
		}
		checkCloudSCPRows(t, database.db, result, "SELECT n.code,COUNT(*)"+strings.Replace(from, "u.tenant_id=n.id", "d.tenant_id=n.id", 1)+" GROUP BY n.code", []any{1, 1, 1, 1, 1}, false)
	})
	t.Run("五表INNER关联", func(t *testing.T) {
		request.Model, request.ExpectedSourceVersion = cloudSCPUserModel(source.DatabaseName, "inner"), 3
		if _, err = sourceCase.SaveBiSourceModel(ctx, request); err != nil {
			t.Fatal(err)
		}
		var result bi.QueryResult
		result, err = sourceCase.Execute(ctx, scope, bi.QuerySpec{SourceCode: source.Code, Measures: []bi.MeasureSpec{{Code: "users"}}})
		if err != nil {
			t.Fatal(err)
		}
		checkCloudSCPRows(t, database.db, result, "SELECT COUNT(*)"+strings.ReplaceAll(from, "LEFT JOIN", "INNER JOIN"), []any{1, 1, 1, 1, 1}, false)
		t.Logf("INNER 匹配数量 %s", result.Rows[0][0])
	})

	t.Run("非法关系拒绝", func(t *testing.T) {
		for _, test := range []struct {
			name   string
			mutate func(*adminv1.BiSourceModel)
		}{
			{name: "非唯一右键", mutate: func(m *adminv1.BiSourceModel) { m.Relations[0].RightColumn = "tenant_id" }},
			{name: "字段类型不同", mutate: func(m *adminv1.BiSourceModel) { m.Relations[0].LeftColumn = "user_name" }},
			{name: "重复父节点", mutate: func(m *adminv1.BiSourceModel) {
				m.Relations[1] = proto.Clone(m.Relations[0]).(*adminv1.BiSourceRelation)
			}},
			{name: "关系回环", mutate: func(m *adminv1.BiSourceModel) { m.Relations[3].RightTable = m.BaseTable }},
			{name: "断开子图", mutate: func(m *adminv1.BiSourceModel) {
				m.Relations[0].LeftTable = m.Relations[1].RightTable
				m.Relations[1].LeftTable = m.Relations[0].RightTable
			}},
			{name: "重复表", mutate: func(m *adminv1.BiSourceModel) { m.Tables[1] = proto.Clone(m.Tables[0]).(*adminv1.BiSourceModelTable) }},
		} {
			t.Run(test.name, func(t *testing.T) {
				candidate := proto.Clone(model).(*adminv1.BiSourceModel)
				test.mutate(candidate)
				_, err = configuredSourceModel(candidate, metadata.Tables)
				if errors.Code(err) != 400 {
					t.Fatalf("非法关系未被拒绝: %v", err)
				}
			})
		}
	})

	request.Model, request.ExpectedSourceVersion = model, 4
	if _, err = sourceCase.SaveBiSourceModel(ctx, request); err != nil {
		t.Fatal(err)
	}
	t.Run("多表卡片公共指标同步发布闭环", func(t *testing.T) {
		var savedCards []*adminv1.BiCard
		for _, dimensions := range [][]string{nil, {"dept", "role"}} {
			spec := bi.QuerySpec{SourceCode: source.Code, Dimensions: dimensions, Measures: []bi.MeasureSpec{{Code: "users"}}, Limit: 100}
			chartType := "metric"
			if len(dimensions) > 0 {
				chartType = "bar"
			}
			var result bi.QueryResult
			result, err = cards.biQueryCase.execute(ctx, scope, spec, &dto.BiQueryRecordContext{Question: "测试用户数"})
			if err != nil {
				t.Fatal(err)
			}
			var saved *adminv1.BiCard
			saved, err = cards.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{ExpectedSourceVersion: 5, Card: &adminv1.BiCardForm{Name: "测试卡片", Query: toProtoQuerySpec(spec), QueryRecordId: result.QueryRecordID, ChartType: chartType}})
			if err != nil {
				t.Fatal(err)
			}
			saved, err = cards.GetBiCard(ctx, &adminv1.GetBiCardRequest{Id: saved.Id})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(saved.Sql, "JOIN") != 4 {
				t.Fatal("卡片未保存完整多表 SQL")
			}
			savedCards = append(savedCards, saved)
		}
		layout := fmt.Sprintf(`{"schemaVersion":4,"name":"关联测试","cards":[{"id":"a","card_id":%d,"x":0,"y":0,"w":6,"h":4},{"id":"b","card_id":%d,"x":6,"y":0,"w":6,"h":4}]}`, savedCards[0].Id, savedCards[1].Id)
		var dashboard *adminv1.BiDashboard
		dashboard, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: &adminv1.BiDashboard{Code: "cloud_scp_test", Name: "关联测试", SchemaJson: layout}})
		if err != nil {
			t.Fatal(err)
		}
		dashboard, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: dashboard.Id, Version: dashboard.DraftVersion})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = dashboards.GetBiDashboardData(ctx, &adminv1.GetBiDashboardDataRequest{Id: dashboard.Id}); err != nil {
			t.Fatal(err)
		}
		changes := []*adminv1.BiMetricChange{{MetricCode: "users", Name: "用户数", Expression: "COUNT(*) * 2", Unit: "人", ExpectedRevision: 2}}
		var preview *adminv1.PreviewBiCardSqlResponse
		preview, err = cards.PreviewBiCardSql(ctx, &adminv1.PreviewBiCardSqlRequest{Id: savedCards[0].Id, ExpectedCardVersion: 1, ExpectedSourceVersion: 5, QueryRecordId: savedCards[0].QueryRecordId, Query: savedCards[0].Query, MeasureChanges: changes})
		if err != nil {
			t.Fatal(err)
		}
		if preview.Impact.AffectedCards != 2 || preview.Impact.AffectedDashboards != 1 {
			t.Fatal("多表模型影响范围错误")
		}
		if _, err = cards.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{Id: savedCards[0].Id, ExpectedCardVersion: 1, ExpectedSourceVersion: 5, PreviewQueryRecordId: preview.PreviewQueryRecordId, MeasureChanges: changes, Card: &adminv1.BiCardForm{Name: "测试双倍人数", Query: savedCards[0].Query, QueryRecordId: savedCards[0].QueryRecordId, ChartType: "metric"}}); err != nil {
			t.Fatal(err)
		}
		for _, card := range savedCards {
			var item *models.BiCard
			item, err = cards.cardRepo.FindByID(ctx, card.Id)
			if err != nil || item.Version != 2 || !strings.Contains(item.ExecutionJSON, "* 2") {
				t.Fatal("关联卡片未同步公共指标", err)
			}
			var execution dto.CardExecution
			if err = json.Unmarshal([]byte(item.ExecutionJSON), &execution); err != nil {
				t.Fatal(err)
			}
			spec := fromProtoQuerySpec(card.Query)
			var result bi.QueryResult
			result, err = sourceCase.executeDefinition(ctx, scope, spec, &execution, 0, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			selectSQL, tail := "COUNT(*)*2", ""
			if len(spec.Dimensions) > 0 {
				selectSQL, tail = "d.name,r.name,COUNT(*)*2", " GROUP BY d.name,r.name"
			}
			checkCloudSCPRows(t, database.db, result, "SELECT "+selectSQL+from+tail, []any{1, 1, 1, 1, 1}, false)
		}
		for _, version := range []int64{0, dashboard.PublishedVersion} {
			if _, err = dashboards.GetBiDashboardData(ctx, &adminv1.GetBiDashboardDataRequest{Id: dashboard.Id, Version: version}); errors.Code(err) != 409 {
				t.Fatalf("旧发布仍可执行: %v", err)
			}
		}
		dashboard, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: dashboard})
		if err != nil {
			t.Fatal(err)
		}
		dashboard, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: dashboard.Id, Version: dashboard.DraftVersion})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = dashboards.GetBiDashboardData(ctx, &adminv1.GetBiDashboardDataRequest{Id: dashboard.Id}); err != nil {
			t.Fatal("重新发布后无法执行", err)
		}
	})
}

// TestCloudSCPDictionaryJoins 用真实字典明细覆盖多分组、数值排序、截断和不同连接形式。
func TestCloudSCPDictionaryJoins(t *testing.T) {
	cards, _, ctx, source, database := newCloudSCPTestCases(t, "cloud-scp/default")
	sourceCase := cards.biQueryCase.biSourceCase
	base, dictionary := sourceTableKey(source.DatabaseName, "base_dict_item"), sourceTableKey(source.DatabaseName, "base_dict")
	model := &adminv1.BiSourceModel{BaseTable: base, Tables: []*adminv1.BiSourceModelTable{{Key: base}, {Key: dictionary}}, Relations: []*adminv1.BiSourceRelation{{LeftTable: base, LeftColumn: "dict_id", RightTable: dictionary, RightColumn: "id", JoinType: "left"}}, Dimensions: []*adminv1.BiSourceDimension{{Code: "dict", Name: "字典", Type: "string", Table: dictionary, Column: "code", Filterable: true}, {Code: "value", Name: "选项", Type: "string", Table: base, Column: "value", Filterable: true}, {Code: "status", Name: "状态", Type: "number", Table: base, Column: "status", Filterable: true}}, Measures: []*adminv1.BiSourceMeasure{{Code: "items", Name: "选项数", Type: "number", Expression: "COUNT(*)"}, {Code: "sort_total", Name: "排序合计", Type: "number", Expression: "SUM(sort)"}}}
	_, err := sourceCase.SaveBiSourceModel(ctx, &adminv1.SaveBiSourceModelRequest{Id: source.ID, ExpectedSourceVersion: 1, Model: model})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		dimensions []string
		orders     []bi.OrderSpec
		limit      int32
		sql        string
	}{
		{name: "两表明细聚合", sql: "SELECT COUNT(*),SUM(i.sort) FROM base_dict_item i LEFT JOIN base_dict d ON i.dict_id=d.id"},
		{name: "字典分组", dimensions: []string{"dict"}, sql: "SELECT d.code,COUNT(*),SUM(i.sort) FROM base_dict_item i LEFT JOIN base_dict d ON i.dict_id=d.id GROUP BY d.code"},
		{name: "双维排序Top10", dimensions: []string{"dict", "value"}, orders: []bi.OrderSpec{{Field: "dict", Direction: "asc"}, {Field: "value", Direction: "asc"}}, limit: 10, sql: "SELECT d.code,i.value,COUNT(*),SUM(i.sort) FROM base_dict_item i LEFT JOIN base_dict d ON i.dict_id=d.id GROUP BY d.code,i.value ORDER BY d.code ASC,i.value ASC LIMIT 10"},
		{name: "指标数值降序", dimensions: []string{"dict"}, orders: []bi.OrderSpec{{Field: "items", Direction: "desc"}, {Field: "dict", Direction: "asc"}}, limit: 5, sql: "SELECT d.code,COUNT(*),SUM(i.sort) FROM base_dict_item i LEFT JOIN base_dict d ON i.dict_id=d.id GROUP BY d.code ORDER BY COUNT(*) DESC,d.code ASC LIMIT 5"},
		{name: "源上限100截断", dimensions: []string{"dict", "value"}, orders: []bi.OrderSpec{{Field: "dict", Direction: "asc"}, {Field: "value", Direction: "asc"}}, limit: 1000, sql: "SELECT d.code,i.value,COUNT(*),SUM(i.sort) FROM base_dict_item i LEFT JOIN base_dict d ON i.dict_id=d.id GROUP BY d.code,i.value ORDER BY d.code ASC,i.value ASC LIMIT 100"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var result bi.QueryResult
			result, err = cards.biQueryCase.execute(ctx, bi.Scope{TenantID: 1, UserID: 10}, bi.QuerySpec{SourceCode: source.Code, Dimensions: test.dimensions, Measures: []bi.MeasureSpec{{Code: "items"}, {Code: "sort_total"}}, Orders: test.orders, Limit: test.limit}, nil)
			if err != nil {
				t.Fatal(err)
			}
			checkCloudSCPRows(t, database.db, result, test.sql, nil, len(test.orders) > 0)
			if test.limit >= 100 && !result.Truncated {
				t.Fatal("已知超过100组的真实字典结果未报告截断")
			}
			t.Logf("结果 %d 行，截断=%v，BI 耗时 %dms", len(result.Rows), result.Truncated, result.ElapsedMS)
		})
	}
}

// TestCloudSCPDoris 验证百万行真实事实表及复合键模型的安全拒绝，不伪造单列唯一性。
func TestCloudSCPDoris(t *testing.T) {
	cards, _, ctx, source, database := newCloudSCPTestCases(t, "cloud-scp/scp")
	sourceCase := cards.biQueryCase.biSourceCase
	metadata, err := sourceCase.ListBiSourceTables(ctx, &adminv1.ListBiSourceTablesRequest{Id: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	base, users := sourceTableKey(source.DatabaseName, "account_debit_rec"), sourceTableKey(source.DatabaseName, "base_user")
	model := &adminv1.BiSourceModel{BaseTable: base, Tables: []*adminv1.BiSourceModelTable{{Key: base, TenantColumn: "tenant_id"}}, Dimensions: []*adminv1.BiSourceDimension{{Code: "project", Name: "项目", Type: "number", Table: base, Column: "project_id", Filterable: true}, {Code: "status", Name: "状态", Type: "number", Table: base, Column: "status", Filterable: true}}, Measures: []*adminv1.BiSourceMeasure{{Code: "trades", Name: "交易笔数", Type: "number", Expression: "COUNT(*)"}, {Code: "amount", Name: "交易金额", Type: "number", Expression: "SUM(pay_amount) / 100", Unit: "元"}, {Code: "average", Name: "平均交易额", Type: "number", Expression: "AVG(pay_amount)", Unit: "分"}}}
	t.Run("复合键关联阻断", func(t *testing.T) {
		candidate := proto.Clone(model).(*adminv1.BiSourceModel)
		candidate.Tables = append(candidate.Tables, &adminv1.BiSourceModelTable{Key: users, TenantColumn: "tenant_id"})
		candidate.Relations = []*adminv1.BiSourceRelation{{LeftTable: base, LeftColumn: "user_id", RightTable: users, RightColumn: "user_id", JoinType: "left"}}
		_, err = configuredSourceModel(candidate, metadata.Tables)
		if errors.Code(err) != 400 || !strings.Contains(err.Error(), "唯一") {
			t.Fatalf("未安全阻断无法证明唯一性的关联: %v", err)
		}
		t.Log("限制已复现：Doris 未返回可用单列唯一键，BI 当前不能建立真实交易表的复合键多表模型")
	})
	if _, err = sourceCase.SaveBiSourceModel(ctx, &adminv1.SaveBiSourceModelRequest{Id: source.ID, ExpectedSourceVersion: 1, Model: model}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		dimensions []string
		filters    []bi.FilterSpec
		selectSQL  string
		tail       string
		args       []any
	}{
		{name: "百万行事实表租户聚合", selectSQL: "COUNT(*),SUM(pay_amount)*1.0/100,AVG(pay_amount)"},
		{name: "跨项目分组", dimensions: []string{"project"}, selectSQL: "project_id,COUNT(*),SUM(pay_amount)*1.0/100,AVG(pay_amount)", tail: " GROUP BY project_id"},
		{name: "项目IN状态区间组合", dimensions: []string{"project", "status"}, filters: []bi.FilterSpec{{Field: "project", Operator: "in", Values: []string{"2", "10001"}}, {Field: "status", Operator: "between", Values: []string{"3", "4"}}}, selectSQL: "project_id,status,COUNT(*),SUM(pay_amount)*1.0/100,AVG(pay_amount)", tail: " AND project_id IN (?,?) AND status BETWEEN ? AND ? GROUP BY project_id,status", args: []any{"2", "10001", "3", "4"}},
		{name: "空集精度和NULL", filters: []bi.FilterSpec{{Field: "project", Operator: "eq", Values: []string{"-999"}}}, selectSQL: "COUNT(*),SUM(pay_amount)*1.0/100,AVG(pay_amount)", tail: " AND project_id=?", args: []any{"-999"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var result bi.QueryResult
			result, err = cards.biQueryCase.execute(ctx, bi.Scope{TenantID: 1, UserID: 10}, bi.QuerySpec{SourceCode: source.Code, Dimensions: test.dimensions, Measures: []bi.MeasureSpec{{Code: "trades"}, {Code: "amount"}, {Code: "average"}}, Filters: test.filters}, nil)
			if err != nil {
				t.Fatal(err)
			}
			checkCloudSCPRows(t, database.db, result, "SELECT "+test.selectSQL+" FROM account_debit_rec WHERE tenant_id=?"+test.tail, append([]any{1}, test.args...), false)
			t.Logf("结果 %d 行，BI 耗时 %dms", len(result.Rows), result.ElapsedMS)
		})
	}
	t.Run("可信租户参数隔离", func(t *testing.T) {
		var configured dto.ConfiguredSourceSchema
		configured, err = configuredSourceModel(model, metadata.Tables)
		if err != nil {
			t.Fatal(err)
		}
		configured = reviseSourceModel(dto.ConfiguredSourceSchema{}, configured, 2)
		spec := bi.QuerySpec{SourceCode: source.Code, Measures: []bi.MeasureSpec{{Code: "trades"}}}
		var execution *dto.CardExecution
		execution, err = query.Compile("doris", 100, configured, spec)
		if err != nil {
			t.Fatal(err)
		}
		for _, tenantID := range []int64{1, 2, 1000, 999999} {
			var args []any
			args, err = bindExecutionParameters(execution, spec, bi.Scope{TenantID: tenantID}, 0, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			var result bi.QueryResult
			result, err = executeConfiguredDefinition(ctx, database, execution, args, 100)
			if err != nil {
				t.Fatal(err)
			}
			checkCloudSCPRows(t, database.db, result, "SELECT COUNT(*) FROM account_debit_rec WHERE tenant_id=?", []any{tenantID}, false)
			t.Logf("租户 %d 可见 %s 条", tenantID, result.Rows[0][0])
		}
	})
	t.Run("项目授权边界", func(t *testing.T) {
		_, err = sourceCase.Execute(ctx, bi.Scope{TenantID: 1, UserID: 10, ProjectID: 2}, bi.QuerySpec{SourceCode: source.Code, Measures: []bi.MeasureSpec{{Code: "trades"}}})
		if errors.Code(err) != 403 {
			t.Fatalf("配置源项目权限边界失效: %v", err)
		}
	})
	t.Run("五表复合键独立对照", func(t *testing.T) {
		var baseline, correct, tenantOnly, baselineAmount, correctAmount, tenantOnlyAmount int64
		err = database.db.QueryRowContext(ctx, "SELECT COUNT(*),SUM(pay_amount) FROM account_debit_rec WHERE tenant_id=1").Scan(&baseline, &baselineAmount) //nolint:forbidigo // 固定聚合仅用于证明真实数据的关联基数，不返回业务明细。
		if err != nil {
			t.Fatal(err)
		}
		joins := " FROM account_debit_rec f LEFT JOIN base_user u ON f.user_id=u.user_id AND f.tenant_id=u.tenant_id%s LEFT JOIN base_dept d ON u.dept_id=d.dept_id AND u.tenant_id=d.tenant_id%s LEFT JOIN dev_info v ON f.device_code=v.device_code AND f.tenant_id=v.tenant_id%s LEFT JOIN dev_group_catalog g ON v.group_id=g.group_id AND v.tenant_id=g.tenant_id%s WHERE f.tenant_id=1"
		err = database.db.QueryRowContext(ctx, "SELECT COUNT(*),SUM(f.pay_amount)"+fmt.Sprintf(joins, " AND f.project_id=u.project_id", " AND u.project_id=d.project_id", " AND f.project_id=v.project_id", " AND v.project_id=g.project_id")).Scan(&correct, &correctAmount) //nolint:forbidigo // 独立验证完整复合键关系是否保持事实笔数和金额。
		if err != nil {
			t.Fatal(err)
		}
		err = database.db.QueryRowContext(ctx, "SELECT COUNT(*),SUM(f.pay_amount)"+fmt.Sprintf(joins, "", "", "", "")).Scan(&tenantOnly, &tenantOnlyAmount) //nolint:forbidigo // 只读反例量化遗漏项目关联导致的重复累计。
		if err != nil {
			t.Fatal(err)
		}
		if baseline != correct || baselineAmount != correctAmount {
			t.Fatal("完整复合键的五表关联仍放大了事实统计")
		}
		t.Logf("事实基准=%d条/%d分；完整复合键=%d条/%d分；遗漏项目条件=%d条/%d分", baseline, baselineAmount, correct, correctAmount, tenantOnly, tenantOnlyAmount)
	})
}

// newCloudSCPTestCases 仅将真实连接用于读取，所有 BI 写入均使用可自动清理的 SQLite 仓储。
func newCloudSCPTestCases(t *testing.T, key string) (*BiCardCase, *BiDashboardCase, context.Context, *models.BiSource, *configuredDatabase) {
	t.Helper()
	path := os.Getenv("BI_CLOUD_SCP_CONNECTIONS")
	if path == "" {
		t.Skip("设置 BI_CLOUD_SCP_CONNECTIONS 指向本机私有连接 JSON 后运行真实数据源测试")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var connections map[string]struct {
		Driver   string `json:"driver"`
		Address  string `json:"address"`
		Database string `json:"database"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err = json.Unmarshal(payload, &connections); err != nil {
		t.Fatal("私有连接 JSON 无效")
	}
	entry, exists := connections[key]
	if !exists {
		t.Fatalf("私有连接 JSON 未配置 %s", key)
	}
	var host, portText string
	host, portText, err = net.SplitHostPort(entry.Address)
	if err != nil {
		t.Fatal("连接地址无效")
	}
	var port int
	port, err = strconv.Atoi(portText)
	if err != nil {
		t.Fatal("连接端口无效")
	}
	var dsn string
	_, dsn, err = sourceDSN(&models.BiSource{Driver: entry.Driver, Host: host, Port: int32(port), DatabaseName: entry.Database, Username: entry.Username}, entry.Password)
	if err != nil {
		t.Fatal(err)
	}
	previousOpen, registered := driver.Opens[entry.Driver]
	driver.Opens[entry.Driver] = mysql.Open
	t.Cleanup(func() {
		if registered {
			driver.Opens[entry.Driver] = previousOpen
		} else {
			delete(driver.Opens, entry.Driver)
		}
	})
	var client *gorm.Client
	var cleanup func()
	client, cleanup, err = gorm.NewGormClient(&configv1.Data_Database{Driver: entry.Driver, Source: dsn}, gorm.WithName(key))
	if err != nil {
		t.Fatal("打开真实数据源失败，请检查本机私有连接配置")
	}
	t.Cleanup(cleanup)
	cards, dashboards := newBiWorkflowTestCases(t)
	cards.GormClients[key] = client
	sourceCase := cards.biQueryCase.biSourceCase
	sourceCase.authorizer = modelTestAuthorizer{}
	identity := &data.UserTokenPayload{TenantId: 1, TenantCode: gorm.DefaultTenantCode, UserId: 10, DataScope: gorm.DataScopeAll}
	ctx, cancel := context.WithTimeout(engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims()), 90*time.Second)
	t.Cleanup(cancel)
	source := &models.BiSource{TenantID: 1, Code: "cloud_scp_test", Name: "cloud-scp 临时测试源", Driver: entry.Driver, DatabaseName: entry.Database, SourceName: key, SchemaJSON: "{}", Version: 1, MaxRows: 100, Status: 1}
	if err = sourceCase.sourceRepo.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	var database *configuredDatabase
	database, err = sourceCase.openSource(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)
	return cards, dashboards, ctx, source, database
}

// cloudSCPUserModel 描述真实用户到部门、角色、岗位和租户的五表关系。
func cloudSCPUserModel(schemaName, joinType string) *adminv1.BiSourceModel {
	users := sourceTableKey(schemaName, "base_user")
	model := &adminv1.BiSourceModel{BaseTable: users, Tables: []*adminv1.BiSourceModelTable{{Key: users, TenantColumn: "tenant_id"}}, Measures: []*adminv1.BiSourceMeasure{{Code: "users", Name: "用户数", Type: "number", Expression: "COUNT(*)", Unit: "人"}, {Code: "status_sum", Name: "状态合计", Type: "number", Expression: "SUM(status)"}, {Code: "status_avg", Name: "状态均值", Type: "number", Expression: "AVG(status)"}, {Code: "status_ratio", Name: "状态比值", Type: "number", Expression: "SUM(status)/NULLIF(COUNT(*),0)"}, {Code: "identified", Name: "编号计数", Type: "number", Expression: "COUNT(id)"}}}
	for _, relation := range []struct{ table, left, column, code, tenant string }{{"base_dept", "dept_id", "name", "dept", "tenant_id"}, {"base_role", "role_id", "name", "role", "tenant_id"}, {"base_post", "post_id", "name", "post", "tenant_id"}, {"base_tenant", "tenant_id", "code", "tenant", "id"}} {
		key := sourceTableKey(schemaName, relation.table)
		model.Tables = append(model.Tables, &adminv1.BiSourceModelTable{Key: key, TenantColumn: relation.tenant})
		model.Relations = append(model.Relations, &adminv1.BiSourceRelation{LeftTable: users, LeftColumn: relation.left, RightTable: key, RightColumn: "id", JoinType: joinType})
		model.Dimensions = append(model.Dimensions, &adminv1.BiSourceDimension{Code: relation.code, Name: relation.code, Type: "string", Table: key, Column: relation.column, Filterable: true})
	}
	model.Dimensions = append(model.Dimensions, &adminv1.BiSourceDimension{Code: "status", Name: "状态", Type: "number", Table: users, Column: "status", Filterable: true}, &adminv1.BiSourceDimension{Code: "created", Name: "创建日期", Type: "date", Table: users, Column: "created_at", Filterable: true})
	return model
}

// checkCloudSCPRows 用独立手写只读 SQL 比对 BI 结果，保留小数精度并避免在失败日志输出个人数据。
func checkCloudSCPRows(t *testing.T, db *sql.DB, actual bi.QueryResult, queryText string, args []any, ordered bool) {
	t.Helper()
	rows, err := db.QueryContext(t.Context(), queryText, args...) //nolint:forbidigo // 固定只读 SQL 是与 BI 编译器独立的结果对照。
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = rows.Close(); err != nil {
			t.Error(err)
		}
	}()
	want := make([][]string, 0)
	for rows.Next() {
		values := make([]any, len(actual.Columns))
		pointers := make([]any, len(values))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err = rows.Scan(pointers...); err != nil {
			t.Fatal(err)
		}
		row := make([]string, len(values))
		for index, value := range values {
			row[index] = formatConfiguredValue(value)
		}
		want = append(want, row)
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	got := make([][]string, len(actual.Rows))
	for index, row := range actual.Rows {
		got[index] = slices.Clone(row)
	}
	for _, result := range [][][]string{got, want} {
		for _, row := range result {
			for index, value := range row {
				if actual.Columns[index].Type == bi.FieldTypeNumber && value != "" {
					number, ok := new(big.Rat).SetString(value)
					if !ok {
						t.Fatal("结果包含非数值指标")
					}
					row[index] = number.RatString()
				}
			}
		}
		if !ordered {
			slices.SortFunc(result, slices.Compare[[]string])
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BI 与独立 SQL 结果不一致：BI %d 行，对照 %d 行", len(got), len(want))
	}
}
