package biz

import (
	"context"
	"fmt"
	"path"
	"strconv"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/report"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/id"
)

// AiReportCase 生成并保存 AI 助手对话报告。
type AiReportCase struct {
	*biz.BaseCase
	tx       data.Transaction
	fileCase *FileCase
}

// NewAiReportCase 创建 AI 助手报告业务实例。
func NewAiReportCase(baseCase *biz.BaseCase, tx data.Transaction, fileCase *FileCase) *AiReportCase {
	return &AiReportCase{BaseCase: baseCase, tx: tx, fileCase: fileCase}
}

// GenerateAiReport 将当前对话的 Markdown 生成 HTML/PDF，并以同一事务登记文件元数据。
func (c *AiReportCase) GenerateAiReport(ctx context.Context, req *basev1.GenerateAiReportRequest) (*basev1.GenerateAiReportResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var html []byte
	html, err = report.HTML(req.GetTitle(), req.GetContent())
	if err != nil {
		return nil, errorsx.Internal("生成报告 HTML 失败").WithCause(err)
	}
	var pdf []byte
	pdf, err = report.PDF(ctx, html)
	if err != nil {
		return nil, errorsx.Internal("生成报告 PDF 失败").WithCause(err)
	}
	if len(html) > maxFileContentBytes || len(pdf) > maxFileContentBytes {
		return nil, errorsx.InvalidArgument("生成的报告文件不能超过 20 MB")
	}
	directory := fmt.Sprintf("ai/reports/%d/%d", authInfo.TenantId, authInfo.UserId)
	filename := strconv.FormatInt(id.GenSnowflakeID(), 10)
	files := []*basev1.FileInfo{
		{Name: "分析报告-" + filename + ".html", Extname: "html"},
		{Name: "分析报告-" + filename + ".pdf", Extname: "pdf"},
	}
	contents := [][]byte{html, pdf}
	storedPaths := make([]string, 0, len(files))
	saved := false
	defer func() {
		if saved {
			return
		}
		for _, objectPath := range storedPaths {
			var cleanupErr error
			cleanupErr = c.OSS.DeleteFile(objectPath)
			if cleanupErr != nil {
				log.Warn(fmt.Sprintf("清理未完成的报告文件失败，路径 %s: %v", objectPath, cleanupErr))
			}
		}
	}()
	for index, file := range files {
		storedName := filename + "." + file.Extname
		objectPath := path.Join(directory, storedName)
		storedPaths = append(storedPaths, objectPath)
		_, err = c.OSS.UploadByByte(storedName, directory, contents[index])
		if err != nil {
			return nil, errorsx.Internal("保存报告文件失败").WithCause(err)
		}
		file.Url = publicFileURL(objectPath)
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		for index, file := range files {
			err = c.fileCase.recordUploadedFile(txCtx, authInfo.TenantId, file.Name, file.Extname, contents[index], storedPaths[index], basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_AUTHORIZED)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	saved = true
	return &basev1.GenerateAiReportResponse{Html: files[0], Pdf: files[1]}, nil
}
