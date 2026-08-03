package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/v1/dto"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
)

var _ = new(context.Context)

const _ = kratosHTTP.SupportPackageIsVersion3

const OperationAiMessageServiceDeleteAiMessage = "/base.v1.AiMessageService/DeleteAiMessage"
const OperationAiMessageServiceRegenerateAiMessage = "/base.v1.AiMessageService/RegenerateAiMessage"
const OperationAiMessageServiceRetryAiUserMessage = "/base.v1.AiMessageService/RetryAiUserMessage"
const OperationAiMessageServiceSendAiMessage = "/base.v1.AiMessageService/SendAiMessage"
const OperationAiMessageServiceUpdateAiMessage = "/base.v1.AiMessageService/UpdateAiMessage"

// AiMessageServiceHTTPServer 定义 AI 助手消息发送 HTTP 服务。
type AiMessageServiceHTTPServer interface {
	// DeleteAiMessage 删除 AI 助手消息。
	DeleteAiMessage(context.Context, *basev1.DeleteAiMessageRequest) (*basev1.DeleteAiMessageResponse, error)
	// RegenerateAiMessage 重新生成 AI 助手输出。
	RegenerateAiMessage(context.Context, *basev1.RegenerateAiMessageRequest) (*basev1.SendAiMessageResponse, error)
	// RetryAiUserMessage 重试失败的 AI 助手消息。
	RetryAiUserMessage(context.Context, *basev1.RetryAiUserMessageRequest) (*basev1.SendAiMessageResponse, error)
	// StreamAiMessage 流式发送 AI 助手消息。
	StreamAiMessage(context.Context, *basev1.SendAiMessageRequest, dto.AiStreamEmitter) error
	// UpdateAiMessage 更新 AI 助手消息并重新生成输出。
	UpdateAiMessage(context.Context, *basev1.UpdateAiMessageRequest) (*basev1.SendAiMessageResponse, error)
}

// RegisterAiMessageServiceHTTPServer 注册 AI 助手消息发送 HTTP 接口。
func RegisterAiMessageServiceHTTPServer(s *kratosHTTP.Server, srv AiMessageServiceHTTPServer) {
	r := s.Route("/")
	r.POST("/api/v1/base/ai/session/{session_id}/message", _AiMessageService_SendAiMessage0_HTTP_Handler(srv))
	r.DELETE("/api/v1/base/ai/session/{session_id}/message/{message_id}", _AiMessageService_DeleteAiMessage0_HTTP_Handler(srv))
	r.PUT("/api/v1/base/ai/session/{session_id}/message/{message_id}", _AiMessageService_UpdateAiMessage0_HTTP_Handler(srv))
	r.POST("/api/v1/base/ai/session/{session_id}/message/{message_id}/retry", _AiMessageService_RetryAiUserMessage0_HTTP_Handler(srv))
	r.POST("/api/v1/base/ai/session/{session_id}/message/{message_id}/regeneration", _AiMessageService_RegenerateAiMessage0_HTTP_Handler(srv))
}

// EmitAiStream 写入单条 AI 助手 SSE 事件。
func (e *aiStreamEmitter) EmitAiStream(event dto.AiStreamEvent, payload dto.AiStreamPayload) error {
	if e == nil || e.writer == nil {
		return fmt.Errorf("AI助手流式响应写入器未初始化")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if _, err = fmt.Fprintf(e.writer, "event: %s\n", event); err != nil {
		return err
	}
	if _, err = fmt.Fprintf(e.writer, "data: %s\n\n", data); err != nil {
		return err
	}
	if e.flusher != nil {
		e.flusher.Flush()
	}
	return nil
}

// _AiMessageService_SendAiMessage0_HTTP_Handler 处理 AI 助手消息发送流式请求。
func _AiMessageService_SendAiMessage0_HTTP_Handler(srv AiMessageServiceHTTPServer) func(ctx kratosHTTP.Context) error {
	return func(ctx kratosHTTP.Context) error {
		var in basev1.SendAiMessageRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		kratosHTTP.SetOperation(ctx, OperationAiMessageServiceSendAiMessage)
		response := ctx.Response()
		flusher, ok := response.(http.Flusher)
		if !ok {
			return errorsx.Internal("流式响应不支持")
		}
		emitter := &aiStreamEmitter{
			writer:  response,
			flusher: flusher,
		}
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			header := response.Header()
			header.Set("Content-Type", "text/event-stream; charset=utf-8")
			header.Set("Cache-Control", "no-cache")
			header.Set("Connection", "keep-alive")
			header.Set("X-Accel-Buffering", "no")
			return nil, srv.StreamAiMessage(ctx, req.(*basev1.SendAiMessageRequest), emitter)
		})
		_, err := h(ctx, &in)
		return err
	}
}

// _AiMessageService_DeleteAiMessage0_HTTP_Handler 处理 AI 助手消息删除请求。
func _AiMessageService_DeleteAiMessage0_HTTP_Handler(srv AiMessageServiceHTTPServer) func(ctx kratosHTTP.Context) error {
	return func(ctx kratosHTTP.Context) error {
		var in basev1.DeleteAiMessageRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		kratosHTTP.SetOperation(ctx, OperationAiMessageServiceDeleteAiMessage)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DeleteAiMessage(ctx, req.(*basev1.DeleteAiMessageRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out.(*basev1.DeleteAiMessageResponse))
	}
}

// _AiMessageService_UpdateAiMessage0_HTTP_Handler 处理 AI 助手消息文本更新请求。
func _AiMessageService_UpdateAiMessage0_HTTP_Handler(srv AiMessageServiceHTTPServer) func(ctx kratosHTTP.Context) error {
	return func(ctx kratosHTTP.Context) error {
		var in basev1.UpdateAiMessageRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		kratosHTTP.SetOperation(ctx, OperationAiMessageServiceUpdateAiMessage)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.UpdateAiMessage(ctx, req.(*basev1.UpdateAiMessageRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out.(*basev1.SendAiMessageResponse))
	}
}

// _AiMessageService_RetryAiUserMessage0_HTTP_Handler 处理失败消息重试请求。
func _AiMessageService_RetryAiUserMessage0_HTTP_Handler(srv AiMessageServiceHTTPServer) func(ctx kratosHTTP.Context) error {
	return func(ctx kratosHTTP.Context) error {
		var in basev1.RetryAiUserMessageRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		kratosHTTP.SetOperation(ctx, OperationAiMessageServiceRetryAiUserMessage)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.RetryAiUserMessage(ctx, req.(*basev1.RetryAiUserMessageRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out.(*basev1.SendAiMessageResponse))
	}
}

type aiStreamEmitter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	mutex   sync.Mutex
}

// _AiMessageService_RegenerateAiMessage0_HTTP_Handler 处理 AI 助手输出重新生成请求。
func _AiMessageService_RegenerateAiMessage0_HTTP_Handler(srv AiMessageServiceHTTPServer) func(ctx kratosHTTP.Context) error {
	return func(ctx kratosHTTP.Context) error {
		var in basev1.RegenerateAiMessageRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		kratosHTTP.SetOperation(ctx, OperationAiMessageServiceRegenerateAiMessage)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.RegenerateAiMessage(ctx, req.(*basev1.RegenerateAiMessageRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out.(*basev1.SendAiMessageResponse))
	}
}
