package model

import (
	"time"

	"gorm.io/datatypes"
)

// SystemSmsChannel 短信渠道表
type SystemSmsChannel struct {
	ID          int64  `gorm:"primaryKey;autoIncrement;comment:渠道编号" json:"id"`
	Signature   string `gorm:"size:10;not null;comment:短信签名" json:"signature"`
	Code        string `gorm:"size:63;not null;comment:渠道编码" json:"code"`
	Status      int32  `gorm:"not null;comment:启用状态" json:"status"`
	Remark      string `gorm:"size:255;comment:备注" json:"remark"`
	ApiKey      string `gorm:"size:63;not null;comment:短信 API 的账号" json:"apiKey"`
	ApiSecret   string `gorm:"size:63;comment:短信 API 的密钥" json:"apiSecret"`
	CallbackUrl string `gorm:"size:255;comment:短信发送回调 URL" json:"callbackUrl"`

	// Base fields
	TenantBaseDO
}

func (SystemSmsChannel) TableName() string {
	return "system_sms_channel"
}

// SystemSmsTemplate 短信模板表
type SystemSmsTemplate struct {
	ID            int64                       `gorm:"primaryKey;autoIncrement;comment:自增编号" json:"id"`
	Type          int32                       `gorm:"not null;comment:短信类型" json:"type"`
	Status        int32                       `gorm:"not null;comment:启用状态" json:"status"`
	Code          string                      `gorm:"size:63;not null;comment:模板编码" json:"code"`
	Name          string                      `gorm:"size:63;not null;comment:模板名称" json:"name"`
	Content       string                      `gorm:"size:255;not null;comment:模板内容" json:"content"`
	Params        datatypes.JSONSlice[string] `gorm:"serializer:json;comment:参数数组" json:"params"`
	Remark        string                      `gorm:"size:255;comment:备注" json:"remark"`
	ApiTemplateId string                      `gorm:"size:63;not null;comment:短信 API 的模板编号" json:"apiTemplateId"`
	ChannelId     int64                       `gorm:"not null;comment:短信渠道编号" json:"channelId"`
	ChannelCode   string                      `gorm:"size:63;not null;comment:短信渠道编码" json:"channelCode"`

	// Base fields
	TenantBaseDO
}

func (SystemSmsTemplate) TableName() string {
	return "system_sms_template"
}

// SystemSmsLog 短信日志表
type SystemSmsLog struct {
	ID              int64                  `gorm:"primaryKey;autoIncrement;comment:自增编号" json:"id"`
	ChannelId       int64                  `gorm:"not null;comment:短信渠道编号" json:"channelId"`
	ChannelCode     string                 `gorm:"size:63;not null;comment:短信渠道编码" json:"channelCode"`
	TemplateId      int64                  `gorm:"not null;comment:模板编号" json:"templateId"`
	TemplateCode    string                 `gorm:"size:63;not null;comment:模板编码" json:"templateCode"`
	TemplateType    int32                  `gorm:"not null;comment:短信类型" json:"templateType"`
	TemplateContent string                 `gorm:"size:255;not null;comment:模板内容" json:"templateContent"`
	TemplateParams  map[string]interface{} `gorm:"serializer:json;comment:模板参数" json:"templateParams"`
	ApiTemplateId   string                 `gorm:"size:63;not null;comment:短信 API 的模板编号" json:"apiTemplateId"`
	Mobile          string                 `gorm:"size:11;not null;comment:手机号" json:"mobile"`
	UserId          int64                  `gorm:"comment:用户编号" json:"userId"`
	UserType        int32                  `gorm:"comment:用户类型" json:"userType"`
	SendStatus      int32                  `gorm:"not null;default:0;comment:发送状态" json:"sendStatus"`
	SendTime        *time.Time             `gorm:"comment:发送时间" json:"sendTime"`
	ApiSendCode     string                 `gorm:"size:63;comment:短信 API 发送结果的编码" json:"apiSendCode"`
	ApiSendMsg      string                 `gorm:"size:255;comment:短信 API 发送失败的提示" json:"apiSendMsg"`
	ApiRequestId    string                 `gorm:"size:63;comment:短信 API 发送返回的唯一请求 ID" json:"apiRequestId"`
	ApiSerialNo     string                 `gorm:"size:63;comment:短信 API 发送返回的序号" json:"apiSerialNo"`
	ReceiveStatus   int32                  `gorm:"not null;default:0;comment:接收状态" json:"receiveStatus"`
	ReceiveTime     *time.Time             `gorm:"comment:接收时间" json:"receiveTime"`
	ApiReceiveCode  string                 `gorm:"size:63;comment:短信 API 接收结果的编码" json:"apiReceiveCode"`
	ApiReceiveMsg   string                 `gorm:"size:255;comment:短信 API 接收结果的提示" json:"apiReceiveMsg"`

	// Base fields
	TenantBaseDO
}

func (SystemSmsLog) TableName() string {
	return "system_sms_log"
}

// SystemSmsCode 短信验证码表
type SystemSmsCode struct {
	ID         int64      `gorm:"primaryKey;autoIncrement;comment:自增编号" json:"id"`
	Mobile     string     `gorm:"size:11;not null;index;comment:手机号" json:"mobile"`
	Code       string     `gorm:"size:6;not null;comment:验证码" json:"code"`
	Scene      int32      `gorm:"not null;comment:发送场景" json:"scene"`
	Used       bool       `gorm:"not null;default:false;comment:是否使用" json:"used"`
	UsedTime   *time.Time `gorm:"comment:使用时间" json:"usedTime"`
	TodayIndex int32      `gorm:"not null;default:1;comment:今日发送的第几条" json:"todayIndex"`
	CreateIp   string     `gorm:"size:30;comment:创建 IP" json:"createIp"`
	UsedIp     string     `gorm:"size:30;comment:使用 IP" json:"usedIp"`

	// Base fields
	TenantBaseDO
}

func (SystemSmsCode) TableName() string {
	return "system_sms_code"
}
