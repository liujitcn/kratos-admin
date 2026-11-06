package report

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var renderSlots = make(chan struct{}, 2)

// HTML 将 Markdown 转成禁止脚本和外部资源的报告，原始 HTML 保持禁用。
func HTML(title, content string) ([]byte, error) {
	var body bytes.Buffer
	err := goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough)).Convert([]byte(content), &body)
	if err != nil {
		return nil, fmt.Errorf("转换报告正文失败: %w", err)
	}
	var output bytes.Buffer
	err = reportTemplate.Execute(&output, struct {
		Title string
		Body  template.HTML
	}{Title: title, Body: template.HTML(body.String())})
	if err != nil {
		return nil, fmt.Errorf("生成报告 HTML 失败: %w", err)
	}
	return output.Bytes(), nil
}

// PDF 在受限并发和超时内调用本机 Chromium，以开启沙箱的普通用户导出矢量 PDF。
func PDF(ctx context.Context, html []byte) ([]byte, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	err := ctx.Err()
	if err != nil {
		return nil, err
	}
	select {
	case renderSlots <- struct{}{}:
		defer func() { <-renderSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	var directory string
	directory, err = os.MkdirTemp("", "kratos-report-")
	if err != nil {
		return nil, fmt.Errorf("创建报告临时目录失败: %w", err)
	}
	defer func() {
		var cleanupErr error
		cleanupErr = os.RemoveAll(directory)
		if cleanupErr != nil {
			log.Warn(fmt.Sprintf("清理报告临时文件失败: %v", cleanupErr))
		}
	}()
	filename := filepath.Join(directory, "report.html")
	err = os.WriteFile(filename, html, 0o600)
	if err != nil {
		return nil, fmt.Errorf("写入报告临时文件失败: %w", err)
	}
	options := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("no-sandbox", false))
	allocator, cancelAllocator := chromedp.NewExecAllocator(ctx, options...)
	defer cancelAllocator()
	browser, cancelBrowser := chromedp.NewContext(allocator)
	defer cancelBrowser()
	location := &url.URL{Scheme: "file", Path: filename}
	err = chromedp.Do(browser, chromedp.EmulateViewport(673, 1050), chromedp.Navigate(location.String()))
	if err != nil {
		return nil, fmt.Errorf("加载报告页面失败，请检查 Chromium 和沙箱配置: %w", err)
	}
	_, err = chromedp.Run(browser, chromedp.Poll[bool]("document.fonts.status === 'loaded'"))
	if err != nil {
		return nil, fmt.Errorf("加载报告字体失败: %w", err)
	}
	var pdf []byte
	pdf, err = chromedp.Run(browser, chromedp.PrintToPDF(
		chromedp.PDFPaper(chromedp.PaperA4),
		chromedp.PDFMargin(16.0/25.4),
		chromedp.PDFPrintBackground(),
		chromedp.PDFHeaderTemplate("<span></span>"),
		chromedp.PDFFooterTemplate("<div style='font-family:Noto Sans SC,sans-serif;font-size:10px;color:#475569;width:100%;text-align:center'><span class='pageNumber'></span> / <span class='totalPages'></span></div>"),
	))
	if err != nil {
		return nil, fmt.Errorf("导出报告 PDF 失败: %w", err)
	}
	return pdf, nil
}

var reportTemplate = template.Must(template.New("report").Parse(`<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'">
<title>{{.Title}}</title><style>
@page { size:A4; }
* { box-sizing:border-box; }
body { margin:0; color:#1f2937; background:#fff; font-family:"Noto Sans SC","PingFang SC","Microsoft YaHei",sans-serif; font-size:16px; line-height:1.75; overflow-wrap:anywhere; }
main { width:100%; max-width:1120px; margin:0 auto; padding:36px 32px; }
h1,h2,h3,h4 { color:#10243a; line-height:1.4; break-after:avoid; }
h1 { font-size:32px; margin:0 0 24px; }
h2 { font-size:23px; margin:28px 0 12px; }
h3 { font-size:19px; margin:22px 0 10px; }
p,ul,ol { margin:10px 0; }
ul,ol { padding-left:24px; }
table { width:100%; table-layout:fixed; border-collapse:collapse; font-size:14px; margin:18px 0; }
thead { display:table-header-group; }
th,td { padding:10px 12px; vertical-align:top; }
th { text-align:left; color:#173858; background:#edf3fb; }
td { border-bottom:1px solid #e2e8f0; }
tbody tr:nth-child(even) { background:#f8fafc; }
tr { break-inside:avoid; }
pre { padding:14px; background:#f7f9fc; white-space:pre-wrap; overflow-wrap:anywhere; }
code { font-size:.9em; }
blockquote { margin:16px 0; padding:12px 16px; background:#f0fdfa; }
a { color:#2563eb; }
hr { border:0; height:1px; background:#e2e8f0; margin:24px 0; }
@media screen and (max-width:600px) { main { padding:24px 20px; } h1 { font-size:28px; } table { font-size:12px; } th,td { padding:8px 6px; } }
@media print { main { max-width:none; margin:0; padding:0; } }
</style></head><body><main><h1>{{.Title}}</h1>{{.Body}}</main></body></html>`))
