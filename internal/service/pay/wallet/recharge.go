package wallet

import (
	"context"
	stdErrors "errors"
	"strconv"
	"time"

	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	paySvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"gorm.io/gorm"
)

type PayWalletRechargeService struct {
	q          *query.Query
	walletSvc  *PayWalletService
	trxSvc     *PayWalletTransactionService
	pkgSvc     *PayWalletRechargePackageService
	orderSvc   *paySvc.PayOrderService
	refundSvc  *paySvc.PayRefundService
	notifySvc  *paySvc.PayNotifyService
	channelSvc *paySvc.PayChannelService
}

func NewPayWalletRechargeService(
	q *query.Query,
	walletSvc *PayWalletService,
	trxSvc *PayWalletTransactionService,
	pkgSvc *PayWalletRechargePackageService,
	orderSvc *paySvc.PayOrderService,
	refundSvc *paySvc.PayRefundService,
	notifySvc *paySvc.PayNotifyService,
	channelSvc *paySvc.PayChannelService,
) *PayWalletRechargeService {
	return &PayWalletRechargeService{
		q:          q,
		walletSvc:  walletSvc,
		trxSvc:     trxSvc,
		pkgSvc:     pkgSvc,
		orderSvc:   orderSvc,
		refundSvc:  refundSvc,
		notifySvc:  notifySvc,
		channelSvc: channelSvc,
	}
}

// CreateWalletRecharge 创建充值记录 (发起充值)
func (s *PayWalletRechargeService) CreateWalletRecharge(ctx context.Context, req *pay2.PayWalletRechargeCreateReq, userIP string) (*pay.PayWalletRecharge, error) {
	// 1. 校验钱包是否存在
	wallet, err := s.walletSvc.GetOrCreateWallet(ctx, req.UserID, req.UserType)
	if err != nil {
		return nil, err
	}

	// 2. 计算金额
	payPrice := req.PayPrice
	bonusPrice := req.BonusPrice
	if req.PackageID > 0 {
		pkg, err := s.pkgSvc.GetWalletRechargePackage(ctx, req.PackageID)
		if err != nil {
			return nil, err
		}
		if pkg == nil {
			return nil, errors.NewBizError(1006004001, "充值套餐不存在")
		}
		payPrice = pkg.PayPrice
		bonusPrice = pkg.BonusPrice
	}

	// 3. 创建充值记录
	recharge := &pay.PayWalletRecharge{
		WalletID:   wallet.ID,
		TotalPrice: payPrice + bonusPrice,
		PayPrice:   payPrice,
		BonusPrice: bonusPrice,
		PackageID:  req.PackageID,
		PayStatus:  false, // Waiting
	}
	err = s.q.PayWalletRecharge.WithContext(ctx).Create(recharge)
	if err != nil {
		return nil, err
	}

	return recharge, nil
}

// UpdateWalletRechargerPaid 更新钱包充值为已支付
func (s *PayWalletRechargeService) UpdateWalletRechargerPaid(ctx context.Context, id int64, payOrderID int64) error {
	// 1.1 获取充值记录
	recharge, err := s.q.PayWalletRecharge.WithContext(ctx).Where(s.q.PayWalletRecharge.ID.Eq(id)).First()
	if err != nil {
		return err
	}
	if recharge.PayStatus {
		return nil // 已经支付，重复回调
	}

	// 1.2 校验支付订单
	payOrder, err := s.orderSvc.GetOrder(ctx, payOrderID)
	if err != nil || payOrder == nil {
		return stdErrors.New("支付订单不存在")
	}
	if payOrder.Status != consts.PayOrderStatusSuccess {
		return stdErrors.New("支付订单未支付")
	}

	// 2. 更新钱包充值的支付状态
	now := time.Now()
	res, err := s.q.PayWalletRecharge.WithContext(ctx).
		Where(s.q.PayWalletRecharge.ID.Eq(id), s.q.PayWalletRecharge.PayStatus.Is(false)).
		Updates(map[string]interface{}{
			"pay_status":       true,
			"pay_order_id":     payOrderID,
			"pay_time":         now,
			"pay_channel_code": payOrder.ChannelCode,
		})
	if err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return stdErrors.New("更新充值状态失败(非未支付状态)")
	}

	// 3. 更新钱包余额
	err = s.walletSvc.AddWalletBalance(ctx, recharge.WalletID, strconv.FormatInt(id, 10), consts.PayWalletBizTypeRecharge, recharge.TotalPrice)
	if err != nil {
		return err
	}
	return nil
}

// RefundWalletRecharge 发起钱包充值退款
func (s *PayWalletRechargeService) RefundWalletRecharge(ctx context.Context, id int64, userIP string) error {
	// 1.1 获取钱包充值记录
	recharge, err := s.q.PayWalletRecharge.WithContext(ctx).Where(s.q.PayWalletRecharge.ID.Eq(id)).First()
	if err != nil || recharge == nil {
		return errors.ErrNotFound
	}
	// 1.2 校验
	if !recharge.PayStatus {
		return stdErrors.New("未支付，无法退款")
	}
	if recharge.PayRefundID > 0 { // 已经申请过退款
		return stdErrors.New("已经申请过退款")
	}

	// 2. 冻结退款的余额
	if err := s.walletSvc.FreezePrice(ctx, recharge.WalletID, recharge.TotalPrice); err != nil {
		return err
	}

	// 3. 创建退款单
	appKey := config.C.Pay.WalletPayAppKey // 需要配置
	if appKey == "" {
		appKey = "wallet" // fallback
	}
	// MerchantRefundId gen
	refundNo := "R" + strconv.FormatInt(id, 10) + "_" + strconv.FormatInt(time.Now().Unix(), 10)

	payRefundID, err := s.refundSvc.CreateRefund(ctx, &pay2.PayRefundCreateReq{
		AppKey:           appKey,
		UserIP:           userIP,
		MerchantOrderId:  strconv.FormatInt(id, 10),
		MerchantRefundId: refundNo,
		Reason:           "想退钱",
		Price:            recharge.PayPrice,
	})
	if err != nil {
		return err
	}

	// 4. 更新充值记录退款单号
	_, err = s.q.PayWalletRecharge.WithContext(ctx).
		Where(s.q.PayWalletRecharge.ID.Eq(id)).
		Updates(map[string]interface{}{
			"pay_refund_id": payRefundID,
			"refund_status": consts.PayRefundStatusWaiting,
		})
	return err
}

// UpdateWalletRechargeRefunded 更新钱包充值为已退款
func (s *PayWalletRechargeService) UpdateWalletRechargeRefunded(ctx context.Context, id int64, refundID int64) error {
	recharge, err := s.q.PayWalletRecharge.WithContext(ctx).Where(s.q.PayWalletRecharge.ID.Eq(id)).First()
	if err != nil || recharge == nil {
		return errors.ErrNotFound
	}
	if recharge.PayRefundID != refundID {
		return stdErrors.New("退款单号不匹配")
	}

	payRefund, err := s.refundSvc.GetRefund(ctx, refundID)
	if err != nil {
		return err
	}

	// 2. 处理退款结果
	updates := map[string]interface{}{}
	if payRefund.Status == consts.PayRefundStatusSuccess {
		// 2.1 退款成功: 真正的扣除余额 (ReduceFrozen)
		// Manual update for deduacting frozen and total recharge
		res, err := s.walletSvc.q.PayWallet.WithContext(ctx).
			Where(s.walletSvc.q.PayWallet.ID.Eq(recharge.WalletID)).
			Updates(map[string]interface{}{
				"freeze_price":   gorm.Expr("freeze_price - ?", recharge.TotalPrice),
				"total_recharge": gorm.Expr("total_recharge - ?", recharge.TotalPrice),
			})
		if err != nil {
			return err
		}
		if res.RowsAffected == 0 {
			// ignore?
		}
		// Record Transaction
		s.trxSvc.CreateWalletTransaction(ctx, &pay.PayWallet{ID: recharge.WalletID, Balance: 0}, // approximate
			consts.PayWalletBizTypeRechargeRefund, strconv.FormatInt(recharge.ID, 10), "充值退款", -recharge.TotalPrice)

		updates["refund_status"] = consts.PayRefundStatusSuccess
		updates["refund_time"] = payRefund.SuccessTime
		updates["refund_total_price"] = recharge.TotalPrice
		updates["refund_pay_price"] = recharge.PayPrice
		updates["refund_bonus_price"] = recharge.BonusPrice
	} else if payRefund.Status == consts.PayRefundStatusFailure {
		// 2.2 退款失败: 解冻
		err = s.walletSvc.UnfreezePrice(ctx, recharge.WalletID, recharge.TotalPrice)
		if err != nil {
			return err
		}
		updates["refund_status"] = consts.PayRefundStatusFailure
	} else {
		return nil // Still waiting
	}

	_, err = s.q.PayWalletRecharge.WithContext(ctx).
		Where(s.q.PayWalletRecharge.ID.Eq(id)).
		Updates(updates)
	return err
}

func (s *PayWalletRechargeService) GetWalletRechargePage(ctx context.Context, req *pay2.PayWalletRechargePageReq) (*pagination.PageResult[*pay.PayWalletRecharge], error) {
	q := s.q.PayWalletRecharge.WithContext(ctx)
	if req.PayStatus != nil {
		q = q.Where(s.q.PayWalletRecharge.PayStatus.Is(*req.PayStatus))
	}
	q = q.Order(s.q.PayWalletRecharge.ID.Desc())

	list, total, err := q.FindByPage(req.PageNo, req.PageSize)
	if err != nil {
		return nil, err
	}
	return pagination.NewPageResult(list, total), nil
}
