package system

import (
	"context"
	"errors"

	"github.com/samber/lo"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"

	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type RoleService struct {
	q *query.Query
}

func NewRoleService(q *query.Query) *RoleService {
	return &RoleService{
		q: q,
	}
}

// CreateRole 创建角色
func (s *RoleService) CreateRole(ctx context.Context, req *system.RoleSaveReq) (int64, error) {
	if reservedRole(req.Code) {
		return 0, errors.New("reserved role requires offline provisioning")
	}
	if err := s.checkDuplicate(ctx, req.Name, req.Code, 0); err != nil {
		return 0, err
	}

	role := &model.SystemRole{
		Name:             req.Name,
		Code:             req.Code,
		Sort:             req.Sort,
		Status:           int32(*req.Status),
		Remark:           req.Remark,
		Type:             consts.RoleTypeCustom,
		DataScope:        consts.DataScopeAll,
		DataScopeDeptIds: model.Int64ListFromCSV{}, // Initialize to avoid NULL error
	}

	err := s.q.SystemRole.WithContext(ctx).Create(role)
	return role.ID, err
}

// UpdateRole 更新角色
func (s *RoleService) UpdateRole(ctx context.Context, req *system.RoleSaveReq) error {
	if reservedRole(req.Code) {
		return errors.New("reserved role cannot be assigned through HTTP")
	}
	r := s.q.SystemRole
	role, err := r.WithContext(ctx).Where(r.ID.Eq(req.ID)).First()
	if err != nil {
		return errors.New("角色不存在")
	}
	if reservedRole(role.Code) {
		return errors.New("reserved role requires offline management")
	}
	if role.Type == consts.RoleTypeSystem {
		// Allow updating basic info even for system roles, but maybe restricted in some systems.
		// For now allow it.
	}

	if err := s.checkDuplicate(ctx, req.Name, req.Code, req.ID); err != nil {
		return err
	}

	_, err = r.WithContext(ctx).Where(r.ID.Eq(req.ID)).Updates(&model.SystemRole{
		Name:   req.Name,
		Code:   req.Code,
		Sort:   req.Sort,
		Status: int32(*req.Status),
		Remark: req.Remark,
	})
	return err
}

// UpdateRoleStatus 更新角色状态
func (s *RoleService) UpdateRoleStatus(ctx context.Context, req *system.RoleUpdateStatusReq) error {
	r := s.q.SystemRole
	role, err := r.WithContext(ctx).Where(r.ID.Eq(req.ID)).First()
	if err != nil {
		return errors.New("角色不存在")
	}
	if reservedRole(role.Code) {
		return errors.New("reserved role requires offline management")
	}
	if role.Type == consts.RoleTypeSystem {
		return errors.New("内置角色不能修改状态")
	}
	_, err = r.WithContext(ctx).Where(r.ID.Eq(req.ID)).Update(r.Status, *req.Status)
	return err
}

// UpdateRoleDataScope 更新数据权限
func (s *RoleService) UpdateRoleDataScope(ctx context.Context, roleId int64, dataScope int, deptIds []int64) error {
	if dataScope != consts.DataScopeAll || len(deptIds) > 0 {
		return errors.New("departmental/custom/self data scopes are not enabled; tenant-wide RBAC only")
	}
	r := s.q.SystemRole
	role, err := r.WithContext(ctx).Where(r.ID.Eq(roleId)).First()
	if err != nil {
		return errors.New("角色不存在")
	}

	if reservedRole(role.Code) {
		return errors.New("reserved role requires offline management")
	}
	_, err = r.WithContext(ctx).Where(r.ID.Eq(roleId)).Updates(&model.SystemRole{
		DataScope:        int32(dataScope),
		DataScopeDeptIds: model.Int64ListFromCSV(deptIds), // Handled by serializer:json
	})
	return err
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(ctx context.Context, id int64) error {
	r := s.q.SystemRole
	role, err := r.WithContext(ctx).Where(r.ID.Eq(id)).First()
	if err != nil {
		return errors.New("角色不存在")
	}
	if reservedRole(role.Code) {
		return errors.New("reserved role requires offline management")
	}
	if role.Type == consts.RoleTypeSystem {
		return errors.New("内置角色不能删除")
	}
	// Check assigned users count
	userRoleCount, err := s.q.SystemUserRole.WithContext(ctx).Where(s.q.SystemUserRole.RoleID.Eq(id)).Count()
	if err != nil {
		return err
	}
	if userRoleCount > 0 {
		return errors.New("角色已分配给用户，无法删除")
	}
	_, err = r.WithContext(ctx).Where(r.ID.Eq(id)).Delete()
	return err
}

// GetRole 获得角色
func (s *RoleService) GetRole(ctx context.Context, id int64) (*system.RoleRespVO, error) {
	r := s.q.SystemRole
	item, err := r.WithContext(ctx).Where(r.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return s.convertResp(item), nil
}

// GetRolePage 分页
func (s *RoleService) GetRolePage(ctx context.Context, req *system.RolePageReq) (*pagination.PageResult[*system.RoleRespVO], error) {
	r := s.q.SystemRole
	qb := r.WithContext(ctx)

	if req.Name != "" {
		qb = qb.Where(r.Name.Like("%" + req.Name + "%"))
	}
	if req.Code != "" {
		qb = qb.Where(r.Code.Like("%" + req.Code + "%"))
	}
	if req.Status != nil {
		qb = qb.Where(r.Status.Eq(int32(*req.Status)))
	}
	if req.CreateTimeGe != nil {
		qb = qb.Where(r.CreateTime.Gte(*req.CreateTimeGe))
	}
	if req.CreateTimeLe != nil {
		qb = qb.Where(r.CreateTime.Lte(*req.CreateTimeLe))
	}

	total, err := qb.Count()
	if err != nil {
		return nil, err
	}
	list, err := qb.Order(r.Sort, r.ID).Offset(req.GetOffset()).Limit(req.PageSize).Find()
	if err != nil {
		return nil, err
	}

	return &pagination.PageResult[*system.RoleRespVO]{
		List:  lo.Map(list, func(item *model.SystemRole, _ int) *system.RoleRespVO { return s.convertResp(item) }),
		Total: total,
	}, nil
}

// GetRoleListByStatus 获取全列表
func (s *RoleService) GetRoleListByStatus(ctx context.Context, status int) ([]*system.RoleRespVO, error) {
	r := s.q.SystemRole
	list, err := r.WithContext(ctx).Where(r.Status.Eq(int32(status))).Order(r.Sort, r.ID).Find()
	if err != nil {
		return nil, err
	}
	return lo.Map(list, func(item *model.SystemRole, _ int) *system.RoleRespVO { return s.convertResp(item) }), nil
}

// GetRoleList IDs (Already existed, keep it)
func (s *RoleService) GetRoleList(ctx context.Context, ids []int64) ([]*model.SystemRole, error) {
	if len(ids) == 0 {
		return []*model.SystemRole{}, nil
	}
	r := s.q.SystemRole
	return r.WithContext(ctx).Where(r.ID.In(ids...)).Find()
}

// Helpers

func (s *RoleService) checkDuplicate(ctx context.Context, name, code string, excludeId int64) error {
	r := s.q.SystemRole
	// Name unique
	qb := r.WithContext(ctx).Where(r.Name.Eq(name))
	if excludeId > 0 {
		qb = qb.Where(r.ID.Neq(excludeId))
	}
	count, err := qb.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("角色名称已存在")
	}

	// Code unique
	qb = r.WithContext(ctx).Where(r.Code.Eq(code))
	if excludeId > 0 {
		qb = qb.Where(r.ID.Neq(excludeId))
	}
	count, err = qb.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("角色编码已存在")
	}
	return nil
}

func (s *RoleService) convertResp(item *model.SystemRole) *system.RoleRespVO {
	return &system.RoleRespVO{
		ID:               item.ID,
		Name:             item.Name,
		Code:             item.Code,
		Sort:             item.Sort,
		Status:           item.Status,
		Type:             item.Type,
		Remark:           item.Remark,
		DataScope:        item.DataScope,
		DataScopeDeptIDs: []int64(item.DataScopeDeptIds),
		CreateTime:       item.CreateTime,
	}
}

// HasAnySuperAdmin 判断角色列表中是否包含超级管理员角色
// 对应 Java: RoleServiceImpl.hasAnySuperAdmin
func (s *RoleService) HasAnySuperAdmin(ctx context.Context, roleIds []int64) (bool, error) {
	if len(roleIds) == 0 {
		return false, nil
	}

	r := s.q.SystemRole
	roles, err := r.WithContext(ctx).Where(r.ID.In(roleIds...)).Find()
	if err != nil {
		return false, err
	}

	for _, role := range roles {
		if role.Code == consts.RoleCodeSuperAdmin && role.Status == 0 && role.DataScope == consts.DataScopeAll {
			return true, nil
		}
	}
	return false, nil
}
