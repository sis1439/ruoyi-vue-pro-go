package system

import (
	"context"
	"crypto/tls"
	"encoding/json"
	stderrors "errors"
	"fmt"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"regexp"
	"strings"
	"time"

	"gopkg.in/gomail.v2"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"gorm.io/gorm"
)

type MailService struct {
	db *gorm.DB
}

func NewMailService(db *gorm.DB) *MailService {
	return &MailService{db: db}
}

// ================= Mail Account CRUD =================

func (s *MailService) CreateMailAccount(ctx context.Context, r *system.MailAccountSaveReq) (int64, error) {
	account := &model.SystemMailAccount{
		Mail:           r.Mail,
		Username:       r.Username,
		Password:       r.Password,
		Host:           r.Host,
		Port:           r.Port,
		SslEnable:      model.BitBool(r.SslEnable),
		StarttlsEnable: model.BitBool(r.StarttlsEnable),
	}
	if err := s.db.WithContext(ctx).Create(account).Error; err != nil {
		return 0, err
	}
	return account.ID, nil
}

func (s *MailService) UpdateMailAccount(ctx context.Context, r *system.MailAccountSaveReq) error {
	// 校验存在
	if _, err := s.GetMailAccount(ctx, r.ID); err != nil {
		return err
	}
	account := &model.SystemMailAccount{
		ID:             r.ID,
		Mail:           r.Mail,
		Username:       r.Username,
		Password:       r.Password,
		Host:           r.Host,
		Port:           r.Port,
		SslEnable:      model.BitBool(r.SslEnable),
		StarttlsEnable: model.BitBool(r.StarttlsEnable),
	}
	if err := s.db.WithContext(ctx).Updates(account).Error; err != nil {
		return err
	}
	return nil
}

// 移除旧的 UpdateMailAccount (已合并)

func (s *MailService) DeleteMailAccount(ctx context.Context, id int64) error {
	// 校验是否有关联模板
	if err := s.validateMailAccountCanDelete(ctx, id); err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&model.SystemMailAccount{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (s *MailService) DeleteMailAccountList(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if err := s.DeleteMailAccount(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *MailService) validateMailAccountCanDelete(ctx context.Context, id int64) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.SystemMailTemplate{}).Where("account_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return consts.ErrMailAccountRelateTemplateExists
	}
	return nil
}

func (s *MailService) GetMailAccount(ctx context.Context, id int64) (*model.SystemMailAccount, error) {
	tenant, ok := pkgContext.TenantID(ctx)
	if !ok {
		return nil, fmt.Errorf("trusted tenant required")
	}
	if s.db == nil {
		return nil, fmt.Errorf("mail database unavailable")
	}
	var account model.SystemMailAccount
	if err := s.db.WithContext(ctx).Where("tenant_id = ?", tenant).First(&account, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, consts.ErrMailAccountNotExists
		}
		return nil, err
	}
	return &account, nil
}

func (s *MailService) GetMailAccountPage(ctx context.Context, r *system.MailAccountPageReq) (*pagination.PageResult[*model.SystemMailAccount], error) {
	db := s.db.WithContext(ctx).Model(&model.SystemMailAccount{})
	if r.Mail != "" {
		db = db.Where("mail LIKE ?", "%"+r.Mail+"%")
	}
	if r.Username != "" {
		db = db.Where("username LIKE ?", "%"+r.Username+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []*model.SystemMailAccount
	offset := (r.PageNo - 1) * r.PageSize
	if err := db.Order("id desc").Offset(offset).Limit(r.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &pagination.PageResult[*model.SystemMailAccount]{List: list, Total: total}, nil
}

func (s *MailService) GetSimpleMailAccountList(ctx context.Context) ([]*model.SystemMailAccount, error) {
	var list []*model.SystemMailAccount
	if err := s.db.WithContext(ctx).Select("id", "mail").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ================= Mail Template CRUD =================

func (s *MailService) CreateMailTemplate(ctx context.Context, r *system.MailTemplateSaveReq) (int64, error) {
	// 校验 code 唯一
	if err := s.validateMailTemplateCodeUnique(ctx, 0, r.Code); err != nil {
		return 0, err
	}
	template := &model.SystemMailTemplate{
		Name:      r.Name,
		Code:      r.Code,
		AccountID: r.AccountID,
		Nickname:  r.Nickname,
		Title:     r.Title,
		Content:   r.Content,
		Status:    r.Status,
		Remark:    r.Remark,
		Params:    s.parseTemplateTitleAndContentParams(r.Title, r.Content),
	}
	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return 0, err
	}
	return template.ID, nil
}

func (s *MailService) UpdateMailTemplate(ctx context.Context, r *system.MailTemplateSaveReq) error {
	// 校验存在
	if _, err := s.GetMailTemplate(ctx, r.ID); err != nil {
		return err
	}
	// 校验 code 唯一
	if err := s.validateMailTemplateCodeUnique(ctx, r.ID, r.Code); err != nil {
		return err
	}
	template := &model.SystemMailTemplate{
		ID:        r.ID,
		Name:      r.Name,
		Code:      r.Code,
		AccountID: r.AccountID,
		Nickname:  r.Nickname,
		Title:     r.Title,
		Content:   r.Content,
		Status:    r.Status,
		Remark:    r.Remark,
		Params:    s.parseTemplateTitleAndContentParams(r.Title, r.Content),
	}
	if err := s.db.WithContext(ctx).Updates(template).Error; err != nil {
		return err
	}
	return nil
}

func (s *MailService) DeleteMailTemplate(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Delete(&model.SystemMailTemplate{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (s *MailService) DeleteMailTemplateList(ctx context.Context, ids []int64) error {
	if err := s.db.WithContext(ctx).Delete(&model.SystemMailTemplate{}, ids).Error; err != nil {
		return err
	}
	return nil
}

func (s *MailService) GetMailTemplate(ctx context.Context, id int64) (*model.SystemMailTemplate, error) {
	var template model.SystemMailTemplate
	if err := s.db.WithContext(ctx).First(&template, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, consts.ErrMailTemplateNotExists
		}
		return nil, err
	}
	return &template, nil
}

func (s *MailService) GetSimpleMailTemplateList(ctx context.Context) ([]*model.SystemMailTemplate, error) {
	var list []*model.SystemMailTemplate
	if err := s.db.WithContext(ctx).Select("id", "name").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *MailService) GetMailTemplatePage(ctx context.Context, r *system.MailTemplatePageReq) (*pagination.PageResult[*model.SystemMailTemplate], error) {
	db := s.db.WithContext(ctx).Model(&model.SystemMailTemplate{})
	if r.Name != "" {
		db = db.Where("name LIKE ?", "%"+r.Name+"%")
	}
	if r.Code != "" {
		db = db.Where("code LIKE ?", "%"+r.Code+"%")
	}
	if r.AccountID != 0 {
		db = db.Where("account_id = ?", r.AccountID)
	}
	if r.Status != nil {
		db = db.Where("status = ?", *r.Status)
	}
	if len(r.CreateTime) == 2 {
		db = db.Where("create_time BETWEEN ? AND ?", r.CreateTime[0], r.CreateTime[1])
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []*model.SystemMailTemplate
	offset := (r.PageNo - 1) * r.PageSize
	if err := db.Order("id desc").Offset(offset).Limit(r.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &pagination.PageResult[*model.SystemMailTemplate]{List: list, Total: total}, nil
}

func (s *MailService) validateMailTemplateCodeUnique(ctx context.Context, id int64, code string) error {
	var template model.SystemMailTemplate
	db := s.db.WithContext(ctx).Where("code = ?", code)
	if id > 0 {
		db = db.Where("id != ?", id)
	}
	if err := db.First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return consts.ErrMailTemplateCodeExists
}

func (s *MailService) parseTemplateTitleAndContentParams(title, content string) []string {
	// Java regex: \\{(.*?)\\}
	re := regexp.MustCompile(`\{(.*?)\}`)
	paramMap := make(map[string]struct{})

	// From title
	matches := re.FindAllStringSubmatch(title, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramMap[match[1]] = struct{}{}
		}
	}

	// From content
	matches = re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramMap[match[1]] = struct{}{}
		}
	}

	params := make([]string, 0, len(paramMap))
	for p := range paramMap {
		params = append(params, p)
	}
	return params
}

// ================= Mail Sending Logic =================

// SendSingleMail 发送单条邮件 (核心逻辑)
func (s *MailService) SendSingleMail(ctx context.Context, toMails, ccMails, bccMails []string, userID int64, userType int, templateCode string, params map[string]interface{}) (int64, error) {
	// Resolve both records for this request; never cache templates globally by code.
	tenant, ok := pkgContext.TenantID(ctx)
	if !ok {
		return 0, fmt.Errorf("trusted tenant required")
	}
	if s.db == nil {
		return 0, fmt.Errorf("mail database unavailable")
	}
	var template model.SystemMailTemplate
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND code = ?", tenant, templateCode).First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, consts.ErrMailTemplateNotExists
		}
		return 0, err
	}
	// 模板开启状态校验
	if template.Status != 0 { // CommonStatus: 0=ENABLE
		return 0, nil
	}

	// 2. 邮箱地址处理：如果为空，尝试从用户 ID 获取
	if len(toMails) == 0 && userID > 0 {
		mail, err := s.getUserMail(ctx, userID, userType)
		if err != nil {
			return 0, err
		}
		if mail != "" {
			toMails = []string{mail}
		}
	}
	if len(toMails) == 0 {
		return 0, consts.ErrMailSendMailNotExists
	}

	// 3. 校验参数
	if err := s.validateTemplateParams(&template, params); err != nil {
		return 0, err
	}

	// The account must belong to the same trusted tenant as the selected template.
	account, err := s.GetMailAccount(ctx, template.AccountID)
	if err != nil {
		return 0, err
	}

	// 5. 渲染内容
	title := template.Title
	content := template.Content
	for k, v := range params {
		placeholder := "{" + k + "}"
		valStr := fmt.Sprintf("%v", v)
		title = strings.ReplaceAll(title, placeholder, valStr)
		content = strings.ReplaceAll(content, placeholder, valStr)
	}

	// 6. 构造日志 (先行构造以获取 ID，且即使发送失败也会记录)
	paramsStr, err := json.Marshal(params)
	if err != nil {
		return 0, err
	}
	log := &model.SystemMailLog{
		TenantBaseDO:     model.TenantBaseDO{TenantID: tenant},
		UserID:           userID,
		UserType:         userType,
		ToMails:          model.StringListFromCSV(toMails),
		CcMails:          model.StringListFromCSV(ccMails),
		BccMails:         model.StringListFromCSV(bccMails),
		AccountID:        account.ID,
		FromMail:         account.Mail,
		TemplateID:       template.ID,
		TemplateCode:     template.Code,
		TemplateNickname: template.Nickname,
		TemplateTitle:    title,
		TemplateContent:  content,
		TemplateParams:   string(paramsStr),
		SendStatus:       consts.MailSendStatusInit,
	}
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		return 0, err
	}

	// 7. 执行发送
	// TODO: 考虑引入更强大的邮件库支持 Cc/Bcc。此处暂使用简版封装。
	messageID, err := s.doSend(account, toMails, ccMails, bccMails, template.Nickname, title, content)

	// 8. 更新日志
	updateData := map[string]interface{}{}
	now := time.Now()
	updateData["send_time"] = &now
	if err != nil {
		updateData["send_status"] = consts.MailSendStatusFailure
		updateData["send_exception"] = err.Error()
	} else {
		updateData["send_status"] = consts.MailSendStatusSuccess
		updateData["send_message_id"] = messageID
	}
	err = stderrors.Join(err, s.db.WithContext(ctx).Model(log).Updates(updateData).Error)

	return log.ID, err
}

func (s *MailService) validateTemplateParams(template *model.SystemMailTemplate, params map[string]interface{}) error {
	// 注意：此处 model.SystemMailTemplate.Params 是 StringListFromCSV (即 []string)
	if len(template.Params) == 0 {
		return nil
	}
	for _, p := range template.Params {
		if _, ok := params[p]; !ok {
			return consts.ErrMailSendTemplateParamMiss
		}
	}
	return nil
}

func (s *MailService) getUserMail(ctx context.Context, userID int64, userType int) (string, error) {
	if userType != consts.UserTypeAdmin {
		// MemberUser has no email field. Callers must provide an explicit recipient.
		return "", consts.ErrMailSendMailNotExists
	}
	tenant, ok := pkgContext.TenantID(ctx)
	if !ok {
		return "", fmt.Errorf("trusted tenant required")
	}
	var user model.SystemUser
	if err := s.db.WithContext(ctx).Select("email").Where("tenant_id = ? AND id = ?", tenant, userID).First(&user).Error; err != nil {
		return "", err
	}
	return user.Email, nil
}

func (s *MailService) doSend(account *model.SystemMailAccount, toMails, ccMails, bccMails []string, nickname, subject, body string) (string, error) {
	m := gomail.NewMessage()

	// 设置发件人
	from := account.Mail
	if nickname != "" {
		from = fmt.Sprintf("%s <%s>", nickname, account.Mail)
	}
	m.SetHeader("From", from)

	// 设置收件人
	m.SetHeader("To", toMails...)
	if len(ccMails) > 0 {
		m.SetHeader("Cc", ccMails...)
	}
	if len(bccMails) > 0 {
		m.SetHeader("Bcc", bccMails...)
	}

	// 设置主题和内容
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// 设置 SMTP 拨号器
	d := gomail.NewDialer(account.Host, account.Port, account.Username, account.Password)
	if bool(account.SslEnable) {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		return "", err
	}

	// Message-ID 通常在发送后由服务器生成，库中有时可以直接获得或是模拟获取
	return m.GetHeader("Message-ID")[0], nil
}

// ================= Mail Log =================

func (s *MailService) GetMailLogPage(ctx context.Context, r *system.MailLogPageReq) (*pagination.PageResult[*model.SystemMailLog], error) {
	db := s.db.WithContext(ctx).Model(&model.SystemMailLog{})
	if r.UserID != 0 {
		db = db.Where("user_id = ?", r.UserID)
	}
	if r.UserType != 0 {
		db = db.Where("user_type = ?", r.UserType)
	}
	if r.ToMail != "" {
		db = db.Where("to_mails LIKE ?", "%"+r.ToMail+"%")
	}
	if r.AccountID != 0 {
		db = db.Where("account_id = ?", r.AccountID)
	}
	if r.TemplateID != 0 {
		db = db.Where("template_id = ?", r.TemplateID)
	}
	if r.SendStatus != nil {
		db = db.Where("send_status = ?", *r.SendStatus)
	}
	if len(r.SendTime) == 2 {
		db = db.Where("send_time BETWEEN ? AND ?", r.SendTime[0], r.SendTime[1])
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []*model.SystemMailLog
	offset := (r.PageNo - 1) * r.PageSize
	if err := db.Order("id desc").Offset(offset).Limit(r.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &pagination.PageResult[*model.SystemMailLog]{List: list, Total: total}, nil
}

func (s *MailService) GetMailLog(ctx context.Context, id int64) (*model.SystemMailLog, error) {
	var log model.SystemMailLog
	if err := s.db.WithContext(ctx).First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}
