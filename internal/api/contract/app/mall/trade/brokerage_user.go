package trade

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"

	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

// AppBrokerageUserBindReqVO 绑定推广员 Request
type AppBrokerageUserBindReqVO struct {
	BindUserID model.FlexInt64 `json:"bindUserId" binding:"required"` // 推广员编号
}

// AppBrokerageUserChildSummaryPageReqVO 下级分销统计分页 Request
type AppBrokerageUserChildSummaryPageReqVO struct {
	pagination.PageParam
	Nickname      string `json:"nickname" form:"nickname"`                        // 下级昵称
	Level         int    `json:"level" form:"level" binding:"required,oneof=1 2"` // 分销层级
	Sorting       string `json:"sortingField" form:"sortingField.field"`
	SortingOrder  string `form:"sortingField.order"`
	LegacySorting string `form:"sortingField"` // 排序字段: brokerageTime, userCount, brokeragePrice
}

// AppBrokerageUserRankPageReqVO 分销用户排行分页 Request
type AppBrokerageUserRankPageReqVO struct {
	pagination.PageParam
	Times []string `json:"times" form:"times[]" binding:"required,len=2"` // 时间范围 [start, end]
}

// AppBrokerageUserRespVO 分销用户信息 Response
type AppBrokerageUserRespVO struct {
	BrokerageEnabled bool `json:"brokerageEnabled"` // 是否成为分销员
	BrokeragePrice   int  `json:"brokeragePrice"`   // 佣金金额
	FrozenPrice      int  `json:"frozenPrice"`      // 冻结金额
}

// AppBrokerageUserMySummaryRespVO 个人分销统计 Response
type AppBrokerageUserMySummaryRespVO struct {
	YesterdayPrice           int `json:"yesterdayPrice"`           // 昨日佣金
	WithdrawPrice            int `json:"withdrawPrice"`            // 提现佣金
	FirstBrokerageUserCount  int `json:"firstBrokerageUserCount"`  // 一级分销用户数量
	SecondBrokerageUserCount int `json:"secondBrokerageUserCount"` // 二级分销用户数量
	BrokeragePrice           int `json:"brokeragePrice"`           // 可用佣金
	FrozenPrice              int `json:"frozenPrice"`              // 冻结佣金
}

// AppBrokerageUserChildSummaryRespVO 下级分销统计 Response
type AppBrokerageUserChildSummaryRespVO struct {
	BrokerageOrderCount int                 `json:"brokerageOrderCount"`
	ID                  int64               `json:"id"`
	Nickname            string              `json:"nickname"`           // 用户昵称
	Avatar              string              `json:"avatar"`             // 用户头像
	BrokerageTime       *types.JsonDateTime `json:"brokerageTime"`      // 成为分销员时间
	BrokerageUserCount  int                 `json:"brokerageUserCount"` // 下级累计推广人数
	BrokeragePrice      int                 `json:"brokeragePrice"`     // 累计推广佣金
}

// AppBrokerageUserRankByUserCountRespVO 分销用户排行（基于用户量） Response
type AppBrokerageUserRankByUserCountRespVO struct {
	ID                 int64  `json:"id"`
	Nickname           string `json:"nickname"`           // 用户昵称
	Avatar             string `json:"avatar"`             // 用户头像
	BrokerageUserCount int    `json:"brokerageUserCount"` // 推广用户数量
}

// AppBrokerageUserRankByPriceRespVO 分销用户排行（基于佣金） Response
type AppBrokerageUserRankByPriceRespVO struct {
	ID             int64  `json:"id"`
	Nickname       string `json:"nickname"`       // 用户昵称
	Avatar         string `json:"avatar"`         // 用户头像
	BrokeragePrice int    `json:"brokeragePrice"` // 佣金金额
}
