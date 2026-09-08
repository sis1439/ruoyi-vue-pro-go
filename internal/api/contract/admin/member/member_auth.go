package member

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

// AppAuthLoginReq 手机+密码登录
type AppAuthLoginReq struct {
	Mobile   string `json:"mobile" binding:"required,len=11"` // 简单校验
	Password string `json:"password" binding:"required,min=4,max=16"`
	// Social
	SocialType  int    `json:"socialType"`
	SocialCode  string `json:"socialCode"`
	SocialState string `json:"socialState"`
}

// AppAuthSmsLoginReq 手机+验证码登录
type AppAuthSmsLoginReq struct {
	Mobile string `json:"mobile" binding:"required,len=11"`
	Code   string `json:"code" binding:"required,min=4,max=6"`
	// Social
	SocialType  int    `json:"socialType"`
	SocialCode  string `json:"socialCode"`
	SocialState string `json:"socialState"`
}

// AppAuthSmsSendReq 发送手机验证码
type AppAuthSmsSendReq struct {
	Mobile string `json:"mobile" binding:"omitempty,len=11"`
	Scene  int    `json:"scene" binding:"required,oneof=1 2 3 4"` // 对应 SmsSceneEnum
}

// AppAuthSmsValidateReq 校验手机验证码
type AppAuthSmsValidateReq struct {
	Mobile string `json:"mobile" binding:"omitempty,len=11"`
	Code   string `json:"code" binding:"required"`
	Scene  int    `json:"scene" binding:"required,oneof=1 2 3 4"`
}

// AppAuthSocialLoginReq 社交登录
type AppAuthSocialLoginReq struct {
	Type  int32  `json:"type" binding:"required"`
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// AppAuthWeixinMiniAppLoginReq 微信小程序登录
type AppAuthWeixinMiniAppLoginReq struct {
	PhoneCode string `json:"phoneCode" binding:"required"`
	LoginCode string `json:"loginCode" binding:"required"`
	State     string `json:"state" binding:"required"`
}

// AppAuthLoginResp 登录响应
type AppAuthLoginResp struct {
	UserID       int64              `json:"userId"`
	AccessToken  string             `json:"accessToken"`
	RefreshToken string             `json:"refreshToken"`
	ExpiresTime  types.JsonDateTime `json:"expiresTime"`
	OpenID       string             `json:"openid"`
}

// AppAuthWeixinJsapiSignatureResp 微信 JSAPI 签名响应
type AppAuthWeixinJsapiSignatureResp struct {
	AppID     string `json:"appId"`
	NonceStr  string `json:"nonceStr"`
	Timestamp int64  `json:"timestamp"`
	URL       string `json:"url"`
	Signature string `json:"signature"`
}
