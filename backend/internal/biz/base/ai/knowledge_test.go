package ai

import (
	"reflect"
	"testing"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/rag/data"
	databasegorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSearchChunksUsesReadySelectedKnowledge 验证引用只取指定库中存续且完成处理的文档，并遵循召回数量与排序。
func TestSearchChunksUsesReadySelectedKnowledge(t *testing.T) {
	ctx := t.Context()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(data.Models()...); err != nil {
		t.Fatal(err)
	}
	client := &databasegorm.Client{DB: db}
	databases := map[string]*databasegorm.Client{databasegorm.DefaultClientName: client, data.RagGormClientName: client}
	store, err := data.NewData(databases)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewKnowledgeEngine(databases, store, nil, nil)
	docs := []*Doc{
		{ID: 1, KnowledgeBaseID: 10, Name: "退款手册", Status: DocStatusReady},
		{ID: 2, KnowledgeBaseID: 10, Name: "处理失败文档", Status: DocStatusFailed},
		{ID: 3, KnowledgeBaseID: 11, Name: "未选择的文档", Status: DocStatusReady},
		{ID: 4, KnowledgeBaseID: 10, Name: "已删除的文档", Status: DocStatusReady},
	}
	if err = db.WithContext(ctx).Create(docs).Error; err != nil {
		t.Fatal(err)
	}
	docQuery := engine.docRepo.Query(ctx).AiKnowledgeDoc
	if err = engine.docRepo.Delete(ctx, repository.Where(docQuery.ID.Eq(4))); err != nil {
		t.Fatal(err)
	}
	chunks := []*Chunk{
		{ID: 1, DocID: 1, Content: "备用说明", Embedding: `[0,1]`},
		{ID: 2, DocID: 1, Content: "渠道退款规则", Embedding: `[0.3,0.7]`},
		{ID: 3, DocID: 1, Content: "常规退款规则", Embedding: `[0.8,0.2]`},
		{ID: 4, DocID: 2, Content: "处理失败", Embedding: `[1,0]`},
		{ID: 5, DocID: 3, Content: "未选择", Embedding: `[1,0]`},
		{ID: 6, DocID: 4, Content: "已删除", Embedding: `[1,0]`},
	}
	if err = db.WithContext(ctx).Create(chunks).Error; err != nil {
		t.Fatal(err)
	}
	hits, err := engine.SearchChunks(ctx, 10, []float32{1, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, 0, len(hits))
	for _, hit := range hits {
		ids = append(ids, hit.ChunkID)
		if hit.DocName != "退款手册" {
			t.Fatalf("错误记录进入可引用资料: %+v", hit)
		}
	}
	if !reflect.DeepEqual(ids, []int64{3, 2}) {
		t.Fatalf("召回数量或顺序不正确: %v", ids)
	}
}
