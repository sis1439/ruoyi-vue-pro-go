package brokerage

import (
	"github.com/wxlbd/ruoyi-mall-go/internal/model"

	"time"
)

// BrokerageUser 分销用户
type BrokerageUser struct {
	ID               int64         `gorm:"primaryKey;autoIncrement;comment:用户编号"`
	BindUserID       int64         `gorm:"column:bind_user_id;default:0;comment:推广员编号"`
	BindUserTime     *time.Time    `gorm:"column:bind_user_time;comment:推广员绑定时间"`
	BrokerageEnabled model.BitBool `gorm:"column:brokerage_enabled;type:tinyint(1);default:(0);comment:是否有分销资格"`
	BrokerageTime    *time.Time    `gorm:"column:brokerage_time;comment:成为分销员时间"`
	BrokeragePrice   int           `gorm:"column:brokerage_price;default:0;comment:可用佣金"`
	FrozenPrice      int           `gorm:"column:frozen_price;default:0;comment:冻结佣金"`
	model.TenantBaseDO
}

func (BrokerageUser) TableName() string {
	return "trade_brokerage_user"
}
