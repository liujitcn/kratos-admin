package admin

import (
	"context"
	"errors"
	"testing"
)

// TestDeleteExpiredRowsRetriesAfterDeleteFailure 验证批量删除失败时不会吞掉错误或虚增数量。
func TestDeleteExpiredRowsRetriesAfterDeleteFailure(t *testing.T) {
	fetchCalls := 0
	removeCalls := 0
	deleted, err := deleteExpiredRows(context.Background(), func(context.Context) ([]int64, error) {
		fetchCalls++
		if fetchCalls == 1 {
			return []int64{1, 2}, nil
		}
		return nil, nil
	}, func(context.Context, []int64) error {
		removeCalls++
		return errors.New("delete failed")
	}, "删除过期日志")
	if err == nil {
		t.Fatal("delete failure should be returned")
	}
	if deleted != 0 {
		t.Fatalf("failed deletion changed count to %d", deleted)
	}
	if removeCalls != 1 {
		t.Fatalf("expected one delete attempt, got %d", removeCalls)
	}
}
