package biz

import (
	"context"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	agent "github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/biz"
	databasegorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/oss/local"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var generatedTestDSNFlag = flag.String("generated-test-dsn", "", "生成文件持久化集成测试数据库 DSN")

// TestCollectGeneratedProducts 验证工具输出的结构化产物与裸链接两类解析形态。
func TestCollectGeneratedProducts(t *testing.T) {
	// Gotenberg 真实输出形态：外层 content parts，内层 JSON 携带无后缀签名链接与 filename。
	output := `[{"type":"text","text":"{\"download_url\": \"http://192.168.60.52:8002/files/2JVMRX0l9tznRFzHOCSqNPGx?exp=1794556954&sig=abc\", \"filename\": \"html.pdf\", \"bytes\": 546606, \"expires_at\": \"2026-11-13T08:02:34+00:00\"}"}]`
	products := collectGeneratedProducts([]agent.ToolUsage{{Name: "gotenberg", Output: output}})
	if len(products) != 1 {
		t.Fatalf("结构化产物数量 = %d, 期望 1", len(products))
	}
	if products[0].downloadURL != "http://192.168.60.52:8002/files/2JVMRX0l9tznRFzHOCSqNPGx?exp=1794556954&sig=abc" {
		t.Fatalf("解析链接 = %s", products[0].downloadURL)
	}
	if products[0].filename != "html.pdf" {
		t.Fatalf("解析文件名 = %s, 期望 html.pdf", products[0].filename)
	}

	// 裸链接兜底：带文档后缀的普通链接。
	products = collectGeneratedProducts([]agent.ToolUsage{{Name: "tool", Output: "已生成文件 https://example.com/data/report.docx 请查收。"}})
	if len(products) != 1 || products[0].filename != "report.docx" {
		t.Fatalf("裸链接解析结果 = %+v", products)
	}

	// 非文档链接必须被拦截，避免任意接口地址被当作文件拉取。
	products = collectGeneratedProducts([]agent.ToolUsage{{Name: "tool", Output: "详情见 http://example.com/api/detail?id=1"}})
	if len(products) != 0 {
		t.Fatalf("非文档链接不应产生产物 = %+v", products)
	}
}

// TestPersistGeneratedToolFilesIntegration 验证生成文件拉取、转存、登记与正文链接替换的完整链路。
func TestPersistGeneratedToolFilesIntegration(t *testing.T) {
	dsn := *generatedTestDSNFlag
	if dsn == "" {
		t.Skip("使用 -args -generated-test-dsn=... 指向测试库以执行数据库集成测试")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&models.AiFile{}); err != nil {
		t.Fatal(err)
	}
	store, err := data.NewData(map[string]*databasegorm.Client{databasegorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}

	ossClient := local.NewOSS(t.TempDir())
	baseCase := &biz.BaseCase{OSS: ossClient}
	aiFileCase := NewAiFileCase(baseCase, data.NewAiFileRepository(store))

	// 模拟 Gotenberg：返回无扩展名签名链接与 filename，正文 PDF 字节固定。
	pdfBytes := []byte("%PDF-1.4 测试内容")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer server.Close()
	signedURL := server.URL + "/files/2JVMRX0l9tznRFzHOCSqNPGx?exp=1794556954&sig=abc"
	output := `[{"type":"text","text":"{\"download_url\": \"` + signedURL + `\", \"filename\": \"安全策略.pdf\", \"bytes\": ` + "13" + `}"}]`

	reply := &ai.Response{
		Content: "已生成文件，下载链接：" + signedURL,
		Tools:   []agent.ToolUsage{{Name: "gotenberg_pdf", Status: "success", Output: output}},
	}
	c := &AiMessageCase{BaseCase: baseCase, aiFileCase: aiFileCase}
	c.persistGeneratedToolFiles(context.Background(), 7, reply)

	if len(reply.Attachments) != 1 {
		t.Fatalf("附件数量 = %d, 期望 1", len(reply.Attachments))
	}
	attachment := reply.Attachments[0]
	if attachment.Name != "安全策略.pdf" {
		t.Fatalf("附件文件名 = %s", attachment.Name)
	}
	if attachment.MimeType != "application/pdf" {
		t.Fatalf("附件 MIME = %s", attachment.MimeType)
	}
	if !strings.Contains(attachment.Url, "/data/ai/generated/") {
		t.Fatalf("附件链接未指向持久地址 = %s", attachment.Url)
	}
	if strings.Contains(reply.Content, signedURL) {
		t.Fatalf("正文仍包含临时签名链接")
	}
	if !strings.Contains(reply.Content, attachment.Url) {
		t.Fatalf("正文未替换为持久链接 = %s", reply.Content)
	}

	// 本地对象存储应真实落盘，且字节内容与源一致。
	objectPath := strings.TrimPrefix(attachment.Url, "/data/")
	stored, err := ossClient.GetFileByte(objectPath)
	if err != nil {
		t.Fatalf("对象未落盘: %v", err)
	}
	if string(stored) != string(pdfBytes) {
		t.Fatalf("落盘内容与源不一致")
	}

	// 数据库应登记为生成文件。
	row, err := store.Query(context.Background()).AiFile.WithContext(context.Background()).
		Where(store.Query(context.Background()).AiFile.TenantID.Eq(7)).First()
	if err != nil {
		t.Fatalf("查询生成文件失败: %v", err)
	}
	if row.SourceType != "generated" {
		t.Fatalf("来源类型 = %s, 期望 generated", row.SourceType)
	}
	if row.FileName != "安全策略.pdf" || row.Extension != "pdf" || row.Size != int64(len(pdfBytes)) {
		t.Fatalf("文件元数据异常 = %s/%s/%d", row.FileName, row.Extension, row.Size)
	}
	if !strings.HasPrefix(row.LinkURL, "ai/generated/") {
		t.Fatalf("对象路径 = %s, 期望 ai/generated/ 前缀", row.LinkURL)
	}
	if row.ContentHash == "" || row.AccessMode == 0 {
		t.Fatalf("内容哈希或访问方式未填充")
	}
}
