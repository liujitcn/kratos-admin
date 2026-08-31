package admin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-kit/transport/cron"
)

const (
	// AuditRetentionTaskName 是审计日志归档任务的稳定调用目标。
	AuditRetentionTaskName = "system.admin.BaseAuditRetention"
	auditRetentionDays     = 180
	auditArchiveBatchSize  = 5000
)

var _ cron.TaskExec = (*AuditRetentionTask)(nil)

// AuditRetentionTask 归档并清理超过保留期的六类审计日志。
// 归档文件使用 JSONL 保存并生成 SHA-256/HMAC 校验文件，在线数据仅在归档完整性校验成功后删除。
type AuditRetentionTask struct {
	loginRepo            *data.BaseLoginLogRepository
	apiRepo              *data.BaseAPILogRepository
	operationRepo        *data.BaseOperationLogRepository
	dataAccessRepo       *data.BaseDataAccessLogRepository
	permissionRepo       *data.BasePermissionLogRepository
	policyEvaluationRepo *data.BasePolicyEvaluationLogRepository
}

// NewAuditRetentionTask 创建审计日志归档任务。
func NewAuditRetentionTask(
	loginRepo *data.BaseLoginLogRepository,
	apiRepo *data.BaseAPILogRepository,
	operationRepo *data.BaseOperationLogRepository,
	dataAccessRepo *data.BaseDataAccessLogRepository,
	permissionRepo *data.BasePermissionLogRepository,
	policyEvaluationRepo *data.BasePolicyEvaluationLogRepository,
) *AuditRetentionTask {
	return &AuditRetentionTask{
		loginRepo:            loginRepo,
		apiRepo:              apiRepo,
		operationRepo:        operationRepo,
		dataAccessRepo:       dataAccessRepo,
		permissionRepo:       permissionRepo,
		policyEvaluationRepo: policyEvaluationRepo,
	}
}

// Task 返回交由 base_job 调度的任务定义。
func (t *AuditRetentionTask) Task() cron.Task {
	return cron.Task{Name: AuditRetentionTaskName, Exec: t}
}

// Exec 将过期日志按表导出 JSONL 后从在线表删除。
func (t *AuditRetentionTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	days := auditRetentionDays
	var err error
	if value := os.Getenv("AUDIT_RETENTION_DAYS"); value != "" {
		var parsed int
		parsed, err = strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("AUDIT_RETENTION_DAYS 配置无效: %w", err)
		}
		days = parsed
	}
	if days <= 0 {
		return []string{"审计日志保留策略未启用"}, nil
	}
	if os.Getenv("AUDIT_ARCHIVE_INTEGRITY_KEY") == "" {
		return nil, fmt.Errorf("AUDIT_ARCHIVE_INTEGRITY_KEY 未配置，拒绝创建无完整性保护的审计归档")
	}
	archiveDir := os.Getenv("AUDIT_ARCHIVE_DIR")
	if archiveDir == "" {
		archiveDir = "./data/audit-archive"
	}
	if err = os.MkdirAll(archiveDir, 0o750); err != nil {
		return nil, fmt.Errorf("创建审计归档目录失败: %w", err)
	}
	before := time.Now().AddDate(0, 0, -days)
	archived := 0
	if err = t.archiveLogin(ctx, before, archiveDir, &archived); err != nil {
		return nil, err
	}
	if err = t.archiveAPI(ctx, before, archiveDir, &archived); err != nil {
		return nil, err
	}
	if err = t.archiveOperation(ctx, before, archiveDir, &archived); err != nil {
		return nil, err
	}
	if err = t.archiveDataAccess(ctx, before, archiveDir, &archived); err != nil {
		return nil, err
	}
	if err = t.archivePermission(ctx, before, archiveDir, &archived); err != nil {
		return nil, err
	}
	if err = t.archivePolicyEvaluation(ctx, before, archiveDir, &archived); err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("审计日志归档 %d 条，保留周期 %d 天", archived, days)}, nil
}

// archiveLogin 归档登录日志。
func (t *AuditRetentionTask) archiveLogin(ctx context.Context, before time.Time, dir string, total *int) error {
	query := t.loginRepo.Query(ctx).BaseLoginLog
	for {
		rows, err := t.loginRepo.List(ctx, repository.Where(query.OccurredAt.Lt(before)), repository.Order(query.ID.Asc()), repository.Limit(auditArchiveBatchSize))
		if err != nil {
			return fmt.Errorf("查询过期登录日志失败: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}
		if err = t.archiveRows(rowsToAny(rows), idsFromLogin(rows), func(ids []int64) error { return t.loginRepo.DeleteByIDs(ctx, ids) }, filepath.Join(dir, "base_login_log"), total); err != nil {
			return err
		}
	}
}

// archiveAPI 归档 API 访问日志。
func (t *AuditRetentionTask) archiveAPI(ctx context.Context, before time.Time, dir string, total *int) error {
	query := t.apiRepo.Query(ctx).BaseAPILog
	for {
		rows, err := t.apiRepo.List(ctx, repository.Where(query.OccurredAt.Lt(before)), repository.Order(query.ID.Asc()), repository.Limit(auditArchiveBatchSize))
		if err != nil {
			return fmt.Errorf("查询过期 API 日志失败: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}
		if err = t.archiveRows(rowsToAny(rows), idsFromAPI(rows), func(ids []int64) error { return t.apiRepo.DeleteByIDs(ctx, ids) }, filepath.Join(dir, "base_api_log"), total); err != nil {
			return err
		}
	}
}

// archiveOperation 归档业务操作日志。
func (t *AuditRetentionTask) archiveOperation(ctx context.Context, before time.Time, dir string, total *int) error {
	query := t.operationRepo.Query(ctx).BaseOperationLog
	for {
		rows, err := t.operationRepo.List(ctx, repository.Where(query.OccurredAt.Lt(before)), repository.Order(query.ID.Asc()), repository.Limit(auditArchiveBatchSize))
		if err != nil {
			return fmt.Errorf("查询过期操作日志失败: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}
		if err = t.archiveRows(rowsToAny(rows), idsFromOperation(rows), func(ids []int64) error { return t.operationRepo.DeleteByIDs(ctx, ids) }, filepath.Join(dir, "base_operation_log"), total); err != nil {
			return err
		}
	}
}

// archiveDataAccess 归档数据访问日志。
func (t *AuditRetentionTask) archiveDataAccess(ctx context.Context, before time.Time, dir string, total *int) error {
	query := t.dataAccessRepo.Query(ctx).BaseDataAccessLog
	for {
		rows, err := t.dataAccessRepo.List(ctx, repository.Where(query.OccurredAt.Lt(before)), repository.Order(query.ID.Asc()), repository.Limit(auditArchiveBatchSize))
		if err != nil {
			return fmt.Errorf("查询过期数据访问日志失败: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}
		if err = t.archiveRows(rowsToAny(rows), idsFromDataAccess(rows), func(ids []int64) error { return t.dataAccessRepo.DeleteByIDs(ctx, ids) }, filepath.Join(dir, "base_data_access_log"), total); err != nil {
			return err
		}
	}
}

// archivePermission 归档权限日志。
func (t *AuditRetentionTask) archivePermission(ctx context.Context, before time.Time, dir string, total *int) error {
	query := t.permissionRepo.Query(ctx).BasePermissionLog
	for {
		rows, err := t.permissionRepo.List(ctx, repository.Where(query.OccurredAt.Lt(before)), repository.Order(query.ID.Asc()), repository.Limit(auditArchiveBatchSize))
		if err != nil {
			return fmt.Errorf("查询过期权限日志失败: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}
		if err = t.archiveRows(rowsToAny(rows), idsFromPermission(rows), func(ids []int64) error { return t.permissionRepo.DeleteByIDs(ctx, ids) }, filepath.Join(dir, "base_permission_log"), total); err != nil {
			return err
		}
	}
}

// archivePolicyEvaluation 归档策略评估日志。
func (t *AuditRetentionTask) archivePolicyEvaluation(ctx context.Context, before time.Time, dir string, total *int) error {
	query := t.policyEvaluationRepo.Query(ctx).BasePolicyEvaluationLog
	for {
		rows, err := t.policyEvaluationRepo.List(ctx, repository.Where(query.OccurredAt.Lt(before)), repository.Order(query.ID.Asc()), repository.Limit(auditArchiveBatchSize))
		if err != nil {
			return fmt.Errorf("查询过期策略评估日志失败: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}
		if err = t.archiveRows(rowsToAny(rows), idsFromPolicyEvaluation(rows), func(ids []int64) error { return t.policyEvaluationRepo.DeleteByIDs(ctx, ids) }, filepath.Join(dir, "base_policy_evaluation_log"), total); err != nil {
			return err
		}
	}
}

// archiveRows 将一批日志原子写入确定性归档文件，并在完整性校验后删除在线记录。
func (t *AuditRetentionTask) archiveRows(rows []any, ids []int64, deleteRows func([]int64) error, pathPrefix string, total *int) error {
	if len(rows) == 0 {
		return nil
	}
	path := fmt.Sprintf("%s-%d-%d.jsonl", pathPrefix, ids[0], ids[len(ids)-1])
	err := writeArchiveRows(path, rows)
	if err != nil {
		return err
	}
	if err = writeArchiveChecksum(path); err != nil {
		return err
	}
	if err = deleteRows(ids); err != nil {
		return fmt.Errorf("删除已归档审计日志失败: %w", err)
	}
	*total += len(rows)
	return nil
}

// writeArchiveRows 通过同目录临时文件写入 JSONL，并以原子重命名发布完整批次。
func writeArchiveRows(path string, rows []any) error {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("创建审计归档临时文件失败: %w", err)
	}
	tempPath := file.Name()
	fileClosed := false
	defer func() {
		if !fileClosed {
			_ = file.Close()
		}
		_ = os.Remove(tempPath)
	}()
	if err = file.Chmod(0o640); err != nil {
		return fmt.Errorf("设置审计归档文件权限失败: %w", err)
	}
	encoder := json.NewEncoder(file)
	for _, row := range rows {
		if err = encoder.Encode(row); err != nil {
			return fmt.Errorf("写入审计归档文件失败: %w", err)
		}
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("同步审计归档文件失败: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("关闭审计归档文件失败: %w", err)
	}
	fileClosed = true
	if err = os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("发布审计归档文件失败: %w", err)
	}
	return nil
}

// writeArchiveChecksum 为审计归档 JSONL 生成 SHA-256 校验文件。
func writeArchiveChecksum(path string) error {
	input, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开审计归档文件失败: %w", err)
	}
	defer input.Close()
	digest := sha256.New()
	if _, err = io.Copy(digest, input); err != nil {
		return fmt.Errorf("计算审计归档校验值失败: %w", err)
	}
	checksumPath := path + ".sha256"
	checksum := hex.EncodeToString(digest.Sum(nil)) + "  " + filepath.Base(path) + "\n"
	if err = os.WriteFile(checksumPath, []byte(checksum), 0o600); err != nil {
		return fmt.Errorf("写入审计归档校验文件失败: %w", err)
	}
	input, err = os.Open(path)
	if err != nil {
		return fmt.Errorf("打开审计归档文件失败: %w", err)
	}
	defer input.Close()
	macValue := hmac.New(sha256.New, []byte(os.Getenv("AUDIT_ARCHIVE_INTEGRITY_KEY")))
	if _, err = io.Copy(macValue, input); err != nil {
		return fmt.Errorf("计算审计归档 HMAC 失败: %w", err)
	}
	hmacPath := path + ".hmac"
	hmacText := hex.EncodeToString(macValue.Sum(nil)) + "  " + filepath.Base(path) + "\n"
	if err = os.WriteFile(hmacPath, []byte(hmacText), 0o600); err != nil {
		return fmt.Errorf("写入审计归档 HMAC 文件失败: %w", err)
	}
	return nil
}

// rowsToAny 将任意日志模型切片转换为归档所需的接口切片。
func rowsToAny[T any](rows []*T) []any {
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	return result
}

// idsFromLogin 提取登录日志主键。
func idsFromLogin(rows []*models.BaseLoginLog) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// idsFromAPI 提取 API 日志主键。
func idsFromAPI(rows []*models.BaseAPILog) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// idsFromOperation 提取操作日志主键。
func idsFromOperation(rows []*models.BaseOperationLog) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// idsFromDataAccess 提取数据访问日志主键。
func idsFromDataAccess(rows []*models.BaseDataAccessLog) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// idsFromPermission 提取权限日志主键。
func idsFromPermission(rows []*models.BasePermissionLog) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// idsFromPolicyEvaluation 提取策略评估日志主键。
func idsFromPolicyEvaluation(rows []*models.BasePolicyEvaluationLog) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}
