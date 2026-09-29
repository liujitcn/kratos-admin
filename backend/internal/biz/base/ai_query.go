package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"

	"github.com/cloudwego/eino/schema"
	"github.com/go-kratos/kratos/v3/log"
)

const (
	// aiQueryDefaultMaxRows 问数结果默认行数上限，超出后截断。
	aiQueryDefaultMaxRows = 50
	// aiQueryMaxResultBytes 问数结果累计文本体积上限，避免工具输出撑爆模型上下文。
	aiQueryMaxResultBytes = 48 << 10
	// aiQuerySchemaBudget 注入提示词的表结构字典文本预算。
	aiQuerySchemaBudget = 16 << 10
	// aiQueryExecTimeout 只读 SQL 执行超时，防止 SLEEP 等慢查询拖垮聊天链路。
	aiQueryExecTimeout = 15 * time.Second
	// aiQueryNotRelevant 模型认为问题与可用数据无关时的标记输出。
	aiQueryNotRelevant = "NOT_RELEVANT"
	// aiQueryStatusSuccess 问数成功。
	aiQueryStatusSuccess = 1
	// aiQueryStatusFailed 问数失败。
	aiQueryStatusFailed = 2
	// aiQueryMaxSnapshotBytes 问数结果快照序列化体积上限，超出后从尾部丢弃数据行。
	aiQueryMaxSnapshotBytes = 60 << 10
)

// aiQueryResultSnapshot 问数结果快照序列化结构。
type aiQueryResultSnapshot struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

// aiQueryWhitelistTable 问数表白名单条目，表说明作为库表注释缺失时的兜底展示。
type aiQueryWhitelistTable struct {
	// Name 数据表名。
	Name string
	// Comment 表说明。
	Comment string
}

// aiQueryTableWhitelist 智能问数允许查询的数据表白名单。
// 只读 SQL 安全边界完全由该清单决定，新增表需要显式登记。
var aiQueryTableWhitelist = []aiQueryWhitelistTable{
	{Name: "ai_provider", Comment: "AI 模型供应商"},
	{Name: "ai_session", Comment: "AI 会话"},
	{Name: "ai_message", Comment: "AI 消息"},
	{Name: "base_user", Comment: "用户"},
	{Name: "base_role", Comment: "角色"},
	{Name: "base_dept", Comment: "部门"},
	{Name: "base_post", Comment: "岗位"},
	{Name: "base_menu", Comment: "菜单"},
	{Name: "base_dict", Comment: "字典"},
	{Name: "base_dict_item", Comment: "字典项"},
	{Name: "base_config", Comment: "系统参数配置"},
	{Name: "base_api", Comment: "接口"},
	{Name: "base_api_log", Comment: "接口调用日志"},
	{Name: "base_login_log", Comment: "登录日志"},
	{Name: "base_operation_log", Comment: "操作日志"},
	{Name: "base_data_access_log", Comment: "数据访问日志"},
	{Name: "base_permission_log", Comment: "权限校验日志"},
	{Name: "base_message", Comment: "站内消息"},
	{Name: "base_message_category", Comment: "消息分类"},
	{Name: "base_message_delivery", Comment: "消息投递记录"},
	{Name: "base_message_template", Comment: "消息模板"},
	{Name: "base_message_provider", Comment: "消息通道"},
	{Name: "base_job", Comment: "定时任务"},
	{Name: "base_job_log", Comment: "定时任务执行日志"},
	{Name: "base_file", Comment: "文件"},
	{Name: "base_area", Comment: "行政区划"},
	{Name: "base_language", Comment: "语言"},
	{Name: "base_tenant", Comment: "租户"},
	{Name: "base_tenant_project", Comment: "租户项目"},
	{Name: "base_third_account", Comment: "第三方账号"},
	{Name: "code_gen_table", Comment: "代码生成数据表"},
	{Name: "code_gen_proto", Comment: "代码生成 Proto"},
	{Name: "code_gen_column", Comment: "代码生成字段"},
	{Name: "oauth_client", Comment: "OAuth 客户端"},
}

var (
	// aiQueryFenceRe 匹配 markdown 代码围栏内容。
	aiQueryFenceRe = regexp.MustCompile("(?s)```(?:sql)?\\s*\\n?(.*?)```")
	// aiQueryCommentRe 匹配 SQL 注释。
	aiQueryCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/|--[^\n]*|#[^\n]*`)
	// aiQueryDangerWordRe 匹配只读查询禁止出现的关键字。
	aiQueryDangerWordRe = regexp.MustCompile(`(?i)\b(insert|update|delete|drop|alter|create|truncate|rename|grant|revoke|call|execute|prepare|deallocate|load|lock|unlock|into|show|use|explain|handler|do|begin|commit|rollback|set|analyze|optimize|repair|purge|flush|kill|shutdown|delimiter)\b`)
	// aiQueryDangerClauseRe 匹配 SELECT 后置危险子句。
	aiQueryDangerClauseRe = regexp.MustCompile(`(?i)\binto\s+(outfile|dumpfile)\b|\bfor\s+update\b|\block\s+in\s+share\s+mode\b`)
	// aiQueryStringLiteralRe 匹配单双引号字符串字面量，关键字校验前先移除避免误判数据内容。
	aiQueryStringLiteralRe = regexp.MustCompile(`'[^']*'|"[^"]*"`)
	// aiQueryTokenRe 表名扫描分词：关键字、括号、逗号、标识符、反引号标识符与字符串字面量。
	aiQueryTokenRe = regexp.MustCompile("(?i)\\b(from|join|where|group|order|having|limit|union|offset|window|qualify)\\b|\\(|\\)|,|`[^`]*`|'[^']*'|\"[^\"]*\"|[a-zA-Z_][a-zA-Z0-9_]*")
	// aiQueryCTENameRe 提取 WITH 公共表表达式名称。
	aiQueryCTENameRe = regexp.MustCompile("(?i)with\\s+`?([a-zA-Z_][a-zA-Z0-9_]*)\\s+as\\s*\\(")
	// aiQueryTenantEqRe 匹配 tenant_id 等值条件。
	aiQueryTenantEqRe = regexp.MustCompile("(?i)([a-zA-Z_][a-zA-Z0-9_]*\\.)?tenant_id\\s*=\\s*-?\\d+")
	// aiQueryTenantBadOpRe 匹配 tenant_id 非等值条件。
	aiQueryTenantBadOpRe = regexp.MustCompile("(?i)tenant_id\\s*(<>|!=|>=|<=|>|<)\\s*\\S+")
	// aiQueryLimitRe 匹配 LIMIT 子句。
	aiQueryLimitRe = regexp.MustCompile(`(?i)\blimit\s+\d+(\s*,\s*\d+)?`)
)

// aiQueryTableMeta 问数表白名单内的库表元数据快照。
type aiQueryTableMeta struct {
	// Name 数据表名。
	Name string
	// Comment 表说明。
	Comment string
	// TenantScoped 是否包含 tenant_id 租户列。
	TenantScoped bool
	// Columns 字段元数据。
	Columns []aiQueryColumnMeta
}

// aiQueryColumnMeta 问数表字段元数据。
type aiQueryColumnMeta struct {
	// Name 字段名。
	Name string
	// Type 字段类型。
	Type string
	// Comment 字段说明。
	Comment string
}

// AiQueryCase 处理 AI 助手智能问数业务。
type AiQueryCase struct {
	*biz.BaseCase
	registry    *model.Registry
	aiQueryRepo *data.AiQueryRepository
}

// NewAiQueryCase 创建 AI 助手智能问数业务实例。
func NewAiQueryCase(baseCase *biz.BaseCase, registry *model.Registry, aiQueryRepo *data.AiQueryRepository) *AiQueryCase {
	return &AiQueryCase{BaseCase: baseCase, registry: registry, aiQueryRepo: aiQueryRepo}
}

// OptionAiQueryTable 查询智能问数可用的数据表目录。
func (c *AiQueryCase) OptionAiQueryTable(ctx context.Context, req *basev1.OptionAiQueryTableRequest) (*basev1.OptionAiQueryTableResponse, error) {
	meta, err := c.loadSchema(ctx)
	if err != nil {
		return nil, err
	}
	tables := make([]*basev1.AiQueryTable, 0, len(aiQueryTableWhitelist))
	for _, item := range aiQueryTableWhitelist {
		table := meta[item.Name]
		if table == nil {
			continue
		}
		tables = append(tables, &basev1.AiQueryTable{Name: table.Name, Comment: table.Comment})
	}
	return &basev1.OptionAiQueryTableResponse{Tables: tables}, nil
}

// AskAiQuery 将自然语言问题转换为只读 SQL 并查询数据库。
func (c *AiQueryCase) AskAiQuery(ctx context.Context, req *basev1.AskAiQueryRequest) (*basev1.AskAiQueryResponse, error) {
	question := strings.TrimSpace(req.GetQuestion())
	if question == "" {
		return nil, errorsx.InvalidArgument("问数问题不能为空")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	meta, err := c.loadSchema(ctx)
	if err != nil {
		c.saveQueryLog(ctx, authInfo.TenantId, authInfo.UserId, question, "", aiQueryStatusFailed, 0, 0, err.Error(), "")
		return nil, err
	}
	if len(meta) == 0 {
		err = errorsx.Internal("智能问数未配置可用数据表")
		c.saveQueryLog(ctx, authInfo.TenantId, authInfo.UserId, question, "", aiQueryStatusFailed, 0, 0, err.Error(), "")
		return nil, err
	}

	generatedAt := time.Now()
	rawSQL, err := c.generateSQL(ctx, question, meta)
	if err != nil {
		c.saveQueryLog(ctx, authInfo.TenantId, authInfo.UserId, question, "", aiQueryStatusFailed, 0, 0, err.Error(), "")
		return nil, err
	}
	genMS := time.Since(generatedAt).Milliseconds()

	finalSQL, err := c.sanitizeSQL(rawSQL, meta, authInfo.TenantId)
	if err != nil {
		c.saveQueryLog(ctx, authInfo.TenantId, authInfo.UserId, question, extractSQL(rawSQL), aiQueryStatusFailed, 0, 0, err.Error(), "")
		return nil, errorsx.InvalidArgument(err.Error())
	}

	columns, rows, rowCount, truncated, err := c.executeReadOnly(ctx, finalSQL)
	elapsedMS := time.Since(generatedAt).Milliseconds() - genMS
	if err != nil {
		c.saveQueryLog(ctx, authInfo.TenantId, authInfo.UserId, question, finalSQL, aiQueryStatusFailed, 0, elapsedMS, "", "")
		return nil, err
	}
	c.saveQueryLog(ctx, authInfo.TenantId, authInfo.UserId, question, finalSQL, aiQueryStatusSuccess, int32(rowCount), elapsedMS, "", buildResultSnapshot(columns, rows))

	return &basev1.AskAiQueryResponse{
		Sql:       finalSQL,
		Columns:   columns,
		Rows:      rows,
		RowCount:  int32(rowCount),
		Truncated: truncated,
		ElapsedMs: elapsedMS,
	}, nil
}

// saveQueryLog 落库智能问数留痕，记录失败不阻断问数主流程。
func (c *AiQueryCase) saveQueryLog(ctx context.Context, tenantID, userID int64, question, sqlText string, status int32, rowCount int32, elapsedMS int64, errorMsg string, resultSnapshot string) {
	if c == nil || c.aiQueryRepo == nil {
		return
	}
	record := &models.AiQuery{
		TenantID:       tenantID,
		UserID:         userID,
		Question:       truncateRunes(question, 500),
		Sql:            sqlText,
		Status:         status,
		RowCount:       rowCount,
		ElapsedMs:      elapsedMS,
		ErrorMsg:       truncateRunes(errorMsg, 500),
		ResultSnapshot: resultSnapshot,
	}
	if err := c.aiQueryRepo.Create(ctx, record); err != nil {
		log.Error(fmt.Sprintf("智能问数留痕失败: %v", err))
	}
}

// buildResultSnapshot 将问数结果序列化为详情快照 JSON，超限时从尾部丢弃数据行。
func buildResultSnapshot(columns []string, rows []*basev1.AiQueryRow) string {
	if len(rows) == 0 {
		return ""
	}
	snapshot := aiQueryResultSnapshot{
		Columns: columns,
		Rows:    make([][]string, 0, len(rows)),
	}
	for _, row := range rows {
		snapshot.Rows = append(snapshot.Rows, row.GetValues())
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		log.Error(fmt.Sprintf("智能问数结果快照序列化失败: %v", err))
		return ""
	}
	for len(encoded) > aiQueryMaxSnapshotBytes && len(snapshot.Rows) > 0 {
		snapshot.Rows = snapshot.Rows[:len(snapshot.Rows)-1]
		if encoded, err = json.Marshal(snapshot); err != nil {
			log.Error(fmt.Sprintf("智能问数结果快照序列化失败: %v", err))
			return ""
		}
	}
	return string(encoded)
}

// truncateRunes 按字符数截断文本，避免超出数据库字段长度。
func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

// generateSQL 注入表结构字典调用大模型生成只读 SQL。
func (c *AiQueryCase) generateSQL(ctx context.Context, question string, meta map[string]*aiQueryTableMeta) (string, error) {
	client := c.registry.DefaultAssistantClient()
	if client == nil || !client.Enabled() {
		return "", errorsx.Internal("AI 助手没有已启用的模型")
	}
	messages := []*schema.AgenticMessage{
		message.SystemText(aiQuerySQLSystemPrompt(c.schemaDictionary(meta))),
		message.UserText("查询状态为启用的用户数量"),
		message.AIText("SELECT tenant_id, COUNT(*) AS total FROM base_user WHERE status = 1 AND tenant_id = -1"),
		message.UserText("按部门统计用户人数，并给出人数最多的部门"),
		message.AIText("SELECT d.name AS dept_name, COUNT(u.id) AS user_count FROM base_dept d LEFT JOIN base_user u ON u.dept_id = d.id AND u.tenant_id = -1 WHERE d.tenant_id = -1 GROUP BY d.id, d.name ORDER BY user_count DESC LIMIT 50"),
		message.UserText(question),
	}
	response, err := client.Generate(ctx, messages)
	if err != nil {
		return "", errorsx.Internal("智能问数 SQL 生成失败").WithCause(err)
	}
	return message.Text(response), nil
}

// sanitizeSQL 对模型输出执行确定性安全校验并重写租户条件与行数上限。
func (c *AiQueryCase) sanitizeSQL(rawSQL string, meta map[string]*aiQueryTableMeta, tenantID int64) (string, error) {
	sqlText := extractSQL(rawSQL)
	if strings.EqualFold(strings.TrimSpace(sqlText), aiQueryNotRelevant) {
		return "", fmt.Errorf("当前问题与可用数据表无关，请直接告知用户无法通过数据库查询回答该问题")
	}
	if err := validateReadOnlySQL(sqlText); err != nil {
		return "", err
	}
	tables := referencedTableNames(sqlText)
	if err := checkTableWhitelist(sqlText, tables, meta); err != nil {
		return "", err
	}
	tenantTables := tenantScopedTables(tables, meta)
	sqlText, err := enforceTenantScoping(sqlText, tenantTables, tenantID)
	if err != nil {
		return "", err
	}
	return applyLimitClamp(sqlText, aiQueryDefaultMaxRows), nil
}

// executeReadOnly 在只读事务中执行 SQL 并转换为字符串矩阵。
func (c *AiQueryCase) executeReadOnly(ctx context.Context, sqlText string) ([]string, []*basev1.AiQueryRow, int, bool, error) {
	database := c.GormClients[gorm.DefaultClientName]
	if database == nil || database.DB == nil {
		return nil, nil, 0, false, errorsx.Internal("默认数据源未初始化")
	}
	execCtx, cancel := context.WithTimeout(ctx, aiQueryExecTimeout)
	defer cancel()
	tx := database.DB.WithContext(execCtx).Begin(&sql.TxOptions{ReadOnly: true})
	if err := tx.Error; err != nil {
		return nil, nil, 0, false, errorsx.Internal("开启只读事务失败").WithCause(err)
	}
	defer func() { _ = tx.Rollback() }()

	// kit 守卫对 raw SQL 一律拒绝，这里的隔离已由 sanitizeSQL 显式保证
	// （只读校验、表白名单、强制租户条件、行数钳制），需显式豁免。
	rows, err := gorm.SkipDataIsolation(tx).Raw(sqlText).Rows()
	if err != nil {
		return nil, nil, 0, false, errorsx.Internal("智能问数查询执行失败").WithCause(err)
	}
	defer func() { _ = rows.Close() }()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, 0, false, errorsx.Internal("读取查询结果列失败").WithCause(err)
	}
	values := make([]*basev1.AiQueryRow, 0, aiQueryDefaultMaxRows)
	totalBytes := 0
	truncated := false
	for rows.Next() {
		if len(values) >= aiQueryDefaultMaxRows {
			truncated = true
			break
		}
		scanArgs := make([]any, len(columns))
		for index := range scanArgs {
			scanArgs[index] = new(any)
		}
		if err = rows.Scan(scanArgs...); err != nil {
			return nil, nil, 0, false, errorsx.Internal("读取查询结果行失败").WithCause(err)
		}
		row := &basev1.AiQueryRow{Values: make([]string, 0, len(columns))}
		for _, arg := range scanArgs {
			cell := formatSQLValue(*(arg.(*any)))
			totalBytes += len(cell)
			row.Values = append(row.Values, cell)
		}
		values = append(values, row)
		if totalBytes > aiQueryMaxResultBytes {
			truncated = true
			break
		}
	}
	if err = rows.Err(); err != nil {
		return nil, nil, 0, false, errorsx.Internal("遍历查询结果失败").WithCause(err)
	}
	return columns, values, len(values), truncated, nil
}

// loadSchema 从 information_schema 读取白名单表结构，缺失的表自动跳过。
func (c *AiQueryCase) loadSchema(ctx context.Context) (map[string]*aiQueryTableMeta, error) {
	database := c.GormClients[gorm.DefaultClientName]
	if database == nil || database.DB == nil {
		return nil, errorsx.Internal("默认数据源未初始化")
	}
	names := make([]string, 0, len(aiQueryTableWhitelist))
	for _, item := range aiQueryTableWhitelist {
		names = append(names, item.Name)
	}
	db := database.DB.WithContext(ctx)

	var tableInfos []struct {
		TableName    string `gorm:"column:table_name"`
		TableComment string `gorm:"column:table_comment"`
	}
	err := db.Table("information_schema.tables").
		Select("table_name, table_comment").
		Where("table_schema = DATABASE()").
		Where("table_name IN ?", names).
		Scan(&tableInfos).Error
	if err != nil {
		return nil, errorsx.Internal("读取数据表元数据失败").WithCause(err)
	}

	var columnInfos []struct {
		TableName     string `gorm:"column:table_name"`
		ColumnName    string `gorm:"column:column_name"`
		ColumnType    string `gorm:"column:column_type"`
		ColumnComment string `gorm:"column:column_comment"`
	}
	err = db.Table("information_schema.columns").
		Select("table_name, column_name, column_type, column_comment").
		Where("table_schema = DATABASE()").
		Where("table_name IN ?", names).
		Order("table_name, ordinal_position").
		Scan(&columnInfos).Error
	if err != nil {
		return nil, errorsx.Internal("读取数据表字段元数据失败").WithCause(err)
	}

	meta := make(map[string]*aiQueryTableMeta, len(tableInfos))
	for _, item := range tableInfos {
		meta[item.TableName] = &aiQueryTableMeta{
			Name:    item.TableName,
			Comment: firstNonEmpty(item.TableComment, aiQueryTableComment(item.TableName)),
			Columns: make([]aiQueryColumnMeta, 0, 24),
		}
	}
	for _, item := range columnInfos {
		table := meta[item.TableName]
		if table == nil {
			continue
		}
		table.Columns = append(table.Columns, aiQueryColumnMeta{
			Name:    item.ColumnName,
			Type:    item.ColumnType,
			Comment: item.ColumnComment,
		})
		if item.ColumnName == "tenant_id" {
			table.TenantScoped = true
		}
	}
	return meta, nil
}

// schemaDictionary 按白名单顺序渲染注入提示词的表结构字典。
func (c *AiQueryCase) schemaDictionary(meta map[string]*aiQueryTableMeta) string {
	var builder strings.Builder
	for _, item := range aiQueryTableWhitelist {
		table := meta[item.Name]
		if table == nil {
			continue
		}
		if builder.Len() > aiQuerySchemaBudget {
			break
		}
		marker := ""
		if table.TenantScoped {
			marker = "【租户表】"
		}
		builder.WriteString(fmt.Sprintf("- %s（%s）%s：", table.Name, table.Comment, marker))
		columns := make([]string, 0, len(table.Columns))
		for _, column := range table.Columns {
			columns = append(columns, firstNonEmpty(column.Comment, column.Name)+"("+column.Name+":"+column.Type+")")
		}
		builder.WriteString(strings.Join(columns, "，"))
		builder.WriteString("\n")
	}
	return builder.String()
}

// aiQuerySQLSystemPrompt 构造 SQL 生成的系统提示词。
func aiQuerySQLSystemPrompt(dictionary string) string {
	return `你是数据库查询专家。请根据「可用表结构」把用户问题转换成一条 MySQL 只读 SELECT 查询。

硬性规则：
1. 只能生成一条 SELECT 或 WITH ... SELECT 语句，禁止任何写操作、多语句、注释与存储过程调用。
2. 只能使用「可用表结构」中列出的表和列，禁止引用其他表或自行构造表名。
3. 查询标注为【租户表】的表时：SELECT 必须包含其 tenant_id 列，并在 WHERE 中写入 tenant_id = -1（系统会替换为当前租户），不要写其他任何租户过滤条件。
4. 结果集行数必须受限，结尾使用 LIMIT 100 以内的值。
5. 时间范围与业务口径按问题原意选择最合理的解读。
6. 只输出 SQL 本身：不要 markdown 围栏、不要注释、不要解释。
7. 如果问题与可用表结构完全无关，只输出：NOT_RELEVANT

可用表结构：
` + dictionary
}

// extractSQL 剥掉模型输出中的围栏与注释并规范化。
func extractSQL(raw string) string {
	value := strings.TrimSpace(raw)
	if match := aiQueryFenceRe.FindStringSubmatch(value); match != nil {
		value = strings.TrimSpace(match[1])
	}
	value = aiQueryCommentRe.ReplaceAllString(value, " ")
	value = strings.TrimSpace(value)
	value = strings.TrimRight(value, ";")
	return strings.TrimSpace(value)
}

// validateReadOnlySQL 校验语句整体形态与危险关键字。
func validateReadOnlySQL(sqlText string) error {
	upper := strings.ToUpper(strings.TrimSpace(sqlText))
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return fmt.Errorf("只允许生成 SELECT 或 WITH 查询，请重新生成")
	}
	if strings.Contains(sqlText, ";") {
		return fmt.Errorf("禁止多语句查询，请重新生成单条 SELECT")
	}
	// 关键字校验忽略字符串字面量，避免把数据内容误判为危险关键字。
	bare := aiQueryStringLiteralRe.ReplaceAllString(sqlText, "''")
	if aiQueryDangerWordRe.MatchString(bare) {
		return fmt.Errorf("SQL 包含禁止的关键字，请重新生成只读查询")
	}
	if aiQueryDangerClauseRe.MatchString(bare) {
		return fmt.Errorf("SQL 包含禁止的危险子句，请重新生成只读查询")
	}
	return nil
}

// referencedTableNames 提取 SQL 中 FROM/JOIN/逗号连接引用的表名（小写去重）。
//
// 按括号深度扫描，FROM 段内（FROM 到 WHERE/GROUP/ORDER 等子句为止）的顶层标识符
// 视为表名；字符串字面量跳过，避免把数据内容误判为表引用。
func referencedTableNames(sqlText string) []string {
	names := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	depth := 0
	inFrom := map[int]bool{0: false}
	expectTable := map[int]bool{0: false}
	for _, token := range aiQueryTokenRe.FindAllString(sqlText, -1) {
		switch strings.ToLower(token) {
		case "(":
			expectTable[depth] = false
			depth++
			if _, ok := inFrom[depth]; !ok {
				inFrom[depth] = false
				expectTable[depth] = false
			}
		case ")":
			expectTable[depth] = false
			inFrom[depth] = false
			depth--
			if depth < 0 {
				depth = 0
			}
		case ",":
			if inFrom[depth] {
				expectTable[depth] = true
			}
		case "from", "join":
			inFrom[depth] = true
			expectTable[depth] = true
		case "where", "group", "order", "having", "limit", "union", "offset", "window", "qualify":
			inFrom[depth] = false
			expectTable[depth] = false
		default:
			if !expectTable[depth] {
				continue
			}
			// 字符串字面量不是表名。
			if strings.HasPrefix(token, "'") || strings.HasPrefix(token, `"`) {
				continue
			}
			name := strings.ToLower(strings.Trim(token, "`"))
			if _, ok := seen[name]; ok {
				expectTable[depth] = false
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
			expectTable[depth] = false
		}
	}
	return names
}

// checkTableWhitelist 校验引用表全部位于白名单内，CTE 名称豁免。
func checkTableWhitelist(sqlText string, tables []string, meta map[string]*aiQueryTableMeta) error {
	cteNames := make(map[string]struct{}, 2)
	for _, match := range aiQueryCTENameRe.FindAllStringSubmatch(strings.ToLower(sqlText), -1) {
		cteNames[match[1]] = struct{}{}
	}
	var unknown []string
	for _, name := range tables {
		if _, ok := cteNames[name]; ok {
			continue
		}
		if meta[name] == nil {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("SQL 引用了白名单外的表 %s，请改用可用表结构中的表重新生成", strings.Join(unknown, "、"))
	}
	return nil
}

// tenantScopedTables 返回引用表中标记为租户表的表名。
func tenantScopedTables(tables []string, meta map[string]*aiQueryTableMeta) []string {
	result := make([]string, 0, len(tables))
	for _, name := range tables {
		if table := meta[name]; table != nil && table.TenantScoped {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

// enforceTenantScoping 校验并重写租户条件：引用租户表时必须有 tenant_id 等值条件且取值统一替换为当前租户。
func enforceTenantScoping(sqlText string, tenantTables []string, tenantID int64) (string, error) {
	if len(tenantTables) == 0 {
		return sqlText, nil
	}
	if aiQueryTenantBadOpRe.MatchString(sqlText) {
		return "", fmt.Errorf("tenant_id 条件必须使用等值形式，请重新生成")
	}
	if !aiQueryTenantEqRe.MatchString(sqlText) {
		return "", fmt.Errorf("查询租户表 %s 时缺少 tenant_id 等值条件（请写入 tenant_id = -1），请重新生成", strings.Join(tenantTables, "、"))
	}
	return aiQueryTenantEqRe.ReplaceAllStringFunc(sqlText, func(match string) string {
		index := strings.LastIndex(match, "=")
		return match[:index+1] + fmt.Sprintf(" %d", tenantID)
	}), nil
}

// applyLimitClamp 钳制结果行数：存在 LIMIT 时替换为上限值，缺失时在结尾追加。
func applyLimitClamp(sqlText string, maxRows int) string {
	if aiQueryLimitRe.MatchString(sqlText) {
		return aiQueryLimitRe.ReplaceAllString(sqlText, fmt.Sprintf("LIMIT %d", maxRows))
	}
	return sqlText + fmt.Sprintf(" LIMIT %d", maxRows)
}

// formatSQLValue 将数据库扫描值转换为稳定的字符串展示。
func formatSQLValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case []byte:
		return string(typed)
	case time.Time:
		return typed.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprint(typed)
	}
}

// aiQueryTableComment 返回白名单表说明。
func aiQueryTableComment(name string) string {
	for _, item := range aiQueryTableWhitelist {
		if item.Name == name {
			return item.Comment
		}
	}
	return ""
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
