package promotion

import (
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"

	"time"
)

// PromotionCoupon 优惠券 (对齐Java CouponDO)
type PromotionCoupon struct {
	ID                 int64                    `gorm:"primaryKey;autoIncrement;comment:优惠券编号"`
	TemplateID         int64                    `gorm:"column:template_id;type:bigint;not null;comment:模板编号"`
	Name               string                   `gorm:"column:name;type:varchar(64);not null;comment:优惠券名称"`
	Status             int                      `gorm:"column:status;type:int;not null;comment:状态"` // 1: Unused, 2: Used, 3: Expired
	UserID             int64                    `gorm:"column:user_id;type:bigint;not null;comment:用户编号"`
	TakeType           int                      `gorm:"column:take_type;type:int;not null;comment:领取类型"` // 1: Manually, 2: Register, 3: Admin
	UsePrice           int                      `gorm:"column:use_price;type:int;not null;comment:使用金额限制"`
	ValidStartTime     time.Time                `gorm:"column:valid_start_time;not null;comment:有效期开始时间"`
	ValidEndTime       time.Time                `gorm:"column:valid_end_time;not null;comment:有效期结束时间"`
	ProductScope       int                      `gorm:"column:product_scope;type:int;not null;comment:商品范围"` // 1: All, 2: Category, 3: Spu
	ProductScopeValues types.ListFromCSV[int64] `gorm:"column:product_scope_values;type:varchar(255);comment:商品范围值"`
	DiscountType       int                      `gorm:"column:discount_type;type:int;not null;comment:优惠类型"`
	DiscountPrice      int                      `gorm:"column:discount_price;type:int;comment:优惠金额"`
	DiscountPercent    int                      `gorm:"column:discount_percent;type:int;comment:折扣百分比"`
	DiscountLimitPrice *int                     `gorm:"column:discount_limit_price;type:int;comment:折扣上限"`
	UseOrderID         int64                    `gorm:"column:use_order_id;type:bigint;comment:使用订单编号"`
	UseTime            *time.Time               `gorm:"column:use_time;comment:使用时间"`
	model.TenantBaseDO
}

func (PromotionCoupon) TableName() string {
	return "promotion_coupon"
}
