package base

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	stdhttp "net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/go-utils/id"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the Nms package it is being compiled against.
var _ = new(context.Context)

const _ = http.SupportPackageIsVersion3

const OperationFileServiceDownloadFile = "/base.v1.FileService/DownloadFile"
const OperationFileServiceMultiUploadFile = "/base.v1.FileService/MultiUploadFile"
const OperationFileServiceUploadFile = "/base.v1.FileService/UploadFile"

const maxMultipartUploadBytes int64 = 20 << 20

type FileServiceHTTPServer interface {
	// DownloadFile 下载文件
	DownloadFile(context.Context, *basev1.DownloadFileRequest) (*wrapperspb.BytesValue, error)
	// MultiUploadFile 多个文件上传
	MultiUploadFile(context.Context, *basev1.MultiUploadFileRequest) (*basev1.MultiUploadFileResponse, error)
	// UploadFile 单个文件上传
	UploadFile(context.Context, *basev1.UploadFileRequest) (*basev1.FileInfo, error)
}

func RegisterFileServiceHTTPServer(s *http.Server, srv FileServiceHTTPServer) {
	r := s.Route("/")
	r.POST("/api/v1/base/file/multi", _FileService_MultiUploadFile0_HTTP_Handler(srv))
	r.POST("/api/v1/base/file", _FileService_UploadFile0_HTTP_Handler(srv))
	r.GET("/api/v1/base/file", _FileService_DownloadFile0_HTTP_Handler(srv))
}

// _FileService_MultiUploadFile0_HTTP_Handler 处理 multipart 批量文件上传请求。
func _FileService_MultiUploadFile0_HTTP_Handler(srv FileServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in basev1.MultiUploadFileRequest
		http.SetOperation(ctx, OperationFileServiceMultiUploadFile)
		r := ctx.Request()
		r.Body = stdhttp.MaxBytesReader(ctx.Response(), r.Body, maxMultipartUploadBytes)
		var err error
		if r.MultipartForm == nil {
			err = r.ParseMultipartForm(32 << 20)
			if err != nil {
				return errorsx.InvalidArgument("上传文件格式错误").WithCause(err)
			}
		}
		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			for _, headers := range r.MultipartForm.File {
				for _, header := range headers {
					var formFile multipart.File
					formFile, err = header.Open()
					if err != nil {
						return errorsx.InvalidArgument("上传文件打开失败").WithCause(err)
					}
					contentType := header.Header.Get("Content-Type")
					var uploadFileInfo *basev1.UploadFileInfo
					uploadFileInfo, err = convertUploadFileInfo(formFile, r.FormValue("fileType"), contentType, header.Filename)
					if err != nil {
						return errorsx.InvalidArgument("上传文件解析失败").WithCause(err)
					}
					in.Files = append(in.Files, uploadFileInfo)
				}
			}
		}
		h := ctx.Middleware(func(requestCtx context.Context, req interface{}) (interface{}, error) {
			return srv.MultiUploadFile(requestCtx, req.(*basev1.MultiUploadFileRequest))
		})
		var out interface{}
		out, err = h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*basev1.MultiUploadFileResponse)
		return ctx.Result(200, reply)
	}
}

// _FileService_UploadFile0_HTTP_Handler 处理 multipart 单文件上传请求。
func _FileService_UploadFile0_HTTP_Handler(srv FileServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in basev1.UploadFileRequest
		http.SetOperation(ctx, OperationFileServiceUploadFile)
		r := ctx.Request()
		r.Body = stdhttp.MaxBytesReader(ctx.Response(), r.Body, maxMultipartUploadBytes)
		formFile, header, err := r.FormFile("file")
		if err != nil {
			return errorsx.InvalidArgument("未上传文件").WithCause(err)
		}
		contentType := header.Header.Get("Content-Type")
		var uploadFileInfo *basev1.UploadFileInfo
		uploadFileInfo, err = convertUploadFileInfo(formFile, r.FormValue("fileType"), contentType, header.Filename)
		if err != nil {
			return errorsx.InvalidArgument("上传文件解析失败").WithCause(err)
		}
		in.File = uploadFileInfo
		h := ctx.Middleware(func(requestCtx context.Context, req interface{}) (interface{}, error) {
			return srv.UploadFile(requestCtx, req.(*basev1.UploadFileRequest))
		})
		var out interface{}
		out, err = h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*basev1.FileInfo)
		return ctx.Result(200, reply)
	}
}

func _FileService_DownloadFile0_HTTP_Handler(srv FileServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in basev1.DownloadFileRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationFileServiceDownloadFile)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DownloadFile(ctx, req.(*basev1.DownloadFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*wrapperspb.BytesValue)
		filename := in.GetName()
		if len(filename) == 0 {
			filename = path.Base(in.GetPath())
		}
		filename = path.Base(strings.ReplaceAll(strings.ReplaceAll(filename, "\\", "/"), "\r", ""))
		filename = strings.ReplaceAll(filename, "\n", "")
		contentDisposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
		if contentDisposition == "" {
			contentDisposition = "attachment"
		}
		// 设置响应头，支持文件下载
		ctx.Response().Header().Set("Content-Type", "application/octet-stream")
		ctx.Response().Header().Set("Content-Disposition", contentDisposition)
		ctx.Response().Header().Set("Content-Length", strconv.Itoa(len(reply.Value)))

		// 直接写入二进制数据
		_, err = ctx.Response().Write(reply.Value)
		if err != nil {
			return err
		}

		return nil
	}
}

func convertUploadFileInfo(multipartFile multipart.File, fileType, contentType, fileName string) (*basev1.UploadFileInfo, error) {
	defer func(multipartFile multipart.File) {
		err := multipartFile.Close()
		if err != nil {
			log.Error(fmt.Sprintf("form file close err: %v", err))
		}
	}(multipartFile)

	limited := io.LimitReader(multipartFile, maxMultipartUploadBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxMultipartUploadBytes {
		return nil, errorsx.InvalidArgument("文件内容不能超过 20 MB")
	}
	filePath := fmt.Sprintf("/%s", _const.BASE_PATH)
	if len(fileType) != 0 {
		filePath += "/" + fileType
	}
	extname := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	contentTypes := strings.Split(contentType, "/")
	if len(contentTypes) != 2 {
		filePath += "/files"
	} else if extname == "" {
		extname = strings.ToLower(contentTypes[1])
	} else {
		switch contentTypes[0] {
		case "image":
			filePath += "/images"
			break
		case "video":
			filePath += "/videos"
			break
		case "audio":
			filePath += "/audios"
			break
		case "application", "text":
			filePath += "/docs"
			break
		default:
			filePath += "/files"
			break
		}
	}

	datePath := time.Now().Format("2006/01/02")
	return &basev1.UploadFileInfo{
		Name:    fmt.Sprintf("%d.%s", id.GenSnowflakeID(), extname),
		Extname: extname,
		Path:    fmt.Sprintf("%s/%s", filePath, datePath),
		Content: content,
	}, nil
}
