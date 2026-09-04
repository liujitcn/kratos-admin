package data

import (
	"context"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"gorm.io/gen/field"
	"gorm.io/gorm/clause"
)

// MessageDeliveryWriter 封装消息投递需要的幂等批量写入和可空时间更新。
type MessageDeliveryWriter struct {
	queryProvider data.QueryProvider
}

// NewMessageDeliveryWriter 创建消息投递写入器。
func NewMessageDeliveryWriter(queryProvider data.QueryProvider) *MessageDeliveryWriter {
	return &MessageDeliveryWriter{queryProvider: queryProvider}
}

// CreateIgnore 幂等批量创建用户投递记录并返回实际新增数量。
func (w *MessageDeliveryWriter) CreateIgnore(ctx context.Context, list []*models.BaseMessageDelivery, batchSize int) (int64, error) {
	if len(list) == 0 {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 500
	}
	query := w.queryProvider.Query(ctx).BaseMessageDelivery
	userIDs := make([]int64, 0, len(list))
	for _, item := range list {
		userIDs = append(userIDs, item.UserID)
	}
	existing, err := query.WithContext(ctx).Unscoped().Where(
		query.MessageID.Eq(list[0].MessageID),
		query.UserID.In(userIDs...),
	).Find()
	if err != nil {
		return 0, err
	}
	if len(existing) > 0 {
		existingUsers := make(map[int64]struct{}, len(existing))
		for _, item := range existing {
			existingUsers[item.UserID] = struct{}{}
		}
		filtered := make([]*models.BaseMessageDelivery, 0, len(list))
		for _, item := range list {
			if _, ok := existingUsers[item.UserID]; !ok {
				filtered = append(filtered, item)
			}
		}
		list = filtered
		if len(list) == 0 {
			return 0, nil
		}
	}
	// gorm/gen 当前没有带 OnConflict 的批量 Create API；此处仅用于唯一键幂等写入，输入字段来自受控模型。
	//nolint:forbidigo // 受控的批量幂等写入需要底层 CreateInBatches。
	result := query.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).UnderlyingDB().CreateInBatches(list, batchSize)
	return result.RowsAffected, result.Error
}

// SetReadAt 批量设置或清空当前用户投递记录的已读时间。
func (w *MessageDeliveryWriter) SetReadAt(ctx context.Context, userID int64, ids []int64, readAt *time.Time) error {
	query := w.queryProvider.Query(ctx).BaseMessageDelivery
	dao := query.WithContext(ctx).Where(
		query.UserID.Eq(userID), query.ID.In(ids...), query.RevokedAt.Eq(0),
		field.Or(query.ExpiresAt.Eq(0), query.ExpiresAt.Gt(time.Now().UnixMilli())),
	)
	var readAtValue int64
	if readAt == nil {
		readAtValue = 0
	} else {
		readAtValue = readAt.UnixMilli()
	}
	_, err := dao.UpdateSimple(query.ReadAt.Value(readAtValue), query.UpdatedAt.Value(time.Now()))
	return err
}

// SetAllReadAt 按水位线批量设置当前用户消息已读时间。
func (w *MessageDeliveryWriter) SetAllReadAt(ctx context.Context, userID, beforeDeliveryID int64, readAt time.Time) error {
	query := w.queryProvider.Query(ctx).BaseMessageDelivery
	_, err := query.WithContext(ctx).
		Where(query.UserID.Eq(userID), query.ID.Lte(beforeDeliveryID), query.ReadAt.Eq(0), query.RevokedAt.Eq(0), field.Or(query.ExpiresAt.Eq(0), query.ExpiresAt.Gt(time.Now().UnixMilli()))).
		UpdateSimple(query.ReadAt.Value(readAt.UnixMilli()), query.UpdatedAt.Value(readAt))
	return err
}

// SetArchivedAt 设置或清空当前用户单条投递记录的归档时间。
func (w *MessageDeliveryWriter) SetArchivedAt(ctx context.Context, userID, id int64, archivedAt *time.Time) error {
	query := w.queryProvider.Query(ctx).BaseMessageDelivery
	dao := query.WithContext(ctx).Where(
		query.UserID.Eq(userID), query.ID.Eq(id), query.RevokedAt.Eq(0),
		field.Or(query.ExpiresAt.Eq(0), query.ExpiresAt.Gt(time.Now().UnixMilli())),
	)
	var archivedAtValue int64
	if archivedAt == nil {
		archivedAtValue = 0
	} else {
		archivedAtValue = archivedAt.UnixMilli()
	}
	_, err := dao.UpdateSimple(query.ArchivedAt.Value(archivedAtValue), query.UpdatedAt.Value(time.Now()))
	return err
}

// RevokeMessage 批量撤回指定消息的全部用户投递记录。
func (w *MessageDeliveryWriter) RevokeMessage(ctx context.Context, messageID int64, revokedAt time.Time) error {
	query := w.queryProvider.Query(ctx).BaseMessageDelivery
	_, err := query.WithContext(ctx).
		Where(query.MessageID.Eq(messageID), query.RevokedAt.Eq(0)).
		UpdateSimple(query.RevokedAt.Value(revokedAt.UnixMilli()), query.UpdatedAt.Value(revokedAt))
	return err
}
