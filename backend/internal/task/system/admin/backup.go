package admin

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/liujitcn/kratos-kit/transport/cron"
)

const (
	// BackupTaskName 是数据库备份任务的稳定调用目标。
	BackupTaskName         = "system.admin.BaseBackup"
	backupDefaultRetention = 7
)

var _ cron.TaskExec = (*BackupTask)(nil)

// BackupTask 执行受控的 MySQL 逻辑备份并按数量轮换归档文件。
// 任务生成的压缩文件使用独立密钥加密，并用独立完整性密钥生成 HMAC，避免明文备份直接落盘。
type BackupTask struct{}

// NewBackupTask 创建数据库备份任务。
func NewBackupTask() *BackupTask {
	return &BackupTask{}
}

// Task 返回交由 base_job 调度的任务定义。
func (t *BackupTask) Task() cron.Task {
	return cron.Task{Name: BackupTaskName, Exec: t}
}

// Exec 执行一次数据库备份；未显式启用或缺少数据库名时跳过任务。
func (t *BackupTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	if !envBool("BACKUP_ENABLED", false) {
		return []string{"数据库备份未启用"}, nil
	}
	if strings.TrimSpace(os.Getenv("BACKUP_INTEGRITY_KEY")) == "" {
		return nil, fmt.Errorf("BACKUP_INTEGRITY_KEY 未配置，拒绝创建无完整性保护的备份")
	}
	if strings.TrimSpace(os.Getenv("BACKUP_ENCRYPTION_KEY")) == "" {
		return nil, fmt.Errorf("BACKUP_ENCRYPTION_KEY 未配置，拒绝创建明文备份")
	}
	database := os.Getenv("MYSQLDUMP_DATABASE")
	if database == "" {
		return nil, fmt.Errorf("MYSQLDUMP_DATABASE 未配置")
	}
	directory := os.Getenv("BACKUP_DIR")
	if directory == "" {
		directory = "./data/backups"
	}
	var err error
	err = os.MkdirAll(directory, 0o750)
	if err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %w", err)
	}
	backupPath := filepath.Join(directory, fmt.Sprintf("kratos-admin-%s.sql", time.Now().UTC().Format("20060102-150405")))
	temporaryPath := backupPath + ".tmp"
	err = t.dump(ctx, database, temporaryPath)
	if err != nil {
		if removeErr := os.Remove(temporaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return nil, fmt.Errorf("%w；清理临时备份失败: %v", err, removeErr)
		}
		return nil, err
	}
	finalPath := backupPath
	if envBool("BACKUP_GZIP", true) {
		finalPath += ".gz"
		err = gzipFile(temporaryPath, finalPath)
		if err != nil {
			if removeErr := os.Remove(temporaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return nil, fmt.Errorf("%w；清理临时备份失败: %v", err, removeErr)
			}
			return nil, err
		}
		err = os.Remove(temporaryPath)
		if err != nil {
			return nil, fmt.Errorf("删除临时备份文件失败: %w", err)
		}
	} else {
		err = os.Rename(temporaryPath, finalPath)
		if err != nil {
			return nil, fmt.Errorf("提交备份文件失败: %w", err)
		}
	}
	encryptedPath := finalPath + ".enc"
	err = encryptBackup(ctx, finalPath, encryptedPath)
	if err != nil {
		if removeErr := os.Remove(finalPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return nil, fmt.Errorf("%w；清理明文备份失败: %v", err, removeErr)
		}
		if removeErr := os.Remove(encryptedPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return nil, fmt.Errorf("%w；清理加密备份失败: %v", err, removeErr)
		}
		return nil, err
	}
	err = os.Remove(finalPath)
	if err != nil {
		if removeErr := os.Remove(encryptedPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return nil, fmt.Errorf("删除明文备份失败: %v；清理加密备份失败: %w", err, removeErr)
		}
		return nil, fmt.Errorf("删除明文备份失败: %w", err)
	}
	finalPath = encryptedPath
	err = writeBackupChecksum(finalPath)
	if err != nil {
		return nil, err
	}
	err = rotateBackups(directory)
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("数据库备份完成: %s", finalPath)}, nil
}

// dump 调用 mysqldump，密码通过子进程环境传递，避免出现在进程参数中。
func (t *BackupTask) dump(ctx context.Context, database, output string) error {
	command := os.Getenv("MYSQLDUMP_BIN")
	if command == "" {
		command = "mysqldump"
	}
	args := []string{"--single-transaction", "--routines", "--events"}
	if host := os.Getenv("MYSQLDUMP_HOST"); host != "" {
		args = append(args, "--host="+host)
	}
	if port := os.Getenv("MYSQLDUMP_PORT"); port != "" {
		args = append(args, "--port="+port)
	}
	if user := os.Getenv("MYSQLDUMP_USER"); user != "" {
		args = append(args, "--user="+user)
	}
	args = append(args, "--databases", database)
	file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建临时备份文件失败: %w", err)
	}
	defer file.Close()
	commandValue := exec.CommandContext(ctx, command, args...)
	commandValue.Stdout = file
	commandValue.Stderr = os.Stderr
	if password := os.Getenv("MYSQLDUMP_PASSWORD"); password != "" {
		commandValue.Env = append(os.Environ(), "MYSQL_PWD="+password)
	}
	if err = commandValue.Run(); err != nil {
		return fmt.Errorf("执行 mysqldump 失败: %w", err)
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("同步备份文件失败: %w", err)
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
		if closeErr := compressor.Close(); closeErr != nil {
			return fmt.Errorf("压缩备份失败且关闭压缩器失败: %v: %w", err, closeErr)
		}
		if closeErr := output.Close(); closeErr != nil {
			return fmt.Errorf("压缩备份失败且关闭文件失败: %v: %w", err, closeErr)
		}
		return fmt.Errorf("压缩备份失败: %w", err)
	}
	if err = compressor.Close(); err != nil {
		if closeErr := output.Close(); closeErr != nil {
			return fmt.Errorf("完成备份压缩失败且关闭文件失败: %v: %w", err, closeErr)
		}
		return fmt.Errorf("完成备份压缩失败: %w", err)
	}
	if err = output.Sync(); err != nil {
		if closeErr := output.Close(); closeErr != nil {
			return fmt.Errorf("同步压缩备份失败且关闭文件失败: %v: %w", err, closeErr)
		}
		return fmt.Errorf("同步压缩备份失败: %w", err)
	}
	if err = output.Close(); err != nil {
		return fmt.Errorf("关闭压缩备份失败: %w", err)
	}
	return nil
}

// rotateBackups 按配置保留最近的备份及其校验文件。
func rotateBackups(directory string) error {
	retention := backupDefaultRetention
	if value := os.Getenv("BACKUP_RETENTION_COUNT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return fmt.Errorf("BACKUP_RETENTION_COUNT 配置无效")
		}
		retention = parsed
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %w", err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "kratos-admin-") {
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
		if err = os.Remove(paths[0] + ".sha256"); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("删除过期备份校验文件失败: %w", err)
		}
		if err = os.Remove(paths[0] + ".hmac"); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("删除过期备份 HMAC 文件失败: %w", err)
		}
		paths = paths[1:]
	}
	return nil
}

// encryptBackup 使用 OpenSSL AES-256-CBC 加密备份文件，密钥仅通过环境变量传入子进程。
func encryptBackup(ctx context.Context, source, target string) error {
	command := os.Getenv("OPENSSL_BIN")
	if command == "" {
		command = "openssl"
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建加密备份文件失败: %w", err)
	}
	defer file.Close()
	commandValue := exec.CommandContext(ctx, command, "enc", "-aes-256-cbc", "-pbkdf2", "-iter", "100000", "-salt", "-in", source, "-out", "/dev/stdout", "-pass", "env:BACKUP_ENCRYPTION_KEY")
	commandValue.Stdout = file
	commandValue.Stderr = os.Stderr
	commandValue.Env = append(os.Environ(), "BACKUP_ENCRYPTION_KEY="+os.Getenv("BACKUP_ENCRYPTION_KEY"))
	if err = commandValue.Run(); err != nil {
		return fmt.Errorf("加密备份失败: %w", err)
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("同步加密备份失败: %w", err)
	}
	return nil
}

// writeBackupChecksum 为备份文件生成 SHA-256 校验文件。
func writeBackupChecksum(path string) error {
	input, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer input.Close()
	digest := sha256.New()
	if _, err = io.Copy(digest, input); err != nil {
		return fmt.Errorf("计算备份校验值失败: %w", err)
	}
	checksumPath := path + ".sha256"
	checksum := hex.EncodeToString(digest.Sum(nil)) + "  " + filepath.Base(path) + "\n"
	if err = os.WriteFile(checksumPath, []byte(checksum), 0o600); err != nil {
		return fmt.Errorf("写入备份校验文件失败: %w", err)
	}
	key := []byte(os.Getenv("BACKUP_INTEGRITY_KEY"))
	if len(key) == 0 {
		return fmt.Errorf("BACKUP_INTEGRITY_KEY 未配置")
	}
	input, err = os.Open(path)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer input.Close()
	macValue := hmac.New(sha256.New, key)
	if _, err = io.Copy(macValue, input); err != nil {
		return fmt.Errorf("计算备份 HMAC 失败: %w", err)
	}
	hmacPath := path + ".hmac"
	hmacText := hex.EncodeToString(macValue.Sum(nil)) + "  " + filepath.Base(path) + "\n"
	if err = os.WriteFile(hmacPath, []byte(hmacText), 0o600); err != nil {
		return fmt.Errorf("写入备份 HMAC 文件失败: %w", err)
	}
	return nil
}

// envBool 读取布尔环境变量，未配置或格式无效时返回兜底值。
func envBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}
