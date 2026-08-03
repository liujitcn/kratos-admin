package biz

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"

	"github.com/liujitcn/gorm-kit/repository"
)

// LanguageCase 处理语言公共查询业务。
type LanguageCase struct {
	*data.BaseLanguageRepository
}

// NewLanguageCase 创建语言公共查询业务实例。
func NewLanguageCase(baseLanguageRepo *data.BaseLanguageRepository) *LanguageCase {
	return &LanguageCase{BaseLanguageRepository: baseLanguageRepo}
}

// GetLanguage 查询当前支持的语言和主语言。
func (c *LanguageCase) GetLanguage(ctx context.Context, _ *basev1.GetLanguageRequest) (*basev1.GetLanguageResponse, error) {
	query := c.Query(ctx).BaseLanguage
	opts := []repository.QueryOption{
		repository.Where(query.Status.Eq(_const.STATUS_ENABLE)),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	languages := make([]*basev1.LanguageItem, 0, len(list))
	primaryLanguageCode := ""
	for _, item := range list {
		languages = append(languages, &basev1.LanguageItem{
			LanguageCode: item.LanguageCode,
			LanguageName: item.LanguageName,
			NativeName:   item.NativeName,
			Sort:         item.Sort,
		})
		if item.IsPrimary && primaryLanguageCode == "" {
			primaryLanguageCode = item.LanguageCode
		}
	}
	if primaryLanguageCode == "" && len(languages) > 0 {
		primaryLanguageCode = languages[0].LanguageCode
	}
	return &basev1.GetLanguageResponse{
		Languages:           languages,
		PrimaryLanguageCode: primaryLanguageCode,
	}, nil
}
