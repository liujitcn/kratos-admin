package biz

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/securityconfig"
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

// TestGeneratedFilesRespectOutboundPolicy 验证转存登记成功，以及安全拒绝时的可见提示。
func TestGeneratedFilesRespectOutboundPolicy(t *testing.T) {
	previous := securityconfig.Current()
	t.Cleanup(func() { securityconfig.Set(previous) })
	for _, allowed := range []bool{false, true} {
		name := "拒绝私网"
		if allowed {
			name = "精确授权后转存"
		}
		t.Run(name, func(t *testing.T) {
			c, ctx, _, _ := newAiEventTestCase(t)
			c.OSS = local.NewOSS(t.TempDir())
			var requested atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requested.Store(true)
				_, _ = w.Write([]byte("%PDF-1.4 test"))
			}))
			t.Cleanup(server.Close)
			config := securityconfig.Default()
			if allowed {
				config.AIAllowedCIDRs = []string{"127.0.0.1/32"}
			}
			securityconfig.Set(config)
			signedURL := server.URL + "/files/test?sig=secret"
			reply := &ai.Response{Content: "下载：" + signedURL, Tools: []agent.ToolUsage{{Output: `{"download_url":"` + signedURL + `","filename":"验证.pdf"}`}}}
			c.persistGeneratedToolFiles(ctx, 7, reply)
			if requested.Load() != allowed || (len(reply.Attachments) == 1) != allowed {
				t.Fatalf("出站边界或附件转存结果错误: requested=%v attachments=%d", requested.Load(), len(reply.Attachments))
			}
			if !allowed {
				if !strings.Contains(reply.BlocksJSON, "generated_file_issue") {
					t.Fatal("文件转存失败仍静默显示生成成功")
				}
				return
			}
			page, err := c.aiFileCase.PageAiFile(ctx, &basev1.PageAiFileRequest{PageNum: 1, PageSize: 10})
			if err != nil || page.Total != 1 || page.Files[0].SourceType != aiFileSourceGenerated {
				t.Fatalf("生成成果未登记: %v, %v", page, err)
			}
			var stored []byte
			stored, err = c.OSS.GetFileByte(strings.TrimPrefix(reply.Attachments[0].Url, "/data/"))
			if err != nil || string(stored) != "%PDF-1.4 test" || strings.Contains(reply.Content, signedURL) {
				t.Fatalf("持久文件内容或正文链接错误: %q, %v", stored, err)
			}
		})
	}
}

// TestCollectGeneratedProducts 验证工具输出的结构化产物与裸链接两类解析形态。
func TestCollectGeneratedProducts(t *testing.T) {
	// Gotenberg 真实输出形态：外层 content parts，内层 JSON 携带无后缀签名链接与 filename。
	output := `[{"type":"text","text":"{\"download_url\": \"http://192.168.60.83:8002/files/2JVMRX0l9tznRFzHOCSqNPGx?exp=1794556954&sig=abc\", \"filename\": \"html.pdf\", \"bytes\": 546606, \"expires_at\": \"2026-11-13T08:02:34+00:00\"}"}]`
	products := collectGeneratedProducts([]agent.ToolUsage{{Name: "gotenberg", Output: output}})
	if len(products) != 1 {
		t.Fatalf("结构化产物数量 = %d, 期望 1", len(products))
	}
	if products[0].downloadURL != "http://192.168.60.83:8002/files/2JVMRX0l9tznRFzHOCSqNPGx?exp=1794556954&sig=abc" {
		t.Fatalf("解析链接 = %s", products[0].downloadURL)
	}
	if products[0].filename != "html.pdf" {
		t.Fatalf("解析文件名 = %s, 期望 html.pdf", products[0].filename)
	}
	if products[0].expiresAt != "2026-11-13T08:02:34+00:00" {
		t.Fatalf("未保留工具返回的有效期: %s", products[0].expiresAt)
	}
	reply := &ai.Response{Content: "汇总已完成"}
	reportGeneratedFileFailure(reply, products[0], "拉取", errors.New("链接到期"))
	var blocks []dto.AiGeneratedFileIssueBlock
	err := json.Unmarshal([]byte(reply.BlocksJSON), &blocks)
	if err != nil || len(blocks) != 1 || blocks[0].Type != "generated_file_issue" || blocks[0].Name != "html.pdf" || blocks[0].ExpiresAt != products[0].expiresAt || reply.Content != "汇总已完成" {
		t.Fatalf("产物失败应保留汇总并附带有效期: %+v, %v, %s", blocks, err, reply.Content)
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
