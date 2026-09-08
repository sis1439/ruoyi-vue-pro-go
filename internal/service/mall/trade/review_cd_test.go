package trade

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	"go.uber.org/zap"
	"testing"
)

func TestReviewCanceledOrderMustNotAcknowledgePaid(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Exec("INSERT INTO trade_order(id,no,type,terminal,user_id,user_ip,status,product_count,pay_order_id,total_price,discount_price,delivery_price,adjust_price,pay_price,delivery_type,receiver_name,receiver_mobile,receiver_area_id,receiver_detail_address,refund_status,refund_price,coupon_price) VALUES(1,'review',0,10,1,'127.0.0.1',?,1,1,100,0,0,0,100,1,'review','13900000001',1,'review',0,0,0)", consts.TradeOrderStatusCanceled).Error; err != nil {
		t.Fatal(err)
	}
	// Nil pay service also proves the 'success' path never revalidates the payment record.
	p := NewPayOrderProcessor(query.Use(db), nil, zap.NewNop())
	result, err := p.Handle(context.Background(), &OrderHandleRequest{OrderID: 1, PayOrderID: 1})
	if err == nil && result != nil && result.Success {
		t.Fatal("canceled unpaid order is acknowledged as already paid; no payment validation or compensation")
	}
}
