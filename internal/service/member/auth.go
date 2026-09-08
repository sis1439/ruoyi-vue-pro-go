package member

import (
	"context"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"strings"

	member2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/member"
	system2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
)

// 确保 utils 包被使用（用于密码校验）
var _ = utils.CheckPasswordHash

type MemberAuthService struct {
	repo        *query.Query
	smsCodeSvc  *system.SmsCodeService
	userSvc     *MemberUserService
	socialSvc   *system.SocialUserService
	tokenSvc    *system.OAuth2TokenService
	loginLogSvc *system.LoginLogService
}

func NewMemberAuthService(repo *query.Query, smsCodeSvc *system.SmsCodeService, userSvc *MemberUserService, socialSvc *system.SocialUserService, tokenSvc *system.OAuth2TokenService, loginLogSvc *system.LoginLogService) *MemberAuthService {
	return &MemberAuthService{
		repo:        repo,
		smsCodeSvc:  smsCodeSvc,
		userSvc:     userSvc,
		socialSvc:   socialSvc,
		tokenSvc:    tokenSvc,
		loginLogSvc: loginLogSvc,
	}
}

// Login 手机+密码登录
func (s *MemberAuthService) Login(ctx context.Context, r *member2.AppAuthLoginReq, ip, userAgent string, terminal int32) (*member2.AppAuthLoginResp, error) {
	// 1. 查询用户
	userRepo := s.repo.MemberUser
	user, err := userRepo.WithContext(ctx).Where(userRepo.Mobile.Eq(r.Mobile)).First()
	if err != nil {
		return nil, member.ErrAuthLoginBadCredentials
	}

	// 2. 校验状态. 0:开启, 1:关闭
	if user.Status != 0 {
		return nil, member.ErrAuthLoginUserDisabled
	}

	// 3. 校验密码
	if !utils.CheckPasswordHash(r.Password, user.Password) {
		return nil, member.ErrAuthLoginBadCredentials
	}

	// 4. Check Social Bind need?
	var openid string
	if r.SocialType != 0 {
		bindReq := &system2.SocialUserBindReq{
			Type:  r.SocialType,
			Code:  r.SocialCode,
			State: r.SocialState,
		}
		var err error
		openid, err = s.socialSvc.BindSocialUser(ctx, user.ID, 1, bindReq) // 1=Member
		if err != nil {
			return nil, err
		}
	}

	// 5. 生成 Token（使用 OAuth2TokenService，UserType=1 表示会员）
	return s.createTokenAfterLoginSuccess(ctx, user, consts.LoginLogTypeUsername, openid, ip, userAgent)
}

// SmsLogin 手机+验证码登录
func (s *MemberAuthService) SmsLogin(ctx context.Context, r *member2.AppAuthSmsLoginReq, ip, userAgent string, terminal int32) (*member2.AppAuthLoginResp, error) {
	// 1. 校验验证码
	if err := s.smsCodeSvc.UseSmsCode(ctx, r.Mobile, 1, r.Code, ip); err != nil { // 1 = SmsSceneMemberLogin
		return nil, err
	}

	// 2-4. 使用事务处理用户创建和社交绑定
	var user *member.MemberUser
	var openid string
	err := s.repo.Transaction(func(tx *query.Query) error {
		// 2. 查询用户，不存在则注册
		u := tx.MemberUser
		var err error
		user, err = u.WithContext(ctx).Where(u.Mobile.Eq(r.Mobile)).First()
		if err != nil {
			// 用户不存在，创建新用户
			user = &member.MemberUser{
				Mobile:           r.Mobile,
				Nickname:         "手机用户" + r.Mobile[len(r.Mobile)-4:],
				RegisterIP:       ip,
				RegisterTerminal: terminal,
				Status:           0, // Enabled
				Point:            0,
				Experience:       0,
			}
			if err := tx.MemberUser.WithContext(ctx).Create(user); err != nil {
				return err
			}
		}

		// 3. 校验状态
		if user.Status != 0 {
			return member.ErrAuthLoginUserDisabled
		}

		// 4. Bind Social if needed
		if r.SocialType != 0 {
			bindReq := &system2.SocialUserBindReq{
				Type:  r.SocialType,
				Code:  r.SocialCode,
				State: r.SocialState,
			}
			var err error
			openid, err = s.socialSvc.BindSocialUser(ctx, user.ID, 1, bindReq)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 5. 在事务外记录登录日志和更新登录信息，避免锁冲突
	result, err := s.createTokenAfterLoginSuccess(ctx, user, consts.LoginLogTypeSms, openid, ip, userAgent)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// SocialLogin 社交快捷登录
func (s *MemberAuthService) SocialLogin(ctx context.Context, r *member2.AppAuthSocialLoginReq, ip, userAgent string, terminal int32) (*member2.AppAuthLoginResp, error) {
	// 1. 获得社交用户
	socialUser, bindUserId, err := s.socialSvc.GetSocialUserByCode(ctx, 1, int(r.Type), r.Code, r.State) // 1=Member
	if err != nil {
		return nil, err
	}
	if socialUser == nil {
		return nil, member.ErrAuthSocialUserNotFound
	}

	// 2-4. 使用事务处理用户创建和社交绑定
	var result *member2.AppAuthLoginResp
	err = s.repo.Transaction(func(tx *query.Query) error {
		var user *member.MemberUser
		if bindUserId != 0 {
			// Case 1: Already bound
			u := tx.MemberUser
			var err error
			user, err = u.WithContext(ctx).Where(u.ID.Eq(bindUserId)).First()
			if err != nil {
				return err
			}
		} else {
			// Case 2: Not bound -> Auto Register + Bind
			// Create User
			user = &member.MemberUser{
				Nickname:         socialUser.Nickname,
				Avatar:           socialUser.Avatar,
				RegisterIP:       ip,
				RegisterTerminal: terminal,
				Status:           0, // Enabled
				Point:            0,
				Experience:       0,
			}
			if err := tx.MemberUser.WithContext(ctx).Create(user); err != nil {
				return err
			}

			// Bind
			bindReq := &system2.SocialUserBindReq{
				Type:  int(r.Type),
				Code:  r.Code,
				State: r.State,
			}
			openid, err := s.socialSvc.BindSocialUser(ctx, user.ID, 1, bindReq)
			if err != nil {
				return err
			}
			socialUser.Openid = openid
		}

		if user == nil {
			return member.ErrAuthUserNotFound
		}

		// Create Token（使用 OAuth2TokenService）
		var err error
		result, err = s.createTokenAfterLoginSuccess(ctx, user, consts.LoginLogTypeSocial, socialUser.Openid, ip, userAgent)
		return err
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// SendSmsCode 发送验证码
func (s *MemberAuthService) SendSmsCode(ctx context.Context, r *member2.AppAuthSmsSendReq, createIp string) error {
	return s.smsCodeSvc.SendSmsCode(ctx, r.Mobile, int32(r.Scene), createIp)
}

// ValidateSmsCode 校验验证码
func (s *MemberAuthService) ValidateSmsCode(ctx context.Context, r *member2.AppAuthSmsValidateReq) error {
	return s.smsCodeSvc.ValidateSmsCode(ctx, r.Mobile, int32(r.Scene), r.Code)
}

// RefreshToken 刷新访问令牌
func (s *MemberAuthService) RefreshToken(ctx context.Context, refreshToken, ip, userAgent string) (*member2.AppAuthLoginResp, error) {
	// 1. 验证 refreshToken（从 Redis 获取原令牌信息）
	oldToken, err := s.tokenSvc.GetRefreshToken(ctx, refreshToken, consts.UserTypeMember)
	if err != nil || oldToken == nil {
		return nil, member.ErrAuthUserNotTokenValid
	}

	// 2. 获取用户信息
	ctx = pkgContext.WithTenant(ctx, oldToken.TenantID)
	userRepo := s.repo.MemberUser
	user, err := userRepo.WithContext(ctx).Where(userRepo.ID.Eq(oldToken.UserID)).First()
	if err != nil {
		return nil, member.ErrAuthUserNotTokenValid // 对应 Java 处理，当用户不存在时也返回 Token 无效
	}

	// 3. 校验状态
	if user.Status != 0 {
		return nil, member.ErrAuthLoginUserDisabled
	}

	// 4. 创建新的访问令牌
	tokenDO, err := s.tokenSvc.RefreshAccessToken(ctx, refreshToken, user.ID, consts.UserTypeMember, user.TenantID, map[string]string{"nickname": user.Nickname})
	if err != nil {
		return nil, err
	}
	return &member2.AppAuthLoginResp{UserID: user.ID, AccessToken: tokenDO.AccessToken, RefreshToken: tokenDO.RefreshToken, ExpiresTime: types.ToJsonDateTime(tokenDO.ExpiresTime)}, nil
}

// Logout 退出登录
func (s *MemberAuthService) Logout(ctx context.Context, token, ip, userAgent string) error {
	// 1. 处理 token，移除 Bearer 前缀
	if strings.HasPrefix(strings.ToUpper(token), "BEARER ") {
		token = token[7:]
	}
	if token == "" {
		return nil
	}

	// 2. 使用 OAuth2TokenService 删除访问令牌
	_, err := s.tokenSvc.RemoveAccessToken(ctx, token)
	return err
}

// createTokenAfterLoginSuccess 创建令牌并记录登录成功日志
func (s *MemberAuthService) createTokenAfterLoginSuccess(ctx context.Context, user *member.MemberUser,
	logType int, openid, ip, userAgent string) (*member2.AppAuthLoginResp, error) {
	// 1. 记录登录日志
	s.loginLogSvc.CreateLoginLog(ctx, user.ID, consts.UserTypeMember, user.Mobile, ip, userAgent, logType, consts.LoginResultSuccess)
	// 2. 更新最后登录时间
	_ = s.userSvc.UpdateUserLogin(ctx, user.ID, ip)

	// 3. 创建 Token
	return s.createToken(ctx, user, openid)
}

// createToken 创建访问令牌（使用 OAuth2TokenService，与 Java 对齐）
func (s *MemberAuthService) createToken(ctx context.Context, user *member.MemberUser, openid string) (*member2.AppAuthLoginResp, error) {
	// 构建用户信息
	userInfo := map[string]string{
		"nickname": user.Nickname,
	}

	// 令牌绑定数据库中的会员租户
	tokenDO, err := s.tokenSvc.CreateAccessToken(ctx, user.ID, consts.UserTypeMember, user.TenantID, userInfo)
	if err != nil {
		return nil, errors.ErrUnknown
	}

	return &member2.AppAuthLoginResp{
		UserID:       user.ID,
		AccessToken:  tokenDO.AccessToken,
		RefreshToken: tokenDO.RefreshToken,
		ExpiresTime:  types.ToJsonDateTime(tokenDO.ExpiresTime),
		OpenID:       openid,
	}, nil
}

// GetSocialAuthorizeUrl 获取社交授权链接
func (s *MemberAuthService) GetSocialAuthorizeUrl(ctx context.Context, socialType int, redirectUri string) (string, error) {
	return s.socialSvc.GetAuthorizeUrl(ctx, socialType, 1, redirectUri) // 1=Member
}

// WeixinMiniAppLogin 微信小程序登录
func (s *MemberAuthService) WeixinMiniAppLogin(ctx context.Context, r *member2.AppAuthWeixinMiniAppLoginReq, ip, userAgent string, terminal int32) (*member2.AppAuthLoginResp, error) {
	// 1. 获得社交用户
	mobile, err := s.socialSvc.GetMobile(ctx, 1, 31, r.PhoneCode) // 1=Member, 31=Mini App
	if err != nil {
		return nil, err
	}

	// 2. 获得注册用户
	user, err := s.userSvc.CreateUserIfAbsent(ctx, mobile, ip, terminal) // 使用传入的 terminal
	if err != nil {
		return nil, err
	}

	// 3. 绑定社交用户
	bindReq := &system2.SocialUserBindReq{
		Type:  31,
		Code:  r.LoginCode,
		State: r.State,
	}
	openid, err := s.socialSvc.BindSocialUser(ctx, user.ID, 1, bindReq)
	if err != nil {
		return nil, err
	}

	// 4. 创建 Token
	return s.createTokenAfterLoginSuccess(ctx, user, consts.LoginLogTypeSocial, openid, ip, userAgent)
}

// CreateWeixinMpJsapiSignature 创建微信 MP JSAPI 签名
func (s *MemberAuthService) CreateWeixinMpJsapiSignature(ctx context.Context, url string) (*member2.AppAuthWeixinJsapiSignatureResp, error) {
	signature, err := s.socialSvc.CreateWxMpJsapiSignature(ctx, 1, url) // 1 = Member
	if err != nil {
		return nil, err
	}
	return &member2.AppAuthWeixinJsapiSignatureResp{
		AppID:     signature.AppID,
		NonceStr:  signature.NonceStr,
		Timestamp: signature.Timestamp,
		URL:       signature.URL,
		Signature: signature.Signature,
	}, nil
}
