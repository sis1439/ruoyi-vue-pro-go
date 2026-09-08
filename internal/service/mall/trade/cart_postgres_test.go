package trade

import (
	"context"
	model "github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"testing"
)

func TestPostgresCartEmptyAndIsolatedCount(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	s := NewCartService(query.Use(db), nil, nil)
	a := pkgcontext.WithTenant(context.Background(), 1)
	b := pkgcontext.WithTenant(context.Background(), 2)
	for _, ctx := range []context.Context{a, b} {
		count, err := s.GetCartCount(ctx, 17)
		if err != nil || count != 0 {
			t.Fatalf("empty cart: count=%d err=%v", count, err)
		}
	}
	cart := &model.Cart{UserID: 17, SkuID: 1, SpuID: 1, Count: 2, Selected: true}
	if err := db.WithContext(a).Create(cart).Error; err != nil {
		t.Fatal(err)
	}
	if count, err := s.GetCartCount(a, 17); err != nil || count != 2 {
		t.Fatalf("owner count: %d %v", count, err)
	}
	if count, err := s.GetCartCount(b, 17); err != nil || count != 0 {
		t.Fatalf("cross tenant count: %d %v", count, err)
	}
	if count, err := s.GetCartCount(a, 18); err != nil || count != 0 {
		t.Fatalf("other user count: %d %v", count, err)
	}
	if err := s.DeleteCart(a, 17, []int64{cart.ID}); err != nil {
		t.Fatal(err)
	}
	if count, err := s.GetCartCount(a, 17); err != nil || count != 0 {
		t.Fatalf("deleted cart: %d %v", count, err)
	}
}
