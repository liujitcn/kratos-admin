package middleware

import (
	"context"
	"errors"
	"time"

	einoadk "github.com/cloudwego/eino/adk"
	componentsModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	einoModel "github.com/liujitcn/kratos-admin/backend/internal/agent/model"
)

// ModelRetryOptions 表示项目侧模型重试配置。
type ModelRetryOptions struct {
	// MaxRetries 最大重试次数。
	MaxRetries int
	// BackoffFunc 返回每次重试前的等待时间。
	BackoffFunc func(ctx context.Context, attempt int) time.Duration
}

// ModelRetryConfig 构造 ADK AgenticMessage 模型重试配置。
func ModelRetryConfig(options ModelRetryOptions) *einoadk.TypedModelRetryConfig[*schema.AgenticMessage] {
	// 未配置重试次数时交给 ADK 默认行为处理，避免无意放大模型请求量。
	if options.MaxRetries <= 0 {
		return nil
	}
	return &einoadk.TypedModelRetryConfig[*schema.AgenticMessage]{
		MaxRetries: options.MaxRetries,
		IsRetryAble: func(ctx context.Context, err error) bool {
			// 请求被调用方主动取消或超时时不重试，避免用户已经离开后继续消耗模型资源。
			return err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
		},
		BackoffFunc: options.BackoffFunc,
	}
}

// ResponsesServerToolHandler 将 Responses 服务端工具选项和调用记录挂到模型调用中。
type ResponsesServerToolHandler struct {
	*einoadk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]
}

// NewResponsesServerToolHandler 创建 Responses 服务端工具中间件。
func NewResponsesServerToolHandler() *ResponsesServerToolHandler {
	return &ResponsesServerToolHandler{
		TypedBaseChatModelAgentMiddleware: &einoadk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]{},
	}
}

// WrapModel 包装模型调用，注入 Responses 服务端工具选项。
func (h *ResponsesServerToolHandler) WrapModel(
	ctx context.Context,
	model componentsModel.BaseModel[*schema.AgenticMessage],
	_ *einoadk.TypedModelContext[*schema.AgenticMessage],
) (componentsModel.BaseModel[*schema.AgenticMessage], error) {
	// Responses 服务端工具需要附加到每一次模型调用，所以这里包一层 BaseModel。
	return &responsesServerToolModel{inner: model}, nil
}

// responsesServerToolModel 为底层模型统一注入 Responses 服务端工具。
type responsesServerToolModel struct {
	inner componentsModel.BaseModel[*schema.AgenticMessage]
}

// Generate 注入 Responses 服务端工具选项，并记录非流式响应中的服务端工具事件。
func (m *responsesServerToolModel) Generate(ctx context.Context, input []*schema.AgenticMessage, opts ...componentsModel.Option) (*schema.AgenticMessage, error) {
	return m.inner.Generate(ctx, input, append(opts, einoModel.ResponsesServerToolOptions()...)...)
}

// Stream 注入 Responses 服务端工具选项，并在流式 chunk 中捕获服务端工具事件。
func (m *responsesServerToolModel) Stream(ctx context.Context, input []*schema.AgenticMessage, opts ...componentsModel.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	reader, err := m.inner.Stream(ctx, input, append(opts, einoModel.ResponsesServerToolOptions()...)...)
	// 模型启动流失败或没有 reader 时，直接返回原始结果，后续统计由模型指标中间件负责。
	if err != nil || reader == nil {
		return reader, err
	}
	return reader, nil
}
