package biz

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gen/field"
)

const (
	aiFileSourceUpload    = "upload"
	aiFileSourceGenerated = "generated"
	aiFileStatusActive    = "active"
	aiFileStatusTrashed   = "trashed"
)

// AiFileCase 管理当前租户的 AI 文件元数据和对象生命周期。
type AiFileCase struct {
	*biz.BaseCase
	*data.AiFileRepository
}

// NewAiFileCase 创建 AI 文件业务实例。
func NewAiFileCase(baseCase *biz.BaseCase, aiFileRepo *data.AiFileRepository) *AiFileCase {
	return &AiFileCase{BaseCase: baseCase, AiFileRepository: aiFileRepo}
}

// PageAiFile 分页查询当前租户的 AI 文件。
func (c *AiFileCase) PageAiFile(ctx context.Context, req *basev1.PageAiFileRequest) (*basev1.PageAiFileResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if c.AiFileRepository == nil {
		return nil, errorsx.Internal("AI文件元数据仓储未配置")
	}
	query := c.Query(ctx).AiFile
	opts := []repository.QueryOption{
		repository.Where(query.TenantID.Eq(authInfo.TenantId)),
		repository.Order(query.CreatedAt.Desc()),
		repository.Order(query.ID.Desc()),
	}
	switch req.GetView() {
	case basev1.AiFileView_AI_FILE_VIEW_TRASHED:
		opts = append(opts, repository.Where(query.Status.Eq(aiFileStatusTrashed)))
	default:
		opts = append(opts, repository.Where(query.Status.Eq(aiFileStatusActive)))
		if req.GetView() == basev1.AiFileView_AI_FILE_VIEW_UPLOADED {
			opts = append(opts, repository.Where(query.SourceType.Eq(aiFileSourceUpload)))
		}
		if req.GetView() == basev1.AiFileView_AI_FILE_VIEW_GENERATED {
			opts = append(opts, repository.Where(query.SourceType.Eq(aiFileSourceGenerated)))
		}
	}
	if keyword := req.GetKeyword(); keyword != "" {
		pattern := "%" + keyword + "%"
		opts = append(opts, repository.Where(field.Or(query.FileName.Like(pattern), query.ContentHash.Like(pattern))))
	}
	var rows []*models.AiFile
	var total int64
	rows, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	files := make([]*basev1.AiFile, 0, len(rows))
	for _, row := range rows {
		files = append(files, aiFileDTO(row))
	}
	return &basev1.PageAiFileResponse{Files: files, Total: total}, nil
}

// RenameAiFile 修改当前租户 AI 文件的展示名称。
func (c *AiFileCase) RenameAiFile(ctx context.Context, req *basev1.RenameAiFileRequest) (*basev1.AiFile, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	row, err := c.ownedAiFile(ctx, req.GetId(), aiFileStatusActive)
	if err != nil {
		return nil, err
	}
	updatedAt := time.Now()
	query := c.Query(ctx).AiFile
	_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID), query.TenantID.Eq(authInfo.TenantId)).UpdateSimple(query.FileName.Value(req.GetFileName()), query.UpdatedBy.Value(authInfo.UserId), query.UpdatedAt.Value(updatedAt))
	if err != nil {
		return nil, err
	}
	row.FileName = req.GetFileName()
	row.UpdatedBy = authInfo.UserId
	row.UpdatedAt = updatedAt
	return aiFileDTO(row), nil
}

// TrashAiFile 将当前租户的 AI 文件移入回收站。
func (c *AiFileCase) TrashAiFile(ctx context.Context, req *basev1.TrashAiFileRequest) (*basev1.AiFile, error) {
	return c.updateAiFileStatus(ctx, req.GetId(), aiFileStatusActive, aiFileStatusTrashed)
}

// RestoreAiFile 恢复当前租户回收站中的 AI 文件。
func (c *AiFileCase) RestoreAiFile(ctx context.Context, req *basev1.RestoreAiFileRequest) (*basev1.AiFile, error) {
	return c.updateAiFileStatus(ctx, req.GetId(), aiFileStatusTrashed, aiFileStatusActive)
}

// DeleteAiFile 彻底删除当前租户的 AI 文件及对象。
func (c *AiFileCase) DeleteAiFile(ctx context.Context, req *basev1.DeleteAiFileRequest) (*emptypb.Empty, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	row, err := c.ownedAiFile(ctx, req.GetId(), aiFileStatusTrashed)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).AiFile
	if err = c.Delete(ctx, repository.Unscoped(), repository.Where(query.ID.Eq(row.ID), query.TenantID.Eq(authInfo.TenantId))); err != nil {
		return nil, err
	}
	if objectPath := aiFileObjectPath(row); objectPath != "" {
		if err = c.OSS.DeleteFile(objectPath); err != nil {
			return nil, errorsx.Internal("删除AI文件对象失败").WithCause(err)
		}
	}
	return new(emptypb.Empty), nil
}

// RegisterGeneratedFile 登记受信生成链路产生的 AI 文件，不写入 base_file。
func (c *AiFileCase) RegisterGeneratedFile(ctx context.Context, file *models.AiFile) error {
	if c.AiFileRepository == nil {
		return errorsx.Internal("AI文件元数据仓储未配置")
	}
	if file == nil || file.TenantID <= 0 || file.FileName == "" || file.LinkURL == "" {
		return errorsx.InvalidArgument("生成文件元数据不完整")
	}
	file.SourceType = aiFileSourceGenerated
	if file.Status == "" {
		file.Status = aiFileStatusActive
	}
	if file.FileGUID == "" {
		file.FileGUID = id.NewGUIDv7NoHyphen()
	}
	if file.FileDirectory == "" {
		file.FileDirectory = path.Dir(file.LinkURL)
	}
	if file.SaveFileName == "" {
		file.SaveFileName = path.Base(file.LinkURL)
	}
	if file.AccessMode == 0 {
		file.AccessMode = int32(basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_AUTHORIZED)
	}
	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now()
	}
	if file.UpdatedAt.IsZero() {
		file.UpdatedAt = file.CreatedAt
	}
	return c.Create(ctx, file)
}

// ownedAiFile 按租户和状态读取文件，隐藏跨租户和错误状态记录。
func (c *AiFileCase) ownedAiFile(ctx context.Context, id int64, status string) (*models.AiFile, error) {
	if c.AiFileRepository == nil {
		return nil, errorsx.Internal("AI文件元数据仓储未配置")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).AiFile
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id)), repository.Where(query.TenantID.Eq(authInfo.TenantId))}
	if status != "" {
		opts = append(opts, repository.Where(query.Status.Eq(status)))
	}
	row, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, errorsx.ResourceNotFound("AI文件不存在或无权访问").WithCause(err)
	}
	return row, nil
}

// updateAiFileStatus 更新当前租户文件的回收站状态。
func (c *AiFileCase) updateAiFileStatus(ctx context.Context, id int64, from, to string) (*basev1.AiFile, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	row, err := c.ownedAiFile(ctx, id, from)
	if err != nil {
		return nil, err
	}
	updatedAt := time.Now()
	query := c.Query(ctx).AiFile
	_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID), query.TenantID.Eq(authInfo.TenantId), query.Status.Eq(from)).UpdateSimple(query.Status.Value(to), query.UpdatedBy.Value(authInfo.UserId), query.UpdatedAt.Value(updatedAt))
	if err != nil {
		return nil, err
	}
	row.Status = to
	row.UpdatedBy = authInfo.UserId
	row.UpdatedAt = updatedAt
	return aiFileDTO(row), nil
}

// aiFileDTO 转换 AI 文件响应并统一返回浏览器访问地址。
func aiFileDTO(row *models.AiFile) *basev1.AiFile {
	if row == nil {
		return nil
	}
	return &basev1.AiFile{
		Id:          row.ID,
		TenantId:    row.TenantID,
		SourceType:  row.SourceType,
		Status:      row.Status,
		FileName:    row.FileName,
		Extension:   row.Extension,
		MimeType:    row.MimeType,
		Size:        row.Size,
		LinkUrl:     publicFileURL(row.LinkURL),
		ContentHash: row.ContentHash,
		AccessMode:  row.AccessMode,
		CreatedAt:   timestamppb.New(row.CreatedAt),
		UpdatedAt:   timestamppb.New(row.UpdatedAt),
	}
}

// aiFileObjectPath 返回本地对象路径，外部生成链接不交给当前 OSS 删除。
func aiFileObjectPath(row *models.AiFile) string {
	if row == nil || strings.HasPrefix(row.LinkURL, "http://") || strings.HasPrefix(row.LinkURL, "https://") {
		return ""
	}
	objectPath, err := objectFilePath(path.Join(row.FileDirectory, row.SaveFileName))
	if err != nil {
		return ""
	}
	return objectPath
}
