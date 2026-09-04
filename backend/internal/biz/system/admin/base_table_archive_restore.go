package biz

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/backup"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

type restoreIDRange struct {
	StartID int64 `json:"start_id"`
	EndID   int64 `json:"end_id"`
}

// BaseTableArchiveRestoreCase 管理归档恢复记录。
type BaseTableArchiveRestoreCase struct {
	*biz.BaseCase
	*data.BaseTableArchiveRestoreRepository
	archiveRecordRepo *data.BaseTableArchiveRecordRepository
}

// NewBaseTableArchiveRestoreCase 创建归档恢复业务实例。
func NewBaseTableArchiveRestoreCase(baseCase *biz.BaseCase, repo *data.BaseTableArchiveRestoreRepository, archiveRecordRepo *data.BaseTableArchiveRecordRepository) *BaseTableArchiveRestoreCase {
	return &BaseTableArchiveRestoreCase{BaseCase: baseCase, BaseTableArchiveRestoreRepository: repo, archiveRecordRepo: archiveRecordRepo}
}

// PageBaseTableArchiveRestore 分页查询归档恢复记录。
func (c *BaseTableArchiveRestoreCase) PageBaseTableArchiveRestore(ctx context.Context, req *adminv1.PageBaseTableArchiveRestoreRequest) (*adminv1.PageBaseTableArchiveRestoreResponse, error) {
	query := c.Query(ctx).BaseTableArchiveRestore
	opts := []repository.QueryOption{repository.Order(query.ID.Desc())}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseTableArchiveRestore, 0, len(list))
	for _, item := range list {
		items = append(items, toBaseTableArchiveRestore(item))
	}
	return &adminv1.PageBaseTableArchiveRestoreResponse{BaseTableArchiveRestores: items, Total: int32(total)}, nil
}

// GetBaseTableArchiveRestore 查询归档恢复记录。
func (c *BaseTableArchiveRestoreCase) GetBaseTableArchiveRestore(ctx context.Context, id int64) (*adminv1.BaseTableArchiveRestore, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBaseTableArchiveRestore(item), nil
}

// ExecuteBaseTableArchiveRestore 人工执行归档恢复并记录结果。
func (c *BaseTableArchiveRestoreCase) ExecuteBaseTableArchiveRestore(ctx context.Context, req *adminv1.BaseTableArchiveRestore) error {
	if req.GetArchiveRecordId() <= 0 {
		return errorsx.InvalidArgument("归档记录不能为空")
	}
	if req.GetRestoreMode() != adminv1.BaseTableArchiveRestoreMode_BASE_TABLE_ARCHIVE_RESTORE_MODE_ALL && req.GetRestoreMode() != adminv1.BaseTableArchiveRestoreMode_BASE_TABLE_ARCHIVE_RESTORE_MODE_SELECTED {
		return errorsx.InvalidArgument("归档恢复模式无效")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var archiveRecord *models.BaseTableArchiveRecord
	archiveRecord, err = c.archiveRecordRepo.FindByID(ctx, req.GetArchiveRecordId())
	if err != nil {
		return err
	}
	now := time.Now()
	entity := &models.BaseTableArchiveRestore{ArchiveRecordID: req.GetArchiveRecordId(), TableName_: archiveRecord.TableName_, RestoreMode: int32(req.GetRestoreMode()), RestoreRange: req.GetRestoreRange(), RestoredRows: 0, OperatorID: authInfo.UserId, Status: int32(adminv1.BaseTableArchiveRestoreStatus_BASE_TABLE_ARCHIVE_RESTORE_STATUS_RUNNING), Error: "", StartedAt: now, FinishedAt: now}
	if err = c.Create(ctx, entity); err != nil {
		return err
	}
	var restoredRows int64
	restoredRows, err = restoreArchiveRecord(ctx, c.BaseCase, archiveRecord, req.GetRestoreMode(), req.GetRestoreRange())
	if err != nil {
		entity.Status = int32(adminv1.BaseTableArchiveRestoreStatus_BASE_TABLE_ARCHIVE_RESTORE_STATUS_FAILED)
		entity.Error = err.Error()
		entity.FinishedAt = time.Now()
		if updateErr := c.UpdateByID(ctx, entity); updateErr != nil {
			return fmt.Errorf("%w；更新归档恢复失败记录失败: %v", err, updateErr)
		}
		return err
	}
	entity.RestoredRows = restoredRows
	entity.Status = int32(adminv1.BaseTableArchiveRestoreStatus_BASE_TABLE_ARCHIVE_RESTORE_STATUS_SUCCESS)
	entity.FinishedAt = time.Now()
	return c.UpdateByID(ctx, entity)
}

func toBaseTableArchiveRestore(item *models.BaseTableArchiveRestore) *adminv1.BaseTableArchiveRestore {
	return &adminv1.BaseTableArchiveRestore{Id: item.ID, ArchiveRecordId: item.ArchiveRecordID, TableName: item.TableName_, RestoreMode: adminv1.BaseTableArchiveRestoreMode(item.RestoreMode), RestoreRange: item.RestoreRange, RestoredRows: item.RestoredRows, OperatorId: item.OperatorID, Status: adminv1.BaseTableArchiveRestoreStatus(item.Status), Error: item.Error, StartedAt: item.StartedAt.Format(time.RFC3339), FinishedAt: item.FinishedAt.Format(time.RFC3339)}
}

// restoreArchiveRecord 将指定归档记录恢复到当前数据源。
func restoreArchiveRecord(ctx context.Context, baseCase *biz.BaseCase, archiveRecord *models.BaseTableArchiveRecord, mode adminv1.BaseTableArchiveRestoreMode, restoreRange string) (int64, error) {
	if archiveRecord.ArchiveMode == int32(adminv1.BaseTableArchiveMode_BASE_TABLE_ARCHIVE_MODE_INTERNAL_DATABASE) {
		return restoreInternalArchive(ctx, baseCase, archiveRecord, mode, restoreRange)
	}
	if archiveRecord.ArchiveMode == int32(adminv1.BaseTableArchiveMode_BASE_TABLE_ARCHIVE_MODE_OSS) {
		if mode == adminv1.BaseTableArchiveRestoreMode_BASE_TABLE_ARCHIVE_RESTORE_MODE_SELECTED {
			return 0, errorsx.InvalidArgument("OSS 归档暂不支持选择性恢复")
		}
		return restoreOSSArchive(ctx, baseCase, archiveRecord)
	}
	return 0, fmt.Errorf("归档模式无效")
}

// restoreInternalArchive 将当前数据源内部归档表中的记录幂等恢复到在线表。
func restoreInternalArchive(ctx context.Context, baseCase *biz.BaseCase, archiveRecord *models.BaseTableArchiveRecord, mode adminv1.BaseTableArchiveRestoreMode, restoreRange string) (int64, error) {
	if !tableNamePattern.MatchString(archiveRecord.TableName_) || !tableNamePattern.MatchString(archiveRecord.ArchiveTableName) {
		return 0, fmt.Errorf("归档表名不合法")
	}
	var client *gorm.Client
	var err error
	client, err = GormClientBySourceName(baseCase, archiveRecord.SourceName)
	if err != nil {
		return 0, err
	}
	var whereSQL string
	var values []interface{}
	whereSQL, values, err = restoreRangeSQL(mode, restoreRange)
	if err != nil {
		return 0, err
	}
	source := "`" + archiveRecord.TableName_ + "`"
	archive := "`" + archiveRecord.ArchiveTableName + "`"
	//nolint:forbidigo // 表名来自归档记录并通过白名单校验，恢复范围值使用参数绑定。
	query := "INSERT IGNORE INTO " + source + " SELECT * FROM " + archive + whereSQL
	result := client.DB.WithContext(ctx).Exec(query, values...)
	if result.Error != nil {
		return 0, fmt.Errorf("恢复内部归档数据失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// restoreOSSArchive 下载 OSS 归档 SQL 并恢复到当前数据源。
func restoreOSSArchive(ctx context.Context, baseCase *biz.BaseCase, archiveRecord *models.BaseTableArchiveRecord) (int64, error) {
	if baseCase.OSS == nil || archiveRecord.ObjectKey == "" {
		return 0, fmt.Errorf("OSS 归档对象未配置")
	}
	dataValue, err := baseCase.OSS.GetFileByte(archiveRecord.ObjectKey)
	if err != nil {
		return 0, fmt.Errorf("下载归档对象失败: %w", err)
	}
	if archiveRecord.Sha256 != "" {
		digest := sha256.Sum256(dataValue)
		if !hmac.Equal([]byte(archiveRecord.Sha256), []byte(hex.EncodeToString(digest[:]))) {
			return 0, fmt.Errorf("归档对象 SHA-256 校验失败")
		}
	}
	var dsn *mysql.Config
	dsn, err = databaseConfigBySourceName(baseCase, archiveRecord.SourceName)
	if err != nil {
		return 0, err
	}
	return importSQLBytes(ctx, dsn, dsn.DBName, dataValue)
}

func restoreRangeSQL(mode adminv1.BaseTableArchiveRestoreMode, restoreRange string) (string, []interface{}, error) {
	if mode == adminv1.BaseTableArchiveRestoreMode_BASE_TABLE_ARCHIVE_RESTORE_MODE_ALL {
		return "", nil, nil
	}
	var value restoreIDRange
	if err := json.Unmarshal([]byte(restoreRange), &value); err != nil || value.StartID <= 0 || value.EndID < value.StartID {
		return "", nil, fmt.Errorf("选择性恢复范围必须是有效的 start_id/end_id")
	}
	return " WHERE `id` BETWEEN ? AND ?", []interface{}{value.StartID, value.EndID}, nil
}

func databaseConfigBySourceName(baseCase *biz.BaseCase, sourceName string) (*mysql.Config, error) {
	dataConfig := baseCase.GetConfig().GetData()
	if dataConfig == nil {
		return nil, fmt.Errorf("数据源配置为空")
	}
	databaseConfig := dataConfig.GetDatabases()[sourceName]
	if sourceName == gorm.DefaultClientName && databaseConfig == nil {
		databaseConfig = dataConfig.GetDatabase()
	}
	if databaseConfig == nil {
		return nil, fmt.Errorf("目标数据源 %s 未配置", sourceName)
	}
	dsn, err := mysql.ParseDSN(databaseConfig.GetSource())
	if err != nil {
		return nil, fmt.Errorf("解析目标数据源失败: %w", err)
	}
	return dsn, nil
}

func importSQLBytes(ctx context.Context, dsn *mysql.Config, database string, content []byte) (int64, error) {
	temporaryDirectory, err := os.MkdirTemp("", "kratos-table-archive-restore-")
	if err != nil {
		return 0, fmt.Errorf("创建归档恢复目录失败: %w", err)
	}
	defer os.RemoveAll(temporaryDirectory)
	path := filepath.Join(temporaryDirectory, "archive.sql")
	if err = os.WriteFile(path, content, 0o600); err != nil {
		return 0, fmt.Errorf("写入归档恢复文件失败: %w", err)
	}
	return importSQLFile(ctx, dsn, database, path)
}

func importSQLFile(ctx context.Context, dsn *mysql.Config, database, sqlPath string) (int64, error) {
	var file *os.File
	var err error
	file, err = os.Open(sqlPath)
	if err != nil {
		return 0, fmt.Errorf("打开 SQL 恢复文件失败: %w", err)
	}
	defer file.Close()
	if backup.CommandAvailable(backup.MysqlCommand) {
		args := mysqlCommandArgs(dsn, database)
		var passwordFile string
		passwordFile, err = backup.WriteMySQLDefaultsFile(dsn.Passwd)
		if err != nil {
			return 0, err
		}
		defer os.Remove(passwordFile)
		args = append([]string{"--defaults-extra-file=" + passwordFile}, args...)
		command := exec.CommandContext(ctx, backup.MysqlCommand, args...)
		command.Stdin = file
		if err = command.Run(); err != nil {
			return 0, fmt.Errorf("执行 SQL 恢复失败: %w", err)
		}
		return 0, nil
	}
	sqlDB, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		return 0, fmt.Errorf("创建 Go MySQL 恢复连接失败: %w", err)
	}
	defer sqlDB.Close()
	if err = sqlDB.PingContext(ctx); err != nil {
		return 0, fmt.Errorf("连接 Go MySQL 恢复数据库失败: %w", err)
	}
	if err = backup.RestoreMySQL(ctx, sqlDB, database, file); err != nil {
		return 0, fmt.Errorf("Go 执行 SQL 恢复失败: %w", err)
	}
	return 0, nil
}

func mysqlCommandArgs(dsn *mysql.Config, database string) []string {
	args := make([]string, 0, 8)
	if dsn.Net == "unix" && dsn.Addr != "" {
		args = append(args, "--socket="+dsn.Addr)
	} else if dsn.Addr != "" {
		host := dsn.Addr
		port := "3306"
		parsedHost, parsedPort, err := net.SplitHostPort(dsn.Addr)
		if err == nil {
			host = parsedHost
			port = parsedPort
		}
		args = append(args, "--host="+host, "--port="+port)
	}
	if dsn.User != "" {
		args = append(args, "--user="+dsn.User)
	}
	return append(args, database)
}
