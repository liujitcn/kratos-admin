package errorsx

import (
	stderrs "errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// WrapInternal 在错误尚未完成分类时，包装成内部错误。
func WrapInternal(err error, message string) error {
	// 仓储层统一返回 gorm.ErrRecordNotFound，这类可预期查询空结果应对外表现为资源不存在。
	if stderrs.Is(err, gorm.ErrRecordNotFound) {
		return ResourceNotFound(message).WithCause(err)
	}
	return WrapIfNeeded(err, Internal(message))
}

// IsMySQLDuplicateKey 判断是否命中了 MySQL 唯一键冲突。
func IsMySQLDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return stderrs.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
