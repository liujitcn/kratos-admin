package sessionregistry

import (
	"sync"
	"testing"
	"time"

	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

// indexReadBarrierCache 固定两个注册请求读取相同索引的并发时序。
type indexReadBarrierCache struct {
	cache.Cache
	readers sync.WaitGroup
}

// Get 在两个注册请求都读完旧索引后才放行写入。
func (c *indexReadBarrierCache) Get(key string) (string, error) {
	value, err := c.Cache.Get(key)
	if key == indexKey {
		c.readers.Done()
		c.readers.Wait()
	}
	return value, err
}

// TestConcurrentRegistrationKeepsBothSessions 验证不同用户同时登录不会丢失在线索引。
func TestConcurrentRegistrationKeepsBothSessions(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err = store.Set(indexKey, "[]", time.Hour); err != nil {
		t.Fatal(err)
	}
	barrier := &indexReadBarrierCache{Cache: store}
	barrier.readers.Add(2)
	results := make(chan error, 2)
	for _, record := range []Record{{SessionID: "one", UserID: 1}, {SessionID: "two", UserID: 2}} {
		go func() { results <- Register(barrier, record) }()
	}
	for range 2 {
		if err = <-results; err != nil {
			t.Fatal(err)
		}
	}
	var ids []string
	ids, err = loadIndex(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("并发登录应保留两个会话，实际索引: %v", ids)
	}
}
