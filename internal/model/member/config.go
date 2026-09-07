package member

import (
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
)

// MemberConfig 会员配置
// Table: member_config
type MemberConfig struct {
	ID                        int64         `gorm:"primaryKey;autoIncrement;comment:编号" json:"id"`
	PointTradeDeductEnable    model.BitBool `gorm:"column:point_trade_deduct_enable;default:(0);comment:积分抵扣开关" json:"pointTradeDeductEnable"`          // 1-开启 0-关闭
	PointTradeDeductUnitPrice int           `gorm:"column:point_trade_deduct_unit_price;default:0;comment:积分抵扣单位价格" json:"pointTradeDeductUnitPrice"` // 积分抵扣，单位：分
	PointTradeDeductMaxPrice  int           `gorm:"column:point_trade_deduct_max_price;default:0;comment:积分抵扣最大值" json:"pointTradeDeductMaxPrice"`
	PointTradeGivePoint       int           `gorm:"column:point_trade_give_point;default:0;comment:1 元赠送多少分" json:"pointTradeGivePoint"`
	model.TenantBaseDO
}

func (MemberConfig) TableName() string {
	return "member_config"
}
