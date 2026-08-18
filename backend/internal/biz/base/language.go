package biz

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"

	"github.com/liujitcn/gorm-kit/repository"
)

// LanguageCase 处理语言公共查询业务。
type LanguageCase struct {
	*biz.BaseCase
	*data.BaseLanguageRepository
}

// NewLanguageCase 创建语言公共查询业务实例。
func NewLanguageCase(baseCase *biz.BaseCase, baseLanguageRepo *data.BaseLanguageRepository) *LanguageCase {
	return &LanguageCase{BaseCase: baseCase, BaseLanguageRepository: baseLanguageRepo}
}

// OptionLanguage 查询当前支持的语言选项。
func (c *LanguageCase) OptionLanguage(ctx context.Context, _ *basev1.OptionLanguageRequest) (*basev1.OptionLanguageResponse, error) {
	query := c.Query(ctx).BaseLanguage
	opts := []repository.QueryOption{
		repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	languages := make([]*basev1.LanguageItem, 0, len(list))
	for _, item := range list {
		languages = append(languages, &basev1.LanguageItem{
			LanguageCode: item.LanguageCode,
			NativeName:   item.NativeName,
		})
	}
	return &basev1.OptionLanguageResponse{
		Languages: languages,
	}, nil
}
