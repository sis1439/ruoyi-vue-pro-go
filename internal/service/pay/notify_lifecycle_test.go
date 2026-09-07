package pay

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// Tests lifetime/tenant isolation only; the legacy merchant payload/protocol remains batch C.
func TestNotifyJobWaitsForTenantWork(t *testing.T) {
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("T09_REDIS_ADDR required")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 3})
	defer rdb.Close()
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	var delivered atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Millisecond)
		delivered.Add(1)
		_, _ = w.Write([]byte("SUCCESS"))
	}))
	defer server.Close()
	a := pkgcontext.WithTenant(context.Background(), time.Now().UnixNano())
	b := pkgcontext.WithTenant(context.Background(), 2)
	now := time.Now().Add(-time.Second)
	task := &pay.PayNotifyTask{Type: PayNotifyTypeOrder, Status: PayNotifyStatusWaiting, NotifyURL: server.URL, NextNotifyTime: &now, MaxNotifyTimes: 3}
	if err := db.WithContext(a).Create(task).Error; err != nil {
		t.Fatal(err)
	}
	s := NewPayNotifyService(query.Use(db), zap.NewNop(), rdb)
	if n, err := s.ExecuteNotify(b); err != nil || n != 0 {
		t.Fatalf("other tenant: %d %v", n, err)
	}
	if n, err := s.ExecuteNotify(a); err != nil || n != 1 {
		t.Fatalf("own task: %d %v", n, err)
	}
	if delivered.Load() != 1 {
		t.Fatal("job returned before child work completed")
	}
	var saved pay.PayNotifyTask
	if err := db.WithContext(a).First(&saved, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != PayNotifyStatusSuccess || saved.NotifyTimes != 1 {
		t.Fatalf("job returned before result persistence: %+v", saved)
	}
	if n, err := s.ExecuteNotify(a); err != nil || n != 0 {
		t.Fatalf("completed replay: %d %v", n, err)
	}
}
