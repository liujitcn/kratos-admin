package biz

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-kratos/kratos/v3/log"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"

	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseAreaCase 行政区域业务处理对象
type BaseAreaCase struct {
	*biz.BaseCase
	*data.BaseAreaRepository
	mapper *mapper.CopierMapper[commonv1.AppTreeOptionResponse_Option, models.BaseArea]
}

// NewBaseAreaCase 创建行政区域业务处理对象
func NewBaseAreaCase(
	baseCase *biz.BaseCase,
	baseAreaRepo *data.BaseAreaRepository,
) *BaseAreaCase {
	return &BaseAreaCase{
		BaseCase:           baseCase,
		BaseAreaRepository: baseAreaRepo,
		mapper:             mapper.NewCopierMapper[commonv1.AppTreeOptionResponse_Option, models.BaseArea](),
	}
}

// TreeBaseArea 查询行政区域树形列表
func (c *BaseAreaCase) TreeBaseArea(ctx context.Context) (*commonv1.AppTreeOptionResponse, error) {
	var err error
	if c.Cache != nil {
		var cached string
		cached, err = c.Cache.Get(_const.BASE_AREA_CACHE_KEY)
		if err == nil {
			response := &commonv1.AppTreeOptionResponse{}
			err = json.Unmarshal([]byte(cached), response)
			if err == nil {
				return response, nil
			}
		}
	}

	query := c.Query(ctx).BaseArea
	var list []*models.BaseArea
	list, err = c.List(ctx, repository.Order(query.ID.Asc()))
	if err != nil {
		return nil, err
	}
	response := &commonv1.AppTreeOptionResponse{
		List: c.buildTree(list, 0),
	}
	if c.Cache != nil {
		var payload []byte
		payload, err = json.Marshal(response)
		if err != nil {
			log.Error("MarshalBaseAreaCache", "error", err)
		} else {
			err = c.Cache.Set(_const.BASE_AREA_CACHE_KEY, string(payload), _const.BASE_AREA_CACHE_EXPIRE)
			if err != nil {
				log.Error("SetBaseAreaCache", "error", err)
			}
		}
	}
	return response, nil
}

// 递归构建行政区域树
func (c *BaseAreaCase) buildTree(list []*models.BaseArea, parentID int64) []*commonv1.AppTreeOptionResponse_Option {
	var res []*commonv1.AppTreeOptionResponse_Option
	for _, item := range list {
		// 仅把当前父节点下的区域挂到本层结果里。
		if item.ParentID == parentID {
			option := c.mapper.ToDTO(item)
			option.Value = strconv.FormatInt(item.ID, 10)
			option.Text = item.Name
			option.Children = c.buildTree(list, item.ID)
			res = append(res, option)
		}
	}
	return res
}
