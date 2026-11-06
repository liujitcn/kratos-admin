package base

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/transport"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

// TestAiMessageApprovalHTTPRoute 检查 SSE 适配器保留工具确认的 HTTP 入口和操作标识。
func TestAiMessageApprovalHTTPRoute(t *testing.T) {
	service := &approvalHTTPService{}
	server := kratosHTTP.NewServer()
	RegisterAiMessageServiceHTTPServer(server, service)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/base/ai/session/2/message/5/approval", strings.NewReader(`{"approval_id":"21f3bdd8-a446-4e7d-b2b1-af2318409343","approved":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("确认请求返回 %d: %s", response.Code, response.Body.String())
	}
	if service.request.GetSessionId() != "2" || service.request.GetMessageId() != "5" || service.request.GetApprovalId() != "21f3bdd8-a446-4e7d-b2b1-af2318409343" || !service.request.GetApproved() {
		t.Fatalf("确认参数未正确绑定: %v", service.request)
	}
	if service.operation != basev1.OperationAiMessageServiceDecideAiToolApproval {
		t.Fatalf("确认操作标识不正确: %q", service.operation)
	}
}

type approvalHTTPService struct {
	AiMessageServiceHTTPServer
	request   *basev1.DecideAiToolApprovalRequest
	operation string
}

// DecideAiToolApproval 捕获 HTTP 层绑定的确认参数和权限操作标识。
func (s *approvalHTTPService) DecideAiToolApproval(ctx context.Context, request *basev1.DecideAiToolApprovalRequest) (*basev1.SendAiMessageResponse, error) {
	s.request = request
	if tr, ok := transport.FromServerContext(ctx); ok {
		s.operation = tr.Operation()
	}
	return &basev1.SendAiMessageResponse{}, nil
}
