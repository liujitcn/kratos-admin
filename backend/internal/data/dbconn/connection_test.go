package dbconn

import (
	"net/url"
	"testing"

	"github.com/go-sql-driver/mysql"
)

// TestDSNIPv6Host 验证驱动可使用有无方括号的 IPv6 地址，且作用域标识不丢失。
func TestDSNIPv6Host(t *testing.T) {
	for _, driver := range []string{"mysql", "doris", "tidb", "postgres"} {
		for _, tc := range []struct{ host, address string }{
			{"::1", "[::1]:1234"},
			{"[::1]", "[::1]:1234"},
			{"[fe80::1%eth0]", "[fe80::1%eth0]:1234"},
			{"db.example", "db.example:1234"},
		} {
			t.Run(driver+"/"+tc.host, func(t *testing.T) {
				driverName, dsn, err := DSN(Config{Driver: driver, Host: tc.host, Port: 1234, DatabaseName: "demo", Username: "reader"}, "")
				if err != nil {
					t.Fatal(err)
				}
				var address string
				if driverName == "postgres" {
					var parsed *url.URL
					parsed, err = url.Parse(dsn)
					if err != nil {
						t.Fatal(err)
					}
					address = parsed.Host
				} else {
					var parsed *mysql.Config
					parsed, err = mysql.ParseDSN(dsn)
					if err != nil {
						t.Fatal(err)
					}
					address = parsed.Addr
				}
				if address != tc.address {
					t.Errorf("连接地址为 %q，预期 %q", address, tc.address)
				}
			})
		}
	}
}
