package backup

import "os/exec"

const (
	// MysqldumpCommand 是数据库 SQL 导出的固定命令名。
	MysqldumpCommand = "mysqldump"
	// OpensslCommand 是备份文件加解密的固定命令名。
	OpensslCommand = "openssl"
	// MysqlCommand 是数据库 SQL 恢复的固定命令名。
	MysqlCommand = "mysql"
	// PgdumpCommand 是 PostgreSQL 数据库 SQL 导出的固定命令名。
	PgdumpCommand = "pg_dump"
	// PsqlCommand 是 PostgreSQL 数据库 SQL 恢复的固定命令名。
	PsqlCommand = "psql"
)

// CommandAvailable 判断指定命令是否可以通过当前进程的 PATH 或绝对路径执行。
func CommandAvailable(commandName string) bool {
	if commandName == "" {
		return false
	}
	_, err := exec.LookPath(commandName)
	return err == nil
}
