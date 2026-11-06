package data

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"gorm.io/gen"
	"gorm.io/gorm"
)

const aiCheckpointMaxSize = 192 * 1024 * 1024

// AiCheckPointStore 将 Eino 快照与对应运行的审批状态保存在同一数据库记录中。
type AiCheckPointStore struct {
	repo    *data.AiToolApprovalRepository
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

// NewAiCheckPointStore 创建压缩的数据库 checkpoint 存储。
func NewAiCheckPointStore(repo *data.AiToolApprovalRepository) (*AiCheckPointStore, error) {
	decoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(aiCheckpointMaxSize))
	if err != nil {
		return nil, err
	}
	var encoder *zstd.Encoder
	// Eino 在一份快照中重复保存上下文，窗口必须覆盖单轮内联附件，才能消除重复副本。
	encoder, err = zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), zstd.WithEncoderLevel(zstd.SpeedBetterCompression), zstd.WithWindowSize(64*1024*1024))
	if err != nil {
		decoder.Close()
		return nil, err
	}
	return &AiCheckPointStore{repo: repo, encoder: encoder, decoder: decoder}, nil
}

// Get 只读取已经由业务入口取得执行权且尚未过期的运行快照。
func (s *AiCheckPointStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	id, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return nil, false, err
	}
	var item *models.AiToolApproval
	item, err = s.repo.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if item.Status != 0 || !time.Now().Before(item.ExpiresAt) {
		return nil, false, fmt.Errorf("agent checkpoint is not running or has expired")
	}
	if len(item.CheckpointData) == 0 {
		return nil, false, nil
	}
	var value []byte
	value, err = s.decoder.DecodeAll(item.CheckpointData, nil)
	if err != nil {
		return nil, false, fmt.Errorf("decode agent checkpoint: %w", err)
	}
	return value, true, nil
}

// Set 只为仍拥有执行权的运行写入快照，存储失败时不向用户发出可确认状态。
func (s *AiCheckPointStore) Set(ctx context.Context, key string, value []byte) error {
	id, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return err
	}
	if len(value) > aiCheckpointMaxSize {
		return fmt.Errorf("agent checkpoint exceeds 192 MiB")
	}
	compressed := s.encoder.EncodeAll(value, nil)
	if len(compressed) > 48*1024*1024 {
		return fmt.Errorf("compressed agent checkpoint exceeds 48 MiB")
	}
	query := s.repo.Query(ctx).AiToolApproval
	var result gen.ResultInfo
	result, err = query.WithContext(ctx).Where(query.ID.Eq(id), query.Status.Eq(0), query.ExpiresAt.Gt(time.Now())).UpdateSimple(query.CheckpointData.Value(compressed), query.UpdatedAt.Value(time.Now()))
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("agent checkpoint execution has expired")
	}
	return nil
}
