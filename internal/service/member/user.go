package member

import (
	"context"
	"errors"
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"
	"math"
	"time"

	member2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	query "github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
)

type MemberUserService struct {
	q             *query.Query
	smsCodeSvc    *system.SmsCodeService
	levelSvc      *MemberLevelService
	socialUserSvc *system.SocialUserService
}

func NewMemberUserService(q *query.Query, smsCodeSvc *system.SmsCodeService, levelSvc *MemberLevelService, socialUserSvc *system.SocialUserService) *MemberUserService {
	return &MemberUserService{
		q:             q,
		smsCodeSvc:    smsCodeSvc,
		levelSvc:      levelSvc,
		socialUserSvc: socialUserSvc,
	}
}

// GetUserInfo 获取用户个人信息
func (s *MemberUserService) GetUserInfo(ctx context.Context, id int64) (*member2.AppMemberUserInfoResp, error) {
	u := s.q.MemberUser
	user, err := u.WithContext(ctx).Where(u.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	var levelResp *member2.AppMemberUserLevelResp
	if user.LevelID > 0 {
		level, _ := s.levelSvc.GetLevel(ctx, user.LevelID)
		if level != nil {
			levelResp = &member2.AppMemberUserLevelResp{
				ID:    level.ID,
				Name:  level.Name,
				Level: level.Level,
				Icon:  level.Icon,
			}
		}
	}

	// 获取分销资格（对齐 Java: BrokerageUserService.getUserBrokerageEnabled）
	// 直接查询 trade_brokerage_user 表以避免循环依赖
	brokerageEnabled := false
	if brokerageUser, err := s.q.BrokerageUser.WithContext(ctx).Where(s.q.BrokerageUser.ID.Eq(id)).First(); err == nil && brokerageUser != nil {
		brokerageEnabled = bool(brokerageUser.BrokerageEnabled)
	}

	return &member2.AppMemberUserInfoResp{
		ID:               user.ID,
		Nickname:         user.Nickname,
		Avatar:           user.Avatar,
		Mobile:           user.Mobile,
		Sex:              user.Sex,
		Point:            user.Point,
		Experience:       user.Experience,
		Level:            levelResp,
		BrokerageEnabled: brokerageEnabled,
	}, nil
}

// CreateUser 创建会员用户（对齐 Java: MemberUserServiceImpl.createUser）
func (s *MemberUserService) CreateUser(ctx context.Context, nickname, avatar, regIp string, terminal int32) (*member.MemberUser, error) {
	// 生成随机密码（对齐 Java: IdUtil.fastSimpleUUID()）
	password := utils.GenerateRandomString(32)
	hashedPwd, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &member.MemberUser{
		Nickname:         nickname,
		Avatar:           avatar,
		Password:         hashedPwd,
		RegisterIP:       regIp,
		RegisterTerminal: terminal,
		Status:           0, // CommonStatusEnum.ENABLE = 0
		Point:            0,
		Experience:       0,
	}

	// 昵称为空时，随机生成（对齐 Java: "用户" + RandomUtil.randomNumbers(6)）
	if user.Nickname == "" {
		user.Nickname = "用户" + utils.GenerateRandomString(6)
	}

	if err := s.q.MemberUser.WithContext(ctx).Create(user); err != nil {
		return nil, err
	}

	// TODO: 发送 MQ 消息：用户创建（对齐 Java: memberUserProducer.sendUserCreateMessage）
	// 需要在事务提交后发送，避免事务回滚导致消息已发送

	return user, nil
}

// CreateUserIfAbsent 如果用户不存在则创建
func (s *MemberUserService) CreateUserIfAbsent(ctx context.Context, mobile, regIp string, terminal int32) (*member.MemberUser, error) {
	u := s.q.MemberUser
	user, err := u.WithContext(ctx).Where(u.Mobile.Eq(mobile)).First()
	if err == nil {
		return user, nil
	}
	// Need to check if error is RecordNotFound
	return s.CreateUser(ctx, "手机用户"+mobile[len(mobile)-4:], "", regIp, terminal)
}

// GetUser 获得用户信息 (Internal)
func (s *MemberUserService) GetUser(ctx context.Context, id int64) (*member.MemberUser, error) {
	return s.q.MemberUser.WithContext(ctx).Where(s.q.MemberUser.ID.Eq(id)).First()
}

// UpdateUser 修改用户基本信息
func (s *MemberUserService) UpdateUser(ctx context.Context, id int64, req *member2.AppMemberUserUpdateReq) error {
	// 校验用户存在
	u := s.q.MemberUser
	count, err := u.WithContext(ctx).Where(u.ID.Eq(id)).Count()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("user not found")
	}

	// 更新字段
	_, err = u.WithContext(ctx).Where(u.ID.Eq(id)).
		Select(u.Nickname, u.Avatar, u.Sex).
		Updates(&member.MemberUser{
			Nickname: req.Nickname,
			Avatar:   req.Avatar,
			Sex:      int32(req.Sex),
		})
	return err
}

// UpdateUserMobile 修改用户手机
func (s *MemberUserService) UpdateUserMobile(ctx context.Context, id int64, req *member2.AppMemberUserUpdateMobileReq) error {
	// 使用定义的场景值（对应 Java 中的 MEMBER_UPDATE_MOBILE）
	scene := system.SmsSceneMemberUpdateMob.Scene

	// 1. 校验验证码
	if err := s.smsCodeSvc.UseSmsCode(ctx, req.Mobile, int32(scene), req.Code, ""); err != nil {
		return err
	}

	// 2. Check if mobile already used by another user
	u := s.q.MemberUser
	count, err := u.WithContext(ctx).Where(u.Mobile.Eq(req.Mobile), u.ID.Neq(id)).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("mobile already used")
	}

	_, err = u.WithContext(ctx).Where(u.ID.Eq(id)).Update(u.Mobile, req.Mobile)
	return err
}

// ResetUserPassword 重置用户密码 (忘记密码)
// 对齐 Java: MemberUserServiceImpl.resetUserPassword
func (s *MemberUserService) ResetUserPassword(ctx context.Context, req *member2.AppMemberUserResetPasswordReq) error {
	// 1. 校验验证码 (场景: 重置密码 = SmsSceneEnum.MEMBER_RESET_PASSWORD)
	if err := s.smsCodeSvc.UseSmsCode(ctx, req.Mobile, system.SmsSceneMemberResetPwd.Scene, req.Code, ""); err != nil {
		return err
	}

	// 2. 查询用户
	u := s.q.MemberUser
	user, err := u.WithContext(ctx).Where(u.Mobile.Eq(req.Mobile)).First()
	if err != nil {
		return errors.New("mobile not registered")
	}

	// 3. Hash Password
	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		return err
	}

	// 4. Update Password
	_, err = u.WithContext(ctx).Where(u.ID.Eq(user.ID)).Update(u.Password, hashedPwd)
	return err
}

// UpdateUserPassword 修改用户密码
func (s *MemberUserService) UpdateUserPassword(ctx context.Context, id int64, req *member2.AppMemberUserUpdatePasswordReq) error {
	// 1. Check User
	u := s.q.MemberUser
	user, err := u.WithContext(ctx).Where(u.ID.Eq(id)).First()
	if err != nil {
		return errors.New("user not found")
	}

	// 2. Validate Old Password
	if !utils.CheckPasswordHash(req.OldPassword, user.Password) {
		return errors.New("invalid old password")
	}

	// 3. Hash New Password
	hashedPwd, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	// 4. Update
	_, err = u.WithContext(ctx).Where(u.ID.Eq(id)).Update(u.Password, hashedPwd)
	return err
}

// GetUserCountByTagId 获得标签下的用户数量
// 对齐 Java: MemberUserMapper.selectCountByTagId (使用 FIND_IN_SET)
func (s *MemberUserService) GetUserCountByTagId(ctx context.Context, tagId int64) (int64, error) {
	var count int64
	err := s.q.MemberUser.WithContext(ctx).UnderlyingDB().
		Where("? = ANY(string_to_array(tag_ids, ','))", fmt.Sprint(tagId)).
		Count(&count).Error
	return count, err
}

// GetUserCountByLevelId 获得等级下的用户数量
func (s *MemberUserService) GetUserCountByLevelId(ctx context.Context, levelId int64) (int64, error) {
	u := s.q.MemberUser
	return u.WithContext(ctx).Where(u.LevelID.Eq(levelId)).Count()
}

// UpdateUserPoint 更新用户积分
// 对齐 Java: MemberUserMapper.updatePointIncr / updatePointDecr
// point: 增加积分 (正数) 或 消费积分 (负数)
func (s *MemberUserService) UpdateUserPoint(ctx context.Context, id int64, point int) bool {
	if point == 0 {
		return true
	}
	if point < -math.MaxInt32 || point > math.MaxInt32 {
		return false
	}
	u := repo.QueryFromContext(ctx, s.q).MemberUser
	update := u.WithContext(ctx).Where(u.ID.Eq(id))
	if point < 0 {
		update = update.Where(u.Point.Gte(int32(-point)))
	} else {
		update = update.Where(u.Point.Lte(int32(math.MaxInt32 - point)))
	}
	info, err := update.Update(u.Point, u.Point.Add(int32(point)))
	if err != nil {
		return false
	}
	return info.RowsAffected > 0
}

// UpdateUserLogin 更新用户登录信息
func (s *MemberUserService) UpdateUserLogin(ctx context.Context, id int64, ip string) error {
	u := s.q.MemberUser
	now := time.Now()
	_, err := u.WithContext(ctx).Where(u.ID.Eq(id)).Updates(&member.MemberUser{
		LoginIP:   ip,
		LoginDate: &now,
	})
	return err
}

// UpdateUserMobileByWeixin 微信小程序更新手机号
func (s *MemberUserService) UpdateUserMobileByWeixin(ctx context.Context, userId int64, code string) error {
	// 1. 获得手机号
	mobile, err := s.socialUserSvc.GetMobile(ctx, 1, 31, code) // 1=Member, 31=WECHAT_MINI_APP
	if err != nil {
		return err
	}
	if mobile == "" {
		return errors.New("获取手机号失败")
	}

	// 2. 更新手机号
	return s.updateUserMobile(ctx, userId, mobile)
}

func (s *MemberUserService) updateUserMobile(ctx context.Context, id int64, mobile string) error {
	u := s.q.MemberUser
	_, err := u.WithContext(ctx).Where(u.ID.Eq(id)).Update(u.Mobile, mobile)
	return err
}

// UpdateUserLevel 更新用户等级信息
func (s *MemberUserService) UpdateUserLevel(ctx context.Context, id int64, levelId int64, experience int) error {
	u := s.q.MemberUser
	_, err := u.WithContext(ctx).Where(u.ID.Eq(id)).Updates(&member.MemberUser{
		LevelID:    levelId,
		Experience: int32(experience),
	})
	return err
}

func (s *MemberUserService) GetUserListByNickname(ctx context.Context, nickname string) ([]*member.MemberUser, error) {
	return s.q.MemberUser.WithContext(ctx).Where(s.q.MemberUser.Nickname.Like("%" + nickname + "%")).Find()
}

// GetUserMap 获得用户 Map (Entity)
func (s *MemberUserService) GetUserMap(ctx context.Context, ids []int64) (map[int64]*member.MemberUser, error) {
	users, err := s.q.MemberUser.WithContext(ctx).Where(s.q.MemberUser.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	userMap := make(map[int64]*member.MemberUser)
	for _, u := range users {
		userMap[u.ID] = u
	}
	return userMap, nil
}

// GetUserRespMap 获得用户 Map (Response VO)
func (s *MemberUserService) GetUserRespMap(ctx context.Context, ids []int64) (map[int64]*member2.MemberUserResp, error) {
	if len(ids) == 0 {
		return make(map[int64]*member2.MemberUserResp), nil
	}
	u := s.q.MemberUser
	list, err := u.WithContext(ctx).Where(u.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]*member2.MemberUserResp, len(list))
	for _, user := range list {
		// Fetch Level info if needed or just basic info
		// For now, basic info is enough as per TradeOrder requirements
		userMap[user.ID] = &member2.MemberUserResp{
			ID:         user.ID,
			Mobile:     user.Mobile,
			Status:     user.Status,
			Nickname:   user.Nickname,
			Avatar:     user.Avatar,
			Sex:        user.Sex,
			AreaID:     int64(user.AreaID),
			Birthday:   user.Birthday,
			Mark:       user.Mark,
			LevelID:    user.LevelID,
			GroupID:    user.GroupID,
			CreateTime: user.CreateTime,
		}
	}
	return userMap, nil
}

// ========== Admin API Service Methods ==========

// AdminUpdateUser Admin 更新会员用户
func (s *MemberUserService) AdminUpdateUser(ctx context.Context, r *member2.MemberUserUpdateReq) error {
	u := s.q.MemberUser
	count, err := u.WithContext(ctx).Where(u.ID.Eq(r.ID)).Count()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("user not found")
	}

	// Convert []int64 to []int for model.IntListFromCSV
	tagIds := make([]int, len(r.TagIDs))
	for i, v := range r.TagIDs {
		tagIds[i] = int(v)
	}

	// 构建更新对象，支持所有字段
	updateUser := &member.MemberUser{
		Mobile:   r.Mobile,
		Status:   int32(r.Status),
		Nickname: r.Nickname,
		Avatar:   r.Avatar,
		Name:     r.Name,
		Sex:      int32(r.Sex),
		AreaID:   int32(r.AreaID),
		Birthday: r.Birthday,
		Mark:     r.Mark,
		TagIds:   tagIds,
	}
	if r.LevelID != nil {
		updateUser.LevelID = *r.LevelID
	}
	if r.GroupID != nil {
		updateUser.GroupID = *r.GroupID
	}

	_, err = u.WithContext(ctx).Where(u.ID.Eq(r.ID)).
		Select(u.Mobile, u.Status, u.Nickname, u.Avatar, u.Name, u.Sex, u.AreaID, u.Birthday, u.Mark, u.TagIds, u.LevelID, u.GroupID).
		Updates(updateUser)
	return err
}

// GetUserPage Admin 获得会员用户分页
func (s *MemberUserService) GetUserPage(ctx context.Context, r *member2.MemberUserPageReq) (*pagination.PageResult[*member.MemberUser], error) {
	u := s.q.MemberUser
	q := u.WithContext(ctx)

	// 动态条件
	if r.Mobile != "" {
		q = q.Where(u.Mobile.Like("%" + r.Mobile + "%"))
	}
	if r.Nickname != "" {
		q = q.Where(u.Nickname.Like("%" + r.Nickname + "%"))
	}
	if r.LevelID != nil {
		q = q.Where(u.LevelID.Eq(*r.LevelID))
	}
	if r.GroupID != nil {
		q = q.Where(u.GroupID.Eq(*r.GroupID))
	}
	if len(r.LoginDate) == 2 {
		if r.LoginDate[0] != nil {
			q = q.Where(u.LoginDate.Gte(*r.LoginDate[0]))
		}
		if r.LoginDate[1] != nil {
			q = q.Where(u.LoginDate.Lte(*r.LoginDate[1]))
		}
	}
	if len(r.CreateTime) == 2 {
		if r.CreateTime[0] != nil {
			q = q.Where(u.CreateTime.Gte(*r.CreateTime[0]))
		}
		if r.CreateTime[1] != nil {
			q = q.Where(u.CreateTime.Lte(*r.CreateTime[1]))
		}
	}

	// 统计总数
	total, err := q.Count()
	if err != nil {
		return nil, err
	}

	// 分页查询
	offset := (r.PageNo - 1) * r.PageSize
	list, err := q.Order(u.ID.Desc()).Offset(offset).Limit(r.PageSize).Find()
	if err != nil {
		return nil, err
	}

	return &pagination.PageResult[*member.MemberUser]{
		Total: total,
		List:  list,
	}, nil
}
