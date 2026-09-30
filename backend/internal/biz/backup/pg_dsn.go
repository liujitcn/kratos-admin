package backup

import (
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// PostgresDSN 汇总 PostgreSQL 连接配置的关键字段，供命令通道构造参数与环境变量。
type PostgresDSN struct {
	// Host 是主机名或 Unix 套接字目录。
	Host string
	// Port 是服务端口。
	Port string
	// User 是登录用户。
	User string
	// Password 是登录密码。
	Password string
	// Database 是目标数据库名称。
	Database string
}

// ParsePostgresDSN 解析 PostgreSQL 关键字值或 URL 形式的连接串。
func ParsePostgresDSN(source string) (PostgresDSN, error) {
	config, err := pgx.ParseConfig(source)
	if err != nil {
		return PostgresDSN{}, fmt.Errorf("解析 PostgreSQL 数据源失败: %w", err)
	}
	return PostgresDSN{
		Host:     config.Host,
		Port:     strconv.Itoa(int(config.Port)),
		User:     config.User,
		Password: config.Password,
		Database: config.Database,
	}, nil
}
