package member

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type MemberSignInConfigResp struct {
	ID         int64     `json:"id"`
	Day        int       `json:"day"`
	Point      int       `json:"point"`
	Experience int       `json:"experience"`
	Status     int       `json:"status"`
	CreateTime time.Time `json:"createTime"`
}

type MemberSignInConfigCreateReq struct {
	Day        int `json:"day" binding:"required"`
	Point      int `json:"point" binding:"required"`
	Experience int `json:"experience"`
	Status     int `json:"status"`
}

type MemberSignInConfigUpdateReq struct {
	ID         int64 `json:"id" binding:"required"`
	Day        int   `json:"day" binding:"required"`
	Point      int   `json:"point" binding:"required"`
	Experience int   `json:"experience"`
	Status     int   `json:"status"`
}

// MemberSignInRecordResp 签到记录响应 (Admin)
type MemberSignInRecordResp struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"userId"`
	Nickname   string    `json:"nickname"`
	Day        int       `json:"day"`
	Point      int       `json:"point"`
	Experience int       `json:"experience"`
	CreateTime time.Time `json:"createTime"`
}

// AppMemberSignInRecordResp App签到记录响应
type AppMemberSignInRecordResp struct {
	ID         int64              `json:"id"`
	Day        int                `json:"day"`
	Point      int                `json:"point"`
	Experience int                `json:"experience"`
	CreateTime types.JsonDateTime `json:"createTime"`
}

// AppMemberSignInRecordSummaryResp App签到统计响应
type AppMemberSignInRecordSummaryResp struct {
	TotalDay      int  `json:"totalDay"`
	ContinuousDay int  `json:"continuousDay"`
	TodaySignIn   bool `json:"todaySignIn"`
}

// AppMemberSignInConfigResp App签到配置响应 (对齐 Java AppMemberSignInConfigRespVO)
type AppMemberSignInConfigResp struct {
	Day   int `json:"day"`   // 签到第 x 天
	Point int `json:"point"` // 奖励积分
}

type MemberSignInRecordPageReq struct {
	pagination.PageParam
	UserID     int64    `form:"userId"`
	Nickname   string   `form:"nickname"`
	Day        *int     `form:"day"`
	CreateTime []string `form:"createTime[]"`
}

// AppMemberSignInRecordPageReq App签到记录分页请求
type AppMemberSignInRecordPageReq struct {
	pagination.PageParam
}
