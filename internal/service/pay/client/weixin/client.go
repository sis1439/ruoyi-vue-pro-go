package weixin

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"

	"io"
	"net/http"
	"strings"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/app"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/services/transferbatch"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

func init() {
	client.RegisterCreator("wx_pub", NewWxPayClientAsClient)
	client.RegisterCreator("wx_lite", NewWxPayClientAsClient)
	client.RegisterCreator("wx_app", NewWxPayClientAsClient)
	client.RegisterCreator("wx_native", NewWxPayClientAsClient)
	client.RegisterCreator("wx_wap", NewWxPayClientAsClient)
	client.RegisterCreator("wx_bar", NewWxPayClientAsClient)
}

func NewWxPayClientAsClient(channelID int64, channelCode string, config string) (client.PayClient, error) {
	return NewWxPayClient(channelID, channelCode, config)
}

// deref 安全解引用 SDK 返回的字符串指针
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// expireTime 支付过期时间；零值表示不传，避免下单即过期
func expireTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

type WxPayClient struct {
	*client.BaseClient
	config     *WxPayClientConfig
	coreClient *core.Client
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

func NewWxPayClient(channelID int64, channelCode string, config string) (*WxPayClient, error) {
	return &WxPayClient{
		BaseClient: client.NewBaseClient(channelID, channelCode, config),
	}, nil
}

func (c *WxPayClient) Init() error {
	// 1. 解析配置
	var cfg WxPayClientConfig
	if err := json.Unmarshal([]byte(c.Config), &cfg); err != nil {
		return fmt.Errorf("解析微信支付配置失败: %w", err)
	}
	c.config = &cfg

	// 2. V3 版本初始化
	if cfg.APIVersion == APIVersionV3 {
		return c.initV3Client()
	}

	// V2 版本暂不支持
	return errors.New("暂不支持微信支付 V2 版本")
}

func (c *WxPayClient) initV3Client() error {
	cfg := c.config

	// 加载商户私钥
	privateKey, err := utils.LoadPrivateKey(cfg.PrivateKeyContent)
	if err != nil {
		return fmt.Errorf("加载商户私钥失败: %w", err)
	}
	c.privateKey = privateKey

	// 加载微信支付公钥
	publicKey, err := utils.LoadPublicKey(cfg.PublicKeyContent)
	if err != nil {
		return fmt.Errorf("加载微信支付公钥失败: %w", err)
	}
	c.publicKey = publicKey

	// 使用微信支付公钥模式创建客户端
	opts := []core.ClientOption{
		option.WithWechatPayPublicKeyAuthCipher(
			cfg.MchID,
			cfg.CertSerialNo,
			privateKey,
			cfg.PublicKeyID,
			publicKey,
		),
	}

	coreClient, err := core.NewClient(context.Background(), opts...)
	if err != nil {
		return fmt.Errorf("创建微信支付客户端失败: %w", err)
	}
	c.coreClient = coreClient

	fmt.Printf("微信支付客户端初始化成功 [Channel: %d, MchID: %s]\n", c.ChannelID, cfg.MchID)
	return nil
}

// 支付方式
const (
	methodNative = "native"
	methodJSAPI  = "jsapi"
	methodH5     = "h5"
	methodApp    = "app"
)

// payMethodOf 渠道编码 → 下单方式。渠道编码以固定版 uni-app 与 Java 契约为准：
// wx_wap 是浏览器 H5 支付（wx_h5 作为历史别名保留），wx_pub/wx_lite 均走 JSAPI。
func payMethodOf(channelCode string) (string, bool) {
	switch channelCode {
	case "wx_native":
		return methodNative, true
	case "wx_pub", "wx_lite":
		return methodJSAPI, true
	case "wx_wap", "wx_h5":
		return methodH5, true
	case "wx_app":
		return methodApp, true
	default:
		return "", false
	}
}

// UnifiedOrder 统一下单
func (c *WxPayClient) UnifiedOrder(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	method, ok := payMethodOf(c.ChannelCode)
	if !ok {
		return nil, fmt.Errorf("暂不支持的微信支付渠道: %s", c.ChannelCode)
	}
	switch method {
	case methodNative:
		return c.nativeOrder(ctx, req)
	case methodJSAPI:
		return c.jsapiOrder(ctx, req)
	case methodH5:
		return c.h5Order(ctx, req)
	default:
		return c.appOrder(ctx, req)
	}
}

// nativeOrder Native 扫码支付
func (c *WxPayClient) nativeOrder(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	svc := native.NativeApiService{Client: c.coreClient}

	resp, result, err := svc.Prepay(ctx, native.PrepayRequest{
		Appid:       core.String(c.config.AppID),
		Mchid:       core.String(c.config.MchID),
		Description: core.String(req.Subject),
		OutTradeNo:  core.String(req.OutTradeNo),
		NotifyUrl:   core.String(req.NotifyURL),
		Amount: &native.Amount{
			Total:    core.Int64(int64(req.Price)),
			Currency: core.String("CNY"),
		},
	})

	if err != nil {
		return &client.OrderResp{
			Status:           consts.PayOrderStatusClosed, // CLOSED
			OutTradeNo:       req.OutTradeNo,
			ChannelErrorCode: "NATIVE_PREPAY_ERROR",
			ChannelErrorMsg:  err.Error(),
		}, nil
	}

	_ = result

	return &client.OrderResp{
		Status:         consts.PayOrderStatusWaiting, // WAITING
		OutTradeNo:     req.OutTradeNo,
		DisplayMode:    "qr_code",
		DisplayContent: *resp.CodeUrl,
	}, nil
}

// jsapiOrder JSAPI 公众号/小程序支付
func (c *WxPayClient) jsapiOrder(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	svc := jsapi.JsapiApiService{Client: c.coreClient}

	// OpenID 从 ChannelExtras 获取
	openID := ""
	if req.ChannelExtras != nil {
		openID = req.ChannelExtras["openid"]
	}
	if openID == "" {
		return nil, errors.New("JSAPI 支付需要 openid")
	}

	// PrepayWithRequestPayment 由官方 SDK 用商户私钥完成签名，商户私钥不下发前端
	resp, result, err := svc.PrepayWithRequestPayment(ctx, jsapi.PrepayRequest{
		Appid:       core.String(c.config.AppID),
		Mchid:       core.String(c.config.MchID),
		Description: core.String(req.Subject),
		OutTradeNo:  core.String(req.OutTradeNo),
		NotifyUrl:   core.String(req.NotifyURL),
		TimeExpire:  expireTime(req.ExpireTime),
		Amount: &jsapi.Amount{
			Total:    core.Int64(int64(req.Price)),
			Currency: core.String("CNY"),
		},
		Payer: &jsapi.Payer{
			Openid: core.String(openID),
		},
	})

	if err != nil {
		return &client.OrderResp{
			Status:           consts.PayOrderStatusClosed, // CLOSED
			OutTradeNo:       req.OutTradeNo,
			ChannelErrorCode: "JSAPI_PREPAY_ERROR",
			ChannelErrorMsg:  err.Error(),
		}, nil
	}

	_ = result

	// 字段名对齐固定版 uni-app：sheep/platform/pay.js 与 sheep/libs/sdk-h5-weixin.js
	// 均读取 timeStamp / nonceStr / packageValue / signType / paySign
	content, err := json.Marshal(map[string]string{
		"appId":        deref(resp.Appid),
		"timeStamp":    deref(resp.TimeStamp),
		"nonceStr":     deref(resp.NonceStr),
		"packageValue": deref(resp.Package),
		"signType":     deref(resp.SignType),
		"paySign":      deref(resp.PaySign),
	})
	if err != nil {
		return nil, fmt.Errorf("序列化 JSAPI 支付参数失败: %w", err)
	}

	return &client.OrderResp{
		Status:         consts.PayOrderStatusWaiting, // WAITING
		OutTradeNo:     req.OutTradeNo,
		DisplayMode:    client.DisplayModeApp,
		DisplayContent: string(content),
	}, nil
}

// h5Order H5 支付
func (c *WxPayClient) h5Order(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	svc := h5.H5ApiService{Client: c.coreClient}

	// 构造 H5 场景信息
	sceneInfo := &h5.SceneInfo{
		PayerClientIp: core.String(req.UserIP),
		H5Info: &h5.H5Info{
			Type: core.String("Wap"),
		},
	}
	// 如果 channelExtras 中有 info，可以解析覆盖默认值
	if req.ChannelExtras != nil {
		if appName, ok := req.ChannelExtras["app_name"]; ok {
			sceneInfo.H5Info.AppName = core.String(appName)
		}
		if bundleId, ok := req.ChannelExtras["bundle_id"]; ok {
			sceneInfo.H5Info.BundleId = core.String(bundleId)
		}
	}

	resp, result, err := svc.Prepay(ctx, h5.PrepayRequest{
		Appid:       core.String(c.config.AppID),
		Mchid:       core.String(c.config.MchID),
		Description: core.String(req.Subject),
		OutTradeNo:  core.String(req.OutTradeNo),
		NotifyUrl:   core.String(req.NotifyURL),
		Amount: &h5.Amount{
			Total:    core.Int64(int64(req.Price)),
			Currency: core.String("CNY"),
		},
		SceneInfo: sceneInfo,
	})

	if err != nil {
		return &client.OrderResp{
			Status:           consts.PayOrderStatusClosed, // CLOSED
			OutTradeNo:       req.OutTradeNo,
			ChannelErrorCode: "H5_PREPAY_ERROR",
			ChannelErrorMsg:  err.Error(),
		}, nil
	}
	_ = result

	return &client.OrderResp{
		Status:         consts.PayOrderStatusWaiting, // WAITING
		OutTradeNo:     req.OutTradeNo,
		DisplayMode:    "url",
		DisplayContent: *resp.H5Url,
	}, nil
}

// appOrder APP 支付
func (c *WxPayClient) appOrder(ctx context.Context, req *client.UnifiedOrderReq) (*client.OrderResp, error) {
	svc := app.AppApiService{Client: c.coreClient}

	resp, result, err := svc.PrepayWithRequestPayment(ctx, app.PrepayRequest{
		Appid:       core.String(c.config.AppID),
		Mchid:       core.String(c.config.MchID),
		Description: core.String(req.Subject),
		OutTradeNo:  core.String(req.OutTradeNo),
		NotifyUrl:   core.String(req.NotifyURL),
		TimeExpire:  expireTime(req.ExpireTime),
		Amount: &app.Amount{
			Total:    core.Int64(int64(req.Price)),
			Currency: core.String("CNY"),
		},
	})

	if err != nil {
		return &client.OrderResp{
			Status:           consts.PayOrderStatusClosed, // CLOSED
			OutTradeNo:       req.OutTradeNo,
			ChannelErrorCode: "APP_PREPAY_ERROR",
			ChannelErrorMsg:  err.Error(),
		}, nil
	}
	_ = result

	// App 端 uni.requestPayment 要求全小写键名，见 sheep/platform/pay.js wechatAppPay
	content, err := json.Marshal(map[string]string{
		"appid":     c.config.AppID,
		"partnerid": deref(resp.PartnerId),
		"prepayid":  deref(resp.PrepayId),
		"package":   deref(resp.Package),
		"noncestr":  deref(resp.NonceStr),
		"timestamp": deref(resp.TimeStamp),
		"sign":      deref(resp.Sign),
	})
	if err != nil {
		return nil, fmt.Errorf("序列化 App 支付参数失败: %w", err)
	}

	return &client.OrderResp{
		Status:         consts.PayOrderStatusWaiting, // WAITING
		OutTradeNo:     req.OutTradeNo,
		DisplayMode:    client.DisplayModeApp,
		DisplayContent: string(content),
	}, nil
}

// UnifiedRefund 统一退款
func (c *WxPayClient) UnifiedRefund(ctx context.Context, req *client.UnifiedRefundReq) (*client.RefundResp, error) {
	svc := refunddomestic.RefundsApiService{Client: c.coreClient}

	resp, _, err := svc.Create(ctx, refunddomestic.CreateRequest{
		OutTradeNo:  core.String(req.OutTradeNo),
		OutRefundNo: core.String(req.OutRefundNo),
		Reason:      core.String(req.Reason),
		NotifyUrl:   core.String(req.NotifyURL),
		Amount: &refunddomestic.AmountReq{
			Currency: core.String("CNY"),
			Refund:   core.Int64(int64(req.RefundPrice)),
			Total:    core.Int64(int64(req.PayPrice)),
		},
	})

	if err != nil {
		return &client.RefundResp{
			Status:           consts.PayRefundStatusFailure, // FALLBACK/FAILURE (Need better mapping)
			OutTradeNo:       req.OutTradeNo,
			OutRefundNo:      req.OutRefundNo,
			ChannelErrorCode: "REFUND_ERROR",
			ChannelErrorMsg:  err.Error(),
		}, nil
	}

	status := consts.PayRefundStatusWaiting
	switch *resp.Status {
	case refunddomestic.STATUS_SUCCESS:
		status = consts.PayRefundStatusSuccess
	case refunddomestic.STATUS_CLOSED, refunddomestic.STATUS_ABNORMAL:
		status = consts.PayRefundStatusFailure
	}

	return &client.RefundResp{
		Status:          status,
		OutTradeNo:      req.OutTradeNo,
		OutRefundNo:     req.OutRefundNo,
		ChannelRefundNo: *resp.RefundId,
		SuccessTime:     time.Now(), // TODO: Parse SuccessTime if available
	}, nil
}

// GetOrder 查询订单
func (c *WxPayClient) GetOrder(ctx context.Context, outTradeNo string) (*client.OrderResp, error) {
	svc := jsapi.JsapiApiService{Client: c.coreClient}

	resp, _, err := svc.QueryOrderByOutTradeNo(ctx, jsapi.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(outTradeNo),
		Mchid:      core.String(c.config.MchID),
	})

	if err != nil {
		return nil, err
	}

	status := consts.PayOrderStatusWaiting
	var successTime time.Time
	switch *resp.TradeState {
	case "SUCCESS":
		status = consts.PayOrderStatusSuccess
		successTime, _ = time.Parse(time.RFC3339, *resp.SuccessTime)
	case "CLOSED", "PAYERROR":
		status = consts.PayOrderStatusClosed
	}

	out := &client.OrderResp{
		Status:         status,
		OutTradeNo:     outTradeNo,
		ChannelOrderNo: deref(resp.TransactionId),
		SuccessTime:    successTime,
	}
	if resp.Payer != nil {
		out.ChannelUserID = deref(resp.Payer.Openid)
	}
	if resp.Amount != nil && resp.Amount.Total != nil {
		out.Price = int(*resp.Amount.Total)
	}
	return out, nil
}

// GetRefund 查询退款
func (c *WxPayClient) GetRefund(ctx context.Context, outTradeNo, outRefundNo string) (*client.RefundResp, error) {
	return nil, errors.New("退款查询功能暂未实现")
}

// ParseOrderNotify 解析支付回调
func (c *WxPayClient) ParseOrderNotify(req *client.NotifyData) (*client.OrderResp, error) {
	// 1. 构造 http.Request
	httpReq := &http.Request{
		Header: http.Header{},
		Body:   io.NopCloser(strings.NewReader(req.Body)),
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 2. 初始化 NotifyHandler
	verifier := verifiers.NewSHA256WithRSAPubkeyVerifier(c.config.PublicKeyID, *c.publicKey)
	handler, err := notify.NewRSANotifyHandler(c.config.APIV3Key, verifier)
	if err != nil {
		return nil, fmt.Errorf("创建回调处理器失败: %v", err)
	}
	// 3. 解析并验证签名
	transaction := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(context.Background(), httpReq, transaction)
	if err != nil {
		return nil, fmt.Errorf("解析支付回调失败: %w", err)
	}

	_ = notifyReq

	// 4. 转换结果
	status := 0
	switch *transaction.TradeState {
	case "SUCCESS":
		status = consts.PayOrderStatusSuccess
	case "CLOSED", "PAYERROR":
		status = consts.PayOrderStatusClosed
	}

	var successTime time.Time
	if transaction.SuccessTime != nil {
		successTime, _ = time.Parse(time.RFC3339, *transaction.SuccessTime)
	}

	// 5. 校验商户号与 AppID 归属，避免其它商户的合法回调被当作本渠道结果
	if mch := deref(transaction.Mchid); mch != "" && mch != c.config.MchID {
		return nil, fmt.Errorf("回调商户号不匹配: %s", mch)
	}
	if appID := deref(transaction.Appid); appID != "" && appID != c.config.AppID {
		return nil, fmt.Errorf("回调 AppID 不匹配: %s", appID)
	}

	out := &client.OrderResp{
		Status:         status,
		OutTradeNo:     deref(transaction.OutTradeNo),
		ChannelOrderNo: deref(transaction.TransactionId),
		SuccessTime:    successTime,
		RawData:        req.Body,
	}
	if transaction.Payer != nil {
		out.ChannelUserID = deref(transaction.Payer.Openid)
	}
	if transaction.Amount != nil && transaction.Amount.Total != nil {
		out.Price = int(*transaction.Amount.Total)
	}
	return out, nil
}

// ParseRefundNotify 解析退款回调
func (c *WxPayClient) ParseRefundNotify(req *client.NotifyData) (*client.RefundResp, error) {
	// 1. 构造 http.Request
	httpReq := &http.Request{
		Header: http.Header{},
		Body:   io.NopCloser(strings.NewReader(req.Body)),
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 2. 初始化 NotifyHandler
	verifier := verifiers.NewSHA256WithRSAPubkeyVerifier(c.config.PublicKeyID, *c.publicKey)
	handler := notify.NewNotifyHandler(c.config.APIV3Key, verifier)

	// 3. 解析并验证签名
	refundNotify := new(refunddomestic.Refund)
	_, err := handler.ParseNotifyRequest(context.Background(), httpReq, refundNotify)
	if err != nil {
		return nil, fmt.Errorf("解析退款回调失败: %w", err)
	}

	// 4. 转换结果
	status := 0
	switch *refundNotify.Status {
	case refunddomestic.STATUS_SUCCESS:
		status = consts.PayRefundStatusSuccess
	default:
		status = consts.PayRefundStatusFailure
	}

	var successTime time.Time
	if refundNotify.SuccessTime != nil {
		successTime = *refundNotify.SuccessTime
	}

	return &client.RefundResp{
		Status:          status,
		OutTradeNo:      *refundNotify.OutTradeNo,
		OutRefundNo:     *refundNotify.OutRefundNo,
		ChannelRefundNo: *refundNotify.RefundId,
		SuccessTime:     successTime,
		RawData:         req.Body,
	}, nil
}

// UnifiedTransfer 统一转账
func (c *WxPayClient) UnifiedTransfer(ctx context.Context, req *client.UnifiedTransferReq) (*client.TransferResp, error) {
	svc := transferbatch.TransferBatchApiService{Client: c.coreClient}

	// 创建转账批次
	batchReq := transferbatch.InitiateBatchTransferRequest{
		Appid:       core.String(c.config.AppID),
		OutBatchNo:  core.String(req.OutTradeNo),
		BatchName:   core.String(req.Subject),
		BatchRemark: core.String(req.Subject),
		TotalAmount: core.Int64(int64(req.Price)),
		TotalNum:    core.Int64(1),
		TransferDetailList: []transferbatch.TransferDetailInput{
			{
				OutDetailNo:    core.String(req.OutTradeNo + "_1"),
				TransferAmount: core.Int64(int64(req.Price)),
				TransferRemark: core.String(req.Subject),
				Openid:         core.String(req.ChannelUserID),
				UserName:       nil, // 如需要真实姓名需加密
			},
		},
	}

	resp, _, err := svc.InitiateBatchTransfer(ctx, batchReq)
	if err != nil {
		return &client.TransferResp{
			Status:           20, // CLOSED
			OutTradeNo:       req.OutTradeNo,
			ChannelErrorCode: "TRANSFER_ERROR",
			ChannelErrorMsg:  err.Error(),
		}, nil
	}

	return &client.TransferResp{
		Status:            consts.PayTransferStatusProcessing, // SUCCESS (or PROCESSING)
		OutTradeNo:        req.OutTradeNo,
		ChannelTransferNo: *resp.BatchId,
	}, nil
}

// GetTransfer 查询转账订单
func (c *WxPayClient) GetTransfer(ctx context.Context, outTransferNo string) (*client.TransferResp, error) {
	svc := transferbatch.TransferBatchApiService{Client: c.coreClient}

	resp, _, err := svc.GetTransferBatchByOutNo(ctx, transferbatch.GetTransferBatchByOutNoRequest{
		OutBatchNo:      core.String(outTransferNo),
		NeedQueryDetail: core.Bool(true),
		DetailStatus:    core.String("ALL"),
	})
	if err != nil {
		return nil, fmt.Errorf("查询转账批次失败: %w", err)
	}

	var transferStatus int
	var successTime time.Time
	var channelTransferNo string
	var channelErrorMsg string

	if resp.TransferBatch != nil {
		channelTransferNo = *resp.TransferBatch.BatchId
		batchStatus := *resp.TransferBatch.BatchStatus
		switch batchStatus {
		case "FINISHED":
			transferStatus = consts.PayTransferStatusSuccess
			if resp.TransferBatch.UpdateTime != nil {
				successTime = *resp.TransferBatch.UpdateTime
			}
		case "CLOSED":
			transferStatus = consts.PayTransferStatusClosed
			if resp.TransferBatch.CloseReason != nil {
				channelErrorMsg = string(*resp.TransferBatch.CloseReason)
			}
		default:
			transferStatus = consts.PayTransferStatusProcessing
		}
	}

	return &client.TransferResp{
		Status:            transferStatus,
		OutTradeNo:        outTransferNo,
		ChannelTransferNo: channelTransferNo,
		SuccessTime:       successTime,
		ChannelErrorMsg:   channelErrorMsg,
	}, nil
}

// ParseTransferNotify 解析转账回调
// 对齐 Java: AbstractWxPayClient.parseTransferNotifyV3
// 注意: 仅支持 V3 版本，V2 不支持转账回调
func (c *WxPayClient) ParseTransferNotify(req *client.NotifyData) (*client.TransferResp, error) {
	// 1. 构造 http.Request
	httpReq := &http.Request{
		Header: http.Header{},
		Body:   io.NopCloser(strings.NewReader(req.Body)),
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 2. 初始化 NotifyHandler
	verifier := verifiers.NewSHA256WithRSAPubkeyVerifier(c.config.PublicKeyID, *c.publicKey)
	handler, err := notify.NewRSANotifyHandler(c.config.APIV3Key, verifier)
	if err != nil {
		return nil, fmt.Errorf("创建回调处理器失败: %v", err)
	}

	// 3. 解析并验证签名 (使用 map 接收通用回调数据)
	content := make(map[string]interface{})
	_, err = handler.ParseNotifyRequest(context.Background(), httpReq, &content)
	if err != nil {
		return nil, fmt.Errorf("解析转账回调失败: %w", err)
	}

	// 4. 提取字段
	state, _ := content["state"].(string)
	outBizNo, _ := content["out_bill_no"].(string)
	transferBillNo, _ := content["transfer_bill_no"].(string)
	updateTimeStr, _ := content["update_time"].(string)
	failReason, _ := content["fail_reason"].(string)

	// 5. 解析时间
	var successTime time.Time
	if updateTimeStr != "" {
		successTime, _ = time.Parse(time.RFC3339, updateTimeStr)
	}

	// 6. 根据状态转换 (对齐 Java 的状态判断逻辑)
	var transferStatus int
	// ACCEPTED, PROCESSING, WAIT_USER_CONFIRM, TRANSFERING -> 处理中
	switch state {
	case "ACCEPTED", "PROCESSING", "WAIT_USER_CONFIRM", "TRANSFERING":
		transferStatus = consts.PayTransferStatusProcessing
	case "SUCCESS":
		transferStatus = consts.PayTransferStatusSuccess
	default:
		// 其他状态视为关闭
		transferStatus = consts.PayTransferStatusClosed
	}

	return &client.TransferResp{
		Status:            transferStatus,
		OutTradeNo:        outBizNo,
		ChannelTransferNo: transferBillNo,
		SuccessTime:       successTime,
		ChannelErrorMsg:   failReason,
		RawData:           req.Body,
	}, nil
}
