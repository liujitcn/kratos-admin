package biz

import (
	"context"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
)

// MqttMessageCase 提供 MQTT 消息记录的只读查询能力。
type MqttMessageCase struct {
	*biz.BaseCase
	*data.MqttMessageRepository
}

// NewMqttMessageCase 创建 MQTT 消息记录查询业务实例。
func NewMqttMessageCase(baseCase *biz.BaseCase, repository *data.MqttMessageRepository) *MqttMessageCase {
	return &MqttMessageCase{BaseCase: baseCase, MqttMessageRepository: repository}
}

// PageMqttMessage 分页查询 MQTT 消息记录。
func (c *MqttMessageCase) PageMqttMessage(ctx context.Context, req *adminv1.PageMqttMessageRequest) (*adminv1.PageMqttMessageResponse, error) {
	query := c.Query(ctx).MqttMessage
	opts := []repository.QueryOption{repository.Order(query.CreatedAt.Desc()), repository.Order(query.ID.Desc())}
	if value := mqttMessageDirectionValue(req.GetDirection()); value != "" {
		opts = append(opts, repository.Where(query.Direction.Eq(value)))
	}
	if req.GetTopic() != "" {
		opts = append(opts, repository.Where(query.Topic.Like("%"+req.GetTopic()+"%")))
	}
	if value := mqttMessageStatusValue(req.GetStatus()); value != "" {
		opts = append(opts, repository.Where(query.Status.Eq(value)))
	}
	if value := mqttMessagePayloadFormatValue(req.GetPayloadFormat()); value != "" {
		opts = append(opts, repository.Where(query.PayloadFormat.Eq(value)))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.MqttMessageSummary, 0, len(list))
	for _, item := range list {
		items = append(items, toMqttMessageSummary(item))
	}
	return &adminv1.PageMqttMessageResponse{MqttMessages: items, Total: int32(total)}, nil
}

// GetMqttMessage 查询 MQTT 消息记录详情。
func (c *MqttMessageCase) GetMqttMessage(ctx context.Context, id int64) (*adminv1.MqttMessageDetail, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toMqttMessageDetail(item), nil
}

// toMqttMessageSummary 转换 MQTT 消息记录摘要。
func toMqttMessageSummary(item *models.MqttMessage) *adminv1.MqttMessageSummary {
	return &adminv1.MqttMessageSummary{
		Id:            item.ID,
		Direction:     mqttMessageDirectionEnum(item.Direction),
		Topic:         item.Topic,
		PayloadSize:   item.PayloadSize,
		PayloadFormat: mqttMessagePayloadFormatEnum(item.PayloadFormat),
		Qos:           item.Qos,
		Status:        mqttMessageStatusEnum(item.Status),
		LatencyMs:     item.LatencyMs,
		CreatedAt:     formatLogTime(item.CreatedAt),
	}
}

// toMqttMessageDetail 转换 MQTT 消息记录详情并保留原始 Payload。
func toMqttMessageDetail(item *models.MqttMessage) *adminv1.MqttMessageDetail {
	handledAt := ""
	if item.HandledAt > 0 {
		handledAt = formatLogTime(time.UnixMilli(item.HandledAt))
	}
	return &adminv1.MqttMessageDetail{
		Id:            item.ID,
		Direction:     mqttMessageDirectionEnum(item.Direction),
		Topic:         item.Topic,
		PayloadRaw:    append([]byte(nil), item.PayloadRaw...),
		PayloadJson:   item.PayloadJSON,
		PayloadSize:   item.PayloadSize,
		PayloadFormat: mqttMessagePayloadFormatEnum(item.PayloadFormat),
		Qos:           item.Qos,
		Retained:      item.Retained != 0,
		Duplicate:     item.DuplicateFlag != 0,
		MessageId:     item.MessageID,
		Status:        mqttMessageStatusEnum(item.Status),
		ErrorMessage:  item.ErrorMessage,
		LatencyMs:     item.LatencyMs,
		CreatedAt:     formatLogTime(item.CreatedAt),
		HandledAt:     handledAt,
	}
}

// mqttMessageDirectionValue 将消息方向枚举转换为数据库稳定值。
func mqttMessageDirectionValue(value adminv1.MqttMessageDirection) string {
	switch value {
	case adminv1.MqttMessageDirection_MQTT_MESSAGE_DIRECTION_PUBLISH:
		return "PUBLISH"
	case adminv1.MqttMessageDirection_MQTT_MESSAGE_DIRECTION_SUBSCRIBE:
		return "SUBSCRIBE"
	default:
		return ""
	}
}

// mqttMessagePayloadFormatValue 将 Payload 格式枚举转换为数据库稳定值。
func mqttMessagePayloadFormatValue(value adminv1.MqttMessagePayloadFormat) string {
	switch value {
	case adminv1.MqttMessagePayloadFormat_MQTT_MESSAGE_PAYLOAD_FORMAT_JSON:
		return "JSON"
	case adminv1.MqttMessagePayloadFormat_MQTT_MESSAGE_PAYLOAD_FORMAT_TEXT:
		return "TEXT"
	case adminv1.MqttMessagePayloadFormat_MQTT_MESSAGE_PAYLOAD_FORMAT_BINARY:
		return "BINARY"
	default:
		return ""
	}
}

// mqttMessageStatusValue 将消息状态枚举转换为数据库稳定值。
func mqttMessageStatusValue(value adminv1.MqttMessageStatus) string {
	switch value {
	case adminv1.MqttMessageStatus_MQTT_MESSAGE_STATUS_SENT:
		return "SENT"
	case adminv1.MqttMessageStatus_MQTT_MESSAGE_STATUS_HANDLED:
		return "HANDLED"
	case adminv1.MqttMessageStatus_MQTT_MESSAGE_STATUS_FAILED:
		return "FAILED"
	default:
		return ""
	}
}

// mqttMessageDirectionEnum 将数据库方向值转换为接口枚举。
func mqttMessageDirectionEnum(value string) adminv1.MqttMessageDirection {
	switch value {
	case "PUBLISH":
		return adminv1.MqttMessageDirection_MQTT_MESSAGE_DIRECTION_PUBLISH
	case "SUBSCRIBE":
		return adminv1.MqttMessageDirection_MQTT_MESSAGE_DIRECTION_SUBSCRIBE
	default:
		return adminv1.MqttMessageDirection_MQTT_MESSAGE_DIRECTION_UNSPECIFIED
	}
}

// mqttMessagePayloadFormatEnum 将数据库 Payload 格式转换为接口枚举。
func mqttMessagePayloadFormatEnum(value string) adminv1.MqttMessagePayloadFormat {
	switch value {
	case "JSON":
		return adminv1.MqttMessagePayloadFormat_MQTT_MESSAGE_PAYLOAD_FORMAT_JSON
	case "TEXT":
		return adminv1.MqttMessagePayloadFormat_MQTT_MESSAGE_PAYLOAD_FORMAT_TEXT
	case "BINARY":
		return adminv1.MqttMessagePayloadFormat_MQTT_MESSAGE_PAYLOAD_FORMAT_BINARY
	default:
		return adminv1.MqttMessagePayloadFormat_MQTT_MESSAGE_PAYLOAD_FORMAT_UNSPECIFIED
	}
}

// mqttMessageStatusEnum 将数据库状态转换为接口枚举。
func mqttMessageStatusEnum(value string) adminv1.MqttMessageStatus {
	switch value {
	case "SENT":
		return adminv1.MqttMessageStatus_MQTT_MESSAGE_STATUS_SENT
	case "HANDLED":
		return adminv1.MqttMessageStatus_MQTT_MESSAGE_STATUS_HANDLED
	case "FAILED":
		return adminv1.MqttMessageStatus_MQTT_MESSAGE_STATUS_FAILED
	default:
		return adminv1.MqttMessageStatus_MQTT_MESSAGE_STATUS_UNSPECIFIED
	}
}
