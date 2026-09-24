package biz

import (
	"context"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
)

const (
	aiSearchDefaultLimit = 5
	aiSearchMaxLimit     = 10
	aiSearchTimeout      = 10 * time.Second
	aiSearchMaxBodyBytes = 2 << 20
	bingSearchBaseURL    = "https://cn.bing.com/search"
	bingSearchUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"
)

var (
	bingResultBlockRe       = regexp.MustCompile(`(?s)<li class="b_algo".*?</li>`)
	bingResultLinkRe        = regexp.MustCompile(`(?s)<h2[^>]*><a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	bingResultSnippetRe     = regexp.MustCompile(`(?s)<p[^>]*>(.*?)</p>`)
	bingSnippetDatePrefixRe = regexp.MustCompile(`^(?:\d{4}年\d{1,2}月\d{1,2}日|\d+\s*(?:天|小时|分钟|秒)前)\s*·\s*`)
	bingHtmlTagRe           = regexp.MustCompile(`<[^>]+>`)
)

// AiSearchCase 处理 AI 助手联网搜索业务。
type AiSearchCase struct {
	*biz.BaseCase
	client *http.Client
}

// NewAiSearchCase 创建 AI 助手联网搜索业务实例。
func NewAiSearchCase(baseCase *biz.BaseCase) *AiSearchCase {
	return &AiSearchCase{
		BaseCase: baseCase,
		client:   &http.Client{Timeout: aiSearchTimeout},
	}
}

// SearchAiWeb 调用外部搜索引擎并返回结构化结果。
func (c *AiSearchCase) SearchAiWeb(ctx context.Context, req *basev1.SearchAiWebRequest) (*basev1.SearchAiWebResponse, error) {
	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return nil, errorsx.InvalidArgument("搜索关键词不能为空")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = aiSearchDefaultLimit
	}
	if limit > aiSearchMaxLimit {
		limit = aiSearchMaxLimit
	}
	page, err := c.fetchBingPage(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	return &basev1.SearchAiWebResponse{Items: parseBingResults(page, limit)}, nil
}

// fetchBingPage 请求必应网页搜索结果页。
func (c *AiSearchCase) fetchBingPage(ctx context.Context, query string, limit int) (string, error) {
	target, err := url.Parse(bingSearchBaseURL)
	if err != nil {
		return "", errorsx.Internal("联网搜索地址配置异常").WithCause(err)
	}
	values := target.Query()
	values.Set("q", query)
	values.Set("count", strconv.Itoa(limit))
	values.Set("setlang", "zh-hans")
	target.RawQuery = values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return "", errorsx.Internal("构造联网搜索请求失败").WithCause(err)
	}
	request.Header.Set("User-Agent", bingSearchUserAgent)
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	response, err := c.client.Do(request)
	if err != nil {
		return "", errorsx.Internal("联网搜索请求失败").WithCause(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", errorsx.WithMessageKey(
			errorsx.Internal("联网搜索服务返回异常状态"),
			"base.ai.search.http_status",
			map[string]string{"Status": strconv.Itoa(response.StatusCode)},
		)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, aiSearchMaxBodyBytes))
	if err != nil {
		return "", errorsx.Internal("读取联网搜索响应失败").WithCause(err)
	}
	return string(body), nil
}

// parseBingResults 解析必应结果页中的标题、链接与摘要。
func parseBingResults(page string, limit int) []*basev1.SearchAiWebItem {
	blocks := bingResultBlockRe.FindAllString(page, limit)
	items := make([]*basev1.SearchAiWebItem, 0, len(blocks))
	for _, block := range blocks {
		match := bingResultLinkRe.FindStringSubmatch(block)
		if match == nil {
			continue
		}
		item := &basev1.SearchAiWebItem{
			Url:   html.UnescapeString(match[1]),
			Title: cleanBingText(match[2]),
		}
		if snippet := bingResultSnippetRe.FindStringSubmatch(block); snippet != nil {
			item.Snippet = cleanBingSnippet(snippet[1])
		}
		if item.Title == "" && item.Snippet == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

// cleanBingSnippet 清理摘要文本并去掉必应自带的日期前缀。
func cleanBingSnippet(value string) string {
	text := cleanBingText(value)
	return bingSnippetDatePrefixRe.ReplaceAllString(text, "")
}

// cleanBingText 去除 HTML 标签、实体解码并压缩空白。
func cleanBingText(value string) string {
	text := html.UnescapeString(bingHtmlTagRe.ReplaceAllString(value, ""))
	return strings.Join(strings.Fields(text), " ")
}
