package workflow

import (
	"context"
	"errors"
	"testing"
)

func TestDefaultEngineNotReady(t *testing.T) {
	// 未装配引擎时返回统一错误，不 panic。
	if _, err := Submit(context.Background(), SubmitParams{Code: "X"}); !errors.Is(err, errEngineNotReady) {
		t.Fatalf("期望未装配错误，实际: %v", err)
	}
	if _, err := StatusByBusiness(context.Background(), 1, "X", 1); !errors.Is(err, errEngineNotReady) {
		t.Fatalf("期望未装配错误，实际: %v", err)
	}
}

func TestSetDefaultEngine(t *testing.T) {
	engine := &stubEngine{}
	SetDefaultEngine(engine)
	defer SetDefaultEngine(nil)
	if DefaultEngine() != engine {
		t.Fatalf("默认引擎不一致")
	}
}

// stubEngine 测试桩引擎。
type stubEngine struct{}

// Submit 测试桩提交。
func (*stubEngine) Submit(context.Context, SubmitParams) (int64, error) { return 1, nil }

// StatusByBusiness 测试桩状态查询。
func (*stubEngine) StatusByBusiness(context.Context, int64, string, int64) (InstanceStatus, error) {
	return InstanceStatus{}, nil
}
