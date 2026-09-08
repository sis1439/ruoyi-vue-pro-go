package repo

import (
	"context"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
)

type NotifyTemplateRepositoryImpl struct {
	q *query.Query
}

func NewNotifyTemplateRepository(q *query.Query) *NotifyTemplateRepositoryImpl {
	return &NotifyTemplateRepositoryImpl{q: q}
}

func (r *NotifyTemplateRepositoryImpl) FindAll(ctx context.Context) ([]*model.SystemNotifyTemplate, error) {
	t := r.q.SystemNotifyTemplate
	return t.WithContext(ctx).Find()
}

func (r *NotifyTemplateRepositoryImpl) Create(ctx context.Context, template *model.SystemNotifyTemplate) error {
	t := r.q.SystemNotifyTemplate
	return t.WithContext(ctx).Create(template)
}

func (r *NotifyTemplateRepositoryImpl) Update(ctx context.Context, template *model.SystemNotifyTemplate) error {
	t := r.q.SystemNotifyTemplate
	_, err := t.WithContext(ctx).Where(t.ID.Eq(template.ID)).Select(t.Name, t.Code, t.Nickname, t.Content, t.Type, t.Status, t.Remark).Updates(template)
	return err
}

func (r *NotifyTemplateRepositoryImpl) Delete(ctx context.Context, id int64) error {
	t := r.q.SystemNotifyTemplate
	_, err := t.WithContext(ctx).Where(t.ID.Eq(id)).Delete()
	return err
}

func (r *NotifyTemplateRepositoryImpl) FindByID(ctx context.Context, id int64) (*model.SystemNotifyTemplate, error) {
	t := r.q.SystemNotifyTemplate
	return t.WithContext(ctx).Where(t.ID.Eq(id)).First()
}

func (r *NotifyTemplateRepositoryImpl) FindByCode(ctx context.Context, code string) (*model.SystemNotifyTemplate, error) {
	t := r.q.SystemNotifyTemplate
	return t.WithContext(ctx).Where(t.Code.Eq(code)).First()
}

func (r *NotifyTemplateRepositoryImpl) Page(ctx context.Context, name, code string, status *int, pageNo, pageSize int) ([]*model.SystemNotifyTemplate, int64, error) {
	t := r.q.SystemNotifyTemplate
	qb := t.WithContext(ctx)

	if name != "" {
		qb = qb.Where(t.Name.Like("%" + name + "%"))
	}
	if code != "" {
		qb = qb.Where(t.Code.Like("%" + code + "%"))
	}
	if status != nil {
		qb = qb.Where(t.Status.Eq(*status))
	}

	total, err := qb.Count()
	if err != nil {
		return nil, 0, err
	}

	offset := (pageNo - 1) * pageSize
	list, err := qb.Order(t.ID.Desc()).Offset(offset).Limit(pageSize).Find()
	return list, total, err
}

type NotifyMessageRepositoryImpl struct {
	q *query.Query
}

func NewNotifyMessageRepository(q *query.Query) *NotifyMessageRepositoryImpl {
	return &NotifyMessageRepositoryImpl{q: q}
}

func (r *NotifyMessageRepositoryImpl) Create(ctx context.Context, msg *model.SystemNotifyMessage) error {
	m := r.q.SystemNotifyMessage
	return m.WithContext(ctx).Create(msg)
}

func (r *NotifyMessageRepositoryImpl) Page(ctx context.Context, userID int64, userType int, templateCode string, templateType *int, readStatus *bool, startDate, endDate string, pageNo, pageSize int) ([]*model.SystemNotifyMessage, int64, error) {
	m := r.q.SystemNotifyMessage
	qb := m.WithContext(ctx)

	if userID != 0 {
		qb = qb.Where(m.UserID.Eq(userID))
	}
	if userType != 0 {
		qb = qb.Where(m.UserType.Eq(userType))
	}
	if templateCode != "" {
		qb = qb.Where(m.TemplateCode.Like("%" + templateCode + "%"))
	}
	if templateType != nil {
		qb = qb.Where(m.TemplateType.Eq(*templateType))
	}
	if readStatus != nil {
		qb = qb.Where(m.ReadStatus.Eq(model.BitBool(*readStatus)))
	}
	if startDate != "" && endDate != "" {
		qb = qb.Where(m.CreateTime.Between(r.parseTime(startDate), r.parseTime(endDate)))
	}

	total, err := qb.Count()
	if err != nil {
		return nil, 0, err
	}

	offset := (pageNo - 1) * pageSize
	list, err := qb.Order(m.ID.Desc()).Offset(offset).Limit(pageSize).Find()
	return list, total, err
}

func (r *NotifyMessageRepositoryImpl) MyPage(ctx context.Context, userID int64, userType int, readStatus *bool, pageNo, pageSize int) ([]*model.SystemNotifyMessage, int64, error) {
	m := r.q.SystemNotifyMessage
	qb := m.WithContext(ctx).Where(m.UserID.Eq(userID), m.UserType.Eq(userType))

	if readStatus != nil {
		qb = qb.Where(m.ReadStatus.Eq(model.BitBool(*readStatus)))
	}

	total, err := qb.Count()
	if err != nil {
		return nil, 0, err
	}

	offset := (pageNo - 1) * pageSize
	list, err := qb.Order(m.ID.Desc()).Offset(offset).Limit(pageSize).Find()
	return list, total, err
}

func (r *NotifyMessageRepositoryImpl) UpdateReadStatus(ctx context.Context, userID int64, userType int, ids []int64, readStatus bool, readTime *time.Time) error {
	m := r.q.SystemNotifyMessage
	var rt time.Time
	if readTime != nil {
		rt = *readTime
	}
	_, err := m.WithContext(ctx).
		Where(m.ID.In(ids...), m.UserID.Eq(userID), m.UserType.Eq(userType)).
		UpdateSimple(m.ReadStatus.Value(model.BitBool(readStatus)), m.ReadTime.Value(rt))
	return err
}

func (r *NotifyMessageRepositoryImpl) UpdateAllReadStatus(ctx context.Context, userID int64, userType int, readStatus bool, readTime *time.Time) error {
	m := r.q.SystemNotifyMessage
	var rt time.Time
	if readTime != nil {
		rt = *readTime
	}
	_, err := m.WithContext(ctx).
		Where(m.UserID.Eq(userID), m.UserType.Eq(userType), m.ReadStatus.Eq(model.BitBool(!readStatus))).
		UpdateSimple(m.ReadStatus.Value(model.BitBool(readStatus)), m.ReadTime.Value(rt))
	return err
}

func (r *NotifyMessageRepositoryImpl) CountUnread(ctx context.Context, userID int64, userType int) (int64, error) {
	m := r.q.SystemNotifyMessage
	return m.WithContext(ctx).
		Where(m.UserID.Eq(userID), m.UserType.Eq(userType), m.ReadStatus.Eq(model.BitBool(false))).
		Count()
}

func (r *NotifyMessageRepositoryImpl) FindByID(ctx context.Context, id int64) (*model.SystemNotifyMessage, error) {
	m := r.q.SystemNotifyMessage
	return m.WithContext(ctx).Where(m.ID.Eq(id)).First()
}

func (r *NotifyMessageRepositoryImpl) FindUnreadList(ctx context.Context, userID int64, userType int, size int) ([]*model.SystemNotifyMessage, error) {
	m := r.q.SystemNotifyMessage
	return m.WithContext(ctx).
		Where(m.UserID.Eq(userID), m.UserType.Eq(userType), m.ReadStatus.Eq(model.BitBool(false))).
		Order(m.ID.Desc()).
		Limit(size).
		Find()
}

func (r *NotifyMessageRepositoryImpl) parseTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05", s)
	return t
}
