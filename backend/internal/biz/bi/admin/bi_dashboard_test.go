package admin

import (
	"fmt"
	"strings"
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	dataAccess "github.com/liujitcn/kratos-admin/backend/internal/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestBiDashboardPublicationAndWorkbench 验证发布快照、真实取数、版本冲突及工作台收藏与权限。
func TestBiDashboardPublicationAndWorkbench(t *testing.T) {
	cards, dashboards := newBiWorkflowTestCases(t)
	ctx := biCardTestContext(1, 10, 42)
	spec := &adminv1.BiQuerySpec{SourceCode: "test", Measures: []*adminv1.BiMeasureSpec{{Code: "count", Aggregation: "count"}}}
	asked, err := cards.biQueryCase.AskBiQuery(ctx, &adminv1.AskBiQueryRequest{Question: "统计用户数"})
	if err != nil {
		t.Fatal(err)
	}
	form := &adminv1.BiCardForm{Name: "原始卡片", Query: spec, ChartType: "metric", QueryRecordId: asked.Result.QueryRecordId}
	var savedCard *adminv1.BiCard
	savedCard, err = cards.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{Card: form})
	if err != nil {
		t.Fatal(err)
	}
	input := &adminv1.BiDashboard{Name: "运营总览", Code: "overview", SchemaJson: `{"schemaVersion":3,"name":"运营总览","cards":[]}`}
	var draft *adminv1.BiDashboard
	draft, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: input})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: draft.Id, Version: draft.DraftVersion}); err == nil {
		t.Fatal("允许发布空仪表盘")
	}
	draft.SchemaJson = fmt.Sprintf(`{"schemaVersion":3,"name":"运营总览","cards":[{"id":"card_1","card_id":%d,"x":0,"y":0,"w":6,"h":4,"name":"forgedlabel","query":{"source_code":"forged"}}]}`, savedCard.Id)
	draft, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: draft})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dashboards.GetBiDashboard(biCardTestContext(1, 20, 42), &adminv1.GetBiDashboardRequest{Id: draft.Id}); err == nil {
		t.Fatal("未发布草稿泄漏给其他账号")
	}
	var published *adminv1.BiDashboard
	published, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: draft.Id, Version: draft.DraftVersion})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(published.SchemaJson, "forged") {
		t.Fatal("发布接受了客户端伪造的卡片快照")
	}
	if !strings.Contains(published.SchemaJson, "原始卡片") {
		t.Fatal("发布没有固定卡片快照")
	}
	form.Name = "修改后的卡片"
	if _, err = cards.SaveBiCard(ctx, &adminv1.SaveBiCardRequest{Id: savedCard.Id, Card: form}); err != nil {
		t.Fatal(err)
	}
	var data *adminv1.BiDashboardDataResponse
	data, err = dashboards.GetBiDashboardData(biCardTestContext(1, 20, 42), &adminv1.GetBiDashboardDataRequest{Id: draft.Id})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data.SchemaJson, "原始卡片") || strings.Contains(data.SchemaJson, "修改后的卡片") || len(data.Cards) != 1 || data.Cards[0].Result.Rows[0].Values[0] != "3" {
		t.Fatal("卡片改动影响发布快照，或没有按访问者执行真实查询")
	}
	for _, other := range []struct{ tenant, project int64 }{{2, 42}, {1, 99}} {
		if _, err = dashboards.GetBiDashboardData(biCardTestContext(other.tenant, 20, other.project), &adminv1.GetBiDashboardDataRequest{Id: draft.Id}); err == nil {
			t.Fatal("允许跨租户或项目取数")
		}
	}
	var workbench *adminv1.BiWorkbench
	workbench, err = cards.GetBiWorkbench(ctx, &adminv1.GetBiWorkbenchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(workbench.DashboardIds) != 0 || len(workbench.Dashboards) != 0 {
		t.Fatal("首次进入工作台自动收藏了已发布仪表盘")
	}
	workbench, err = cards.SaveBiWorkbench(ctx, &adminv1.SaveBiWorkbenchRequest{DashboardIds: []int64{draft.Id}, FavoriteIds: []int64{savedCard.Id}})
	if err != nil {
		t.Fatal(err)
	}
	if len(workbench.Dashboards) != 1 || workbench.DashboardIds[0] != draft.Id {
		t.Fatal("工作台未保存仪表盘入口")
	}
	workbench, err = cards.SaveBiWorkbench(ctx, &adminv1.SaveBiWorkbenchRequest{FavoriteIds: []int64{savedCard.Id}})
	if err != nil {
		t.Fatal(err)
	}
	workbench, err = cards.GetBiWorkbench(ctx, &adminv1.GetBiWorkbenchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(workbench.DashboardIds) != 0 || len(workbench.Dashboards) != 0 || len(workbench.FavoriteIds) != 1 || workbench.FavoriteIds[0] != savedCard.Id {
		t.Fatal("取消仪表盘收藏后自动恢复入口，或丢失了独立卡片收藏")
	}
	if _, err = cards.SaveBiWorkbench(biCardTestContext(1, 20, 99), &adminv1.SaveBiWorkbenchRequest{DashboardIds: []int64{draft.Id}}); err == nil {
		t.Fatal("工作台接受越权仪表盘")
	}
	draft.Name = "新的运营总览"
	var latest *adminv1.BiDashboard
	latest, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: draft})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: draft}); err == nil {
		t.Fatal("旧版本覆盖最新草稿")
	}
	published, err = dashboards.PublishBiDashboard(ctx, &adminv1.PublishBiDashboardRequest{Id: latest.Id, Version: latest.DraftVersion})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(published.SchemaJson, "修改后的卡片") {
		t.Fatal("重新发布未读取最新卡片配置")
	}
}

// TestDashboardPublicationCompareAndSwap 验证迟到发布不能覆盖新草稿或重复改写发布快照。
func TestDashboardPublicationCompareAndSwap(t *testing.T) {
	_, dashboards := newBiWorkflowTestCases(t)
	ctx := biCardTestContext(1, 10, 42)
	draft, err := dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: &adminv1.BiDashboard{Name: "草稿", Code: "publication_cas", SchemaJson: `{"schemaVersion":3,"cards":[]}`}})
	if err != nil {
		t.Fatal(err)
	}
	draft.Name = "最新草稿"
	var latest *adminv1.BiDashboard
	latest, err = dashboards.SaveBiDashboard(ctx, &adminv1.SaveBiDashboardRequest{Dashboard: draft})
	if err != nil {
		t.Fatal(err)
	}
	stale := &models.BiDashboard{ID: draft.Id, TenantID: 1, Name: "迟到的发布", DraftVersion: draft.DraftVersion, PublishedVersion: draft.DraftVersion}
	var affected int64
	affected, err = dataAccess.UpdateBiDashboardState(ctx, dashboards.dashboardRepo, stale, draft.DraftVersion)
	if err != nil || affected != 0 {
		t.Fatalf("迟到发布覆盖了新草稿: affected=%d err=%v", affected, err)
	}
	var stored *adminv1.BiDashboard
	stored, err = dashboards.GetBiDashboard(ctx, &adminv1.GetBiDashboardRequest{Id: draft.Id})
	if err != nil || stored.Name != latest.Name || stored.DraftVersion != latest.DraftVersion || stored.PublishedVersion != 0 {
		t.Fatalf("最新草稿被修改: %v %v", stored, err)
	}
	var version *models.BiDashboardVersion
	version, err = dashboards.findVersion(ctx, latest.Id, latest.DraftVersion)
	if err != nil {
		t.Fatal(err)
	}
	version.Status = biDashboardVersionPublished
	version.PublishedAt = time.Now()
	affected, err = dataAccess.PublishBiDashboardVersion(ctx, dashboards.versionRepo, version, biDashboardVersionDraft)
	if err != nil || affected != 1 {
		t.Fatalf("首次固定快照失败: affected=%d err=%v", affected, err)
	}
	version.SchemaJSON = `{"forged":true}`
	affected, err = dataAccess.PublishBiDashboardVersion(ctx, dashboards.versionRepo, version, biDashboardVersionDraft)
	if err != nil || affected != 0 {
		t.Fatalf("重复发布改写了快照: affected=%d err=%v", affected, err)
	}
	version, err = dashboards.findVersion(ctx, latest.Id, latest.DraftVersion)
	if err != nil || strings.Contains(version.SchemaJSON, "forged") {
		t.Fatalf("已发布快照被修改: %v %v", version, err)
	}
}

// TestDashboardRejectsLegacyAndInvalidLayout 验证旧数据集和越界布局不能进入新协议。
func TestDashboardRejectsLegacyAndInvalidLayout(t *testing.T) {
	var err error
	for _, value := range []string{`{"widgets":[]}`, `{"schemaVersion":3,"cards":[{"id":"a","card_id":1,"x":10,"w":6,"h":4}]}`, `{"schemaVersion":3,"cards":[{"id":"a","card_id":1,"w":6,"h":4},{"id":"b","card_id":1,"w":6,"h":4}]}`} {
		if _, err = parseDashboardSchema(value); err == nil {
			t.Fatalf("接受非法或历史布局: %s", value)
		}
	}
}
