package trade

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"

	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type AppBrokerageWithdrawPageReqVO struct {
	pagination.PageParam
	Type       int      `form:"type"`
	Status     int      `form:"status,default=-1"`
	CreateTime []string `form:"createTime[]"`
}

type AppBrokerageWithdrawCreateReqVO struct {
	TransferChannelCode string `json:"transferChannelCode"`
	UserAccount         string `json:"userAccount"`
	UserName            string `json:"userName"`
	Type                int    `json:"type"`                           // 提现类型
	Price               *int   `json:"price" binding:"required,gte=0"` // 提现金额
	Name                string `json:"name"`                           // 真实姓名 (Bank/Alipay)
	Account             string `json:"account"`                        // 账号 (Bank/Alipay)
	BankName            string `json:"bankName"`                       // 银行名称 (Bank)
	BankAddress         string `json:"bankAddress"`                    // 开户地址 (Bank)
	QrCodeUrl           string `json:"qrCodeUrl"`                      // 收款码 (Wechat)
	Code                string `json:"code"`                           // 微信渠道需要
}

type AppBrokerageWithdrawRespVO struct {
	PayTransferID int64               `json:"payTransferId"`
	ID            int64               `json:"id"`
	UserID        int64               `json:"userId"`
	Price         int                 `json:"price"`
	FeePrice      int                 `json:"feePrice"`
	TotalPrice    int                 `json:"totalPrice"`
	Type          int                 `json:"type"`
	Name          string              `json:"name"`
	Account       string              `json:"account"`
	BankName      string              `json:"bankName"`
	Status        int                 `json:"status"`
	AuditReason   string              `json:"auditReason"`
	AuditTime     *types.JsonDateTime `json:"auditTime"`
	Remark        string              `json:"remark"`
	CreateTime    types.JsonDateTime  `json:"createTime"`
	TypeName      string              `json:"typeName"`
	StatusName    string              `json:"statusName"`

	// Wechat specific
	TransferChannelPackageInfo string `json:"transferChannelPackageInfo,omitempty"`
	TransferChannelMchId       string `json:"transferChannelMchId,omitempty"`
}
