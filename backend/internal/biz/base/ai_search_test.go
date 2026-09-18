package biz

import (
	"strings"
	"testing"
)

// bingSamplePage 是 cn.bing.com 搜索 "deepseek" 的真实结果页片段，用于回归验证解析器。
const bingSamplePage = `<ol id="b_results"><li class="b_algo" data-id iid=SERP.5331><h2><a href="https://deepseek.com/en/index.html" h="ID=SERP,5331.1">DeepSeek | Into the Unknown</a></h2><div class="b_caption"><p>2026年9月10日&ensp;&#0183;&ensp;DeepSeek is an AI research company focused on building world-leading general artificial intelligence.</p></div></li><li class="b_algo" data-id iid=SERP.5332><h2><a href="https://cdn.deepseek.com/" h="ID=SERP,5332.1"><strong>DeepSeek</strong></a></h2><div class="b_caption"><p>2025年1月13日&ensp;&#0183;&ensp;Chat with <b>DeepSeek</b> AI your intelligent assistant for coding.</p></div></li></ol>`

// TestParseBingResults 验证必应结果块能解析出标题、链接与摘要。
func TestParseBingResults(t *testing.T) {
	items := parseBingResults(bingSamplePage, 5)
	if len(items) != 2 {
		t.Fatalf("解析结果数量 = %d, 期望 2", len(items))
	}
	first := items[0]
	if first.GetUrl() != "https://deepseek.com/en/index.html" {
		t.Errorf("第一条链接 = %q, 期望必应结果链接", first.GetUrl())
	}
	if first.GetTitle() != "DeepSeek | Into the Unknown" {
		t.Errorf("第一条标题 = %q, 期望原文标题", first.GetTitle())
	}
	if !strings.HasPrefix(first.GetSnippet(), "DeepSeek is an AI research company") {
		t.Errorf("第一条摘要 = %q, 期望去掉日期前缀后的摘要", first.GetSnippet())
	}
	if strings.Contains(first.GetSnippet(), "2026年") {
		t.Errorf("第一条摘要包含日期前缀: %q", first.GetSnippet())
	}
	second := items[1]
	if second.GetTitle() != "DeepSeek" {
		t.Errorf("第二条标题 = %q, 期望去除加粗标签后的标题", second.GetTitle())
	}
}

// TestParseBingResultsLimit 验证结果数量按 limit 截断。
func TestParseBingResultsLimit(t *testing.T) {
	items := parseBingResults(bingSamplePage, 1)
	if len(items) != 1 {
		t.Fatalf("解析结果数量 = %d, 期望 1", len(items))
	}
}
