package admin

import (
	"context"
	"mime"
	"strconv"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

// RuntimeLogService 提供实时控制台和历史日志文件接口。
type RuntimeLogService struct {
	adminv1.UnimplementedRuntimeLogServiceServer
	runtimeLogCase *biz.RuntimeLogCase
}

// NewRuntimeLogService 创建运行日志服务。
func NewRuntimeLogService(runtimeLogCase *biz.RuntimeLogCase) *RuntimeLogService {
	return &RuntimeLogService{runtimeLogCase: runtimeLogCase}
}

// ListRuntimeLogFiles 查询可访问的历史日志文件。
func (s *RuntimeLogService) ListRuntimeLogFiles(_ context.Context, _ *adminv1.ListRuntimeLogFilesRequest) (*adminv1.ListRuntimeLogFilesResponse, error) {
	response, err := s.runtimeLogCase.ListRuntimeLogFiles()
	if err != nil {
		log.Error("ListRuntimeLogFiles", "error", err)
		return nil, errorsx.WrapInternal(err, "查询历史日志文件失败")
	}
	return response, nil
}

// ReadRuntimeLogFile 分页读取历史日志文件内容。
func (s *RuntimeLogService) ReadRuntimeLogFile(_ context.Context, req *adminv1.ReadRuntimeLogFileRequest) (*adminv1.ReadRuntimeLogFileResponse, error) {
	response, err := s.runtimeLogCase.ReadRuntimeLogFile(req)
	if err != nil {
		log.Error("ReadRuntimeLogFile", "error", err)
		return nil, errorsx.WrapInternal(err, "读取历史日志文件失败")
	}
	return response, nil
}

// OpenRuntimeConsole 创建当前用户专属实时控制台频道。
func (s *RuntimeLogService) OpenRuntimeConsole(ctx context.Context, req *adminv1.OpenRuntimeConsoleRequest) (*adminv1.OpenRuntimeConsoleResponse, error) {
	response, err := s.runtimeLogCase.OpenRuntimeConsole(ctx, req)
	if err != nil {
		log.Error("OpenRuntimeConsole", "error", err)
		return nil, errorsx.WrapInternal(err, "打开实时控制台失败")
	}
	return response, nil
}

// DownloadRuntimeLogFile 下载历史日志原文件。
func (s *RuntimeLogService) DownloadRuntimeLogFile(ctx context.Context, req *adminv1.DownloadRuntimeLogFileRequest) (*httpbody.HttpBody, error) {
	download, err := s.runtimeLogCase.DownloadRuntimeLogFile(req.GetFileId())
	if err != nil {
		log.Error("DownloadRuntimeLogFile", "error", err)
		return nil, errorsx.WrapInternal(err, "下载历史日志文件失败")
	}
	if serverTransport, ok := transport.FromServerContext(ctx); ok && serverTransport.ReplyHeader() != nil {
		serverTransport.ReplyHeader().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": download.Name}))
		serverTransport.ReplyHeader().Set("Content-Length", strconv.Itoa(len(download.Data)))
		serverTransport.ReplyHeader().Set("Cache-Control", "no-store")
		serverTransport.ReplyHeader().Set("X-Content-Type-Options", "nosniff")
	}
	return &httpbody.HttpBody{ContentType: download.ContentType, Data: download.Data}, nil
}
