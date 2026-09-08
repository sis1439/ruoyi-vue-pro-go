package trade

// TradeConfigSaveReq 交易配置 - 保存 Request (对齐 Java: TradeConfigSaveReqVO)
type TradeConfigSaveReq struct {
	AfterSaleRefundReasons      []string `json:"afterSaleRefundReasons" binding:"required"` // 售后的退款理由
	AfterSaleReturnReasons      []string `json:"afterSaleReturnReasons" binding:"required"` // 售后的退货理由
	DeliveryExpressFreeEnabled  *bool    `json:"deliveryExpressFreeEnabled"`                // 是否启用全场包邮
	DeliveryExpressFreePrice    *int     `json:"deliveryExpressFreePrice"`                  // 全场包邮的最小金额
	DeliveryPickUpEnabled       *bool    `json:"deliveryPickUpEnabled"`                     // 是否开启自提
	BrokerageWithdrawMinPrice   *int     `json:"brokerageWithdrawMinPrice"`                 // 提现最低金额
	BrokerageWithdrawFeePercent *int     `json:"brokerageWithdrawFeePercent"`               // 提现手续费百分比
	BrokerageEnabled            *bool    `json:"brokerageEnabled"`                          // 是否开启分销
	BrokerageFrozenDays         *int     `json:"brokerageFrozenDays"`                       // 分销佣金冻结时间（天）
	BrokerageFirstPercent       *int     `json:"brokerageFirstPercent"`                     // 一级分销比例
	BrokerageSecondPercent      *int     `json:"brokerageSecondPercent"`                    // 二级分销比例
	BrokerageEnabledCondition   *int     `json:"brokerageEnabledCondition"`                 // 分销资格启用条件
	BrokerageBindMode           *int     `json:"brokerageBindMode"`                         // 分销关系绑定模式
	BrokeragePosterUrls         []string `json:"brokeragePosterUrls"`                       // 分销海报图
	BrokerageWithdrawTypes      []int    `json:"brokerageWithdrawTypes"`                    // 提现方式
}

// TradeConfigResp 交易配置 Response (对齐 Java: TradeConfigRespVO)
type TradeConfigResp struct {
	ID                          int64    `json:"id"`
	AppID                       int64    `json:"appId"`
	AfterSaleDeadlineDays       int      `json:"afterSaleDeadlineDays"`  // 售后期限(天)
	PayTimeoutMinutes           int      `json:"payTimeoutMinutes"`      // 支付超时(分钟)
	AutoReceiveDays             int      `json:"autoReceiveDays"`        // 自动收货(天)
	AutoCommentDays             int      `json:"autoCommentDays"`        // 自动好评(天)
	AfterSaleRefundReasons      []string `json:"afterSaleRefundReasons"` // 售后的退款理由
	AfterSaleReturnReasons      []string `json:"afterSaleReturnReasons"` // 售后的退货理由
	DeliveryExpressFreeEnabled  bool     `json:"deliveryExpressFreeEnabled"`
	DeliveryExpressFreePrice    int      `json:"deliveryExpressFreePrice"`
	DeliveryPickUpEnabled       bool     `json:"deliveryPickUpEnabled"`
	BrokerageWithdrawMinPrice   int      `json:"brokerageWithdrawMinPrice"`
	BrokerageWithdrawFeePercent int      `json:"brokerageWithdrawFeePercent"`
	BrokerageEnabled            bool     `json:"brokerageEnabled"`
	BrokerageFrozenDays         int      `json:"brokerageFrozenDays"`
	BrokerageFirstPercent       int      `json:"brokerageFirstPercent"`
	BrokerageSecondPercent      int      `json:"brokerageSecondPercent"`
	BrokerageEnabledCondition   int      `json:"brokerageEnabledCondition"`
	BrokerageBindMode           int      `json:"brokerageBindMode"`
	BrokeragePosterUrls         []string `json:"brokeragePosterUrls"`
	BrokerageWithdrawTypes      []int    `json:"brokerageWithdrawTypes"`
	TencentLbsKey               string   `json:"tencentLbsKey"`
}

// AppTradeConfigResp App 交易配置 Response (对齐 Java: AppTradeConfigRespVO)
type AppTradeConfigResp struct {
	AfterSaleRefundReasons    []string `json:"afterSaleRefundReasons"` // 售后的退款理由
	AfterSaleReturnReasons    []string `json:"afterSaleReturnReasons"` // 售后的退货理由
	DeliveryPickUpEnabled     bool     `json:"deliveryPickUpEnabled"`
	BrokerageWithdrawMinPrice int      `json:"brokerageWithdrawMinPrice"`
	BrokerageFrozenDays       int      `json:"brokerageFrozenDays"`
	BrokeragePosterUrls       []string `json:"brokeragePosterUrls"`
	BrokerageWithdrawTypes    []int    `json:"brokerageWithdrawTypes"`
	TencentLbsKey             string   `json:"tencentLbsKey"`
}
