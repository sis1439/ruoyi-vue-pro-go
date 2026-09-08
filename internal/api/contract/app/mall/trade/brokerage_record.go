package trade

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"

	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type AppBrokerageRecordPageReqVO struct {
	pagination.PageParam
	BizType    *int     `form:"bizType"`      // 业务类型
	Status     *int     `form:"status"`       // 状态
	CreateTime []string `form:"createTime[]"` // 创建时间
}

type AppBrokerageRecordRespVO struct {
	FinishTime  *types.JsonDateTime `json:"finishTime"`
	ID          int64               `json:"id"`
	UserID      int64               `json:"userId"`
	BizType     int                 `json:"bizType"`
	BizID       string              `json:"bizId"`
	Price       int                 `json:"price"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Status      int                 `json:"status"`
	Total       int                 `json:"total"` // TotalPrice in model
	CreateTime  types.JsonDateTime  `json:"createTime"`
	StatusName  string              `json:"statusName"`
}

type AppBrokerageProductPriceRespVO struct {
	Enabled           bool `json:"enabled"`
	BrokerageMinPrice int  `json:"brokerageMinPrice"`
	BrokerageMaxPrice int  `json:"brokerageMaxPrice"`
}
