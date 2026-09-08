package trade

import (
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"net/url"
	"strings"

	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type AppBrokerageWithdrawPageReqVO struct {
	pagination.PageParam
	Type       int      `form:"type"`
	Status     int      `form:"status,default=-1"`
	CreateTime []string `form:"createTime[]"`
}

type AppBrokerageWithdrawCreateReqVO struct {
	TransferChannelCode string  `json:"transferChannelCode"`
	UserAccount         string  `json:"userAccount"`
	UserName            string  `json:"userName"`
	Type                int     `json:"type" binding:"required,oneof=1 2 3 4 5 6"` // 提现类型
	Price               *int    `json:"price" binding:"required,gte=0"`            // 提现金额
	Name                string  `json:"name"`                                      // 真实姓名 (Bank/Alipay)
	Account             string  `json:"account"`                                   // 账号 (Bank/Alipay)
	BankName            *string `json:"bankName"`                                  // 银行名称 (Bank)
	BankAddress         string  `json:"bankAddress"`                               // 开户地址 (Bank)
	QrCodeUrl           string  `json:"qrCodeUrl"`                                 // 收款码 (Wechat)
	Code                string  `json:"code"`                                      // 微信渠道需要
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

// Validate applies Java's withdrawal-type validation groups.
func (r *AppBrokerageWithdrawCreateReqVO) Validate() error {
	if r.Type < 1 || r.Type > 6 || r.Price == nil || *r.Price < 0 {
		return fmt.Errorf("提现方式或金额无效")
	}
	if r.TransferChannelCode != "" && consts.PayChannelName(r.TransferChannelCode) == "" {
		return fmt.Errorf("转账渠道无效")
	}
	switch r.Type {
	case consts.BrokerageWithdrawTypeBank, consts.BrokerageWithdrawTypeWechatAPI, consts.BrokerageWithdrawTypeAlipayAPI:
		if strings.TrimSpace(r.UserName) == "" || strings.TrimSpace(r.UserAccount) == "" {
			return fmt.Errorf("提现账号和姓名不能为空")
		}
	}
	if r.Type == consts.BrokerageWithdrawTypeBank && r.BankName == nil {
		return fmt.Errorf("提现银行不能为空")
	}
	if r.Type == consts.BrokerageWithdrawTypeWechatAPI && (r.TransferChannelCode == "" || *r.Price < 30) {
		return fmt.Errorf("微信提现渠道不能为空且金额不能小于30分")
	}
	if (r.Type == consts.BrokerageWithdrawTypeWechatQR || r.Type == consts.BrokerageWithdrawTypeAlipayQR) && r.QrCodeUrl != "" {
		u, err := url.Parse(r.QrCodeUrl)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ftp") {
			return fmt.Errorf("收款码必须是URL")
		}
	}
	return nil
}
