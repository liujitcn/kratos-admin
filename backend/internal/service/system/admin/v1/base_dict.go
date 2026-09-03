package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseDictService Admin字典服务
type BaseDictService struct {
	adminv1.UnimplementedBaseDictServiceServer
	baseDictCase *biz.BaseDictCase
}

// NewBaseDictService 创建Admin字典服务
func NewBaseDictService(dictCase *biz.BaseDictCase) *BaseDictService {
	return &BaseDictService{baseDictCase: dictCase}
}

// OptionBaseDict 查询字典下拉选择
func (s *BaseDictService) OptionBaseDict(ctx context.Context, req *adminv1.OptionBaseDictRequest) (*adminv1.OptionBaseDictResponse, error) {
	res, err := s.baseDictCase.OptionBaseDict(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseDict %v", err))
		return nil, errorsx.WrapInternal(err, "查询失败")
	}
	return res, nil
}

// PageBaseDict 查询字典分页列表
func (s *BaseDictService) PageBaseDict(ctx context.Context, req *adminv1.PageBaseDictRequest) (*adminv1.PageBaseDictResponse, error) {
	page, err := s.baseDictCase.PageBaseDict(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseDict %v", err))
		return nil, errorsx.WrapInternal(err, "查询字典分页列表失败")
	}

	return page, nil
}

// GetBaseDict 查询字典
func (s *BaseDictService) GetBaseDict(ctx context.Context, req *adminv1.GetBaseDictRequest) (*adminv1.BaseDictForm, error) {
	baseDict, err := s.baseDictCase.GetBaseDict(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseDict %v", err))
		return nil, errorsx.WrapInternal(err, "查询字典失败")
	}

	return baseDict, nil
}

// CreateBaseDict 创建字典
func (s *BaseDictService) CreateBaseDict(ctx context.Context, req *adminv1.CreateBaseDictRequest) (*emptypb.Empty, error) {
	err := s.baseDictCase.CreateBaseDict(ctx, req.GetBaseDict())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseDict %v", err))
		return nil, errorsx.WrapInternal(err, "创建字典失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseDict 更新字典
func (s *BaseDictService) UpdateBaseDict(ctx context.Context, req *adminv1.UpdateBaseDictRequest) (*emptypb.Empty, error) {
	err := s.baseDictCase.UpdateBaseDict(ctx, req.GetBaseDict())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseDict %v", err))
		return nil, errorsx.WrapInternal(err, "更新字典失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseDict 删除字典
func (s *BaseDictService) DeleteBaseDict(ctx context.Context, req *adminv1.DeleteBaseDictRequest) (*emptypb.Empty, error) {
	err := s.baseDictCase.DeleteBaseDict(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseDict %v", err))
		return nil, errorsx.WrapInternal(err, "删除字典失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseDictStatus 设置状态
func (s *BaseDictService) SetBaseDictStatus(ctx context.Context, req *adminv1.SetBaseDictStatusRequest) (*emptypb.Empty, error) {
	err := s.baseDictCase.SetBaseDictStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseDictStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}
