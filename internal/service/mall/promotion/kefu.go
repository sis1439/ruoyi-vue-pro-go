package promotion

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	tenant "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"log"
	"time"

	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/pkg/websocket"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"gorm.io/gorm"
)

// KefuService 客服 Service
type KefuService interface {
	// CreateMessage 发送消息
	CreateMessage(ctx context.Context, r promotion2.KefuMessageCreateReq, senderID int64, senderType int) (int64, error)
	// GetMessagePage 获得消息分页
	GetMessagePage(ctx context.Context, r promotion2.KefuMessagePageReq) (*pagination.PageResult[promotion2.KefuMessageResp], error)
	// GetMessageList 获得消息列表 (对齐 Java - 管理员端)
	GetMessageList(ctx context.Context, r promotion2.KefuMessageListReq) ([]promotion2.KefuMessageResp, error)
	// GetMessageListForMember 【会员】获得消息列表 (对齐 Java)
	GetMessageListForMember(ctx context.Context, r promotion2.KefuMessageListReq, userID int64) ([]promotion2.KefuMessageResp, error)
	// UpdateMessageReadStatus 更新消息已读状态
	UpdateMessageReadStatus(ctx context.Context, conversationID, senderID int64, senderType int) error
	// GetConversationPage 获得会话分页
	GetConversationPage(ctx context.Context, r promotion2.KefuConversationPageReq) (*pagination.PageResult[promotion2.KefuConversationResp], error)
	// GetConversationList 获得会话列表 (对齐 Java)
	GetConversationList(ctx context.Context) ([]promotion2.KefuConversationResp, error)
	// UpdateConversationPinned 置顶/取消置顶会话
	UpdateConversationPinned(ctx context.Context, r promotion2.KeFuConversationUpdatePinnedReq) error
	// DeleteConversation 删除会话
	DeleteConversation(ctx context.Context, id int64) error
	// GetConversation 获得会话
	GetConversation(ctx context.Context, id int64) (*promotion.PromotionKefuConversation, error)
	// GetConversationByUserId 【会员】获得客服会话 (对齐 Java)
	GetConversationByUserId(ctx context.Context, userID int64) (*promotion.PromotionKefuConversation, error)
}

type kefuService struct {
	q             *query.Query
	memberUserSvc *member.MemberUserService
	systemUserSvc *system.UserService
	wsManager     *websocket.Manager
}

func NewKefuService(q *query.Query, memberUserSvc *member.MemberUserService, systemUserSvc *system.UserService, wsManager *websocket.Manager) KefuService {
	return &kefuService{
		q:             q,
		memberUserSvc: memberUserSvc,
		systemUserSvc: systemUserSvc,
		wsManager:     wsManager,
	}
}

// CreateMessage 发送消息 (对齐 Java: KeFuMessageServiceImpl.sendKefuMessage)
// senderType=1 为会员端发送，senderType=2 为管理员端发送
func (s *kefuService) CreateMessage(ctx context.Context, r promotion2.KefuMessageCreateReq, senderID int64, senderType int) (int64, error) {
	var conversation *promotion.PromotionKefuConversation
	var err error

	// 1. 处理会话 (不同端逻辑略有不同)
	if senderType == 1 { // Member (App)
		// App端：尝试获取或创建会话
		conversation, err = s.getOrCreateConversation(ctx, senderID)
		if err != nil {
			return 0, err
		}
		r.ConversationID = conversation.ID
	} else { // Admin
		// Admin端：必须提供 conversationId
		if r.ConversationID == 0 {
			return 0, errors.NewBizError(400, "客服发送消息必须指定会话ID")
		}
		conversation, err = s.validateKefuConversationExists(ctx, r.ConversationID)
		if err != nil {
			return 0, err
		}
		// 校验接收人是否存在
		if err := s.validateReceiverExist(ctx, conversation.UserID, 1); err != nil {
			return 0, err
		}
	}

	// 2. 保存消息
	msg := &promotion.PromotionKefuMessage{
		ConversationID: r.ConversationID,
		SenderID:       senderID,
		SenderType:     senderType,
		ContentType:    r.ContentType,
		Content:        r.Content,
		ReadStatus:     false,
	}

	// 设置接收人 (对齐 Java 逻辑)
	if senderType == 1 { // Member -> Admin
		// Java App 端不显式设置 receiverId/receiverType，使用数据库默认值
		msg.ReceiverID = 0
		msg.ReceiverType = 0
	} else { // Admin -> Member
		// Java: kefuMessage.setReceiverId(conversation.getUserId()).setReceiverType(MEMBER)
		msg.ReceiverID = conversation.UserID
		msg.ReceiverType = 1
	}

	if err = s.q.PromotionKefuMessage.WithContext(ctx).Create(msg); err != nil {
		return 0, err
	}

	// 3. 更新会话 (LastMessage info, UnreadCount)
	if err := s.updateConversationLastMessage(ctx, conversation, msg); err != nil {
		return 0, err
	}

	// 4. 发送 WebSocket 通知 (严格对齐 Java 逻辑)
	// 构建消息响应对象
	msgResp := promotion2.KefuMessageResp{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		SenderType:     msg.SenderType,
		ReceiverID:     msg.ReceiverID,
		ReceiverType:   msg.ReceiverType,
		ContentType:    msg.ContentType,
		Content:        msg.Content,
		ReadStatus:     false,
		CreateTime:     types.ToJsonDateTime(msg.CreateTime),
	}
	// 获取发送者头像
	if senderType == 1 { // Member
		if user, _ := s.memberUserSvc.GetUser(ctx, senderID); user != nil {
			msgResp.SenderAvatar = user.Avatar
		}
	} else { // Admin
		if user, _ := s.systemUserSvc.GetUser(ctx, senderID); user != nil && user.UserRespVO != nil {
			msgResp.SenderAvatar = user.UserRespVO.Avatar
		}
	}

	// 发送 WebSocket 通知 (对齐 Java 双重通知机制)
	if senderType == 2 { // Admin 发送
		// Java: sendAsyncMessageToMember(conversation.getUserId(), KEFU_MESSAGE_TYPE, message)
		s.sendKefuMessageNotify(ctx, conversation.UserID, 1, consts.KefuMessageType, msgResp)
		// Java: sendAsyncMessageToAdmin(KEFU_MESSAGE_TYPE, message) - 通知所有管理员
		s.sendKefuMessageNotify(ctx, 0, 2, consts.KefuMessageType, msgResp)
	} else { // Member (App) 发送
		// Java: sendAsyncMessageToAdmin(KEFU_MESSAGE_TYPE, message) - 通知所有管理员
		s.sendKefuMessageNotify(ctx, 0, 2, consts.KefuMessageType, msgResp)
		// Java: sendAsyncMessageToMember(conversation.getUserId(), KEFU_MESSAGE_TYPE, message) - 通知会员自己
		s.sendKefuMessageNotify(ctx, conversation.UserID, 1, consts.KefuMessageType, msgResp)
	}

	return msg.ID, nil
}

// updateConversationLastMessage 更新会话的最后一条消息，并处理未读数和删除状态
func (s *kefuService) updateConversationLastMessage(ctx context.Context, convo *promotion.PromotionKefuConversation, msg *promotion.PromotionKefuMessage) error {
	convoRepo := s.q.PromotionKefuConversation

	updates := map[string]interface{}{
		"last_message_time":         msg.CreateTime,
		"last_message_content":      msg.Content,
		"last_message_content_type": msg.ContentType,
	}

	// 2.1 更新管理员未读消息数
	if msg.SenderType == 1 { // Member sent
		// increment admin_unread_message_count
		// 使用 UnderlyingDB 避免 gen 代码未更新问题
		// UPDATE promotion_kefu_conversation SET admin_unread_message_count = admin_unread_message_count + 1 WHERE id = ?
		if err := convoRepo.WithContext(ctx).UnderlyingDB().
			Where("id = ?", convo.ID).
			UpdateColumn("admin_unread_message_count", gorm.Expr("admin_unread_message_count + 1")).Error; err != nil {
			return err
		}
	}

	// 2.2 会员用户发送消息时，如果管理员删除过会话则进行恢复
	if msg.SenderType == 1 && bool(convo.AdminDeleted) {
		updates["admin_deleted"] = model.BitBool(false)
	}

	_, err := convoRepo.WithContext(ctx).Where(convoRepo.ID.Eq(convo.ID)).Updates(updates)
	return err
}

// UpdateMessageReadStatus 更新消息已读状态 (对齐 Java: updateKeFuMessageReadStatus)
func (s *kefuService) UpdateMessageReadStatus(ctx context.Context, conversationID, senderID int64, senderType int) error {
	// 1.1 校验会话是否存在
	convo, err := s.validateKefuConversationExists(ctx, conversationID)
	if err != nil {
		return err
	}

	// 1.2 如果是会员端处理已读，需要校验会话属于该用户
	if senderType == 1 && convo.UserID != senderID {
		return errors.NewBizError(404, "会话不存在")
	}

	msgRepo := s.q.PromotionKefuMessage
	convoRepo := s.q.PromotionKefuConversation

	// 1.3 确定要标记为已读的消息发送者类型
	// 如果我是 User(Type 1)，要标记 Admin(Type 2) 发来的消息为已读
	// 如果我是 Admin(Type 2)，要标记 User(Type 1) 发来的消息为已读
	targetSenderType := 0
	if senderType == 1 {
		targetSenderType = 2 // User reads Admin's messages
	} else {
		targetSenderType = 1 // Admin reads User's messages
	}

	// 1.4 查询会话所有的未读消息 (用于后续 WS 通知)
	unreadMessages, err := msgRepo.WithContext(ctx).
		Where(msgRepo.ConversationID.Eq(conversationID)).
		Where(msgRepo.SenderType.Eq(targetSenderType)).
		Where(msgRepo.ReadStatus.Eq(model.BitBool(false))).
		Find()
	if err != nil {
		return err
	}

	// 如果没有未读消息，直接返回
	if len(unreadMessages) == 0 {
		return nil
	}

	// 2.1 更新未读消息状态为已读
	_, err = msgRepo.WithContext(ctx).
		Where(msgRepo.ConversationID.Eq(conversationID)).
		Where(msgRepo.SenderType.Eq(targetSenderType)).
		Where(msgRepo.ReadStatus.Eq(model.BitBool(false))).
		Update(msgRepo.ReadStatus, model.BitBool(true))
	if err != nil {
		return err
	}

	// 2.2 将管理员未读消息计数更新为零 (Only if Admin is reading)
	if senderType == 2 { // Admin reading
		if err = convoRepo.WithContext(ctx).UnderlyingDB().
			Where("id = ?", conversationID).
			Update("admin_unread_message_count", 0).Error; err != nil {
			return err
		}
	}

	// 2.3 发送 WebSocket 通知 (对齐 Java 逻辑)
	// 找到第一条会员发送的消息，用于通知
	var memberSentMessage *promotion.PromotionKefuMessage
	for _, msg := range unreadMessages {
		if msg.SenderType == 1 { // Member sent
			memberSentMessage = msg
			break
		}
	}

	if memberSentMessage != nil && senderType == 2 { // Admin 阅读了会员消息
		// 构建通知消息
		readNotify := promotion2.KefuMessageResp{
			ConversationID: conversationID,
		}

		// 2.3.1 发送消息通知会员，管理员已读
		s.sendKefuMessageNotify(ctx, memberSentMessage.SenderID, 1, consts.KefuMessageReadStatusChange, readNotify)

		// 2.3.2 通知所有管理员消息已读
		s.sendKefuMessageNotify(ctx, 0, 2, consts.KefuMessageReadStatusChange, readNotify)
	}

	return nil
}

// GetMessageList 获得消息列表 (对齐 Java Controller: AdminKeFuMessageController.getKeFuMessageList)
func (s *kefuService) GetMessageList(ctx context.Context, r promotion2.KefuMessageListReq) ([]promotion2.KefuMessageResp, error) {
	msgRepo := s.q.PromotionKefuMessage
	q := msgRepo.WithContext(ctx).Where(msgRepo.ConversationID.Eq(r.ConversationID))

	if r.CreateTime != nil {
		q = q.Where(msgRepo.CreateTime.Lt(*r.CreateTime))
	}

	list, err := q.Order(msgRepo.CreateTime.Desc()).Limit(r.Limit).Find()
	if err != nil {
		return nil, err
	}

	resList := make([]promotion2.KefuMessageResp, len(list))

	// 收集 Admin 发送者 ID 用于批量查询头像
	adminSenderIDs := make(map[int64]struct{})
	for _, v := range list {
		if v.SenderType == 2 { // Admin
			adminSenderIDs[v.SenderID] = struct{}{}
		}
	}

	// 批量查询 Admin 用户头像 (对齐 Java: adminUserApi.getUserMap)
	adminAvatarMap := make(map[int64]string)
	for adminID := range adminSenderIDs {
		if user, _ := s.systemUserSvc.GetUser(ctx, adminID); user != nil && user.UserRespVO != nil {
			adminAvatarMap[adminID] = user.UserRespVO.Avatar
		}
	}

	for i, v := range list {
		resList[i] = promotion2.KefuMessageResp{
			ID:             v.ID,
			ConversationID: v.ConversationID,
			SenderID:       v.SenderID,
			SenderType:     v.SenderType,
			ReceiverID:     v.ReceiverID,
			ReceiverType:   v.ReceiverType,
			ContentType:    v.ContentType,
			Content:        v.Content,
			ReadStatus:     bool(v.ReadStatus),
			CreateTime:     types.ToJsonDateTime(v.CreateTime),
		}
		// 填充 Admin 发送者头像 (对齐 Java Controller 逻辑)
		if v.SenderType == 2 { // Admin
			if avatar, ok := adminAvatarMap[v.SenderID]; ok {
				resList[i].SenderAvatar = avatar
			}
		}
	}
	return resList, nil
}

// GetMessagePage 获得消息分页 (保持兼容，调用 List 逻辑或独立)
func (s *kefuService) GetMessagePage(ctx context.Context, r promotion2.KefuMessagePageReq) (*pagination.PageResult[promotion2.KefuMessageResp], error) {
	// 简单的分页实现，复用逻辑
	msgRepo := s.q.PromotionKefuMessage
	q := msgRepo.WithContext(ctx).Where(msgRepo.ConversationID.Eq(r.ConversationID))
	list, count, err := q.Order(msgRepo.CreateTime.Desc()).FindByPage(r.PageNo-1, r.PageSize)
	if err != nil {
		return nil, err
	}
	resList := make([]promotion2.KefuMessageResp, len(list))
	for i, v := range list {
		resList[i] = promotion2.KefuMessageResp{
			ID:             v.ID,
			ConversationID: v.ConversationID,
			SenderID:       v.SenderID,
			SenderType:     v.SenderType,
			ReceiverID:     v.ReceiverID,
			ReceiverType:   v.ReceiverType,
			ContentType:    v.ContentType,
			Content:        v.Content,
			ReadStatus:     bool(v.ReadStatus),
			CreateTime:     types.ToJsonDateTime(v.CreateTime),
		}
	}
	return &pagination.PageResult[promotion2.KefuMessageResp]{List: resList, Total: count}, nil
}

// GetConversationList 获得会话列表 (对齐 Java: KeFuConversationController.getConversationList)
func (s *kefuService) GetConversationList(ctx context.Context) ([]promotion2.KefuConversationResp, error) {
	convoRepo := s.q.PromotionKefuConversation
	var list []*promotion.PromotionKefuConversation
	err := convoRepo.WithContext(ctx).UnderlyingDB().
		Where("admin_deleted = ?", false).
		Order("admin_pinned DESC").
		Order("last_message_time DESC").
		Find(&list).Error

	if err != nil {
		return nil, err
	}

	resList := make([]promotion2.KefuConversationResp, len(list))
	for i, v := range list {
		resList[i] = promotion2.KefuConversationResp{
			ID:                      v.ID,
			UserID:                  v.UserID,
			LastMessageTime:         v.LastMessageTime,
			LastMessageContent:      v.LastMessageContent,
			LastMessageContentType:  v.LastMessageContentType,
			AdminPinned:             bool(v.AdminPinned),
			UserDeleted:             bool(v.UserDeleted),
			AdminDeleted:            bool(v.AdminDeleted),
			AdminUnreadMessageCount: v.AdminUnreadMessageCount,
			CreateTime:              v.CreateTime,
		}
		// Fill User Info
		user, _ := s.memberUserSvc.GetUser(ctx, v.UserID)
		if user != nil {
			resList[i].UserNickname = user.Nickname
			resList[i].UserAvatar = user.Avatar
		}
	}
	return resList, nil
}

// GetConversationPage 获得会话分页
func (s *kefuService) GetConversationPage(ctx context.Context, r promotion2.KefuConversationPageReq) (*pagination.PageResult[promotion2.KefuConversationResp], error) {
	convoRepo := s.q.PromotionKefuConversation

	var list []*promotion.PromotionKefuConversation
	var total int64

	db := convoRepo.WithContext(ctx).UnderlyingDB().Where("admin_deleted = ?", false)

	// Count
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// Find
	err := db.Order("admin_pinned DESC").
		Order("last_message_time DESC").
		Offset((r.PageNo - 1) * r.PageSize).
		Limit(r.PageSize).
		Find(&list).Error

	if err != nil {
		return nil, err
	}

	resList := make([]promotion2.KefuConversationResp, len(list))
	for i, v := range list {
		resList[i] = promotion2.KefuConversationResp{
			ID:                      v.ID,
			UserID:                  v.UserID,
			LastMessageTime:         v.LastMessageTime,
			LastMessageContent:      v.LastMessageContent,
			LastMessageContentType:  v.LastMessageContentType,
			AdminPinned:             bool(v.AdminPinned),
			UserDeleted:             bool(v.UserDeleted),
			AdminDeleted:            bool(v.AdminDeleted),
			AdminUnreadMessageCount: v.AdminUnreadMessageCount,
			CreateTime:              v.CreateTime,
		}
		user, _ := s.memberUserSvc.GetUser(ctx, v.UserID)
		if user != nil {
			resList[i].UserNickname = user.Nickname
			resList[i].UserAvatar = user.Avatar
		}
	}
	return &pagination.PageResult[promotion2.KefuConversationResp]{List: resList, Total: total}, nil
}

// UpdateConversationPinned
func (s *kefuService) UpdateConversationPinned(ctx context.Context, r promotion2.KeFuConversationUpdatePinnedReq) error {
	// Validate Exists
	if _, err := s.validateKefuConversationExists(ctx, r.ID); err != nil {
		return err
	}
	convoRepo := s.q.PromotionKefuConversation
	// Use UnderlyingDB
	return convoRepo.WithContext(ctx).UnderlyingDB().
		Model(&promotion.PromotionKefuConversation{}).
		Where("id = ?", r.ID).
		Update("admin_pinned", *r.AdminPinned).Error
}

// DeleteConversation
func (s *kefuService) DeleteConversation(ctx context.Context, id int64) error {
	// Validate Exists
	if _, err := s.validateKefuConversationExists(ctx, id); err != nil {
		return err
	}
	// Soft delete for Admin (set admin_deleted = true)
	convoRepo := s.q.PromotionKefuConversation
	_, err := convoRepo.WithContext(ctx).Where(convoRepo.ID.Eq(id)).Update(convoRepo.AdminDeleted, model.BitBool(true))
	return err
}

// GetConversation
func (s *kefuService) GetConversation(ctx context.Context, id int64) (*promotion.PromotionKefuConversation, error) {
	return s.validateKefuConversationExists(ctx, id)
}

// Helpers

func (s *kefuService) validateKefuConversationExists(ctx context.Context, id int64) (*promotion.PromotionKefuConversation, error) {
	convoRepo := s.q.PromotionKefuConversation
	convo, err := convoRepo.WithContext(ctx).Where(convoRepo.ID.Eq(id)).First()
	if err != nil {
		return nil, errors.NewBizError(404, "会话不存在")
	}
	return convo, nil
}

// validateReceiverExist 校验接收人是否存在 (对齐 Java: validateReceiverExist)
func (s *kefuService) validateReceiverExist(ctx context.Context, receiverID int64, receiverType int) error {
	if receiverType == 1 { // Member
		_, err := s.memberUserSvc.GetUser(ctx, receiverID)
		if err != nil {
			return errors.NewBizError(404, "接收人不存在")
		}
	} else if receiverType == 2 { // Admin
		_, err := s.systemUserSvc.GetUser(ctx, receiverID)
		if err != nil {
			return errors.NewBizError(404, "接收人不存在")
		}
	}
	return nil
}

func (s *kefuService) getOrCreateConversation(ctx context.Context, userID int64) (*promotion.PromotionKefuConversation, error) {
	convoRepo := s.q.PromotionKefuConversation
	// Check existing
	convo, err := convoRepo.WithContext(ctx).Where(convoRepo.UserID.Eq(userID)).First()
	if err == nil {
		return convo, nil
	}
	// Create new
	newConvo := &promotion.PromotionKefuConversation{
		UserID:                  userID,
		LastMessageTime:         time.Now(),
		LastMessageContent:      "",
		LastMessageContentType:  1, // default Text
		AdminPinned:             false,
		UserDeleted:             false,
		AdminDeleted:            false,
		AdminUnreadMessageCount: 0,
	}
	if err := convoRepo.WithContext(ctx).Create(newConvo); err != nil {
		return nil, err
	}
	return newConvo, nil
}

// sendKefuMessageNotify 发送 WebSocket 消息通知
// receiverID: 接收者 ID
// receiverType: 1=Member, 2=Admin
// msgType: 消息类型，使用 internal/consts 的客服通知常量
// content: 消息内容
func (s *kefuService) sendKefuMessageNotify(ctx context.Context, receiverID int64, receiverType int, msgType string, content interface{}) {
	if s.wsManager == nil {
		log.Printf("[WebSocket] Manager not initialized, skip notify")
		return
	}

	// Only administrator notifications may omit a specific receiver.
	if receiverID <= 0 && receiverType != consts.UserTypeAdmin {
		return
	}
	// 构建 WebSocket 消息
	wsMsg, err := websocket.NewMessage(msgType, content)
	if err != nil {
		log.Printf("[WebSocket] Failed to create message: %v", err)
		return
	}

	msgBytes, err := wsMsg.ToJSON()
	if err != nil {
		log.Printf("[WebSocket] Failed to serialize message: %v", err)
		return
	}

	tenantID, ok := tenant.TenantID(ctx)
	if !ok {
		log.Printf("[WebSocket] Missing trusted tenant, skip notify")
		return
	}
	s.wsManager.SendToTenant(tenantID, receiverID, receiverType, msgBytes)
}

// GetConversationByUserId 【会员】获得客服会话 (对齐 Java: getConversationByUserId)
func (s *kefuService) GetConversationByUserId(ctx context.Context, userID int64) (*promotion.PromotionKefuConversation, error) {
	convoRepo := s.q.PromotionKefuConversation
	convo, err := convoRepo.WithContext(ctx).Where(convoRepo.UserID.Eq(userID)).First()
	if err != nil {
		return nil, nil // Java 返回 null 当不存在时
	}
	return convo, nil
}

// GetMessageListForMember 【会员】获得消息列表 (对齐 Java: getKeFuMessageList(pageReqVO, userId))
func (s *kefuService) GetMessageListForMember(ctx context.Context, r promotion2.KefuMessageListReq, userID int64) ([]promotion2.KefuMessageResp, error) {
	// 1. 获得客服会话
	conversation := s.getConversationByUserIdInternal(ctx, userID)
	if conversation == nil {
		return []promotion2.KefuMessageResp{}, nil // Java 返回 empty list
	}

	// 2. 设置会话编号
	r.ConversationID = conversation.ID

	// 3. 调用消息列表查询 (复用 GetMessageList 逻辑)
	return s.GetMessageList(ctx, r)
}

// getConversationByUserIdInternal 内部方法，用于获取用户会话
func (s *kefuService) getConversationByUserIdInternal(ctx context.Context, userID int64) *promotion.PromotionKefuConversation {
	convoRepo := s.q.PromotionKefuConversation
	convo, err := convoRepo.WithContext(ctx).Where(convoRepo.UserID.Eq(userID)).First()
	if err != nil {
		return nil
	}
	return convo
}
