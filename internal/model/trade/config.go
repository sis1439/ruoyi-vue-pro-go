package trade

import (
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"gorm.io/datatypes"
)

// TradeConfig 交易中心 - 交易配置
// Table: trade_config
type TradeConfig struct {
	ID                          int64                       `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	AfterSaleRefundReasons      datatypes.JSONSlice[string] `gorm:"column:after_sale_refund_reasons;type:json;comment:售后的退款理由" json:"afterSaleRefundReasons"`
	AfterSaleReturnReasons      datatypes.JSONSlice[string] `gorm:"column:after_sale_return_reasons;type:json;comment:售后的退货理由" json:"afterSaleReturnReasons"`
	DeliveryExpressFreeEnabled  model.BitBool               `gorm:"column:delivery_express_free_enabled;default:(0);comment:是否启用全场包邮" json:"deliveryExpressFreeEnabled"`
	DeliveryExpressFreePrice    int                         `gorm:"column:delivery_express_free_price;default:0;comment:全场包邮的最小金额" json:"deliveryExpressFreePrice"`
	DeliveryPickUpEnabled       model.BitBool               `gorm:"column:delivery_pick_up_enabled;default:(0);comment:是否开启自提" json:"deliveryPickUpEnabled"`
	BrokerageWithdrawMinPrice   int                         `gorm:"column:brokerage_withdraw_min_price;default:0;comment:提现最低金额" json:"brokerageWithdrawMinPrice"`
	BrokerageWithdrawFeePercent int                         `gorm:"column:brokerage_withdraw_fee_percent;default:0;comment:提现手续费百分比" json:"brokerageWithdrawFeePercent"`
	BrokerageEnabled            model.BitBool               `gorm:"column:brokerage_enabled;default:(0);comment:是否开启分销" json:"brokerageEnabled"`
	BrokerageFrozenDays         int                         `gorm:"column:brokerage_frozen_days;default:0;comment:分销佣金冻结时间" json:"brokerageFrozenDays"`
	BrokerageFirstPercent       int                         `gorm:"column:brokerage_first_percent;default:0;comment:一级分销比例" json:"brokerageFirstPercent"`
	BrokerageSecondPercent      int                         `gorm:"column:brokerage_second_percent;default:0;comment:二级分销比例" json:"brokerageSecondPercent"`
	BrokerageEnabledCondition   int                         `gorm:"column:brokerage_enabled_condition;default:1;comment:分销资格启用条件 1:人人分销 2:仅指定用户" json:"brokerageEnabledCondition"`
	BrokerageBindMode           int                         `gorm:"column:brokerage_bind_mode;default:1;comment:分销关系绑定模式 1:首次绑定 2:注册绑定 3:覆盖绑定" json:"brokerageBindMode"`
	BrokeragePosterUrls         datatypes.JSONSlice[string] `gorm:"column:brokerage_poster_urls;type:json;comment:分销海报图地址数组" json:"brokeragePosterUrls"`
	BrokerageWithdrawTypes      types.ListFromCSV[int]      `gorm:"column:brokerage_withdraw_types;type:varchar(255);comment:提现方式" json:"brokerageWithdrawTypes"`
	model.TenantBaseDO
}

func (TradeConfig) TableName() string {
	return "trade_config"
}
