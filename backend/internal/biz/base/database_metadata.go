package biz

import "github.com/liujitcn/kratos-admin/backend/internal/data/dbmeta"

// MysqlFamilyDriver 判断驱动是否属于 MySQL 方言系（含复用 MySQL 方言的 Doris 与 TiDB）。
func MysqlFamilyDriver(driverName string) bool {
	return dbmeta.MysqlFamilyDriver(driverName)
}
