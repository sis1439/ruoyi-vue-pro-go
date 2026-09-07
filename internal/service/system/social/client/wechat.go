package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/pkg/cache"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"

	"crypto/sha1"
	"encoding/hex"
	"time"
)

type WeChatClient struct {
	Client *model.SocialClient
}

func NewWeChatClient(client *model.SocialClient) *WeChatClient {
	return &WeChatClient{Client: client}
}

type WeChatMiniSessionResp struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type WeChatMPAccessTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type WeChatMPUserInfoResp struct {
	OpenID   string `json:"openid"`
	Nickname string `json:"nickname"`
	Sex      int    `json:"sex"`
	Province string `json:"province"`
	City     string `json:"city"`
	Country  string `json:"country"`
	HeadImg  string `json:"headimgurl"`
	UnionID  string `json:"unionid"`
}

func (c *WeChatClient) GetAuthUser(ctx context.Context, code string, state string) (*AuthUser, error) {
	if c.Client.SocialType == consts.SocialTypeWechatMiniProgram {
		// 微信小程序
		return c.getMiniAuthUser(code)
	}
	// 微信公众号
	return c.getMPAuthUser(code)
}

// 小程序登录
func (c *WeChatClient) getMiniAuthUser(code string) (*AuthUser, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		c.Client.ClientId, c.Client.ClientSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var session WeChatMiniSessionResp
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	if session.ErrCode != 0 {
		return nil, errors.NewBizError(1002004003, fmt.Sprintf("微信登录失败: %s", session.ErrMsg))
	}

	return &AuthUser{
		Openid:       session.OpenID,
		Token:        session.SessionKey, // 小程序使用 session_key 作为 token 凭证
		RawTokenInfo: toJson(session),
		// 小程序登录不直接返回用户信息，需要前端 getUserProfile 配合，这里先返回空
		Nickname:    "",
		Avatar:      "",
		RawUserInfo: "{}",
	}, nil
}

// 公众号登录
func (c *WeChatClient) getMPAuthUser(code string) (*AuthUser, error) {
	// 1. 获取 Access Token
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		c.Client.ClientId, c.Client.ClientSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp WeChatMPAccessTokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}
	if tokenResp.ErrCode != 0 {
		return nil, errors.NewBizError(1002004003, fmt.Sprintf("微信登录失败: %s", tokenResp.ErrMsg))
	}

	// 2. 获取用户信息
	userUrl := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		tokenResp.AccessToken, tokenResp.OpenID)

	userResp, err := http.Get(userUrl)
	if err != nil {
		return nil, err
	}
	defer userResp.Body.Close()

	var userInfo WeChatMPUserInfoResp
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &AuthUser{
		Openid:       tokenResp.OpenID,
		Token:        tokenResp.AccessToken,
		RawTokenInfo: toJson(tokenResp),
		Nickname:     userInfo.Nickname,
		Avatar:       userInfo.HeadImg,
		RawUserInfo:  toJson(userInfo),
	}, nil
}

func toJson(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (c *WeChatClient) GetAuthUrl(state string, redirectUri string) string {
	if c.Client.SocialType == consts.SocialTypeWechatMiniProgram {
		return "" // 小程序不支持跳转登录
	}
	// 公众号
	// scope: snsapi_userinfo (需关注?) or snsapi_base (静默).
	// RuoYi explicitly uses snsapi_userinfo usually for full info.
	return fmt.Sprintf("https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=%s#wechat_redirect",
		c.Client.ClientId, redirectUri, state)
}

// GetMobile 获取微信手机号
func (c *WeChatClient) GetMobile(ctx context.Context, code string) (string, error) {
	if c.Client.SocialType != consts.SocialTypeWechatMiniProgram {
		return "", fmt.Errorf("socialType %d does not support GetMobile", c.Client.SocialType)
	}

	// 1. 获取 Access Token
	accessToken, err := c.getWxaAccessToken()
	if err != nil {
		return "", err
	}

	// 2. 获取手机号
	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s", accessToken)
	payload, _ := json.Marshal(map[string]string{"code": code})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var phoneResp struct {
		ErrCode   int    `json:"errcode"`
		ErrMsg    string `json:"errmsg"`
		PhoneInfo struct {
			PurePhoneNumber string `json:"purePhoneNumber"`
		} `json:"phone_info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&phoneResp); err != nil {
		return "", err
	}
	if phoneResp.ErrCode != 0 {
		return "", fmt.Errorf("wechat get mobile failed: %s", phoneResp.ErrMsg)
	}

	return phoneResp.PhoneInfo.PurePhoneNumber, nil
}

func (c *WeChatClient) getWxaAccessToken() (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		c.Client.ClientId, c.Client.ClientSecret)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if tokenResp.ErrCode != 0 {
		return "", fmt.Errorf("wechat get access token failed: %s", tokenResp.ErrMsg)
	}
	return tokenResp.AccessToken, nil
}

// CreateJsapiSignature 创建微信 JSAPI 签名
func (c *WeChatClient) CreateJsapiSignature(ctx context.Context, url string) (*JsapiSignature, error) {
	// 1. 获取 ticket
	ticket, err := c.getJsapiTicket(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 生成随机串和时间戳
	nonceStr := utils.GenerateRandomString(16)
	timestamp := time.Now().Unix()

	// 3. 计算签名
	// 规则：noncestr=xxx&jsapi_ticket=xxx&timestamp=xxx&url=xxx (按字典序)
	plainText := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s",
		ticket, nonceStr, timestamp, url)
	h := sha1.New()
	h.Write([]byte(plainText))
	signature := hex.EncodeToString(h.Sum(nil))

	return &JsapiSignature{
		AppID:     c.Client.ClientId,
		NonceStr:  nonceStr,
		Timestamp: timestamp,
		URL:       url,
		Signature: signature,
	}, nil
}

func (c *WeChatClient) getJsapiTicket(ctx context.Context) (string, error) {
	tenantID, ok := pkgcontext.TenantID(ctx)
	if !ok || c.Client == nil || c.Client.TenantID != tenantID || c.Client.Status != 0 {
		return "", fmt.Errorf("unavailable WeChat client for tenant")
	}
	redisKey := fmt.Sprintf("mall:tenant:%d:wechat:jsapi_ticket:%s", tenantID, c.Client.ClientId)

	// 1. 尝试从缓存获取
	if cache.RDB != nil {
		ticket, err := cache.RDB.Get(ctx, redisKey).Result()
		if err == nil && ticket != "" {
			return ticket, nil
		}
	}

	// 2. 从微信服务器获取
	accessToken, err := c.getWxaAccessToken() // 注意：这里使用了小程序的，实际上公众号和性质一样，但 appId/secret 需对应
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi", accessToken)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var ticketResp struct {
		ErrCode   int    `json:"errcode"`
		ErrMsg    string `json:"errmsg"`
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ticketResp); err != nil {
		return "", err
	}
	if ticketResp.ErrCode != 0 {
		return "", fmt.Errorf("wechat get jsapi ticket failed: %s", ticketResp.ErrMsg)
	}

	// 3. 存入内存
	if cache.RDB != nil {
		// 提前 10 分钟过期
		cache.RDB.Set(ctx, redisKey, ticketResp.Ticket, time.Duration(ticketResp.ExpiresIn-600)*time.Second)
	}

	return ticketResp.Ticket, nil
}

func (c *WeChatClient) GetWxaQrcode(ctx context.Context, path string, width int) ([]byte, error) {
	// TODO: 实现获取微信小程序码逻辑
	return nil, nil
}

type WeChatWxaTemplateListResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Data    []struct {
		PriTmplID string `json:"priTmplId"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		Example   string `json:"example"`
		Type      int    `json:"type"`
	} `json:"data"`
}

func (c *WeChatClient) GetSubscribeTemplateList(ctx context.Context) ([]*WxaSubscribeTemplate, error) {
	// 1. 获取 Access Token
	accessToken, err := c.getWxaAccessToken()
	if err != nil {
		return nil, err
	}

	// 2. 获取模板列表
	url := fmt.Sprintf("https://api.weixin.qq.com/wxaapi/newtmpl/gettemplate?access_token=%s", accessToken)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listResp WeChatWxaTemplateListResp
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	if listResp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat get subscribe template list failed: %s", listResp.ErrMsg)
	}

	// 3. 转换
	res := make([]*WxaSubscribeTemplate, len(listResp.Data))
	for i, item := range listResp.Data {
		res[i] = &WxaSubscribeTemplate{
			ID:      item.PriTmplID,
			Title:   item.Title,
			Content: item.Content,
			Example: item.Example,
			Type:    item.Type,
		}
	}

	return res, nil
}
