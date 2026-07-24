package biz

import (
	"context"
	"net/http"
	"strings"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	_const "github.com/liujitcn/kratos-admin/backend/pkg/const"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	transportSSE "github.com/liujitcn/kratos-admin/backend/pkg/sse"
	"github.com/liujitcn/kratos-admin/backend/service/system/admin/codegen"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SseCase 处理 SSE 公共业务。
type SseCase struct {
	authenticator authnEngine.Authenticator
	userToken     *authData.UserToken
	sse           *sseServer.Server
	streams       *transportSSE.Registry
}

// NewSseCase 创建 SSE 业务实例。
func NewSseCase(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	userToken *authData.UserToken,
	sse *sseServer.Server,
	streams *transportSSE.Registry,
	publisher *transportSSE.Publisher,
	progress *codegen.Manager,
) (*SseCase, error) {
	handler := &SseCase{
		authenticator: authenticator,
		userToken:     userToken,
		sse:           sse,
		streams:       streams,
	}
	cfg := ctx.GetConfig()
	// 未启用 HTTP 服务时，不创建 SSE HTTP 处理器。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return handler, nil
	}
	err := streams.Register(codegen.NewCodeGenSSEStream(progress))
	if err != nil {
		return nil, err
	}
	progress.SetPublisher(func(ctx context.Context, taskID string, task *systemadminv1.CodeGenTask) {
		publisher.TryPublishJSON(ctx, codegen.StreamID(taskID), codegen.SSEEventCodeGenProgress, task)
	})
	return handler, nil
}

// SubscribeSse 订阅 SSE 事件流。
func (h *SseCase) SubscribeSse(ctx context.Context, req *basev1.SubscribeSseRequest) (*emptypb.Empty, error) {
	if h == nil || h.sse == nil {
		return nil, errorsx.Internal("SSE服务未初始化")
	}
	w, ok := kratosHTTP.ResponseWriterFromServerContext(ctx)
	if !ok || w == nil {
		return nil, errorsx.InvalidArgument("SSE订阅仅支持HTTP访问")
	}
	var r *http.Request
	r, ok = kratosHTTP.RequestFromServerContext(ctx)
	if !ok || r == nil {
		return nil, errorsx.InvalidArgument("SSE订阅仅支持HTTP访问")
	}
	userToken, authorized := h.authorizeRequest(r)
	if !authorized {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil
	}

	streamID, found, err := h.streams.Resolve(req.GetStream(), req.GetChannelId(), userToken.UserId)
	if !found {
		return nil, errorsx.InvalidArgument("SSE流不支持")
	}
	if err != nil {
		return nil, err
	}
	h.sse.ServeStreamHTTP(w, r, sseServer.StreamID(streamID))
	return &emptypb.Empty{}, nil
}

// authorizeRequest 判断当前 SSE 请求是否来自后台登录用户，并返回用户信息。
func (h *SseCase) authorizeRequest(r *http.Request) (*authData.UserTokenPayload, bool) {
	userToken, err := h.authenticatorFromRequest(r)
	if err != nil || userToken == nil || userToken.UserId == 0 {
		return nil, false
	}
	// 工作台只面向管理后台，应用端用户令牌不能订阅后台刷新流。
	if userToken.RoleCode == _const.BASE_ROLE_CODE_USER || userToken.RoleCode == _const.BASE_ROLE_CODE_AUTHUSER {
		return nil, false
	}
	return userToken, true
}

// authenticatorFromRequest 从 SSE 请求中解析并校验后台登录用户。
func (h *SseCase) authenticatorFromRequest(r *http.Request) (*authData.UserTokenPayload, error) {
	token := r.Header.Get(authnEngine.HeaderAuthorize)
	if token == "" {
		return nil, errorsx.Unauthenticated("SSE访问令牌为空")
	}
	if len(token) >= len(authnEngine.BearerWord)+1 && strings.EqualFold(token[:len(authnEngine.BearerWord)+1], authnEngine.BearerWord+" ") {
		token = token[len(authnEngine.BearerWord)+1:]
	}

	authClaims, err := h.authenticator.AuthenticateToken(token)
	if err != nil || authClaims == nil {
		return nil, errorsx.Unauthenticated("SSE访问令牌无效").WithCause(err)
	}
	var userToken *authData.UserTokenPayload
	userToken, err = authData.NewUserTokenPayloadWithClaims(authClaims)
	if err != nil || userToken == nil || userToken.UserId == 0 {
		return nil, errorsx.Unauthenticated("SSE访问令牌无效").WithCause(err)
	}
	if !h.userToken.IsExistAccessToken(userToken.UserId) {
		return nil, errorsx.Unauthenticated("SSE访问令牌已失效")
	}
	return userToken, nil
}
