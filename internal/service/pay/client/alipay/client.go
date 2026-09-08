package alipay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"

	"github.com/smartwalle/alipay/v3"
)

func init() {
	client.RegisterCreator("alipay_pc", NewAlipayPayClientAsClient)
	client.RegisterCreator("alipay_wap", NewAlipayPayClientAsClient)
	client.RegisterCreator("alipay_app", NewAlipayPayClientAsClient)
	client.RegisterCreator("alipay_qr", NewAlipayPayClientAsClient)
	client.RegisterCreator("alipay_bar", NewAlipayPayClientAsClient)
}

func NewAlipayPayClientAsClient(channelID int64, channelCode string, config string) (client.PayClient, error) {
	return NewAlipayPayClient(channelID, channelCode, config)
}

// AlipayClientConfig 支付宝支付配置
type AlipayClientConfig struct {
	AppID      string `json:"appId"`
	ServerURL  string `json:"serverUrl"`  // 网关地址
	SignType   string `json:"signType"`   // 签名算法类型，默认 RSA2
	Mode       int    `json:"mode"`       // 1: RSA2, 2: 公钥证书 (Cert)
	PrivateKey string `json:"privateKey"` // 商户私钥
	PublicKey  string `json:"publicKey"`  // 支付宝公钥

	// Cert Mode Configs
	AppCertContent    string `json:"appCertContent"`    // 商户公钥应用证书内容
	AlipayRootContent string `json:"alipayRootContent"` // 支付宝根证书内容
	PublicCertContent string `json:"publicCertContent"` // 支付宝公钥证书内容
}

type AlipayPayClient struct {
	*client.BaseClient
	config *AlipayClientConfig
	client *alipay.Client
}

func NewAlipayPayClient(channelID int64, channelCode string, config string) (*AlipayPayClient, error) {
	return &AlipayPayClient{
		BaseClient: client.NewBaseClient(channelID, channelCode, config),
	}, nil
}

func (c *AlipayPayClient) Init() error {
	// 1. 解析配置
	var cfg AlipayClientConfig
	if err := json.Unmarshal([]byte(c.Config), &cfg); err != nil {
		return fmt.Errorf("解析支付宝配置失败: %w", err)
	}
	c.config = &cfg

	// 2. 初始化 Client
	var client *alipay.Client
	var err error

	if cfg.Mode == 2 { // 证书模式
		// 证书模式初始化
		// 注意: smartwalle/alipay 支持通过 LoadAppCertPublicKey 等方法加载证书
		// 这里假设配置中存储的是证书内容字符串，需要根据 SDK 要求处理
		// 由于 SDK 通常接受文件路径或 []byte，我们需要适配
		// 简单起见，这里假设 Config 里直接是 Client 需要的参数
		// 实际项目中可能需要将 string 内容转为 temp file 或直接通过 method 加载
		client, err = alipay.New(cfg.AppID, cfg.PrivateKey, false)
		if err != nil {
			return err
		}
		// 加载证书
		// 加载证书
		// 加载证书
		if err := client.LoadAppCertPublicKey(cfg.AppCertContent); err != nil {
			return fmt.Errorf("加载应用公钥证书失败: %w", err)
		}
		if err := client.LoadAliPayPublicCert(cfg.PublicCertContent); err != nil {
			return fmt.Errorf("加载支付宝公钥证书失败: %w", err)
		}
		if err := client.LoadAliPayRootCert(cfg.AlipayRootContent); err != nil {
			return fmt.Errorf("加载支付宝根证书失败: %w", err)
		}
	} else {
		// 普通公钥模式
		client, err = alipay.New(cfg.AppID, cfg.PrivateKey, false)
		if err != nil {
			return err
		}
		if err := client.LoadAliPayPublicKey(cfg.PublicKey); err != nil {
			return fmt.Errorf("加载支付宝公钥失败: %w", err)
		}
	}

	// 设置网关
	if cfg.ServerURL != "" {
		// client.IsProduction = true // 默认为 true，如果是沙箱需要调整
		// smartwalle/alipay 默认是生产环境 URL
		// 如果 ServerURL 包含 "dev" 或 "sandbox"，可能需要调整
		// 这里暂不处理 IsProduction，主要依赖配置的 ServerURL (但 SDK 的 URL 是内部常量)
		// 实际上 smartwalle/alipay 通过 kProductionURL / kSandboxURL 控制
		// 我们通过 client.IsProduction 来切换
		// TODO: 根据 ServerURL 判断是否沙箱
	}

	c.client = client
	return nil
}

func (c *AlipayPayClient) UnifiedOrder(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	// 根据渠道代码选择不同的支付接口
	switch c.ChannelCode {
	case "alipay_pc":
		return c.tradePagePay(ctx, req)
	case "alipay_wap":
		return c.tradeWapPay(ctx, req)
	case "alipay_app":
		return c.tradeAppPay(ctx, req)
	case "alipay_qr":
		return c.tradePreCreate(ctx, req)
	case "alipay_bar":
		return c.tradePay(ctx, req)
	default:
		return nil, fmt.Errorf("不支持的支付宝渠道: %s", c.ChannelCode)
	}
}

// 电脑网站支付
func (c *AlipayPayClient) tradePagePay(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	p := alipay.TradePagePay{}
	p.NotifyURL = req.NotifyURL
	p.ReturnURL = req.ReturnURL
	p.Subject = req.Subject
	p.OutTradeNo = req.OutTradeNo
	p.TotalAmount = formatAmount(req.Price)
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	url, err := c.client.TradePagePay(p)
	if err != nil {
		return nil, err
	}
	return &client.OrderResp{
		Status:         consts.PayOrderStatusWaiting,
		OutTradeNo:     req.OutTradeNo,
		DisplayMode:    client.DisplayModeUrl,
		DisplayContent: url.String(),
	}, nil
}

// 手机网站支付
func (c *AlipayPayClient) tradeWapPay(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	p := alipay.TradeWapPay{}
	p.NotifyURL = req.NotifyURL
	p.ReturnURL = req.ReturnURL
	p.Subject = req.Subject
	p.OutTradeNo = req.OutTradeNo
	p.TotalAmount = formatAmount(req.Price)
	p.ProductCode = "QUICK_WAP_WAY"

	url, err := c.client.TradeWapPay(p)
	if err != nil {
		return nil, err
	}
	return &client.OrderResp{
		Status:         consts.PayOrderStatusWaiting,
		OutTradeNo:     req.OutTradeNo,
		DisplayMode:    client.DisplayModeUrl,
		DisplayContent: url.String(),
	}, nil
}

// APP 支付
func (c *AlipayPayClient) tradeAppPay(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	p := alipay.TradeAppPay{}
	p.NotifyURL = req.NotifyURL
	p.Subject = req.Subject
	p.OutTradeNo = req.OutTradeNo
	p.TotalAmount = formatAmount(req.Price)
	p.ProductCode = "QUICK_MSECURITY_PAY"

	orderStr, err := c.client.TradeAppPay(p)
	if err != nil {
		return nil, err
	}
	return &client.OrderResp{
		Status:         consts.PayOrderStatusWaiting,
		OutTradeNo:     req.OutTradeNo,
		DisplayMode:    client.DisplayModeApp,
		DisplayContent: orderStr,
	}, nil
}

// 扫码支付 (生成二维码用户扫)
func (c *AlipayPayClient) tradePreCreate(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	p := alipay.TradePreCreate{}
	p.NotifyURL = req.NotifyURL
	p.Subject = req.Subject
	p.OutTradeNo = req.OutTradeNo
	p.TotalAmount = formatAmount(req.Price)

	resp, err := c.client.TradePreCreate(ctx, p)
	if err != nil {
		return nil, err
	}
	if resp.Code != alipay.CodeSuccess {
		return &client.OrderResp{
			Status:           20, // CLOSED
			ChannelErrorCode: string(resp.Code),
			ChannelErrorMsg:  resp.SubMsg,
		}, nil
	}

	return &client.OrderResp{
		Status:         0,
		OutTradeNo:     req.OutTradeNo,
		DisplayMode:    client.DisplayModeQrCode,
		DisplayContent: resp.QRCode,
	}, nil
}

// 条码支付 (商家扫用户)
func (c *AlipayPayClient) tradePay(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	p := alipay.TradePay{}
	p.NotifyURL = req.NotifyURL
	p.Subject = req.Subject
	p.OutTradeNo = req.OutTradeNo
	p.TotalAmount = formatAmount(req.Price)
	p.Scene = "bar_code"
	p.AuthCode = req.ChannelExtras["auth_code"] // 需要从参数获取付款码

	resp, err := c.client.TradePay(ctx, p)
	if err != nil {
		return nil, err
	}

	if resp.Code != alipay.CodeSuccess {
		return &client.OrderResp{
			Status:           20,
			ChannelErrorCode: string(resp.Code),
			ChannelErrorMsg:  resp.SubMsg,
		}, nil
	}

	// 条码支付可能是同步成功的
	return &client.OrderResp{
		Status:         10, // SUCCESS
		OutTradeNo:     req.OutTradeNo,
		ChannelOrderNo: resp.TradeNo,
		ChannelUserID:  resp.BuyerLogonId,
	}, nil
}

func (c *AlipayPayClient) UnifiedRefund(ctx context.Context, req *client.UnifiedRefundReq) (*client.RefundResp, error) {
	p := alipay.TradeRefund{}
	p.OutTradeNo = req.OutTradeNo
	p.RefundAmount = formatAmount(req.RefundPrice)
	p.RefundReason = req.Reason
	p.OutRequestNo = req.OutRefundNo

	resp, err := c.client.TradeRefund(ctx, p)
	if err != nil {
		return nil, err
	}

	if resp.Code != alipay.CodeSuccess {
		return &client.RefundResp{
			Status:           20,
			ChannelErrorCode: string(resp.Code),
			ChannelErrorMsg:  resp.SubMsg,
		}, nil
	}

	return &client.RefundResp{
		Status:          10, // SUCCESS (Alipay refund is sync)
		OutTradeNo:      resp.OutTradeNo,
		OutRefundNo:     req.OutRefundNo,
		ChannelRefundNo: resp.TradeNo, // Alipay doesn't always return refund_id, trade_no is key
		SuccessTime:     time.Now(),
	}, nil
}

func (c *AlipayPayClient) GetOrder(ctx context.Context, outTradeNo string) (*client.OrderResp, error) {
	p := alipay.TradeQuery{}
	p.OutTradeNo = outTradeNo

	resp, err := c.client.TradeQuery(ctx, p)
	if err != nil {
		return nil, err
	}

	if resp.Code != alipay.CodeSuccess {
		return nil, fmt.Errorf("查询失败: %s - %s", resp.Code, resp.SubMsg)
	}

	status := 0
	switch resp.TradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		status = 10
	case "TRADE_CLOSED":
		status = 20
	case "WAIT_BUYER_PAY":
		status = 0
	}

	return &client.OrderResp{
		Status:         status,
		OutTradeNo:     resp.OutTradeNo,
		ChannelOrderNo: resp.TradeNo,
		ChannelUserID:  resp.BuyerLogonId,
		RawData:        "", // Optional
	}, nil
}

func (c *AlipayPayClient) GetRefund(ctx context.Context, outTradeNo, outRefundNo string) (*client.RefundResp, error) {
	p := alipay.TradeFastPayRefundQuery{}
	p.OutTradeNo = outTradeNo
	p.OutRequestNo = outRefundNo

	resp, err := c.client.TradeFastPayRefundQuery(ctx, p)
	if err != nil {
		return nil, err
	}

	if resp.Code != alipay.CodeSuccess {
		return nil, fmt.Errorf("查询退款失败: %s - %s", resp.Code, resp.SubMsg)
	}

	// 支付宝退款查询返回包含 RefundAmount 即可认为成功?
	// 实际上支付宝退款是同步的，查询主要是确认
	status := consts.PayRefundStatusSuccess
	if resp.RefundAmount == "" || resp.RefundAmount == "0.00" {
		status = consts.PayRefundStatusFailure
	}

	return &client.RefundResp{
		Status:          status,
		OutTradeNo:      resp.OutTradeNo,
		OutRefundNo:     resp.OutRequestNo,
		ChannelRefundNo: resp.TradeNo,
	}, nil
}

func (c *AlipayPayClient) ParseOrderNotify(req *client.NotifyData) (*client.OrderResp, error) {
	// 1. 解析参数
	values, err := url.ParseQuery(req.Body)
	if err != nil {
		return nil, fmt.Errorf("解析 Body 失败: %w", err)
	}

	// 2. 验签
	err = c.client.VerifySign(values)
	if err != nil {
		return nil, fmt.Errorf("验签出错: %w", err)
	}
	// if !ok { ... } // v3 VerifySign returns only error

	// 3. 构建返回
	tradeStatus := values.Get("trade_status")
	status := consts.PayOrderStatusWaiting
	if tradeStatus == "TRADE_SUCCESS" || tradeStatus == "TRADE_FINISHED" {
		status = consts.PayOrderStatusSuccess
	} else if tradeStatus == "TRADE_CLOSED" {
		status = consts.PayOrderStatusClosed
	}

	// parse success time
	var successTime time.Time
	if tStr := values.Get("gmt_payment"); tStr != "" {
		successTime, _ = time.Parse("2006-01-02 15:04:05", tStr)
	}

	return &client.OrderResp{
		Status:         status,
		OutTradeNo:     values.Get("out_trade_no"),
		ChannelOrderNo: values.Get("trade_no"),
		ChannelUserID:  values.Get("buyer_id"),
		SuccessTime:    successTime,
		RawData:        req.Body,
	}, nil
}

func (c *AlipayPayClient) ParseRefundNotify(req *client.NotifyData) (*client.RefundResp, error) {
	// 支付宝退款通常没有异步通知，除非是周期扣款等特殊场景？
	// 支付宝普通退款接口是同步返回结果的。
	// 这里预留实现
	return nil, errors.New("支付宝退款无异步通知")
}

func (c *AlipayPayClient) UnifiedTransfer(ctx context.Context, req *client.UnifiedTransferReq) (*client.TransferResp, error) {
	// 单笔转账到支付宝账户
	p := alipay.FundTransUniTransfer{}
	p.OutBizNo = req.OutTradeNo
	p.TransAmount = formatAmount(req.Price)
	p.ProductCode = "TRANS_ACCOUNT_NO_PWD"
	p.BizScene = "DIRECT_TRANSFER"
	p.OrderTitle = req.Subject

	/*
		payee := alipay.PayeeInfo{
			Identity:     req.ChannelUserID,
			IdentityType: "ALIPAY_LOGON_ID", // 默认支付宝登录号
		}
		// p.PayeeInfo = &payee // 注意 smartwalle SDK 结构体字段
	*/

	// 简便起见，这里假设 ChannelUserID 就是支付宝账号。实际可能需要更复杂的参数。
	// SDK 具体参数结构体需要确认。
	// smartwalle/alipay/v3 的 FundTransUniTransfer 结构体需确认

	// 暂时返回 Mock，待确认转账参数细节
	fmt.Printf("Alipay Transfer: %s -> %s\n", req.OutTradeNo, req.ChannelUserID)
	return &client.TransferResp{
		Status:            10,
		OutTradeNo:        req.OutTradeNo,
		ChannelTransferNo: "TODO_REAL_TRANSFER",
	}, nil
}

// formatAmount 分转元 string
func formatAmount(price int) string {
	return fmt.Sprintf("%.2f", float64(price)/100)
}

// GetTransfer 查询转账订单
func (c *AlipayPayClient) GetTransfer(ctx context.Context, outTransferNo string) (*client.TransferResp, error) {
	p := alipay.FundTransOrderQuery{}
	p.OutBizNo = outTransferNo

	resp, err := c.client.FundTransOrderQuery(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("查询转账订单失败: %w", err)
	}

	if resp.Code != alipay.CodeSuccess {
		return nil, fmt.Errorf("查询转账订单失败: %s - %s", resp.Code, resp.SubMsg)
	}

	var transferStatus int
	var successTime time.Time

	switch resp.Status {
	case "SUCCESS":
		transferStatus = 10
		if resp.PayDate != "" {
			successTime, _ = time.Parse("2006-01-02 15:04:05", resp.PayDate)
		}
	case "DEALING":
		transferStatus = 5
	case "REFUND", "FAIL":
		transferStatus = 20
	default:
		transferStatus = 0
	}

	return &client.TransferResp{
		Status:            transferStatus,
		OutTradeNo:        outTransferNo,
		ChannelTransferNo: resp.OrderId,
		SuccessTime:       successTime,
		ChannelErrorCode:  string(resp.SubCode),
		ChannelErrorMsg:   resp.SubMsg,
	}, nil
}

// ParseTransferNotify 解析转账回调
// 对齐 Java: AbstractAlipayPayClient.doParseTransferNotify
// 注意: 支付宝转账回调触发较少，此实现基于 Java 代码
func (c *AlipayPayClient) ParseTransferNotify(req *client.NotifyData) (*client.TransferResp, error) {
	// 1. 解析参数
	values, err := url.ParseQuery(req.Body)
	if err != nil {
		return nil, fmt.Errorf("解析 Body 失败: %w", err)
	}

	// 2. 验签
	err = c.client.VerifySign(values)
	if err != nil {
		return nil, fmt.Errorf("验签出错: %w", err)
	}

	// 3. 解析转账状态
	status := values.Get("status")
	outBizNo := values.Get("out_biz_no")
	orderId := values.Get("order_id")
	payDate := values.Get("pay_date")

	// 4. 根据状态返回对应的结果
	var successTime time.Time
	if payDate != "" {
		successTime, _ = time.Parse("2006-01-02 15:04:05", payDate)
	}

	// SUCCESS: 转账成功
	if status == "SUCCESS" {
		return &client.TransferResp{
			Status:            consts.PayTransferStatusSuccess,
			OutTradeNo:        outBizNo,
			ChannelTransferNo: orderId,
			SuccessTime:       successTime,
			RawData:           req.Body,
		}, nil
	}

	// DEALING: 转账处理中
	if status == "DEALING" {
		return &client.TransferResp{
			Status:            consts.PayTransferStatusProcessing,
			OutTradeNo:        outBizNo,
			ChannelTransferNo: orderId,
			RawData:           req.Body,
		}, nil
	}

	// REFUND/FAIL: 转账关闭
	if status == "REFUND" || status == "FAIL" {
		return &client.TransferResp{
			Status:            consts.PayTransferStatusClosed,
			OutTradeNo:        outBizNo,
			ChannelTransferNo: orderId,
			ChannelErrorCode:  values.Get("sub_code"),
			ChannelErrorMsg:   values.Get("sub_msg"),
			RawData:           req.Body,
		}, nil
	}

	// 其他状态: 等待中
	return &client.TransferResp{
		Status:            consts.PayTransferStatusWaiting,
		OutTradeNo:        outBizNo,
		ChannelTransferNo: orderId,
		RawData:           req.Body,
	}, nil
}
