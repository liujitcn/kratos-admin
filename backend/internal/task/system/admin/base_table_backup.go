package admin

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/backup"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coreBiz "github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/transport/cron"
)

const (
	// TableBackupTaskName 是数据库表备份任务的稳定调用目标。
	TableBackupTaskName = "system.admin.BaseTableBackup"
	backupFilePrefix    = "kratos-admin"
)

var _ cron.TaskExec = (*TableBackupTask)(nil)

// TableBackupTask 按备份配置对命名数据源执行加密全量备份并上传 OSS。
type TableBackupTask struct {
	baseCase   *coreBiz.BaseCase
	backupRepo *data.BaseTableBackupRepository
	recordRepo *data.BaseTableBackupRecordRepository
}

// NewTableBackupTask 创建数据库表备份任务。
func NewTableBackupTask(baseCase *coreBiz.BaseCase, backupRepo *data.BaseTableBackupRepository, recordRepo *data.BaseTableBackupRecordRepository) *TableBackupTask {
	return &TableBackupTask{baseCase: baseCase, backupRepo: backupRepo, recordRepo: recordRepo}
}

// Task 返回交由 base_job 调度的任务定义。
func (t *TableBackupTask) Task() cron.Task {
	return cron.Task{Name: TableBackupTaskName, Exec: t}
}

// Exec 执行所有启用的数据库备份配置。
func (t *TableBackupTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	query := t.backupRepo.Query(ctx).BaseTableBackup
	configs, err := t.backupRepo.List(ctx, repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)), repository.Order(query.ID.Asc()))
	if err != nil {
		return nil, fmt.Errorf("查询数据库备份配置失败: %w", err)
	}
	if len(configs) == 0 {
		return []string{"没有启用的数据库备份配置"}, nil
	}
	completed := 0
	for _, config := range configs {
		if err = t.backupOne(ctx, config); err != nil {
			return nil, err
		}
		completed++
	}
	return []string{fmt.Sprintf("数据库备份完成 %d 个数据源", completed)}, nil
}

// backupOne 执行单条数据库备份配置并记录 OSS 对象元数据。
func (t *TableBackupTask) backupOne(ctx context.Context, config *models.BaseTableBackup) error {
	if config.BackupType != int32(adminv1.BaseTableBackupType_BASE_TABLE_BACKUP_TYPE_FULL) {
		return fmt.Errorf("数据源 %s 暂不支持增量备份", config.SourceName)
	}
	databaseConfig, err := databaseConfigByName(t.baseCase, config.SourceName)
	if err != nil {
		return err
	}
	client := t.baseCase.GormClients[config.SourceName]
	if client == nil && config.SourceName == gorm.DefaultClientName {
		client = t.baseCase.GormClients[gorm.DefaultClientName]
	}
	if client == nil || client.DB == nil {
		return fmt.Errorf("数据源 %s 未初始化", config.SourceName)
	}
	dsn, err := mysql.ParseDSN(databaseConfig.GetSource())
	if err != nil {
		return fmt.Errorf("解析数据源 %s 失败: %w", config.SourceName, err)
	}
	var sqlDB *sql.DB
	sqlDB, err = client.DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据源 %s 的 SQL 连接失败: %w", config.SourceName, err)
	}
	var runtime backup.Config
	runtime, err = backup.FromRuntime()
	if err != nil {
		return err
	}
	if strings.TrimSpace(runtime.IntegrityKey) == "" || strings.TrimSpace(runtime.EncryptionKey) == "" {
		return fmt.Errorf("数据库备份完整性密钥和加密密钥必须配置")
	}
	record := &models.BaseTableBackupRecord{
		BackupID: config.ID, SourceName: config.SourceName, DatabaseName: dsn.DBName, BackupType: config.BackupType,
		ObjectKey: "", SizeBytes: 0, Sha256: "", Hmac: "",
		Status: int32(adminv1.BaseTableBackupRecordStatus_BASE_TABLE_BACKUP_RECORD_STATUS_RUNNING), Error: "",
		StartedAt: time.Now(), FinishedAt: time.Now(), VerifiedAt: time.Time{},
	}
	err = t.recordRepo.Create(ctx, record)
	if err != nil {
		return fmt.Errorf("创建数据库备份记录失败: %w", err)
	}
	temporaryDirectory, err := os.MkdirTemp("", "kratos-table-backup-")
	if err != nil {
		return t.failBackupRecord(ctx, record, fmt.Errorf("创建备份临时目录失败: %w", err))
	}
	defer os.RemoveAll(temporaryDirectory)
	sqlPath := path.Join(temporaryDirectory, backupFilePrefix+".sql")
	compressedPath := sqlPath + ".gz"
	encryptedPath := compressedPath + ".enc"
	err = dumpDatabase(ctx, sqlDB, dsn, sqlPath)
	if err == nil {
		err = gzipFile(sqlPath, compressedPath)
	}
	if err == nil {
		err = encryptBackup(ctx, runtime.EncryptionKey, compressedPath, encryptedPath)
	}
	if err != nil {
		return t.failBackupRecord(ctx, record, err)
	}
	dataValue, err := os.ReadFile(encryptedPath)
	if err != nil {
		return t.failBackupRecord(ctx, record, fmt.Errorf("读取加密备份失败: %w", err))
	}
	digest := sha256.Sum256(dataValue)
	macValue := hmac.New(sha256.New, []byte(runtime.IntegrityKey))
	_, _ = macValue.Write(dataValue)
	prefix := strings.Trim(config.OSSPrefix, "/")
	objectKey := fmt.Sprintf("%s/%s/%s/%s.sql.gz.enc", prefix, config.SourceName, dsn.DBName, time.Now().UTC().Format("20060102-150405"))
	if t.baseCase.OSS == nil {
		return t.failBackupRecord(ctx, record, fmt.Errorf("OSS 未配置"))
	}
	_, err = t.baseCase.OSS.UploadByByte(filepath.Base(encryptedPath), objectKey, dataValue)
	if err != nil {
		return t.failBackupRecord(ctx, record, fmt.Errorf("上传数据库备份失败: %w", err))
	}
	record.ObjectKey = objectKey
	record.SizeBytes = int64(len(dataValue))
	record.Sha256 = hex.EncodeToString(digest[:])
	record.Hmac = hex.EncodeToString(macValue.Sum(nil))
	if err = t.verifyUploadedBackup(objectKey, record.Sha256, record.Hmac, runtime.IntegrityKey); err != nil {
		_ = t.baseCase.OSS.DeleteFile(objectKey)
		return t.failBackupRecord(ctx, record, err)
	}
	record.Status = int32(adminv1.BaseTableBackupRecordStatus_BASE_TABLE_BACKUP_RECORD_STATUS_SUCCESS)
	record.FinishedAt = time.Now()
	record.VerifiedAt = record.FinishedAt
	if err = t.recordRepo.UpdateByID(ctx, record); err != nil {
		return fmt.Errorf("更新数据库备份记录失败: %w", err)
	}
	return t.rotateBackupRecords(ctx, config)
}

// verifyUploadedBackup 下载并校验已上传的备份对象，确认 OSS 持久化内容未损坏。
func (t *TableBackupTask) verifyUploadedBackup(objectKey, expectedSHA256, expectedHMAC, integrityKey string) error {
	dataValue, err := t.baseCase.OSS.GetFileByte(objectKey)
	if err != nil {
		return fmt.Errorf("回读数据库备份对象失败: %w", err)
	}
	digest := sha256.Sum256(dataValue)
	if !hmac.Equal([]byte(expectedSHA256), []byte(hex.EncodeToString(digest[:]))) {
		return fmt.Errorf("数据库备份对象 SHA-256 校验失败")
	}
	macValue := hmac.New(sha256.New, []byte(integrityKey))
	_, _ = macValue.Write(dataValue)
	if !hmac.Equal([]byte(expectedHMAC), []byte(hex.EncodeToString(macValue.Sum(nil)))) {
		return fmt.Errorf("数据库备份对象 HMAC 校验失败")
	}
	return nil
}

func (t *TableBackupTask) failBackupRecord(ctx context.Context, record *models.BaseTableBackupRecord, backupErr error) error {
	record.Status = int32(adminv1.BaseTableBackupRecordStatus_BASE_TABLE_BACKUP_RECORD_STATUS_FAILED)
	record.Error = backupErr.Error()
	record.FinishedAt = time.Now()
	if err := t.recordRepo.UpdateByID(ctx, record); err != nil {
		return fmt.Errorf("%w；更新失败备份记录失败: %v", backupErr, err)
	}
	return backupErr
}

func (t *TableBackupTask) rotateBackupRecords(ctx context.Context, config *models.BaseTableBackup) error {
	query := t.recordRepo.Query(ctx).BaseTableBackupRecord
	records, err := t.recordRepo.List(ctx,
		repository.Where(query.BackupID.Eq(config.ID)),
		repository.Where(query.Status.Eq(int32(adminv1.BaseTableBackupRecordStatus_BASE_TABLE_BACKUP_RECORD_STATUS_SUCCESS))),
		repository.Order(query.ID.Desc()),
	)
	if err != nil {
		return fmt.Errorf("查询历史备份记录失败: %w", err)
	}
	retention := int(config.RetentionCount)
	if retention < 1 {
		retention = 1
	}
	for _, record := range records[retention:] {
		if record.ObjectKey != "" && t.baseCase.OSS != nil {
			if err = t.baseCase.OSS.DeleteFile(record.ObjectKey); err != nil {
				return fmt.Errorf("删除过期 OSS 备份失败: %w", err)
			}
		}
		record.Status = int32(adminv1.BaseTableBackupRecordStatus_BASE_TABLE_BACKUP_RECORD_STATUS_DELETED)
		if err = t.recordRepo.UpdateByID(ctx, record); err != nil {
			return fmt.Errorf("更新过期备份记录失败: %w", err)
		}
	}
	return nil
}

func databaseConfigByName(baseCase *coreBiz.BaseCase, sourceName string) (*configv1.Data_Database, error) {
	dataConfig := baseCase.GetConfig().GetData()
	if dataConfig == nil {
		return nil, fmt.Errorf("数据源配置为空")
	}
	if sourceName == gorm.DefaultClientName && dataConfig.GetDatabase() != nil {
		return dataConfig.GetDatabase(), nil
	}
	databaseConfig := dataConfig.GetDatabases()[sourceName]
	if databaseConfig == nil {
		return nil, fmt.Errorf("数据源 %s 未配置", sourceName)
	}
	return databaseConfig, nil
}

func dumpDatabase(ctx context.Context, sqlDB *sql.DB, dsn *mysql.Config, output string) error {
	if !backup.CommandAvailable(backup.MysqldumpCommand) {
		return dumpDatabaseByGo(ctx, sqlDB, dsn, output)
	}
	var err error
	var passwordFile string
	passwordFile, err = backup.WriteMySQLDefaultsFile(dsn.Passwd)
	if err != nil {
		return err
	}
	defer os.Remove(passwordFile)
	args := []string{"--defaults-extra-file=" + passwordFile, "--single-transaction", "--routines", "--events"}
	if dsn.Net == "unix" && dsn.Addr != "" {
		args = append(args, "--socket="+dsn.Addr)
	} else if dsn.Addr != "" {
		host := dsn.Addr
		port := "3306"
		parsedHost, parsedPort, splitErr := net.SplitHostPort(dsn.Addr)
		if splitErr == nil {
			host = parsedHost
			port = parsedPort
		}
		args = append(args, "--host="+host, "--port="+port)
	}
	if dsn.User != "" {
		args = append(args, "--user="+dsn.User)
	}
	// 不使用 --databases，避免导出文件写入源数据库的 CREATE DATABASE/USE 语句，
	// 使恢复流程可以将内容导入调用方明确指定的目标数据库。
	args = append(args, dsn.DBName, "--result-file="+output)
	command := exec.CommandContext(ctx, backup.MysqldumpCommand, args...)
	if err = command.Run(); err != nil {
		return fmt.Errorf("执行 mysqldump 失败: %w", err)
	}
	return nil
}

// dumpDatabaseByGo 使用 Go 生成完整数据库 SQL 导出文件。
func dumpDatabaseByGo(ctx context.Context, sqlDB *sql.DB, dsn *mysql.Config, output string) error {
	file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建 Go 数据库备份文件失败: %w", err)
	}
	if err = backup.DumpMySQL(ctx, sqlDB, backup.MySQLDumpOptions{
		Database:        dsn.DBName,
		IncludeSchema:   true,
		IncludeData:     true,
		IncludeRoutines: true,
		IncludeEvents:   true,
		IncludeTriggers: true,
	}, file); err != nil {
		_ = file.Close()
		return fmt.Errorf("Go 导出数据库失败: %w", err)
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("同步 Go 数据库备份文件失败: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("关闭 Go 数据库备份文件失败: %w", err)
	}
	return nil
}

// gzipFile 将备份文件压缩到目标路径并确保文件落盘。
func gzipFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("打开待压缩备份失败: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建压缩备份失败: %w", err)
	}
	compressor := gzip.NewWriter(output)
	if _, err = io.Copy(compressor, input); err != nil {
		_ = compressor.Close()
		_ = output.Close()
		return fmt.Errorf("压缩备份失败: %w", err)
	}
	if err = compressor.Close(); err != nil {
		_ = output.Close()
		return fmt.Errorf("完成备份压缩失败: %w", err)
	}
	if err = output.Sync(); err != nil {
		_ = output.Close()
		return fmt.Errorf("同步压缩备份失败: %w", err)
	}
	if err = output.Close(); err != nil {
		return fmt.Errorf("关闭压缩备份失败: %w", err)
	}
	return nil
}

// encryptBackup 使用 OpenSSL AES-256-CBC 加密备份文件。
func encryptBackup(ctx context.Context, encryptionKey, source, target string) error {
	return backup.EncryptFile(ctx, encryptionKey, source, target)
}

func rotateBackups(directory string, retention int) error {
	if retention < 1 {
		return fmt.Errorf("备份保留数量必须大于零")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %w", err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), backupFilePrefix+"-") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sql") || strings.HasSuffix(entry.Name(), ".sql.gz") || strings.HasSuffix(entry.Name(), ".sql.enc") || strings.HasSuffix(entry.Name(), ".sql.gz.enc") {
			paths = append(paths, filepath.Join(directory, entry.Name()))
		}
	}
	sort.Strings(paths)
	for len(paths) > retention {
		if err = os.Remove(paths[0]); err != nil {
			return fmt.Errorf("删除过期备份失败: %w", err)
		}
		paths = paths[1:]
	}
	return nil
}

func writeBackupChecksum(path, integrityKey string) error {
	input, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer input.Close()
	digest := sha256.New()
	if _, err = io.Copy(digest, input); err != nil {
		return fmt.Errorf("计算备份校验值失败: %w", err)
	}
	if err = writeSyncedFile(path+".sha256", []byte(hex.EncodeToString(digest.Sum(nil))+"  "+filepath.Base(path)+"\n"), 0o600); err != nil {
		return err
	}
	if strings.TrimSpace(integrityKey) == "" {
		return fmt.Errorf("备份完整性密钥为空")
	}
	input, err = os.Open(path)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer input.Close()
	macValue := hmac.New(sha256.New, []byte(integrityKey))
	if _, err = io.Copy(macValue, input); err != nil {
		return fmt.Errorf("计算备份 HMAC 失败: %w", err)
	}
	if err = writeSyncedFile(path+".hmac", []byte(hex.EncodeToString(macValue.Sum(nil))+"  "+filepath.Base(path)+"\n"), 0o600); err != nil {
		return fmt.Errorf("写入备份 HMAC 文件失败: %w", err)
	}
	return nil
}

func writeSyncedFile(path string, content []byte, perm os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err = file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
