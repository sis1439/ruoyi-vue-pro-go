package promotion

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

// KefuMessageCreateReq 发送消息请求
type KefuMessageCreateReq struct {
	SenderID       int64  `json:"senderId"`
	SenderType     int    `json:"senderType"`
	ConversationID int64  `json:"conversationId"`                 // 会话编号 (可选，发送给客服时如果不传，则自动查找或创建)
	ContentType    int    `json:"contentType" binding:"required"` // 1-文本 2-图片 3-商品 4-订单
	Content        string `json:"content" binding:"required"`     // 消息内容
}

// KefuMessageListReq 消息列表请求 (对齐 Java KeFuMessageListReqVO)
type KefuMessageListReq struct {
	ConversationID int64      `form:"conversationId" binding:"required"`      // 会话编号
	CreateTime     *time.Time `form:"createTime"`                             // 发送时间（用于分页）
	Limit          int        `form:"limit" binding:"required,min=1,max=100"` // 每次查询条数，默认10，最大100
}

// KefuMessagePageReq 消息分页请求 (保留用于其他场景)
type KefuMessagePageReq struct {
	pagination.PageParam
	ConversationID int64 `form:"conversationId" binding:"required"` // 会话编号
}

// KefuConversationPageReq 会话列表请求
type KefuConversationPageReq struct {
	pagination.PageParam
	// 可扩展查询条件，如用户昵称等
}

// KeFuConversationUpdatePinnedReq 置顶客服会话请求 (对齐 Java KeFuConversationUpdatePinnedReqVO)
type KeFuConversationUpdatePinnedReq struct {
	ID          int64 `json:"id" binding:"required"`          // 会话编号
	AdminPinned *bool `json:"adminPinned" binding:"required"` // 管理端置顶
}

// KefuMessageResp 客服消息 Response VO
type KefuMessageResp struct {
	ID             int64              `json:"id"`
	ConversationID int64              `json:"conversationId"`
	ContentType    int                `json:"contentType"`
	Content        string             `json:"content"`
	SenderID       int64              `json:"senderId"`
	SenderType     int                `json:"senderType"`
	SenderAvatar   string             `json:"senderAvatar"`
	ReceiverID     int64              `json:"receiverId"`
	ReceiverType   int                `json:"receiverType"`
	ReadStatus     bool               `json:"readStatus"`
	CreateTime     types.JsonDateTime `json:"createTime"`
}

// KefuConversationResp 客服会话 Response VO
type KefuConversationResp struct {
	ID                      int64     `json:"id"`
	UserID                  int64     `json:"userId"`
	UserNickname            string    `json:"userNickname"`
	UserAvatar              string    `json:"userAvatar"`
	LastMessageTime         time.Time `json:"lastMessageTime"`
	LastMessageContent      string    `json:"lastMessageContent"`
	LastMessageContentType  int       `json:"lastMessageContentType"`
	AdminPinned             bool      `json:"adminPinned"`
	UserDeleted             bool      `json:"userDeleted"`
	AdminDeleted            bool      `json:"adminDeleted"`
	AdminUnreadMessageCount int       `json:"adminUnreadMessageCount"`
	CreateTime              time.Time `json:"createTime"`
}
