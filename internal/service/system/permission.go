package system

import (
	"context"
	"errors"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"

	"github.com/samber/lo"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
)

type PermissionService struct {
	q       *query.Query
	roleSvc *RoleService
}

func NewPermissionService(q *query.Query, roleSvc *RoleService) *PermissionService {
	return &PermissionService{
		q:       q,
		roleSvc: roleSvc,
	}
}

// GetUserRoleIdListByUserId 获取用户的角色ID列表
func (s *PermissionService) GetUserRoleIdListByUserId(ctx context.Context, userId int64) ([]int64, error) {
	ur := s.q.SystemUserRole
	list, err := ur.WithContext(ctx).Where(ur.UserID.Eq(userId)).Find()
	if err != nil {
		return nil, err
	}
	return lo.Map(list, func(item *model.SystemUserRole, _ int) int64 {
		return item.RoleID
	}), nil
}

// GetRoleMenuListByRoleId 获取角色的菜单ID列表
func (s *PermissionService) GetRoleMenuListByRoleId(ctx context.Context, roleIds []int64) ([]int64, error) {
	if len(roleIds) == 0 {
		return []int64{}, nil
	}

	// 如果是管理员的情况下，获取全部菜单编号
	isSuperAdmin, err := s.roleSvc.HasAnySuperAdmin(ctx, roleIds)
	if err != nil {
		return nil, err
	}
	if isSuperAdmin {
		// 超级管理员返回所有菜单
		return s.getAllMenuIds(ctx)
	}

	// 如果是非管理员的情况下，获得拥有的菜单编号
	rm := s.q.SystemRoleMenu
	list, err := rm.WithContext(ctx).Where(rm.RoleID.In(roleIds...)).Find()
	if err != nil {
		return nil, err
	}
	// Extract MenuIDs, distinct
	return lo.Uniq(lo.Map(list, func(item *model.SystemRoleMenu, _ int) int64 {
		return item.MenuID
	})), nil
}

// getAllMenuIds 获取所有菜单ID (用于超级管理员)
func (s *PermissionService) getAllMenuIds(ctx context.Context) ([]int64, error) {
	m := s.q.SystemMenu
	menus, err := m.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	return lo.Map(menus, func(item *model.SystemMenu, _ int) int64 {
		return item.ID
	}), nil
}

// AssignRoleMenu 赋予角色菜单
func (s *PermissionService) AssignRoleMenu(ctx context.Context, roleId int64, menuIds []int64) error {
	role, err := s.GetRoleById(ctx, roleId)
	if err != nil {
		return err
	}
	if reservedRole(role.Code) {
		return errors.New("reserved role requires offline management")
	}

	// 使用事务
	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 删除旧的角色菜单关联
		rm := tx.SystemRoleMenu
		if _, err := rm.WithContext(ctx).Where(rm.RoleID.Eq(roleId)).Delete(); err != nil {
			return err
		}

		// 2. 插入新的角色菜单关联
		if len(menuIds) > 0 {
			var bat []*model.SystemRoleMenu
			for _, mid := range menuIds {
				bat = append(bat, &model.SystemRoleMenu{
					RoleID: roleId,
					MenuID: mid,
				})
			}
			// 批量创建
			if err := rm.WithContext(ctx).Create(bat...); err != nil {
				return err
			}
		}
		return nil
	})
}

// AssignRoleDataScope 赋予角色数据权限
func (s *PermissionService) AssignRoleDataScope(ctx context.Context, roleId int64, dataScope int, deptIds []int64) error {
	return s.roleSvc.UpdateRoleDataScope(ctx, roleId, dataScope, deptIds)
}

// AssignUserRole 赋予用户角色
func (s *PermissionService) AssignUserRole(ctx context.Context, userId int64, roleIds []int64) error {
	if _, err := s.q.SystemUser.WithContext(ctx).Where(s.q.SystemUser.ID.Eq(userId)).First(); err != nil {
		return err
	}
	if err := rejectReservedUser(ctx, s.q, userId); err != nil {
		return err
	}
	for _, id := range lo.Uniq(roleIds) {
		role, err := s.GetRoleById(ctx, id)
		if err != nil {
			return err
		}
		if reservedRole(role.Code) {
			return errors.New("reserved role requires offline assignment")
		}
	}

	return s.q.Transaction(func(tx *query.Query) error {
		ur := tx.SystemUserRole
		// 1. 删除旧的用户角色关联
		if _, err := ur.WithContext(ctx).Where(ur.UserID.Eq(userId)).Delete(); err != nil {
			return err
		}

		// 2. 插入新的用户角色关联
		if len(roleIds) > 0 {
			var bat []*model.SystemUserRole
			for _, rid := range roleIds {
				bat = append(bat, &model.SystemUserRole{
					UserID: userId,
					RoleID: rid,
				})
			}
			if err := ur.WithContext(ctx).Create(bat...); err != nil {
				return err
			}
		}
		return nil
	})
}

// IsSuperAdmin 检查用户是否为超级管理员
func (s *PermissionService) IsSuperAdmin(ctx context.Context, userId int64) (bool, error) {
	if err := s.ValidateAdminScope(ctx, userId); err != nil {
		return false, err
	}
	roleIds, err := s.GetUserRoleIdListByUserId(ctx, userId)
	if err != nil {
		return false, err
	}
	if len(roleIds) == 0 {
		return false, nil
	}
	return s.roleSvc.HasAnySuperAdmin(ctx, roleIds)
}

// GetRoleById 根据角色ID获取角色信息
func (s *PermissionService) GetRoleById(ctx context.Context, roleId int64) (*model.SystemRole, error) {
	r := s.q.SystemRole
	return r.WithContext(ctx).Where(r.ID.Eq(roleId)).First()
}

// GetRoleDeptIdListByRoleId 获取角色的自定义部门ID列表
// 对应 Java: RoleDO.dataScopeDeptIds 字段（JSON 存储在角色表中）
func (s *PermissionService) GetRoleDeptIdListByRoleId(ctx context.Context, roleId int64) ([]int64, error) {
	r := s.q.SystemRole
	role, err := r.WithContext(ctx).Where(r.ID.Eq(roleId)).First()
	if err != nil {
		return []int64{}, nil // 角色不存在时返回空列表
	}
	if role.DataScopeDeptIds == nil {
		return []int64{}, nil
	}
	return []int64(role.DataScopeDeptIds), nil
}

// HasPermission rechecks current enabled roles and menus so cached Casbin grants cannot outlive revocation.
func (s *PermissionService) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	roleIDs, err := s.GetUserRoleIdListByUserId(ctx, userID)
	if err != nil || len(roleIDs) == 0 {
		return false, err
	}
	r := s.q.SystemRole
	roles, err := r.WithContext(ctx).Where(r.ID.In(roleIDs...), r.Status.Eq(0)).Find()
	if err != nil {
		return false, err
	}
	enabled := make([]int64, 0, len(roles))
	for _, role := range roles {
		if role.DataScope != consts.DataScopeAll {
			return false, nil
		}
		enabled = append(enabled, role.ID)
	}
	if len(enabled) == 0 {
		return false, nil
	}
	rm := s.q.SystemRoleMenu
	links, err := rm.WithContext(ctx).Where(rm.RoleID.In(enabled...)).Find()
	if err != nil {
		return false, err
	}
	ids := make([]int64, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.MenuID)
	}
	if len(ids) == 0 {
		return false, nil
	}
	m := s.q.SystemMenu
	count, err := m.WithContext(ctx).Where(m.ID.In(ids...), m.Permission.Eq(permission), m.Status.Eq(0)).Count()
	return count > 0, err
}

// ValidateAdminScope rejects unsupported persisted restrictions instead of treating them as All.
func (s *PermissionService) ValidateAdminScope(ctx context.Context, userID int64) error {
	ids, err := s.GetUserRoleIdListByUserId(ctx, userID)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	r := s.q.SystemRole
	roles, err := r.WithContext(ctx).Where(r.ID.In(ids...), r.Status.Eq(0)).Find()
	if err != nil {
		return err
	}
	for _, role := range roles {
		if role.DataScope != consts.DataScopeAll {
			return errors.New("configured role data scope is not enabled; tenant-wide RBAC only")
		}
	}
	return nil
}
