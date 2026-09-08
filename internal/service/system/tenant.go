package system

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	bizErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
)

type TenantService struct {
	q             *query.Query
	roleSvc       *RoleService
	permissionSvc *PermissionService
}

func NewTenantService(q *query.Query, roleSvc *RoleService, permissionSvc *PermissionService) *TenantService {
	return &TenantService{
		q:             q,
		roleSvc:       roleSvc,
		permissionSvc: permissionSvc,
	}
}

// CreateTenant 创建租户
func (s *TenantService) CreateTenant(ctx context.Context, req *system.TenantCreateReq) (int64, error) {
	// 1. 校验租户名是否重复
	if err := s.checkNameUnique(ctx, req.Name, 0); err != nil {
		return 0, err
	}

	// 2. 校验域名是否重复
	if err := s.validTenantWebsiteDuplicate(ctx, req.Websites, 0); err != nil {
		return 0, err
	}

	// 3. 校验套餐是否存在且启用
	var pkg model.SystemTenantPackage
	if err := s.q.SystemTenantPackage.WithContext(ctx).UnderlyingDB().Where("id = ? AND status = ?", *req.PackageID, consts.CommonStatusEnable).First(&pkg).Error; err != nil {
		return 0, errors.New("租户套餐不存在或已禁用")
	}

	// 4. 事务执行
	var tenantId int64
	err := s.q.Transaction(func(tx *query.Query) error {
		// 4.1 创建租户
		tenant := &model.SystemTenant{
			Name:          req.Name,
			ContactName:   req.ContactName,
			ContactMobile: req.ContactMobile,
			Status:        int32(*req.Status),
			PackageID:     *req.PackageID,
			AccountCount:  int32(req.AccountCount),
			ExpireDate:    time.UnixMilli(req.ExpireTime),
			Websites:      model.StringListFromCSV(req.Websites),
		}
		if err := tx.SystemTenant.WithContext(ctx).Create(tenant); err != nil {
			return err
		}
		tenantId = tenant.ID

		// 4.2 创建角色
		role := &model.SystemRole{
			Name:             "租户管理员",
			Code:             consts.RoleCodeTenantAdmin,
			Sort:             0,
			DataScope:        consts.DataScopeAll,
			DataScopeDeptIds: model.Int64ListFromCSV{},
			Status:           consts.CommonStatusEnable,
			Type:             consts.RoleTypeSystem,
			Remark:           "系统自动生成",
		}
		role.TenantID = tenantId
		if err := tx.SystemRole.WithContext(ctx).Create(role); err != nil {
			return err
		}

		// 4.3 创建用户
		hashedPwd, err := utils.HashPassword(req.Password)
		if err != nil {
			return err
		}
		user := &model.SystemUser{
			Username: req.Username,
			Password: hashedPwd,
			Nickname: req.ContactName,
			Mobile:   req.ContactMobile,
			Status:   consts.CommonStatusEnable,
		}
		user.TenantID = tenantId
		if err := tx.SystemUser.WithContext(ctx).Create(user); err != nil {
			return err
		}

		// 4.4 关联用户与角色
		userRole := &model.SystemUserRole{
			UserID: user.ID,
			RoleID: role.ID,
		}
		userRole.TenantID = tenantId
		if err := tx.SystemUserRole.WithContext(ctx).Create(userRole); err != nil {
			return err
		}

		// 4.5 赋予角色菜单权限
		menuIds := pkg.MenuIDs
		if len(menuIds) > 0 {
			roleMenus := make([]*model.SystemRoleMenu, len(menuIds))
			for i, mid := range menuIds {
				roleMenus[i] = &model.SystemRoleMenu{
					RoleID: role.ID,
					MenuID: mid,
				}
				roleMenus[i].TenantID = tenantId
			}
			if err := tx.SystemRoleMenu.WithContext(ctx).Create(roleMenus...); err != nil {
				return err
			}
		}

		// 4.6 更新租户的联系人用户ID
		if _, err := tx.SystemTenant.WithContext(ctx).Where(tx.SystemTenant.ID.Eq(tenantId)).Update(tx.SystemTenant.ContactUserID, user.ID); err != nil {
			return err
		}

		return nil
	})

	return tenantId, err
}

// UpdateTenant 更新租户
func (s *TenantService) UpdateTenant(ctx context.Context, req *system.TenantUpdateReq) error {
	// 1. 校验存在及系统租户保护
	tenant, err := s.validateUpdateTenant(ctx, req.ID)
	if err != nil {
		return err
	}

	// 2. 校验套餐被禁用 (对齐 Java: validTenantPackage)
	if tenant.PackageID != *req.PackageID && *req.PackageID != 0 {
		if err := s.q.SystemTenantPackage.WithContext(ctx).UnderlyingDB().Where("id = ? AND status = ?", *req.PackageID, 0).First(&model.SystemTenantPackage{}).Error; err != nil {
			return errors.New("租户套餐不存在或已禁用")
		}
	}

	// 3. 校验名字唯一
	if err := s.checkNameUnique(ctx, req.Name, req.ID); err != nil {
		return err
	}

	// 3. 校验域名重复
	if err := s.validTenantWebsiteDuplicate(ctx, req.Websites, req.ID); err != nil {
		return err
	}

	// 4. 更新
	t := s.q.SystemTenant
	tenantObj := &model.SystemTenant{
		Name:          req.Name,
		ContactName:   req.ContactName,
		ContactMobile: req.ContactMobile,
		Status:        int32(*req.Status),
		PackageID:     *req.PackageID,
		AccountCount:  int32(req.AccountCount),
		ExpireDate:    time.UnixMilli(req.ExpireTime),
		Websites:      model.StringListFromCSV(req.Websites),
	}
	_, err = t.WithContext(ctx).Where(t.ID.Eq(req.ID)).Updates(tenantObj)
	if err != nil {
		return err
	}

	// 5. 如果套餐发生变化，则修改其角色的权限
	if tenant.PackageID != *req.PackageID {
		// 获得套餐菜单
		var pkg model.SystemTenantPackage
		if err := s.q.SystemTenant.WithContext(ctx).UnderlyingDB().Model(&model.SystemTenantPackage{}).Where("id = ?", *req.PackageID).First(&pkg).Error; err != nil {
			return err
		}
		menuIds := pkg.MenuIDs
		if err := s.updateTenantRoleMenu(ctx, req.ID, menuIds); err != nil {
			return err
		}
	}

	return nil
}

// DeleteTenant 删除租户
func (s *TenantService) DeleteTenant(ctx context.Context, id int64) error {
	// 校验存在及系统租户保护
	if _, err := s.validateUpdateTenant(ctx, id); err != nil {
		return err
	}

	t := s.q.SystemTenant
	_, err := t.WithContext(ctx).Where(t.ID.Eq(id)).Delete()
	return err
}

// DeleteTenantList 批量删除租户 (对齐 Java: deleteTenantList)
func (s *TenantService) DeleteTenantList(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if _, err := s.validateUpdateTenant(ctx, id); err != nil {
			return err
		}
	}
	t := s.q.SystemTenant
	_, err := t.WithContext(ctx).Where(t.ID.In(ids...)).Delete()
	return err
}

// GetTenant 获得租户
func (s *TenantService) GetTenant(ctx context.Context, id int64) (*system.TenantRespVO, error) {
	currentTenant, ok := pkgContext.TenantID(ctx)
	if !ok || currentTenant != id {
		return nil, bizErrors.NewBizError(403, "无权访问该租户管理资料")
	}
	t := s.q.SystemTenant
	tenant, err := t.WithContext(ctx).Where(t.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return &system.TenantRespVO{
		ID:            tenant.ID,
		Name:          tenant.Name,
		ContactUserID: tenant.ContactUserID,
		ContactName:   tenant.ContactName,
		ContactMobile: tenant.ContactMobile,
		Status:        int(tenant.Status),
		Websites:      []string(tenant.Websites),
		PackageID:     tenant.PackageID,
		AccountCount:  int(tenant.AccountCount),
		ExpireTime:    tenant.ExpireDate.UnixMilli(),
		CreateTime:    tenant.CreateTime,
	}, nil
}

// GetTenantPage 获得租户分页
func (s *TenantService) GetTenantPage(ctx context.Context, req *system.TenantPageReq) (*pagination.PageResult[*system.TenantRespVO], error) {
	tenant, ok := pkgContext.TenantID(ctx)
	if !ok {
		return nil, bizErrors.NewBizError(403, "可信租户缺失")
	}

	t := s.q.SystemTenant
	qb := t.WithContext(ctx).Where(t.ID.Eq(tenant))

	if req.Name != "" {
		qb = qb.Where(t.Name.Like("%" + req.Name + "%"))
	}
	if req.ContactName != "" {
		qb = qb.Where(t.ContactName.Like("%" + req.ContactName + "%"))
	}
	if req.ContactMobile != "" {
		qb = qb.Where(t.ContactMobile.Like("%" + req.ContactMobile + "%"))
	}
	if req.Status != nil {
		qb = qb.Where(t.Status.Eq(int32(*req.Status)))
	}
	if req.CreateTimeGe != nil {
		qb = qb.Where(t.CreateTime.Gte(*req.CreateTimeGe))
	}
	if req.CreateTimeLe != nil {
		qb = qb.Where(t.CreateTime.Lte(*req.CreateTimeLe))
	}

	total, err := qb.Count()
	if err != nil {
		return nil, err
	}

	list, err := qb.Order(t.ID.Desc()).Offset(req.GetOffset()).Limit(req.PageSize).Find()
	if err != nil {
		return nil, err
	}

	var data []*system.TenantRespVO
	for _, item := range list {
		data = append(data, &system.TenantRespVO{
			ID:            item.ID,
			Name:          item.Name,
			ContactUserID: item.ContactUserID,
			ContactName:   item.ContactName,
			ContactMobile: item.ContactMobile,
			Status:        int(item.Status),
			Websites:      []string(item.Websites),
			PackageID:     item.PackageID,
			AccountCount:  int(item.AccountCount),
			ExpireTime:    item.ExpireDate.UnixMilli(),
			CreateTime:    item.CreateTime,
		})
	}

	return &pagination.PageResult[*system.TenantRespVO]{
		List:  data,
		Total: total,
	}, nil
}

// GetTenantList 获得租户列表 (用于导出)
func (s *TenantService) GetTenantList(ctx context.Context, req *system.TenantExportReq) ([]*system.TenantRespVO, error) {
	tenant, ok := pkgContext.TenantID(ctx)
	if !ok {
		return nil, bizErrors.NewBizError(403, "可信租户缺失")
	}

	t := s.q.SystemTenant
	qb := t.WithContext(ctx).Where(t.ID.Eq(tenant))

	if req.Name != "" {
		qb = qb.Where(t.Name.Like("%" + req.Name + "%"))
	}
	if req.ContactName != "" {
		qb = qb.Where(t.ContactName.Like("%" + req.ContactName + "%"))
	}
	if req.ContactMobile != "" {
		qb = qb.Where(t.ContactMobile.Like("%" + req.ContactMobile + "%"))
	}
	if req.Status != nil {
		qb = qb.Where(t.Status.Eq(int32(*req.Status)))
	}
	if req.CreateTimeGe != nil {
		qb = qb.Where(t.CreateTime.Gte(*req.CreateTimeGe))
	}
	if req.CreateTimeLe != nil {
		qb = qb.Where(t.CreateTime.Lte(*req.CreateTimeLe))
	}

	list, err := qb.Order(t.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}

	var data []*system.TenantRespVO
	for _, item := range list {
		data = append(data, &system.TenantRespVO{
			ID:            item.ID,
			Name:          item.Name,
			ContactUserID: item.ContactUserID,
			ContactName:   item.ContactName,
			ContactMobile: item.ContactMobile,
			Status:        int(item.Status),
			Websites:      []string(item.Websites),
			PackageID:     item.PackageID,
			ExpireTime:    item.ExpireDate.UnixMilli(), // 毫秒级时间戳
			AccountCount:  int(item.AccountCount),
			CreateTime:    item.CreateTime,
		})
	}
	return data, nil
}

func (s *TenantService) checkNameUnique(ctx context.Context, name string, excludeId int64) error {
	t := s.q.SystemTenant
	qb := t.WithContext(ctx).Where(t.Name.Eq(name))
	if excludeId > 0 {
		qb = qb.Where(t.ID.Neq(excludeId))
	}
	count, err := qb.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("租户名已存在")
	}
	return nil
}

// ValidTenant 校验租户是否合法 (存在、启用、未过期) - 对齐 Java: validTenant
func (s *TenantService) ValidTenant(ctx context.Context, id int64) error {
	t := s.q.SystemTenant
	tenant, err := t.WithContext(ctx).Where(t.ID.Eq(id)).First()
	if err != nil {
		return errors.New("租户不存在")
	}
	if tenant.Status != 0 {
		return errors.New("租户已禁用: " + tenant.Name)
	}
	if !tenant.ExpireDate.IsZero() && tenant.ExpireDate.Before(time.Now()) {
		return errors.New("租户已过期: " + tenant.Name)
	}
	return nil
}

// GetTenantSimpleList 获取启用状态的租户精简列表
func (s *TenantService) GetTenantSimpleList(ctx context.Context) ([]system.TenantSimpleResp, error) {
	tenantRepo := s.q.SystemTenant
	list, err := tenantRepo.WithContext(ctx).Where(tenantRepo.Status.Eq(0)).Find() // 0 = 启用
	if err != nil {
		return nil, err
	}

	result := make([]system.TenantSimpleResp, 0, len(list))
	for _, t := range list {
		result = append(result, system.TenantSimpleResp{
			ID:   t.ID,
			Name: t.Name,
		})
	}
	return result, nil
}

// GetTenantByWebsite 根据域名查询租户
func (s *TenantService) GetTenantByWebsite(ctx context.Context, website string) (*system.TenantSimpleResp, error) {
	tenantRepo := s.q.SystemTenant
	list, err := tenantRepo.WithContext(ctx).Where(tenantRepo.Status.Eq(0)).Find()
	if err != nil {
		return nil, nil
	}

	// 应用层显式匹配
	for _, t := range list {
		for _, w := range t.Websites {
			if w == website {
				return &system.TenantSimpleResp{
					ID:   t.ID,
					Name: t.Name,
				}, nil
			}
		}
	}
	return nil, nil
}

// isSystemTenant 判断是否为系统租户 (PackageID=0) - 对齐 Java: isSystemTenant
func (s *TenantService) isSystemTenant(tenant *model.SystemTenant) bool {
	return tenant.PackageID == 0
}

// validateUpdateTenant 校验存在并检查是否为系统租户 (对齐 Java: validateUpdateTenant)
func (s *TenantService) validateUpdateTenant(ctx context.Context, id int64) (*model.SystemTenant, error) {
	t := s.q.SystemTenant
	tenant, err := t.WithContext(ctx).Where(t.ID.Eq(id)).First()
	if err != nil {
		return nil, errors.New("租户不存在")
	}
	// 特殊逻辑：系统内置租户，不使用套餐，使用 PackageID=0 标识
	if s.isSystemTenant(tenant) {
		return nil, errors.New("系统内置租户，不允许修改或删除")
	}
	return tenant, nil
}

// validTenantWebsiteDuplicate 校验域名唯一性 (对齐 Java: validTenantWebsiteDuplicate)
func (s *TenantService) validTenantWebsiteDuplicate(ctx context.Context, websites []string, excludeId int64) error {
	if len(websites) == 0 {
		return nil
	}
	t := s.q.SystemTenant
	for _, website := range websites {
		// 这里简单处理：由于 websites 是 JSON 数组，无法直接在 DB 层高效 like 精确匹配（除非用 JSON 函数）
		// 我们先查出所有数据（通常租户不多）或者优化查询
		// TODO: 生产环境建议使用数据库 JSON 函数或 ES 索引
		list, err := t.WithContext(ctx).Find()
		if err != nil {
			return err
		}
		for _, tenant := range list {
			if excludeId > 0 && tenant.ID == excludeId {
				continue
			}
			for _, w := range tenant.Websites {
				if w == website {
					return errors.New("域名 [" + website + "] 已被其他租户绑定")
				}
			}
		}
	}
	return nil
}

// GetTenantIdByName 根据租户名获取租户ID
func (s *TenantService) GetTenantIdByName(ctx context.Context, name string) (*int64, error) {
	tenantRepo := s.q.SystemTenant
	tenant, err := tenantRepo.WithContext(ctx).Where(tenantRepo.Name.Eq(name)).First()
	if err != nil {
		return nil, nil // 未找到返回 nil
	}
	return &tenant.ID, nil
}

// GetTenantByName 根据租户名获取租户（供 AuthService 使用）
func (s *TenantService) GetTenantByName(ctx context.Context, name string) (*system.TenantSimpleResp, error) {
	tenantRepo := s.q.SystemTenant
	tenant, err := tenantRepo.WithContext(ctx).Where(tenantRepo.Name.Eq(name)).First()
	if err != nil {
		return nil, err
	}
	return &system.TenantSimpleResp{
		ID:   tenant.ID,
		Name: tenant.Name,
	}, nil
}

// HandleTenantMenu 处理租户菜单过滤
// handler 接收租户允许的菜单ID列表，并在回调中移除不在列表中的菜单
// 如果是系统租户，传入 nil 表示允许所有菜单
func (s *TenantService) HandleTenantMenu(ctx context.Context, handler func(allowedMenuIds []int64)) error {
	// 1. 获得租户ID
	var tenantId int64
	if c, ok := ctx.(*gin.Context); ok {
		tenantId = pkgContext.GetTenantId(c)
	}
	if tenantId == 0 {
		return nil // 如果没有租户上下文（如admin），不做过滤
	}

	// 2. 获得租户
	tenantRepo := s.q.SystemTenant
	tenant, err := tenantRepo.WithContext(ctx).Where(tenantRepo.ID.Eq(tenantId)).First()
	if err != nil {
		return err
	}

	// 3. 如果是系统租户 (PackageID=0), 允许所有菜单
	if tenant.PackageID == 0 {
		handler(nil) // nil 表示允许所有
		return nil
	}

	// 4. 读取租户套餐
	var pkg model.SystemTenantPackage
	if err := s.q.SystemTenant.WithContext(ctx).UnderlyingDB().Model(&model.SystemTenantPackage{}).Where("id = ?", tenant.PackageID).First(&pkg).Error; err != nil {
		// 如果套餐不存在，当作无权限处理
		handler([]int64{})
		return nil
	}

	allowedMenuIds := pkg.MenuIDs

	// 6. 执行处理
	handler(allowedMenuIds)
	return nil
}

// updateTenantRoleMenu 更新租户下所有角色的菜单权限
func (s *TenantService) updateTenantRoleMenu(ctx context.Context, tenantId int64, menuIds []int64) error {
	// 通过 Q 的 Where 条件保证租户隔离
	r := s.q.SystemRole
	roles, err := r.WithContext(ctx).Where(r.TenantID.Eq(tenantId)).Find()
	if err != nil {
		return err
	}

	for _, role := range roles {
		if role.Code == "tenant_admin" {
			// 超级管理员：直接分配套餐的所有功能
			if err := s.permissionSvc.AssignRoleMenu(ctx, role.ID, menuIds); err != nil {
				return err
			}
		} else {
			// 普通用户：裁剪掉超出套餐功能的权限 (交集)
			// 1. 获取角色当前拥有的菜单
			roleMenuIds, err := s.permissionSvc.GetRoleMenuListByRoleId(ctx, []int64{role.ID})
			if err != nil {
				return err
			}
			// 2. 取交集
			newRoleMenuIds := utils.Intersect(roleMenuIds, menuIds)
			// 3. 重新分配
			if err := s.permissionSvc.AssignRoleMenu(ctx, role.ID, newRoleMenuIds); err != nil {
				return err
			}
		}
	}
	return nil
}
