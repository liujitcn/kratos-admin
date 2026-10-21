package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// MqttMessageService 提供 MQTT 消息记录的只读管理接口。
type MqttMessageService struct {
	adminv1.UnimplementedMqttMessageServiceServer
	mqttMessageCase *biz.MqttMessageCase
}

// NewMqttMessageService 创建 MQTT 消息记录管理服务。
func NewMqttMessageService(mqttMessageCase *biz.MqttMessageCase) *MqttMessageService {
	return &MqttMessageService{mqttMessageCase: mqttMessageCase}
}

// PageMqttMessage 查询 MQTT 消息记录分页列表。
func (s *MqttMessageService) PageMqttMessage(ctx context.Context, req *adminv1.PageMqttMessageRequest) (*adminv1.PageMqttMessageResponse, error) {
	response, err := s.mqttMessageCase.PageMqttMessage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageMqttMessage %v", err))
		return nil, errorsx.WrapInternal(err, "查询 MQTT 消息记录失败")
	}
	return response, nil
}

// GetMqttMessage 查询 MQTT 消息记录详情。
func (s *MqttMessageService) GetMqttMessage(ctx context.Context, req *adminv1.GetMqttMessageRequest) (*adminv1.MqttMessageDetail, error) {
	response, err := s.mqttMessageCase.GetMqttMessage(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetMqttMessage %v", err))
		return nil, errorsx.WrapInternal(err, "查询 MQTT 消息详情失败")
	}
	return response, nil
}
