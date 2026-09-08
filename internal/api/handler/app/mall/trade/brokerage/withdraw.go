package brokerage

import (
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"strconv"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	tradeDto "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/trade"
	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/trade/brokerage"
	brokerageSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade/brokerage"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type AppBrokerageWithdrawHandler struct {
	withdrawSvc    *brokerageSvc.BrokerageWithdrawService
	dictSvc        *system.DictService
	payTransferSvc *pay.PayTransferService // Needed? Java Controller uses PayTransferApi.
}

func NewAppBrokerageWithdrawHandler(withdrawSvc *brokerageSvc.BrokerageWithdrawService, payTransferSvc *pay.PayTransferService, dictSvc *system.DictService) *AppBrokerageWithdrawHandler {
	return &AppBrokerageWithdrawHandler{
		withdrawSvc:    withdrawSvc,
		dictSvc:        dictSvc,
		payTransferSvc: payTransferSvc,
	}
}

// GetBrokerageWithdrawPage 获得分销提现分页
func (h *AppBrokerageWithdrawHandler) GetBrokerageWithdrawPage(c *gin.Context) {
	var reqVO tradeDto.AppBrokerageWithdrawPageReqVO
	if err := c.ShouldBindQuery(&reqVO); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}

	// userId := context.GetLoginUserId(c)
	userId := context.GetLoginUserID(c)
	pageReq := &trade.BrokerageWithdrawPageReq{
		PageParam:  reqVO.PageParam,
		CreateTime: resolveCreateTimeQuery(c, reqVO.CreateTime),
		UserID:     userId,
		Type:       reqVO.Type,
		Status:     reqVO.Status,
	}

	pageResult, err := h.withdrawSvc.GetBrokerageWithdrawPage(c, pageReq)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}

	labels, err := loadBrokerageLabels(c, h.dictSvc)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	writeResp := pagination.PageResult[*tradeDto.AppBrokerageWithdrawRespVO]{
		Total: pageResult.Total,
		List: lo.Map(pageResult.List, func(item *brokerage.BrokerageWithdraw, _ int) *tradeDto.AppBrokerageWithdrawRespVO {
			return &tradeDto.AppBrokerageWithdrawRespVO{
				ID:            item.ID,
				UserID:        item.UserID,
				Price:         item.Price,
				FeePrice:      item.FeePrice,
				TotalPrice:    item.TotalPrice,
				Type:          item.Type,
				Name:          item.UserName,    // Map UserName -> Name
				Account:       item.UserAccount, // Map UserAccount -> Account
				BankName:      item.BankName,
				Status:        item.Status,
				AuditReason:   item.AuditReason,
				AuditTime:     types.ToJsonDateTimePtr(item.AuditTime),
				Remark:        item.Remark,
				CreateTime:    types.ToJsonDateTime(item.CreateTime),
				TypeName:      labels.label("brokerage_withdraw_type", item.Type, tradeModel.BrokerageWithdrawTypeName(item.Type)),
				StatusName:    labels.label("brokerage_withdraw_status", item.Status, tradeModel.BrokerageWithdrawStatusName(item.Status)),
				PayTransferID: item.PayTransferID,
			}
		}),
	}
	response.WriteSuccess(c, writeResp)
}

// GetBrokerageWithdraw 获得佣金提现详情
func (h *AppBrokerageWithdrawHandler) GetBrokerageWithdraw(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}

	userId := context.GetLoginUserID(c)
	withdraw, err := h.withdrawSvc.GetBrokerageWithdraw(c, id)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	if withdraw == nil || withdraw.UserID != userId {
		response.WriteSuccess(c, nil)
		return
	}

	labels, err := loadBrokerageLabels(c, h.dictSvc)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	// VO Conversion
	respVO := &tradeDto.AppBrokerageWithdrawRespVO{
		ID: withdraw.ID, UserID: withdraw.UserID, Price: withdraw.Price, FeePrice: withdraw.FeePrice, TotalPrice: withdraw.TotalPrice,
		Type: withdraw.Type, Name: withdraw.UserName, Account: withdraw.UserAccount, BankName: withdraw.BankName,
		TypeName: labels.label("brokerage_withdraw_type", withdraw.Type, tradeModel.BrokerageWithdrawTypeName(withdraw.Type)), StatusName: labels.label("brokerage_withdraw_status", withdraw.Status, tradeModel.BrokerageWithdrawStatusName(withdraw.Status)), PayTransferID: withdraw.PayTransferID,
		Status:      withdraw.Status,
		AuditReason: withdraw.AuditReason,
		AuditTime:   types.ToJsonDateTimePtr(withdraw.AuditTime),
		Remark:      withdraw.Remark,
		CreateTime:  types.ToJsonDateTime(withdraw.CreateTime),
	}

	// Wechat Transfer Info Logic
	// Status: AUDIT_SUCCESS(10), Type: WECHAT(3)
	// We check against constants.
	if withdraw.Status == tradeModel.BrokerageWithdrawStatusAuditSuccess && withdraw.Type == tradeModel.BrokerageWithdrawTypeWechatAPI && withdraw.PayTransferID > 0 {
		transfer, err := h.payTransferSvc.GetTransfer(c.Request.Context(), int64(withdraw.PayTransferID))
		if err != nil {
			response.WriteError(c, 500, err.Error())
			return
		}
		if transfer != nil {
			if transfer.ChannelExtras != nil {
				if val, ok := transfer.ChannelExtras["package_info"]; ok {
					respVO.TransferChannelPackageInfo = val
				}
				if val, ok := transfer.ChannelExtras["mch_id"]; ok {
					respVO.TransferChannelMchId = val
				}
			}
		}
	}

	response.WriteSuccess(c, respVO)
}

// CreateBrokerageWithdraw 创建分销提现
func (h *AppBrokerageWithdrawHandler) CreateBrokerageWithdraw(c *gin.Context) {
	var reqVO tradeDto.AppBrokerageWithdrawCreateReqVO
	if err := c.ShouldBindJSON(&reqVO); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}

	if err := reqVO.Validate(); err != nil {
		response.WriteError(c, 400, err.Error())
		return
	}
	userId := context.GetLoginUserID(c)
	id, err := h.withdrawSvc.CreateBrokerageWithdraw(c, userId, &reqVO)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	response.WriteSuccess(c, id)
}
