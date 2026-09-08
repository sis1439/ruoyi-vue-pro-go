package trade

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type AppAfterSaleCreateReq struct {
	OrderItemID      int64    `json:"orderItemId" binding:"required"`
	RefundPrice      *int     `json:"refundPrice" binding:"required,gte=0"`
	Way              int      `json:"way" binding:"required"` // 10: Refund only, 20: Return & Refund
	ApplyReason      string   `json:"applyReason" binding:"required"`
	ApplyDescription string   `json:"applyDescription"`
	ApplyPicURLs     []string `json:"applyPicUrls"`
}

type AppAfterSaleCancelReq struct {
	ID int64 `json:"id" binding:"required"`
}

type AppAfterSalePageReq struct {
	pagination.PageParam
	Statuses []int `form:"statuses" collection_format:"csv"`
}

type AppAfterSaleDeliveryReq struct {
	ID          int64  `json:"id" binding:"required"`
	LogisticsId *int64 `json:"logisticsId" binding:"required,gte=0"`
	LogisticsNo string `json:"logisticsNo" binding:"required"`
}

type TradeAfterSalePageReq struct {
	pagination.PageParam
	No          string   `form:"no"`
	UserID      *int64   `form:"userId"` // Admin can filter by userId, App sets it manually
	Status      *int     `form:"status"`
	OrderItemID *int64   `form:"orderItemId"`
	OrderNo     string   `form:"orderNo"`
	SpuName     string   `form:"spuName"`
	CreateTime  []string `form:"createTime[]"`
}

type TradeAfterSaleAgreeReq struct {
	ID int64 `json:"id" binding:"required"`
}

type TradeAfterSaleDisagreeReq struct {
	ID          int64  `json:"id" binding:"required"`
	AuditReason string `json:"auditReason" binding:"required"`
}

type TradeAfterSaleRefundReq struct {
	ID int64 `json:"id" binding:"required"`
}

type TradeAfterSaleRefuseReq struct {
	ID         int64  `json:"id" binding:"required"`
	RefuseMemo string `json:"refuseMemo" binding:"required"`
}

// AppAfterSaleResp App 售后 Response
type AppAfterSaleResp struct {
	ID               int64                            `json:"id"`
	No               string                           `json:"no"`
	Status           int                              `json:"status"`
	Way              int                              `json:"way"`
	Type             int                              `json:"type"`
	ApplyReason      string                           `json:"applyReason"`
	ApplyDescription string                           `json:"applyDescription"`
	ApplyPicURLs     []string                         `json:"applyPicUrls"`
	OrderID          int64                            `json:"orderId"`
	OrderNo          string                           `json:"orderNo"`
	OrderItemID      int64                            `json:"orderItemId"`
	SpuID            int64                            `json:"spuId"`
	SpuName          string                           `json:"spuName"`
	SkuID            int64                            `json:"skuId"`
	Properties       []product.ProductSkuPropertyResp `json:"properties"`
	PicURL           string                           `json:"picUrl"`
	Count            int                              `json:"count"`
	RefundPrice      int                              `json:"refundPrice"`
	AuditUserID      int64                            `json:"auditUserId"`
	AuditReason      string                           `json:"auditReason"`
	AuditTime        *types.JsonDateTime              `json:"auditTime"`
	LogisticsID      int64                            `json:"logisticsId"`
	LogisticsNo      string                           `json:"logisticsNo"`
	DeliveryTime     *types.JsonDateTime              `json:"deliveryTime"`
	ReceiveTime      *types.JsonDateTime              `json:"receiveTime"`
	ReceiveReason    string                           `json:"receiveReason"`
	RefundTime       *types.JsonDateTime              `json:"refundTime"`
	CreateTime       types.JsonDateTime               `json:"createTime"`
	UpdateTime       types.JsonDateTime               `json:"updateTime"`
}

// AfterSalePageItemResp 售后分页项 Response
type AfterSalePageItemResp struct {
	ID          int64       `json:"id"`
	No          string      `json:"no"`
	Status      int         `json:"status"`
	Type        int         `json:"type"`
	Way         int         `json:"way"`
	UserID      int64       `json:"userId"`
	User        interface{} `json:"user"` // MemberUserResp
	ApplyReason string      `json:"applyReason"`
	SpuName     string      `json:"spuName"`
	PicURL      string      `json:"picUrl"`
	Count       int         `json:"count"`
	RefundPrice int         `json:"refundPrice"`
	CreateTime  time.Time   `json:"createTime"`
}

// TradeAfterSaleDetailResp 售后详情 Response
type TradeAfterSaleDetailResp struct {
	ID               int64               `json:"id"`
	No               string              `json:"no"`
	Status           int                 `json:"status"`
	Type             int                 `json:"type"`
	Way              int                 `json:"way"`
	UserID           int64               `json:"userId"`
	User             interface{}         `json:"user"` // MemberUserResp
	ApplyReason      string              `json:"applyReason"`
	ApplyDescription string              `json:"applyDescription"`
	ApplyPicURLs     []string            `json:"applyPicURLs"`
	OrderID          int64               `json:"orderId"`
	OrderNo          string              `json:"orderNo"`
	OrderItemID      int64               `json:"orderItemId"`
	SpuID            int64               `json:"spuId"`
	SpuName          string              `json:"spuName"`
	SkuID            int64               `json:"skuId"`
	PicURL           string              `json:"picUrl"`
	Count            int                 `json:"count"`
	RefundPrice      int                 `json:"refundPrice"`
	AuditUserID      int64               `json:"auditUserId"`
	AuditReason      string              `json:"auditReason"`
	AuditTime        *time.Time          `json:"auditTime"`
	PayRefundID      int64               `json:"payRefundId"`
	RefundTime       *time.Time          `json:"refundTime"`
	LogisticsID      int64               `json:"logisticsId"`
	LogisticsNo      string              `json:"logisticsNo"`
	DeliveryTime     *time.Time          `json:"deliveryTime"`
	ReceiveTime      *time.Time          `json:"receiveTime"`
	ReceiveReason    string              `json:"receiveReason"`
	CreateTime       time.Time           `json:"createTime"`
	Order            *TradeOrderBase     `json:"order"`
	OrderItem        *AfterSaleOrderItem `json:"orderItem"`
	Logs             []AfterSaleLogResp  `json:"logs"`
}

type AfterSaleOrderItem struct {
	TradeOrderItemBase
}

type AfterSaleLogResp struct {
	ID           int64     `json:"id"`
	AfterSaleID  int64     `json:"afterSaleId"`
	BeforeStatus int       `json:"beforeStatus"`
	AfterStatus  int       `json:"afterStatus"`
	OperateType  int       `json:"operateType"`
	UserType     int       `json:"userType"`
	UserID       int64     `json:"userId"`
	Content      string    `json:"content"`
	CreateTime   time.Time `json:"createTime"`
}
