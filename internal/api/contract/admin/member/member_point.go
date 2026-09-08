package member

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type MemberPointRecordResp struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"userId"`
	Nickname    string    `json:"nickname"`
	BizID       string    `json:"bizId"`
	BizType     int       `json:"bizType"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Point       int       `json:"point"`
	TotalPoint  int       `json:"totalPoint"`
	CreateTime  time.Time `json:"createTime"`
}

type AppMemberPointRecordResp struct {
	ID          int64              `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Point       int                `json:"point"`
	CreateTime  types.JsonDateTime `json:"createTime"`
}

type MemberPointRecordPageReq struct {
	pagination.PageParam
	UserID     int64    `form:"userId"`
	Nickname   string   `form:"nickname"`
	BizType    *int     `form:"bizType"`
	Title      string   `form:"title"`
	CreateTime []string `form:"createTime[]"`
}

type AppMemberPointRecordPageReq struct {
	CreateTime []string `form:"createTime[]"`
	pagination.PageParam
	AddStatus *bool `form:"addStatus"` // true: 增加, false: 减少
}
