package core

import (
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	coredata "github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// NewTransaction 从数据库客户端创建事务接口，各适配器通过同一数据包的上下文键复用事务查询。
func NewTransaction(databases map[string]*gorm.Client) (coredata.Transaction, error) {
	provider, err := data.NewData(databases)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
