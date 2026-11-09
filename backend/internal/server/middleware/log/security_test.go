package logmiddleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/query"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestAuditCoverage 验证改密、敏感字段查看和跨模块变更不会被包前缀或日志服务名称排除。
func TestAuditCoverage(t *testing.T) {
	for _, operation := range []string{"/base.v1.AccountService/UpdateUserPassword", "/base.v1.SecretCryptoService/RevealSecretField", "/manager.admin.v1.ProjectService/UpdateProject", "/bi.admin.v1.BiCardService/SaveBiCard", "/rag.admin.v1.AiKnowledgeDocService/CreateAiKnowledgeDoc"} {
		if !isOperation(request{UserID: 9, Operation: operation, Method: "POST"}) {
			t.Fatalf("变更事件未审计: %s", operation)
		}
	}
	for _, operation := range []string{"/base.v1.SecretCryptoService/RevealSecretField", "/system.admin.v1.RuntimeLogService/DownloadRuntimeLogFile"} {
		if !isDataAccess(request{UserID: 9, Operation: operation, Method: "RPC"}) {
			t.Fatalf("敏感访问未审计: %s", operation)
		}
	}
	for _, field := range []string{"id_code", "source", "context_json", "output_json", "kubeconfig"} {
		if !isSensitiveLogField(field) {
			t.Fatalf("敏感快照字段未保护: %s", field)
		}
	}
}

// TestAgreementVersionValidation 验证只有当前协议摘要能成为有效同意，旧摘要不能污染审计。
func TestAgreementVersionValidation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "agreement.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	statement := &gorm.Statement{DB: db}
	err = statement.Parse(new(models.BaseConfig))
	if err != nil {
		t.Fatal(err)
	}
	statement.Schema.FieldsByName["ID"].DataType = schema.Int
	err = db.AutoMigrate(new(models.BaseConfig))
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range []*models.BaseConfig{{ID: 1201, Site: 3, Key: "privacyProtocol", Value: "privacy", Status: 1}, {ID: 1202, Site: 3, Key: "serviceProtocol", Value: "service", Status: 1}} {
		err = db.Create(record).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	service := &Middleware{logQuery: query.Use(db)}
	digest := sha256.Sum256([]byte("privacy\x00service"))
	err = service.validateAgreement(t.Context(), request{AgreementSite: "2", AgreementVersion: hex.EncodeToString(digest[:])})
	if err != nil {
		t.Fatal(err)
	}
	if service.validateAgreement(t.Context(), request{AgreementSite: "2", AgreementVersion: strings.Repeat("a", 64)}) == nil {
		t.Fatal("过期协议摘要被接受")
	}
}

// TestConsentStoredAsOperation 验证协议确认使用现有操作日志通道，记录主体、时间、来源及摘要而非创建无法消费的新事件。
func TestConsentStoredAsOperation(t *testing.T) {
	middleware := &Middleware{}
	event, accepted, err := middleware.buildEvent(adminTask{EventID: "consent-event", Kind: "consent", Request: request{UserID: 9, TenantID: 2, UserName: "operator", ClientIP: "192.0.2.1", AgreementVersion: strings.Repeat("a", 64), AgreementSite: "2", OccurredAt: time.Now()}})
	if err != nil || !accepted || event.Kind != "operation" {
		t.Fatalf("协议事件无法进入现有消费通道: %s, %v", event.Kind, err)
	}
	var record models.BaseOperationLog
	err = json.Unmarshal(event.Payload, &record)
	if err != nil || record.UserID != 9 || record.ResourceType != "user_consent" || !strings.Contains(record.AfterData, "192.0.2.1") {
		t.Fatalf("协议确认上下文缺失: %v", err)
	}
}
