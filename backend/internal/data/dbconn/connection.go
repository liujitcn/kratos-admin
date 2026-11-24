package dbconn

import (
	"context"
	"database/sql"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/liujitcn/kratos-core/errorsx"
)

// Config 保存独立数据库的连接参数，密码单独传递。
type Config struct {
	Driver       string
	Host         string
	Port         int32
	DatabaseName string
	Username     string
	SSLMode      string
}

// Open 按受支持的驱动打开临时连接池，调用方负责关闭。
func Open(config Config, password string) (*sql.DB, error) {
	driver, dsn, err := DSN(config, password)
	if err != nil {
		return nil, err
	}
	return sql.Open(driver, dsn)
}

// Ping 在有界时间内探测数据库连接。
func Ping(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

// DSN 将独立连接配置转换为驱动 DSN，兼容已保存的带方括号 IPv6 地址。
func DSN(config Config, password string) (string, string, error) {
	host := config.Host
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		unbracketed := host[1 : len(host)-1]
		if ip, err := netip.ParseAddr(unbracketed); err == nil && ip.Is6() {
			host = unbracketed
		}
	}
	address := net.JoinHostPort(host, strconv.Itoa(int(config.Port)))
	driverName := strings.ToLower(config.Driver)
	switch driverName {
	case "mysql", "doris", "tidb":
		if config.Host == "" || config.Port <= 0 || config.Port > 65535 || config.DatabaseName == "" || config.Username == "" {
			return "", "", errorsx.InvalidArgument("MySQL兼容数据源连接信息不完整")
		}
		// 零值 Config 不允许 mysql_native_password，而 Doris 等数据源默认使用该认证方式，需显式开启。
		dsn := (&mysql.Config{User: config.Username, Passwd: password, Net: "tcp", Addr: address, DBName: config.DatabaseName, AllowNativePasswords: true, Params: map[string]string{"charset": "utf8mb4", "parseTime": "true", "loc": "Local"}}).FormatDSN()
		return "mysql", dsn, nil
	case "postgres":
		if config.Host == "" || config.Port <= 0 || config.Port > 65535 || config.DatabaseName == "" || config.Username == "" {
			return "", "", errorsx.InvalidArgument("PostgreSQL数据源连接信息不完整")
		}
		sslMode := config.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		connectionURL := &url.URL{Scheme: "postgres", Host: address, Path: "/" + config.DatabaseName, User: url.UserPassword(config.Username, password)}
		params := connectionURL.Query()
		params.Set("sslmode", sslMode)
		connectionURL.RawQuery = params.Encode()
		return "postgres", connectionURL.String(), nil
	default:
		return "", "", errorsx.InvalidArgument("数据库驱动不支持")
	}
}
