package projectaccess

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// TestLifecycleChange 验证业务锁包裹完整变更、项目去重排序以及拒绝后不执行写入。
func TestLifecycleChange(t *testing.T) {
	lifecycle := NewLifecycle()
	events := []string{}
	err := lifecycle.Register("scp", func(ctx context.Context, keys []ProjectKey, next func(context.Context) error) error {
		if !reflect.DeepEqual(keys, []ProjectKey{{TenantID: 1, ProjectID: 101}, {TenantID: 2, ProjectID: 201}}) {
			t.Fatalf("锁顺序错误: %v", keys)
		}
		events = append(events, "locked")
		err := next(ctx)
		events = append(events, "released")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	err = lifecycle.Change(context.Background(), []ProjectKey{{TenantID: 2, ProjectID: 201}, {TenantID: 1, ProjectID: 101}, {TenantID: 1, ProjectID: 101}}, func(context.Context) error { events = append(events, "committed"); return nil })
	if err != nil || !reflect.DeepEqual(events, []string{"locked", "committed", "released"}) {
		t.Fatalf("事务提交前释放了业务锁: %v %v", events, err)
	}
	denied := errors.New("同步尚未停止")
	err = lifecycle.Register("block", func(context.Context, []ProjectKey, func(context.Context) error) error { return denied })
	if err != nil {
		t.Fatal(err)
	}
	err = lifecycle.Change(context.Background(), nil, func(context.Context) error { t.Fatal("拒绝后不应调用变更"); return nil })
	if !errors.Is(err, denied) {
		t.Fatalf("未保留业务拒绝: %v", err)
	}
}
