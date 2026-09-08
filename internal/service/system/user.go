package system

import (
	"context"
	"errors"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
)

type UserService struct {
	q       *query.Query
	deptSvc *DeptService
}

func NewUserService(q *query.Query, deptSvc *DeptService) *UserService {
	return &UserService{
		q:       q,
		deptSvc: deptSvc,
	}
}

// GetSimpleUserList 获取用户精简列表（只包含启用用户）
func (s *UserService) GetSimpleUserList(ctx context.Context) ([]system.UserSimpleRespVO, error) {
	u := s.q.SystemUser
	list, err := u.WithContext(ctx).Where(u.Status.Eq(consts.CommonStatusEnable)).Find()
	if err != nil {
		return nil, err
	}

	result := make([]system.UserSimpleRespVO, 0, len(list))
	for _, user := range list {
		result = append(result, system.UserSimpleRespVO{
			ID:       user.ID,
			Nickname: user.Nickname,
		})
	}
	return result, nil
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, req *system.UserSaveReq) (int64, error) {
	if err := validateAssignableRoles(ctx, s.q, req.RoleIDs); err != nil {
		return 0, err
	}
	// 1. 校验唯一性
	if err := s.checkUsernameUnique(ctx, req.Username, 0); err != nil {
		return 0, err
	}
	if req.Mobile != "" {
		if err := s.checkMobileUnique(ctx, req.Mobile, 0); err != nil {
			return 0, err
		}
	}
	if req.Email != "" {
		if err := s.checkEmailUnique(ctx, req.Email, 0); err != nil {
			return 0, err
		}
	}

	// 2. 加密密码（空密码时使用默认密码）
	if err := validateAdminPassword(req.Password); err != nil {
		return 0, err
	}
	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		return 0, err
	}

	// 3. 构造用户对象
	user := &model.SystemUser{
		Username: req.Username,
		Password: hashedPwd,
		Nickname: req.Nickname,
		DeptID:   req.DeptID,
		PostIDs:  "",
		Email:    req.Email,
		Mobile:   req.Mobile,
		Sex:      req.Sex,
		Avatar:   req.Avatar,
		Status:   int32(req.Status),
		Remark:   req.Remark,
	}

	// 4. 事务执行
	err = s.q.Transaction(func(tx *query.Query) error {
		// 4.1 插入用户
		if err := tx.SystemUser.WithContext(ctx).Create(user); err != nil {
			return err
		}

		// 4.2 关联岗位
		if len(req.PostIDs) > 0 {
			var userPosts []*model.SystemUserPost
			for _, postId := range req.PostIDs {
				userPosts = append(userPosts, &model.SystemUserPost{
					UserID: user.ID,
					PostID: postId,
				})
			}
			if err := tx.SystemUserPost.WithContext(ctx).Create(userPosts...); err != nil {
				return err
			}
		}

		// 4.3 关联角色
		if len(req.RoleIDs) > 0 {
			var userRoles []*model.SystemUserRole
			for _, roleId := range req.RoleIDs {
				userRoles = append(userRoles, &model.SystemUserRole{
					UserID: user.ID,
					RoleID: roleId,
				})
			}
			if err := tx.SystemUserRole.WithContext(ctx).Create(userRoles...); err != nil {
				return err
			}
		}
		return nil
	})

	return user.ID, err
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(ctx context.Context, req *system.UserSaveReq) error {
	if err := validateAssignableRoles(ctx, s.q, req.RoleIDs); err != nil {
		return err
	}
	if err := rejectReservedUser(ctx, s.q, req.ID); err != nil {
		return err
	}
	// 1. 校验存在
	u := s.q.SystemUser
	_, err := u.WithContext(ctx).Where(u.ID.Eq(req.ID)).First()
	if err != nil {
		return errors.New("用户不存在")
	}

	// 2. 校验唯一性
	if err := s.checkUsernameUnique(ctx, req.Username, req.ID); err != nil {
		return err
	}
	if req.Mobile != "" {
		if err := s.checkMobileUnique(ctx, req.Mobile, req.ID); err != nil {
			return err
		}
	}
	if req.Email != "" {
		if err := s.checkEmailUnique(ctx, req.Email, req.ID); err != nil {
			return err
		}
	}

	// 3. 事务更新
	return s.q.Transaction(func(tx *query.Query) error {
		// 3.1 更新基本信息
		_, err := tx.SystemUser.WithContext(ctx).Where(u.ID.Eq(req.ID)).Updates(&model.SystemUser{
			Nickname: req.Nickname,
			DeptID:   req.DeptID,
			Email:    req.Email,
			Mobile:   req.Mobile,
			Sex:      req.Sex,
			Avatar:   req.Avatar,
			Status:   int32(req.Status),
			Remark:   req.Remark,
		})
		if err != nil {
			return err
		}

		// 3.2 更新岗位 (Delete + Insert)
		if _, err := tx.SystemUserPost.WithContext(ctx).Where(tx.SystemUserPost.UserID.Eq(req.ID)).Delete(); err != nil {
			return err
		}
		if len(req.PostIDs) > 0 {
			var userPosts []*model.SystemUserPost
			for _, postId := range req.PostIDs {
				userPosts = append(userPosts, &model.SystemUserPost{
					UserID: req.ID,
					PostID: postId,
				})
			}
			if err := tx.SystemUserPost.WithContext(ctx).Create(userPosts...); err != nil {
				return err
			}
		}

		// 3.3 更新角色 (Delete + Insert)
		if _, err := tx.SystemUserRole.WithContext(ctx).Where(tx.SystemUserRole.UserID.Eq(req.ID)).Delete(); err != nil {
			return err
		}
		if len(req.RoleIDs) > 0 {
			var userRoles []*model.SystemUserRole
			for _, roleId := range req.RoleIDs {
				userRoles = append(userRoles, &model.SystemUserRole{
					UserID: req.ID,
					RoleID: roleId,
				})
			}
			if err := tx.SystemUserRole.WithContext(ctx).Create(userRoles...); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteUser 删除用户
// 对应 Java: AdminUserServiceImpl.deleteUser
func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	if err := rejectReservedUser(ctx, s.q, id); err != nil {
		return err
	}
	// 1. 校验用户存在
	u := s.q.SystemUser
	user, err := u.WithContext(ctx).Where(u.ID.Eq(id)).First()
	if err != nil {
		return errors.New("用户不存在")
	}
	if user == nil {
		return errors.New("用户不存在")
	}

	// 2. 删除用户及关联数据（使用事务）
	return s.q.Transaction(func(tx *query.Query) error {
		// 2.1 删除用户
		if _, err := tx.SystemUser.WithContext(ctx).Where(tx.SystemUser.ID.Eq(id)).Delete(); err != nil {
			return err
		}

		// 2.2 删除用户角色关联
		if _, err := tx.SystemUserRole.WithContext(ctx).Where(tx.SystemUserRole.UserID.Eq(id)).Delete(); err != nil {
			return err
		}

		// 2.3 删除用户岗位关联
		if _, err := tx.SystemUserPost.WithContext(ctx).Where(tx.SystemUserPost.UserID.Eq(id)).Delete(); err != nil {
			return err
		}

		return nil
	})
}

// GetUser 获得用户详情
// 对应 Java: AdminUserServiceImpl.getUser + 返回完整角色和岗位信息
func (s *UserService) GetUser(ctx context.Context, id int64) (*system.UserProfileRespVO, error) {
	u := s.q.SystemUser
	user, err := u.WithContext(ctx).Where(u.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	// 1. 获取用户角色ID列表
	ur := s.q.SystemUserRole
	userRoles, _ := ur.WithContext(ctx).Where(ur.UserID.Eq(id)).Find()
	roleIds := make([]int64, len(userRoles))
	for i, r := range userRoles {
		roleIds[i] = r.RoleID
	}

	// 2. 获取角色详情
	var roles []*system.RoleSimpleRespVO
	if len(roleIds) > 0 {
		roleList, _ := s.q.SystemRole.WithContext(ctx).Where(s.q.SystemRole.ID.In(roleIds...)).Find()
		for _, role := range roleList {
			roles = append(roles, &system.RoleSimpleRespVO{
				ID:   role.ID,
				Name: role.Name,
			})
		}
	}

	// 3. 获取用户岗位ID列表
	up := s.q.SystemUserPost
	userPosts, _ := up.WithContext(ctx).Where(up.UserID.Eq(id)).Find()
	postIds := make([]int64, len(userPosts))
	for i, p := range userPosts {
		postIds[i] = p.PostID
	}

	// 4. 获取岗位详情
	var posts []*system.PostSimpleRespVO
	if len(postIds) > 0 {
		postList, _ := s.q.SystemPost.WithContext(ctx).Where(s.q.SystemPost.ID.In(postIds...)).Find()
		for _, post := range postList {
			posts = append(posts, &system.PostSimpleRespVO{
				ID:   post.ID,
				Name: post.Name,
			})
		}
	}

	// 5. 获取部门信息
	var dept *system.DeptSimpleRespVO
	var deptName string
	if user.DeptID > 0 {
		deptInfo, _ := s.q.SystemDept.WithContext(ctx).Where(s.q.SystemDept.ID.Eq(user.DeptID)).First()
		if deptInfo != nil {
			dept = &system.DeptSimpleRespVO{
				ID:       deptInfo.ID,
				Name:     deptInfo.Name,
				ParentID: deptInfo.ParentID,
			}
			deptName = deptInfo.Name
		}
	}

	return &system.UserProfileRespVO{
		UserRespVO: &system.UserRespVO{
			ID:         user.ID,
			Username:   user.Username,
			Nickname:   user.Nickname,
			Remark:     user.Remark,
			DeptID:     user.DeptID,
			DeptName:   deptName,
			PostIDs:    postIds,
			RoleIDs:    roleIds,
			Email:      user.Email,
			Mobile:     user.Mobile,
			Sex:        user.Sex,
			Avatar:     user.Avatar,
			Status:     user.Status,
			LoginIP:    user.LoginIP,
			CreateTime: user.CreateTime,
		},
		Roles: roles,
		Posts: posts,
		Dept:  dept,
	}, nil
}

// GetUserPage 获得用户分页
func (s *UserService) GetUserPage(ctx context.Context, req *system.UserPageReq) (*pagination.PageResult[*system.UserRespVO], error) {
	u := s.q.SystemUser
	qb := u.WithContext(ctx)

	if req.Username != "" {
		qb = qb.Where(u.Username.Like("%" + req.Username + "%"))
	}
	if req.Mobile != "" {
		qb = qb.Where(u.Mobile.Like("%" + req.Mobile + "%"))
	}
	if req.Status != nil {
		qb = qb.Where(u.Status.Eq(int32(*req.Status)))
	}
	if req.DeptID > 0 {
		// 部门过滤：包含指定部门及其所有子部门
		// 对应 Java: getDeptCondition(deptId)
		deptIds := []int64{req.DeptID}
		childDeptIds, _ := s.deptSvc.GetDeptIdListByParentId(ctx, req.DeptID)
		deptIds = append(deptIds, childDeptIds...)
		qb = qb.Where(u.DeptID.In(deptIds...))
	}
	if req.CreateTimeGe != nil {
		qb = qb.Where(u.CreateTime.Gte(*req.CreateTimeGe))
	}
	if req.CreateTimeLe != nil {
		qb = qb.Where(u.CreateTime.Lte(*req.CreateTimeLe))
	}

	if req.RoleID > 0 {
		// 角色过滤
		ur := s.q.SystemUserRole
		userRoles, _ := ur.WithContext(ctx).Where(ur.RoleID.Eq(req.RoleID)).Find()
		if len(userRoles) == 0 {
			return &pagination.PageResult[*system.UserRespVO]{
				List:  []*system.UserRespVO{},
				Total: 0,
			}, nil
		}
		userIds := make([]int64, len(userRoles))
		for i, v := range userRoles {
			userIds[i] = v.UserID
		}
		qb = qb.Where(u.ID.In(userIds...))
	}

	total, err := qb.Count()
	if err != nil {
		return nil, err
	}

	list, err := qb.Order(u.ID).Offset(req.GetOffset()).Limit(req.PageSize).Find()
	if err != nil {
		return nil, err
	}

	var data []*system.UserRespVO
	// 批量获取部门名称
	deptIDs := make([]int64, 0, len(list))
	for _, item := range list {
		if item.DeptID > 0 {
			deptIDs = append(deptIDs, item.DeptID)
		}
	}
	deptMap := make(map[int64]string)
	if len(deptIDs) > 0 {
		depts, _ := s.q.SystemDept.WithContext(ctx).Where(s.q.SystemDept.ID.In(deptIDs...)).Find()
		for _, d := range depts {
			deptMap[d.ID] = d.Name
		}
	}

	for _, item := range list {
		data = append(data, &system.UserRespVO{
			ID:         item.ID,
			Username:   item.Username,
			Nickname:   item.Nickname,
			Remark:     item.Remark,
			DeptID:     item.DeptID,
			DeptName:   deptMap[item.DeptID],
			Email:      item.Email,
			Mobile:     item.Mobile,
			Sex:        item.Sex,
			Avatar:     item.Avatar,
			Status:     item.Status,
			LoginIP:    item.LoginIP,
			CreateTime: item.CreateTime,
		})
	}

	return &pagination.PageResult[*system.UserRespVO]{
		List:  data,
		Total: total,
	}, nil
}

// UpdateUserStatus 修改用户状态
func (s *UserService) UpdateUserStatus(ctx context.Context, req *system.UserUpdateStatusReq) error {
	if err := rejectReservedUser(ctx, s.q, req.ID); err != nil {
		return err
	}
	u := s.q.SystemUser
	_, err := u.WithContext(ctx).Where(u.ID.Eq(req.ID)).Update(u.Status, int32(*req.Status))
	return err
}

// UpdateUserPassword 修改用户密码
func (s *UserService) UpdateUserPassword(ctx context.Context, req *system.UserUpdatePasswordReq) error {
	if err := validateAdminPassword(req.Password); err != nil {
		return err
	}
	if err := rejectReservedUser(ctx, s.q, req.ID); err != nil {
		return err
	}
	u := s.q.SystemUser
	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		return err
	}
	_, err = u.WithContext(ctx).Where(u.ID.Eq(req.ID)).Update(u.Password, hashedPwd)
	return err
}

// ResetUserPassword 重置用户密码
func (s *UserService) ResetUserPassword(ctx context.Context, req *system.UserResetPasswordReq) error {
	if err := validateAdminPassword(req.Password); err != nil {
		return err
	}
	if err := rejectReservedUser(ctx, s.q, req.ID); err != nil {
		return err
	}
	u := s.q.SystemUser
	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		return err
	}
	_, err = u.WithContext(ctx).Where(u.ID.Eq(req.ID)).Update(u.Password, hashedPwd)
	return err
}

// GetUserList 获得用户列表 (用于导出)
func (s *UserService) GetUserList(ctx context.Context, req *system.UserExportReq) ([]*system.UserRespVO, error) {
	u := s.q.SystemUser
	qb := u.WithContext(ctx)

	if req.Username != "" {
		qb = qb.Where(u.Username.Like("%" + req.Username + "%"))
	}
	if req.Mobile != "" {
		qb = qb.Where(u.Mobile.Like("%" + req.Mobile + "%"))
	}
	if req.Status != nil {
		qb = qb.Where(u.Status.Eq(int32(*req.Status)))
	}
	if req.DeptID > 0 {
		qb = qb.Where(u.DeptID.Eq(req.DeptID))
	}
	if req.CreateTimeGe != nil {
		qb = qb.Where(u.CreateTime.Gte(*req.CreateTimeGe))
	}
	if req.CreateTimeLe != nil {
		qb = qb.Where(u.CreateTime.Lte(*req.CreateTimeLe))
	}

	list, err := qb.Order(u.ID).Find()
	if err != nil {
		return nil, err
	}

	var data []*system.UserRespVO
	// 批量获取部门名称
	deptIDs := make([]int64, 0, len(list))
	for _, item := range list {
		if item.DeptID > 0 {
			deptIDs = append(deptIDs, item.DeptID)
		}
	}
	deptMap := make(map[int64]string)
	if len(deptIDs) > 0 {
		depts, _ := s.q.SystemDept.WithContext(ctx).Where(s.q.SystemDept.ID.In(deptIDs...)).Find()
		for _, d := range depts {
			deptMap[d.ID] = d.Name
		}
	}

	for _, item := range list {
		data = append(data, &system.UserRespVO{
			ID:         item.ID,
			Username:   item.Username,
			Nickname:   item.Nickname,
			Remark:     item.Remark,
			DeptID:     item.DeptID,
			DeptName:   deptMap[item.DeptID],
			Email:      item.Email,
			Mobile:     item.Mobile,
			Sex:        item.Sex,
			Avatar:     item.Avatar,
			Status:     item.Status,
			LoginIP:    item.LoginIP,
			CreateTime: item.CreateTime,
		})
	}
	return data, nil
}

// GetImportTemplate 获得导入模板
func (s *UserService) GetImportTemplate(ctx context.Context) ([]system.UserImportExcelVO, error) {
	return []system.UserImportExcelVO{
		{
			Username: "zhangsan",
			Nickname: "张三",
			Email:    "zhangsan@yudao.cn",
			Mobile:   "15601691300",
			Sex:      "1",
			Status:   "0",
			DeptID:   100,
		},
		{
			Username: "lisi",
			Nickname: "李四",
			Email:    "lisi@yudao.cn",
			Mobile:   "15601691301",
			Sex:      "2",
			Status:   "0",
			DeptID:   100,
		},
	}, nil
}

// Helpers

func (s *UserService) checkUsernameUnique(ctx context.Context, username string, excludeId int64) error {
	u := s.q.SystemUser
	qb := u.WithContext(ctx).Where(u.Username.Eq(username))
	if excludeId > 0 {
		qb = qb.Where(u.ID.Neq(excludeId))
	}
	count, err := qb.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("用户名已存在")
	}
	return nil
}

func (s *UserService) checkMobileUnique(ctx context.Context, mobile string, excludeId int64) error {
	u := s.q.SystemUser
	qb := u.WithContext(ctx).Where(u.Mobile.Eq(mobile))
	if excludeId > 0 {
		qb = qb.Where(u.ID.Neq(excludeId))
	}
	count, err := qb.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("手机号已存在")
	}
	return nil
}

func (s *UserService) checkEmailUnique(ctx context.Context, email string, excludeId int64) error {
	u := s.q.SystemUser
	qb := u.WithContext(ctx).Where(u.Email.Eq(email))
	if excludeId > 0 {
		qb = qb.Where(u.ID.Neq(excludeId))
	}
	count, err := qb.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("邮箱已存在")
	}
	return nil
}

// DeleteUserList 批量删除用户
func (s *UserService) DeleteUserList(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if err := rejectReservedUser(ctx, s.q, id); err != nil {
			return err
		}
	}
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		if err := s.DeleteUser(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func validateAdminPassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return errors.New("explicit 8..72-byte password required; no default password is provided")
	}
	return nil
}
