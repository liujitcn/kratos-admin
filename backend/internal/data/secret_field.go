package data

import (
	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// NewSecretFieldStorage 在默认数据库上绑定密钥字段加解密回调并返回运行时。
func NewSecretFieldStorage(databases map[string]*gorm.Client) (*kit.SecretFieldRuntime, error) {
	return kit.BindSecretFieldStorage(databases[gorm.DefaultClientName].DB)
}
