package biz

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/backup"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseTableBackupRestoreCase 管理数据库备份恢复记录。
type BaseTableBackupRestoreCase struct {
	*biz.BaseCase
	*data.BaseTableBackupRestoreRepository
	backupRecordRepo *data.BaseTableBackupRecordRepository
}

// NewBaseTableBackupRestoreCase 创建数据库备份恢复业务实例。
func NewBaseTableBackupRestoreCase(baseCase *biz.BaseCase, repo *data.BaseTableBackupRestoreRepository, backupRecordRepo *data.BaseTableBackupRecordRepository) *BaseTableBackupRestoreCase {
	return &BaseTableBackupRestoreCase{BaseCase: baseCase, BaseTableBackupRestoreRepository: repo, backupRecordRepo: backupRecordRepo}
}

// PageBaseTableBackupRestore 分页查询数据库备份恢复记录。
func (c *BaseTableBackupRestoreCase) PageBaseTableBackupRestore(ctx context.Context, req *adminv1.PageBaseTableBackupRestoreRequest) (*adminv1.PageBaseTableBackupRestoreResponse, error) {
	query := c.Query(ctx).BaseTableBackupRestore
	opts := []repository.QueryOption{repository.Order(query.ID.Desc())}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseTableBackupRestore, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseTableBackupRestore(item))
	}
	return &adminv1.PageBaseTableBackupRestoreResponse{BaseTableBackupRestores: items, Total: int32(total)}, nil
}

// GetBaseTableBackupRestore 查询数据库备份恢复记录。
func (c *BaseTableBackupRestoreCase) GetBaseTableBackupRestore(ctx context.Context, id int64) (*adminv1.BaseTableBackupRestore, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBaseTableBackupRestore(item), nil
}

// ExecuteBaseTableBackupRestore 人工执行数据库备份恢复并记录结果。
func (c *BaseTableBackupRestoreCase) ExecuteBaseTableBackupRestore(ctx context.Context, req *adminv1.BaseTableBackupRestore) error {
	if req.GetBackupRecordId() <= 0 || req.GetTargetDatabase() == "" || req.GetTargetSourceName() == "" {
		return errorsx.InvalidArgument("备份记录、目标数据源和目标数据库不能为空")
	}
	if req.GetRestoreMode() != adminv1.BaseTableBackupRestoreMode_BASE_TABLE_BACKUP_RESTORE_MODE_VERIFY_ONLY && req.GetRestoreMode() != adminv1.BaseTableBackupRestoreMode_BASE_TABLE_BACKUP_RESTORE_MODE_FULL {
		return errorsx.InvalidArgument("备份恢复模式无效")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var backupRecord *models.BaseTableBackupRecord
	backupRecord, err = c.backupRecordRepo.FindByID(ctx, req.GetBackupRecordId())
	if err != nil {
		return err
	}
	now := time.Now()
	entity := &models.BaseTableBackupRestore{BackupRecordID: req.GetBackupRecordId(), SourceName: backupRecord.SourceName, TargetSourceName: req.GetTargetSourceName(), TargetDatabase: req.GetTargetDatabase(), RestoreMode: int32(req.GetRestoreMode()), OperatorID: authInfo.UserId, Status: int32(adminv1.BaseTableBackupRestoreStatus_BASE_TABLE_BACKUP_RESTORE_STATUS_RUNNING), Error: "", StartedAt: now, FinishedAt: now}
	if err = c.Create(ctx, entity); err != nil {
		return err
	}
	_, err = restoreBackupRecord(ctx, c.BaseCase, backupRecord, req.GetTargetSourceName(), req.GetTargetDatabase(), req.GetRestoreMode())
	if err != nil {
		entity.Status = int32(adminv1.BaseTableBackupRestoreStatus_BASE_TABLE_BACKUP_RESTORE_STATUS_FAILED)
		entity.Error = err.Error()
		entity.FinishedAt = time.Now()
		if updateErr := c.UpdateByID(ctx, entity); updateErr != nil {
			return fmt.Errorf("%w；更新备份恢复失败记录失败: %v", err, updateErr)
		}
		return err
	}
	entity.Status = int32(adminv1.BaseTableBackupRestoreStatus_BASE_TABLE_BACKUP_RESTORE_STATUS_SUCCESS)
	entity.FinishedAt = time.Now()
	return c.UpdateByID(ctx, entity)
}

func toBaseTableBackupRestore(item *models.BaseTableBackupRestore) *adminv1.BaseTableBackupRestore {
	return &adminv1.BaseTableBackupRestore{Id: item.ID, BackupRecordId: item.BackupRecordID, SourceName: item.SourceName, TargetSourceName: item.TargetSourceName, TargetDatabase: item.TargetDatabase, RestoreMode: adminv1.BaseTableBackupRestoreMode(item.RestoreMode), OperatorId: item.OperatorID, Status: adminv1.BaseTableBackupRestoreStatus(item.Status), Error: item.Error, StartedAt: item.StartedAt.Format(time.RFC3339), FinishedAt: item.FinishedAt.Format(time.RFC3339)}
}

// restoreBackupRecord 校验并恢复指定数据库备份记录。
func restoreBackupRecord(ctx context.Context, baseCase *biz.BaseCase, backupRecord *models.BaseTableBackupRecord, targetSourceName, targetDatabase string, mode adminv1.BaseTableBackupRestoreMode) (int64, error) {
	if backupRecord.Status != int32(adminv1.BaseTableBackupRecordStatus_BASE_TABLE_BACKUP_RECORD_STATUS_SUCCESS) {
		return 0, fmt.Errorf("只能恢复已校验成功的数据库备份")
	}
	if strings.TrimSpace(backupRecord.Sha256) == "" || strings.TrimSpace(backupRecord.Hmac) == "" {
		return 0, fmt.Errorf("数据库备份缺少完整性校验元数据")
	}
	if baseCase.OSS == nil || backupRecord.ObjectKey == "" {
		return 0, fmt.Errorf("OSS 备份对象未配置")
	}
	dataValue, err := baseCase.OSS.GetFileByte(backupRecord.ObjectKey)
	if err != nil {
		return 0, fmt.Errorf("下载数据库备份失败: %w", err)
	}
	var runtime backup.Config
	runtime, err = backup.FromRuntime()
	if err != nil {
		return 0, err
	}
	if err = verifyBackupBytes(dataValue, backupRecord, runtime.IntegrityKey); err != nil {
		return 0, err
	}
	if mode == adminv1.BaseTableBackupRestoreMode_BASE_TABLE_BACKUP_RESTORE_MODE_VERIFY_ONLY {
		return 0, nil
	}
	var dsn *mysql.Config
	dsn, err = databaseConfigBySourceName(baseCase, targetSourceName)
	if err != nil {
		return 0, err
	}
	if dsn.DBName != targetDatabase {
		return 0, fmt.Errorf("目标数据库必须与目标数据源配置一致")
	}
	temporaryDirectory, err := os.MkdirTemp("", "kratos-table-restore-")
	if err != nil {
		return 0, fmt.Errorf("创建恢复临时目录失败: %w", err)
	}
	defer os.RemoveAll(temporaryDirectory)
	encryptedPath := filepath.Join(temporaryDirectory, "backup.sql.gz.enc")
	decryptedPath := filepath.Join(temporaryDirectory, "backup.sql.gz")
	sqlPath := filepath.Join(temporaryDirectory, "backup.sql")
	if err = os.WriteFile(encryptedPath, dataValue, 0o600); err != nil {
		return 0, fmt.Errorf("写入恢复临时文件失败: %w", err)
	}
	err = decryptBackupFile(ctx, runtime.EncryptionKey, encryptedPath, decryptedPath)
	if err != nil {
		return 0, err
	}
	if err = gunzipFile(decryptedPath, sqlPath); err != nil {
		return 0, err
	}
	return importSQLFile(ctx, dsn, targetDatabase, sqlPath)
}

func verifyBackupBytes(dataValue []byte, record *models.BaseTableBackupRecord, integrityKey string) error {
	if strings.TrimSpace(record.Sha256) == "" || strings.TrimSpace(record.Hmac) == "" {
		return fmt.Errorf("备份完整性校验元数据为空")
	}
	digest := sha256.Sum256(dataValue)
	if !hmac.Equal([]byte(record.Sha256), []byte(hex.EncodeToString(digest[:]))) {
		return fmt.Errorf("备份 SHA-256 校验失败")
	}
	if strings.TrimSpace(integrityKey) == "" {
		return fmt.Errorf("备份完整性密钥未配置")
	}
	macValue := hmac.New(sha256.New, []byte(integrityKey))
	_, _ = macValue.Write(dataValue)
	if !hmac.Equal([]byte(record.Hmac), []byte(hex.EncodeToString(macValue.Sum(nil)))) {
		return fmt.Errorf("备份 HMAC 校验失败")
	}
	return nil
}

func decryptBackupFile(ctx context.Context, encryptionKey, source, target string) error {
	return backup.DecryptFile(ctx, encryptionKey, source, target)
}

func gunzipFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("打开压缩恢复文件失败: %w", err)
	}
	defer input.Close()
	reader, err := gzip.NewReader(input)
	if err != nil {
		return fmt.Errorf("读取压缩恢复文件失败: %w", err)
	}
	defer reader.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建 SQL 恢复文件失败: %w", err)
	}
	if _, err = io.Copy(output, reader); err != nil {
		_ = output.Close()
		return fmt.Errorf("解压数据库备份失败: %w", err)
	}
	return output.Close()
}
