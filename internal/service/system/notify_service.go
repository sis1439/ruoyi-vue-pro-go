package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

// NotifyTemplateRepository 站内信模板数据访问接口
type NotifyTemplateRepository interface {
	FindAll(ctx context.Context) ([]*model.SystemNotifyTemplate, error)
	Create(ctx context.Context, template *model.SystemNotifyTemplate) error
	Update(ctx context.Context, template *model.SystemNotifyTemplate) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*model.SystemNotifyTemplate, error)
	FindByCode(ctx context.Context, code string) (*model.SystemNotifyTemplate, error)
	Page(ctx context.Context, name, code string, status *int, pageNo, pageSize int) ([]*model.SystemNotifyTemplate, int64, error)
}

// NotifyMessageRepository 站内信消息数据访问接口
type NotifyMessageRepository interface {
	Create(ctx context.Context, msg *model.SystemNotifyMessage) error
	Page(ctx context.Context, userID int64, userType int, templateCode string, templateType *int, readStatus *bool, startDate, endDate string, pageNo, pageSize int) ([]*model.SystemNotifyMessage, int64, error)
	MyPage(ctx context.Context, userID int64, userType int, readStatus *bool, pageNo, pageSize int) ([]*model.SystemNotifyMessage, int64, error)
	UpdateReadStatus(ctx context.Context, userID int64, userType int, ids []int64, readStatus bool, readTime *time.Time) error
	UpdateAllReadStatus(ctx context.Context, userID int64, userType int, readStatus bool, readTime *time.Time) error
	CountUnread(ctx context.Context, userID int64, userType int) (int64, error)
	FindByID(ctx context.Context, id int64) (*model.SystemNotifyMessage, error)
	FindUnreadList(ctx context.Context, userID int64, userType int, size int) ([]*model.SystemNotifyMessage, error)
}

type NotifyService struct {
	templateRepo NotifyTemplateRepository
	messageRepo  NotifyMessageRepository
}

func NewNotifyService(templateRepo NotifyTemplateRepository, messageRepo NotifyMessageRepository) *NotifyService {
	s := &NotifyService{
		templateRepo: templateRepo,
		messageRepo:  messageRepo,
	}
	return s
}

// ================= Template CRUD =================

func (s *NotifyService) CreateNotifyTemplate(ctx context.Context, r *system.NotifyTemplateCreateReq) (int64, error) {
	template := &model.SystemNotifyTemplate{
		Name:     r.Name,
		Code:     r.Code,
		Nickname: r.Nickname,
		Content:  r.Content,
		Type:     r.Type,
		Status:   r.Status,
		Remark:   r.Remark,
	}
	if err := s.templateRepo.Create(ctx, template); err != nil {
		return 0, err
	}
	return template.ID, nil
}

func (s *NotifyService) UpdateNotifyTemplate(ctx context.Context, r *system.NotifyTemplateUpdateReq) error {
	err := s.templateRepo.Update(ctx, &model.SystemNotifyTemplate{
		ID:       r.ID,
		Name:     r.Name,
		Code:     r.Code,
		Nickname: r.Nickname,
		Content:  r.Content,
		Type:     r.Type,
		Status:   r.Status,
		Remark:   r.Remark,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *NotifyService) DeleteNotifyTemplate(ctx context.Context, id int64) error {
	err := s.templateRepo.Delete(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *NotifyService) GetNotifyTemplate(ctx context.Context, id int64) (*model.SystemNotifyTemplate, error) {
	return s.templateRepo.FindByID(ctx, id)
}

func (s *NotifyService) GetNotifyTemplatePage(ctx context.Context, r *system.NotifyTemplatePageReq) (*pagination.PageResult[*model.SystemNotifyTemplate], error) {
	list, total, err := s.templateRepo.Page(ctx, r.Name, r.Code, r.Status, r.PageNo, r.PageSize)
	if err != nil {
		return nil, err
	}
	return &pagination.PageResult[*model.SystemNotifyTemplate]{List: list, Total: total}, nil
}

// ================= Message Logic =================

func (s *NotifyService) SendNotify(ctx context.Context, userID int64, userType int, templateCode string, params map[string]interface{}) (int64, error) {
	template, err := s.templateRepo.FindByCode(ctx, templateCode)
	if err != nil {
		return 0, err
	}
	if template == nil {
		return 0, errors.NewBizError(1002006001, "站内信模板不存在")
	}

	// 对齐 Java: NotifySendServiceImpl.sendSingleNotify - 校验模板状态
	// Status: 0=开启, 1=禁用 (CommonStatusEnum: ENABLE=0, DISABLE=1)
	if template.Status == consts.CommonStatusDisable {
		// 模板已禁用，静默返回（对齐 Java 的 log.info 并 return null）
		return 0, nil
	}

	// 对齐 Java: NotifySendServiceImpl.validateTemplateParams - 校验模板参数完整性
	if template.Params != "" {
		var requiredParams []string
		if err := json.Unmarshal([]byte(template.Params), &requiredParams); err == nil {
			for _, key := range requiredParams {
				if _, exists := params[key]; !exists {
					return 0, errors.NewBizError(1002006002, fmt.Sprintf("站内信模板参数 [%s] 缺失", key))
				}
			}
		}
	}

	content := template.Content
	for k, v := range params {
		content = strings.ReplaceAll(content, "{"+k+"}", fmt.Sprintf("%v", v))
	}

	paramsStr, _ := json.Marshal(params)
	msg := &model.SystemNotifyMessage{
		UserID:           userID,
		UserType:         userType,
		TemplateID:       template.ID,
		TemplateCode:     template.Code,
		TemplateNickname: template.Nickname,
		TemplateContent:  content,
		TemplateType:     template.Type,
		TemplateParams:   string(paramsStr),
		ReadStatus:       false,
	}

	if err := s.messageRepo.Create(ctx, msg); err != nil {
		return 0, err
	}
	return msg.ID, nil
}

func (s *NotifyService) GetNotifyMessagePage(ctx context.Context, r *system.NotifyMessagePageReq) (*pagination.PageResult[*model.SystemNotifyMessage], error) {
	list, total, err := s.messageRepo.Page(ctx, r.UserID, r.UserType, r.TemplateCode, r.TemplateType, r.ReadStatus, r.StartDate, r.EndDate, r.PageNo, r.PageSize)
	if err != nil {
		return nil, err
	}
	return &pagination.PageResult[*model.SystemNotifyMessage]{List: list, Total: total}, nil
}

func (s *NotifyService) GetMyNotifyMessagePage(ctx context.Context, userID int64, userType int, r *system.MyNotifyMessagePageReq) (*pagination.PageResult[*model.SystemNotifyMessage], error) {
	list, total, err := s.messageRepo.MyPage(ctx, userID, userType, r.ReadStatus, r.PageNo, r.PageSize)
	if err != nil {
		return nil, err
	}
	return &pagination.PageResult[*model.SystemNotifyMessage]{List: list, Total: total}, nil
}

func (s *NotifyService) UpdateNotifyMessageRead(ctx context.Context, userID int64, userType int, ids []int64) error {
	now := time.Now()
	return s.messageRepo.UpdateReadStatus(ctx, userID, userType, ids, true, &now)
}

func (s *NotifyService) UpdateAllNotifyMessageRead(ctx context.Context, userID int64, userType int) error {
	now := time.Now()
	return s.messageRepo.UpdateAllReadStatus(ctx, userID, userType, true, &now)
}

func (s *NotifyService) GetUnreadNotifyMessageCount(ctx context.Context, userID int64, userType int) (int64, error) {
	return s.messageRepo.CountUnread(ctx, userID, userType)
}

// GetNotifyMessage 获取单条站内信 (对齐 Java: NotifyMessageService.getNotifyMessage)
func (s *NotifyService) GetNotifyMessage(ctx context.Context, id int64) (*model.SystemNotifyMessage, error) {
	return s.messageRepo.FindByID(ctx, id)
}

// GetUnreadNotifyMessageList 获取未读站内信列表 (对齐 Java: NotifyMessageService.getUnreadNotifyMessageList)
func (s *NotifyService) GetUnreadNotifyMessageList(ctx context.Context, userID int64, userType int, size int) ([]*model.SystemNotifyMessage, error) {
	if size <= 0 {
		size = 10 // Default size
	}
	return s.messageRepo.FindUnreadList(ctx, userID, userType, size)
}
