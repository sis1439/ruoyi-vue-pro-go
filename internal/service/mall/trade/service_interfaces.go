package trade

import (
	"context"

	product2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	trade2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	payModel "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	product "github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
)

// PayOrderServiceAPI 定义支付订单服务接口
type PayOrderServiceAPI interface {
	GetOrder(ctx context.Context, id int64) (*payModel.PayOrder, error)
	UpdatePayOrderPrice(ctx context.Context, id int64, payPrice int) error
	CreateOrder(ctx context.Context, reqDTO *pay.PayOrderCreateReq) (int64, error)
}

// PayRefundServiceAPI 定义退款服务接口
type PayRefundServiceAPI interface {
	CreateRefund(ctx context.Context, reqDTO *pay.PayRefundCreateReq) (int64, error)
	GetRefund(ctx context.Context, id int64) (*payModel.PayRefund, error)
}

// PayAppServiceAPI 定义支付应用服务接口
type PayAppServiceAPI interface {
	GetApp(ctx context.Context, id int64) (*payModel.PayApp, error)
	GetAppByAppKey(ctx context.Context, appKey string) (*payModel.PayApp, error)
}

// ProductCommentServiceAPI 定义商品评价服务接口
type ProductCommentServiceAPI interface {
	CreateAppComment(ctx context.Context, userId int64, req *product2.AppProductCommentCreateReq) (*product.ProductComment, error)
}

// ProductSkuServiceAPI 定义商品 SKU 服务接口
type ProductSkuServiceAPI interface {
	GetSku(ctx context.Context, id int64) (*product.ProductSku, error)
	GetSkuList(ctx context.Context, ids []int64) ([]*product2.ProductSkuResp, error)
	UpdateSkuStock(ctx context.Context, updateReq *product2.ProductSkuUpdateStockReq) error
}

// CouponUserServiceAPI 定义优惠券服务接口
type CouponUserServiceAPI interface {
	UseCoupon(ctx context.Context, userId int64, id int64, orderId int64) error
	ReturnCoupon(ctx context.Context, userId int64, id int64) error
	ReturnCouponForOrder(ctx context.Context, userId, id, orderId int64) error
	GetCoupon(ctx context.Context, userId int64, id int64) (*promotion.PromotionCoupon, error)
}

// MemberUserServiceAPI 定义会员服务接口
type MemberUserServiceAPI interface {
	GetUser(ctx context.Context, id int64) (*member.MemberUser, error)
	UpdateUserPoint(ctx context.Context, id int64, point int) bool
}

// TradeConfigServiceAPI 定义交易配置服务接口
type TradeConfigServiceAPI interface {
	GetTradeConfig(ctx context.Context) (*trade2.TradeConfigResp, error)
}

// TradeNoRedisDAOAPI 定义 Redis 编号生成接口
type TradeNoRedisDAOAPI interface {
	Generate(ctx context.Context, prefix string) (string, error)
}
