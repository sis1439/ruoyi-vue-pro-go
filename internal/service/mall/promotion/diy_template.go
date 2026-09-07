package promotion

import (
	"context"
	stdErrors "errors"
	"time"

	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DiyTemplateService interface {
	CreateDiyTemplate(ctx context.Context, req promotion2.DiyTemplateCreateReq) (int64, error)
	UpdateDiyTemplate(ctx context.Context, req promotion2.DiyTemplateUpdateReq) error
	UseDiyTemplate(ctx context.Context, id int64) error
	DeleteDiyTemplate(ctx context.Context, id int64) error
	GetDiyTemplate(ctx context.Context, id int64) (*promotion2.DiyTemplateResp, error)
	GetDiyTemplateModel(ctx context.Context, id int64) (*promotion.PromotionDiyTemplate, error) // App端使用
	GetDiyTemplatePage(ctx context.Context, req promotion2.DiyTemplatePageReq) (*pagination.PageResult[*promotion2.DiyTemplateResp], error)
	GetDiyTemplateProperty(ctx context.Context, id int64) (*promotion2.DiyTemplatePropertyResp, error)
	UpdateDiyTemplateProperty(ctx context.Context, req promotion2.DiyTemplatePropertyUpdateReq) error
	GetUsedDiyTemplate(ctx context.Context) (*promotion.PromotionDiyTemplate, error)
}

type diyTemplateService struct {
	q *query.Query
}

func NewDiyTemplateService(q *query.Query) DiyTemplateService {
	return &diyTemplateService{q: q}
}

func (s *diyTemplateService) CreateDiyTemplate(ctx context.Context, req promotion2.DiyTemplateCreateReq) (int64, error) {
	if err := s.validateNameUnique(ctx, 0, req.Name); err != nil {
		return 0, err
	}
	template := &promotion.PromotionDiyTemplate{
		Name:           req.Name,
		PreviewPicUrls: types.StringListFromCSV(req.PreviewPicUrls),
		Property:       req.Property,
		Remark:         req.Remark,
		Used:           false,
	}
	err := s.q.PromotionDiyTemplate.WithContext(ctx).Create(template)
	if err != nil {
		return 0, err
	}
	// 创建默认页面
	if err := s.createDefaultPage(ctx, template.ID); err != nil {
		return 0, err
	}
	return template.ID, nil
}

func (s *diyTemplateService) UpdateDiyTemplate(ctx context.Context, req promotion2.DiyTemplateUpdateReq) error {
	_, err := s.validateDiyTemplateExists(ctx, req.ID)
	if err != nil {
		return err
	}
	if err := s.validateNameUnique(ctx, req.ID, req.Name); err != nil {
		return err
	}

	_, err = s.q.PromotionDiyTemplate.WithContext(ctx).Where(s.q.PromotionDiyTemplate.ID.Eq(req.ID)).Updates(promotion.PromotionDiyTemplate{
		Name:           req.Name,
		PreviewPicUrls: types.StringListFromCSV(req.PreviewPicUrls),
		Property:       req.Property,
		Remark:         req.Remark,
	})
	return err
}

func (s *diyTemplateService) DeleteDiyTemplate(ctx context.Context, id int64) error {
	template, err := s.validateDiyTemplateExists(ctx, id)
	if err != nil {
		return err
	}
	if template.Used {
		return errors.NewBizError(400, "该模板正在使用，无法删除")
	}

	_, err = s.q.PromotionDiyTemplate.WithContext(ctx).Where(s.q.PromotionDiyTemplate.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	// Logic to delete pages associated?
	// Java deletes pages too? Yes, usually cascade or manual delete.
	// Java doesn't show explicit page delete in controller snippet but usually Service has it.
	// We should probably delete pages too.
	_, err = s.q.PromotionDiyPage.WithContext(ctx).Where(s.q.PromotionDiyPage.TemplateID.Eq(id)).Delete()
	return err
}

func (s *diyTemplateService) GetDiyTemplate(ctx context.Context, id int64) (*promotion2.DiyTemplateResp, error) {
	template, err := s.validateDiyTemplateExists(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.convertDiyTemplateToResp(template), nil
}

// GetDiyTemplateModel 获取装修模板 Model (用于 App 端)
func (s *diyTemplateService) GetDiyTemplateModel(ctx context.Context, id int64) (*promotion.PromotionDiyTemplate, error) {
	return s.validateDiyTemplateExists(ctx, id)
}

func (s *diyTemplateService) GetDiyTemplatePage(ctx context.Context, req promotion2.DiyTemplatePageReq) (*pagination.PageResult[*promotion2.DiyTemplateResp], error) {
	q := s.q.PromotionDiyTemplate
	do := q.WithContext(ctx)
	if req.Name != "" {
		do = do.Where(q.Name.Like("%" + req.Name + "%"))
	}
	if len(req.CreateTime) == 2 {
		startTime, _ := time.ParseInLocation("2006-01-02 15:04:05", req.CreateTime[0], time.Local)
		endTime, _ := time.ParseInLocation("2006-01-02 15:04:05", req.CreateTime[1], time.Local)
		do = do.Where(q.CreateTime.Between(startTime, endTime))
	}
	list, total, err := do.Order(q.ID.Desc()).FindByPage(req.GetOffset(), req.GetLimit())
	if err != nil {
		return nil, err
	}

	result := make([]*promotion2.DiyTemplateResp, len(list))
	for i, item := range list {
		result[i] = s.convertDiyTemplateToResp(item)
	}
	return &pagination.PageResult[*promotion2.DiyTemplateResp]{List: result, Total: total}, nil
}

func (s *diyTemplateService) GetDiyTemplateProperty(ctx context.Context, id int64) (*promotion2.DiyTemplatePropertyResp, error) {
	template, err := s.validateDiyTemplateExists(ctx, id)
	if err != nil {
		return nil, err
	}

	pages, err := s.q.PromotionDiyPage.WithContext(ctx).Where(s.q.PromotionDiyPage.TemplateID.Eq(id)).Find()
	if err != nil {
		return nil, err
	}

	pageResps := make([]promotion2.DiyPagePropertyResp, len(pages))
	for i, page := range pages {
		pageResps[i] = promotion2.DiyPagePropertyResp{
			DiyPageBase: promotion2.DiyPageBase{
				TemplateID:     page.TemplateID,
				Name:           page.Name,
				Remark:         page.Remark,
				PreviewPicUrls: []string(page.PreviewPicUrls),
			},
			ID:       page.ID,
			Property: string(page.Property),
		}
	}

	return &promotion2.DiyTemplatePropertyResp{
		DiyTemplateBase: promotion2.DiyTemplateBase{
			Name:           template.Name,
			Remark:         template.Remark,
			PreviewPicUrls: []string(template.PreviewPicUrls),
		},
		ID:       template.ID,
		Property: string(template.Property),
		Pages:    pageResps,
	}, nil
}

// UseDiyTemplate 使用装修模板
func (s *diyTemplateService) UseDiyTemplate(ctx context.Context, id int64) error {
	// 校验存在
	_, err := s.validateDiyTemplateExists(ctx, id)
	if err != nil {
		return err
	}

	// 开启事务
	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 将所有已使用的设置为未使用
		err := tx.PromotionDiyTemplate.WithContext(ctx).UnderlyingDB().Model(&promotion.PromotionDiyTemplate{}).Where("used = ?", types.BitBool(true)).Updates(map[string]interface{}{"used": types.BitBool(false)}).Error
		if err != nil {
			return err
		}

		// 2. 更新新的为使用
		now := time.Now()
		_, err = tx.PromotionDiyTemplate.WithContext(ctx).
			Where(tx.PromotionDiyTemplate.ID.Eq(id)).
			Updates(map[string]interface{}{
				"used":      types.BitBool(true),
				"used_time": &now,
			})
		return err
	})
}

func (s *diyTemplateService) GetUsedDiyTemplate(ctx context.Context) (*promotion.PromotionDiyTemplate, error) {
	template := &promotion.PromotionDiyTemplate{}
	err := s.q.PromotionDiyTemplate.WithContext(ctx).UnderlyingDB().Where("used = ?", types.BitBool(true)).First(template).Error
	if err != nil {
		if stdErrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}
	return template, nil
}

// UpdateDiyTemplateProperty 更新装修模板属性
func (s *diyTemplateService) UpdateDiyTemplateProperty(ctx context.Context, req promotion2.DiyTemplatePropertyUpdateReq) error {
	// 校验存在
	_, err := s.validateDiyTemplateExists(ctx, req.ID)
	if err != nil {
		return err
	}
	// 更新属性
	_, err = s.q.PromotionDiyTemplate.WithContext(ctx).
		Where(s.q.PromotionDiyTemplate.ID.Eq(req.ID)).
		Updates(promotion.PromotionDiyTemplate{
			Property: req.Property,
		})
	return err
}

// Helpers
func (s *diyTemplateService) createDefaultPage(ctx context.Context, templateID int64) error {
	pages := []*promotion.PromotionDiyPage{
		{
			TemplateID: templateID,
			Name:       "首页",
			Remark:     "默认首页",
			Property:   datatypes.JSON("{}"),
		},
		{
			TemplateID: templateID,
			Name:       "我的",
			Remark:     "默认我的页面",
			Property:   datatypes.JSON("{}"),
		},
	}
	return s.q.PromotionDiyPage.WithContext(ctx).Create(pages...)
}

func (s *diyTemplateService) validateNameUnique(ctx context.Context, id int64, name string) error {
	q := s.q.PromotionDiyTemplate
	do := q.WithContext(ctx).Where(q.Name.Eq(name))
	if id > 0 {
		do = do.Where(q.ID.Neq(id))
	}
	count, err := do.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.NewBizError(400, "模板名称已存在")
	}
	return nil
}

func (s *diyTemplateService) validateDiyTemplateExists(ctx context.Context, id int64) (*promotion.PromotionDiyTemplate, error) {
	template, err := s.q.PromotionDiyTemplate.WithContext(ctx).Where(s.q.PromotionDiyTemplate.ID.Eq(id)).First()
	if err != nil {
		return nil, errors.NewBizError(404, "装修模板不存在")
	}
	return template, nil
}

func (s *diyTemplateService) convertDiyTemplateToResp(item *promotion.PromotionDiyTemplate) *promotion2.DiyTemplateResp {
	return &promotion2.DiyTemplateResp{
		DiyTemplateBase: promotion2.DiyTemplateBase{
			Name:           item.Name,
			Remark:         item.Remark,
			PreviewPicUrls: []string(item.PreviewPicUrls),
		},
		ID:         item.ID,
		Used:       bool(item.Used),
		UsedTime:   item.UsedTime,
		CreateTime: item.CreateTime,
	}
}
