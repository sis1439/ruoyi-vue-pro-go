package system

import (
	"context"
	"strings"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/cache"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
)

type AuthService struct {
	repo          *query.Query
	permSvc       *PermissionService
	roleSvc       *RoleService
	menuSvc       *MenuService
	tokenSvc      *OAuth2TokenService
	smsCodeSvc    *SmsCodeService
	loginLogSvc   *LoginLogService
	userSvc       *UserService
	socialUserSvc *SocialUserService
}

func NewAuthService(
	repo *query.Query,
	permSvc *PermissionService,
	roleSvc *RoleService,
	menuSvc *MenuService,
	tokenSvc *OAuth2TokenService,
	smsCodeSvc *SmsCodeService,
	loginLogSvc *LoginLogService,
	userSvc *UserService,
	socialUserSvc *SocialUserService,
) *AuthService {
	return &AuthService{
		repo:          repo,
		permSvc:       permSvc,
		roleSvc:       roleSvc,
		menuSvc:       menuSvc,
		tokenSvc:      tokenSvc,
		smsCodeSvc:    smsCodeSvc,
		loginLogSvc:   loginLogSvc,
		userSvc:       userSvc,
		socialUserSvc: socialUserSvc,
	}
}

// GetPermissionInfo 获取登录用户的权限信息
/* ... Unchanged ... */

/* ... Skip to SocialAuthRedirect ... */

// SocialAuthRedirect 社交授权跳转
func (s *AuthService) SocialAuthRedirect(ctx context.Context, socialType int, redirectUri string) (string, error) {
	return s.socialUserSvc.GetAuthorizeUrl(ctx, socialType, consts.UserTypeAdmin, redirectUri)
}

// SocialLogin 社交登录
func (s *AuthService) SocialLogin(ctx context.Context, req *system.AuthSocialLoginReq) (*system.AuthLoginResp, error) {
	// 1. 获取社交用户及绑定用户ID
	_, userId, err := s.socialUserSvc.GetSocialUserByCode(ctx, consts.UserTypeAdmin, req.Type, req.Code, req.State)
	if err != nil {
		return nil, err
	}

	// 2. 检查是否绑定
	if userId == 0 {
		return nil, errors.NewBizError(1002004004, "社交账号未绑定，请先绑定")
	}

	// 3. 获取用户信息
	userRepo := s.repo.SystemUser
	user, err := userRepo.WithContext(ctx).Where(userRepo.ID.Eq(userId)).First()
	if err != nil {
		return nil, errors.NewBizError(1002000002, "用户不存在")
	}

	// 4. 校验状态
	if user.Status != 0 {
		return nil, errors.NewBizError(1002000001, "用户已被禁用")
	}

	// 5. 构建用户信息
	userInfo := map[string]string{
		"nickname": user.Nickname,
	}

	// 6. 创建访问令牌
	tokenDO, err := s.tokenSvc.CreateAccessToken(ctx, user.ID, consts.UserTypeAdmin, user.TenantID, userInfo)
	if err != nil {
		return nil, errors.ErrUnknown
	}

	// 7. 记录登录日志
	s.loginLogSvc.CreateLoginLog(ctx, user.ID, consts.UserTypeAdmin, user.Username, user.LoginIP, "", consts.LoginLogTypeSocial, consts.LoginResultSuccess)

	return &system.AuthLoginResp{
		UserId:       user.ID,
		AccessToken:  tokenDO.AccessToken,
		RefreshToken: tokenDO.RefreshToken,
		ExpiresTime:  tokenDO.ExpiresTime,
	}, nil
}

// GetPermissionInfo 获取登录用户的权限信息
func (s *AuthService) GetPermissionInfo(ctx context.Context) (*system.AuthPermissionInfoResp, error) {
	// 1. 获取当前用户 ID (从 Context)
	userIdVal := ctx.Value(pkgContext.CtxUserIDKey)
	if userIdVal == nil {
		return nil, errors.NewBizError(401, "未登录")
	}
	userId, ok := userIdVal.(int64)
	if !ok {
		return nil, errors.NewBizError(401, "用户标识无效")
	}

	// 2. 获取用户信息
	uRepo := s.repo.SystemUser
	user, err := uRepo.WithContext(ctx).Where(uRepo.ID.Eq(userId)).First()
	if err != nil {
		// 区分是否是 RecordNotFound (已在 InitDB 中忽略日志，这里 err 可能为 gorm.ErrRecordNotFound)
		// 但为了安全，通常模糊返回
		return nil, errors.NewBizError(1002000002, "账号或密码不正确")
	}

	// 3. 获取用户角色
	roleIds, err := s.permSvc.GetUserRoleIdListByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}
	// 获取角色Code列表
	rolesData, err := s.roleSvc.GetRoleList(ctx, roleIds)
	if err != nil {
		return nil, err
	}
	// 过滤禁用的角色 (Java: roles.removeIf(role -> !CommonStatusEnum.ENABLE.getStatus().equals(role.getStatus())))
	var roles []string
	var enabledRoleIds []int64 // 仅保留启用的角色ID
	for _, r := range rolesData {
		if r.Status == consts.CommonStatusEnable {
			roles = append(roles, r.Code)
			enabledRoleIds = append(enabledRoleIds, r.ID)
		}
	}

	// 4. 获取角色菜单 (使用启用的角色ID，而非全部角色ID)
	menuIds, err := s.permSvc.GetRoleMenuListByRoleId(ctx, enabledRoleIds)
	if err != nil {
		return nil, err
	}

	// 5. 获取菜单列表
	menus, err := s.menuSvc.GetMenuListByIds(ctx, menuIds)
	if err != nil {
		return nil, err
	}

	// 5.1 过滤禁用的菜单 (Java: menuList = menuService.filterDisableMenus(menuList))
	var enabledMenus []*system.MenuResp
	for _, m := range menus {
		if m.Status == consts.CommonStatusEnable {
			enabledMenus = append(enabledMenus, m)
		}
	}

	// 6. 获取角色权限 (从菜单中提取)
	permissions := make([]string, 0)
	for _, m := range enabledMenus {
		if m.Permission != "" {
			permissions = append(permissions, m.Permission)
		}
	}

	// 7. 构建菜单树
	menuTree := s.menuSvc.BuildMenuTree(enabledMenus)

	return &system.AuthPermissionInfoResp{
		User: system.UserVO{
			ID:       user.ID,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			DeptID:   user.DeptID,
			Username: user.Username,
			Email:    user.Email,
		},
		Roles:       roles,
		Permissions: permissions,
		Menus:       menuTree,
	}, nil
}

// Login 登录业务
func (s *AuthService) Login(ctx context.Context, req *system.AuthLoginReq) (*system.AuthLoginResp, error) {
	if req.CaptchaVerification != "" {
		if err := NewCaptchaService(cache.RDB).ConsumeVerification(ctx, req.CaptchaVerification); err != nil {
			return nil, err
		}
	}
	// 0. 解析租户
	tenantId, ok := pkgContext.TenantID(ctx)
	if !ok {
		return nil, errors.NewBizError(403, "可信租户缺失")
	}
	if req.TenantName != "" {
		tenantRepo := s.repo.SystemTenant
		tenant, err := tenantRepo.WithContext(ctx).Where(tenantRepo.Name.Eq(req.TenantName)).First()
		if err != nil {
			return nil, errors.NewBizError(1002000003, "租户不存在")
		}
		if tenant.Status != consts.CommonStatusEnable {
			return nil, errors.NewBizError(1002000004, "租户已被禁用")
		}
		if tenant.ID != tenantId {
			return nil, errors.NewBizError(403, "租户不匹配")
		}
	}

	// 1. 查询用户
	userRepo := s.repo.SystemUser
	user, err := userRepo.WithContext(ctx).Where(userRepo.Username.Eq(req.Username), userRepo.TenantID.Eq(tenantId)).First()
	if err != nil {
		// 区分是否是 RecordNotFound (已在 InitDB 中忽略日志，这里 err 可能为 gorm.ErrRecordNotFound)
		// 但为了安全，通常模糊返回
		return nil, errors.NewBizError(1002000002, "账号或密码不正确")
	}

	// 2. 校验状态
	if user.Status != consts.CommonStatusEnable {
		return nil, errors.NewBizError(1002000001, "用户已被禁用")
	}

	// 3. 校验密码
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, errors.NewBizError(1002000002, "账号或密码不正确")
	}

	// 4. 构建用户信息
	userInfo := map[string]string{
		"nickname": user.Nickname,
	}
	if user.DeptID != 0 {
		userInfo["deptId"] = string(rune(user.DeptID))
	}

	// 5. 创建访问令牌（使用 OAuth2TokenService，与 Java 对齐）
	tokenDO, err := s.tokenSvc.CreateAccessToken(ctx, user.ID, consts.UserTypeAdmin, tenantId, userInfo)
	if err != nil {
		return nil, errors.ErrUnknown
	}

	// 6. 返回结果
	return &system.AuthLoginResp{
		UserId:       user.ID,
		AccessToken:  tokenDO.AccessToken,
		RefreshToken: tokenDO.RefreshToken,
		ExpiresTime:  tokenDO.ExpiresTime,
	}, nil
}

// Logout 登出
func (s *AuthService) Logout(ctx context.Context, token string) error {
	// 1. 处理 token，移除 Bearer 前缀
	if strings.HasPrefix(strings.ToUpper(token), "BEARER ") {
		token = token[7:]
	}
	if token == "" {
		return nil
	}

	// 2. 使用 OAuth2TokenService 删除访问令牌
	tokenDO, err := s.tokenSvc.RemoveAccessToken(ctx, token)
	if err != nil {
		return err
	}

	// 3. 记录登出日志
	if tokenDO != nil {
		s.loginLogSvc.CreateLogoutLog(ctx, tokenDO.UserID, tokenDO.UserType, "", "", "")
	}
	return nil
}

// RefreshToken 刷新令牌
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*system.AuthLoginResp, error) {
	// 1. 验证 refreshToken（从 Redis 获取原令牌信息）
	oldToken, err := s.tokenSvc.GetRefreshToken(ctx, refreshToken, consts.UserTypeAdmin)
	if err != nil || oldToken == nil {
		return nil, errors.NewBizError(1002000005, "刷新令牌无效或已过期")
	}

	// 2. 获取用户信息
	ctx = pkgContext.WithTenant(ctx, oldToken.TenantID)
	userRepo := s.repo.SystemUser
	user, err := userRepo.WithContext(ctx).Where(userRepo.ID.Eq(oldToken.UserID)).First()
	if err != nil {
		return nil, errors.NewBizError(1002000002, "用户不存在")
	}

	// 3. 校验状态
	if user.Status != 0 {
		return nil, errors.NewBizError(1002000001, "用户已被禁用")
	}

	// 4. 构建用户信息
	userInfo := map[string]string{
		"nickname": user.Nickname,
	}

	// 5. 创建新的访问令牌
	tokenDO, err := s.tokenSvc.RefreshAccessToken(ctx, refreshToken, user.ID, consts.UserTypeAdmin, user.TenantID, userInfo)
	if err != nil {
		return nil, errors.ErrUnknown
	}

	// 6. 返回结果
	return &system.AuthLoginResp{
		UserId:       user.ID,
		AccessToken:  tokenDO.AccessToken,
		RefreshToken: tokenDO.RefreshToken,
		ExpiresTime:  tokenDO.ExpiresTime,
	}, nil
}

// SmsLogin 短信登录
func (s *AuthService) SmsLogin(ctx context.Context, req *system.AuthSmsLoginReq) (*system.AuthLoginResp, error) {
	// 1. 验证短信验证码 (场景: 登录)
	if err := s.smsCodeSvc.UseSmsCode(ctx, req.Mobile, consts.SmsSceneAdminMemberLogin, req.Code, ""); err != nil {
		return nil, err
	}

	// 2. 根据手机号查询用户
	userRepo := s.repo.SystemUser
	user, err := userRepo.WithContext(ctx).Where(userRepo.Mobile.Eq(req.Mobile)).First()
	if err != nil {
		return nil, errors.NewBizError(1002000002, "账号不存在")
	}

	// 3. 校验状态
	if user.Status != 0 {
		return nil, errors.NewBizError(1002000001, "用户已被禁用")
	}

	// 4. 构建用户信息
	userInfo := map[string]string{
		"nickname": user.Nickname,
	}

	// 5. 创建访问令牌
	tokenDO, err := s.tokenSvc.CreateAccessToken(ctx, user.ID, consts.UserTypeAdmin, user.TenantID, userInfo)
	if err != nil {
		return nil, errors.ErrUnknown
	}

	// 6. 记录登录日志
	s.loginLogSvc.CreateLoginLog(ctx, user.ID, consts.UserTypeAdmin, user.Username, user.LoginIP, "", consts.LoginLogTypeSms, consts.LoginResultSuccess)

	return &system.AuthLoginResp{
		UserId:       user.ID,
		AccessToken:  tokenDO.AccessToken,
		RefreshToken: tokenDO.RefreshToken,
		ExpiresTime:  tokenDO.ExpiresTime,
	}, nil
}

// SendSmsCode 发送短信验证码
func (s *AuthService) SendSmsCode(ctx context.Context, req *system.AuthSmsSendReq, createIp string) error {
	return s.smsCodeSvc.SendSmsCode(ctx, req.Mobile, int32(req.Scene), createIp)
}

// Register 注册
func (s *AuthService) Register(ctx context.Context, r *system.AuthRegisterReq) (*system.AuthLoginResp, error) {
	// 0. 参数校验
	// TODO: 校验密码强度等 (Java: Validator)

	// 1. 创建用户
	createReq := &system.UserSaveReq{
		Username: r.Username,
		Password: r.Password,
		Nickname: r.Username, // 默认昵称
		Status:   consts.CommonStatusEnable,
		RoleIDs:  []int64{}, // 空角色
		PostIDs:  []int64{}, // 空岗位
	}

	_, err := s.userSvc.CreateUser(ctx, createReq)
	if err != nil {
		return nil, err
	}

	// 2. 自动登录
	// 构造登录请求 Mock
	loginReq := &system.AuthLoginReq{
		Username: r.Username,
		Password: r.Password,
	}
	return s.Login(ctx, loginReq)
}

// ResetPassword 重置密码
func (s *AuthService) ResetPassword(ctx context.Context, req *system.AuthResetPasswordReq) error {
	if err := validateAdminPassword(req.Password); err != nil {
		return err
	}
	// 1. 验证短信验证码 (场景: 重置密码)
	if err := s.smsCodeSvc.UseSmsCode(ctx, req.Mobile, consts.SmsSceneAdminResetPassword, req.Code, ""); err != nil {
		return err
	}

	// 2. 根据手机号查询用户
	userRepo := s.repo.SystemUser
	user, err := userRepo.WithContext(ctx).Where(userRepo.Mobile.Eq(req.Mobile)).First()
	if err != nil {
		return errors.NewBizError(1002000002, "用户不存在")
	}

	if err := rejectReservedUser(ctx, s.repo, user.ID); err != nil {
		return err
	}
	// 3. 更新密码
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return errors.ErrUnknown
	}

	_, err = userRepo.WithContext(ctx).Where(userRepo.ID.Eq(user.ID)).Update(userRepo.Password, hashedPassword)
	return err
}

// End of file
