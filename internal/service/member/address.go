package member

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/internal/pkg/area"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"

	member2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"

	"github.com/samber/lo"
)

type MemberAddressService struct {
	q *query.Query
}

func NewMemberAddressService(q *query.Query) *MemberAddressService {
	return &MemberAddressService{q: q}
}

// CreateAddress 创建收件地址
func (s *MemberAddressService) CreateAddress(ctx context.Context, userId int64, req *member2.AppAddressCreateReq) (int64, error) {
	if req.DefaultStatus == nil {
		return 0, errors.ErrParam
	}

	address := &member.MemberAddress{
		UserID:        userId,
		Name:          req.Name,
		Mobile:        req.Mobile,
		AreaID:        req.AreaID,
		DetailAddress: req.DetailAddress,
		DefaultStatus: model.NewBitBool(*req.DefaultStatus),
	}
	err := s.q.Transaction(func(tx *query.Query) error {
		if *req.DefaultStatus {
			if err := s.updateDefaultStatus(ctx, tx, userId, 0); err != nil {
				return err
			}
		}
		return tx.MemberAddress.WithContext(ctx).Create(address)
	})
	return address.ID, err
}

// UpdateAddress 更新收件地址
func (s *MemberAddressService) UpdateAddress(ctx context.Context, userId int64, req *member2.AppAddressUpdateReq) error {
	if req.DefaultStatus == nil {
		return errors.ErrParam
	}
	// 校验存在
	exists, err := s.exists(ctx, userId, req.ID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.NewBizError(1004003005, "收件地址不存在") // ADDRESS_NOT_EXISTS
	}

	return s.q.Transaction(func(tx *query.Query) error {
		if *req.DefaultStatus {
			if err := s.updateDefaultStatus(ctx, tx, userId, req.ID); err != nil {
				return err
			}
		}
		u := tx.MemberAddress
		result, err := u.WithContext(ctx).Where(u.ID.Eq(req.ID), u.UserID.Eq(userId)).Updates(map[string]interface{}{
			"name": req.Name, "mobile": req.Mobile, "area_id": req.AreaID,
			"detail_address": req.DetailAddress, "default_status": model.NewBitBool(*req.DefaultStatus),
		})
		if err != nil {
			return err
		}
		if result.RowsAffected != 1 {
			return errors.NewBizError(1004003005, "收件地址不存在")
		}
		return nil
	})
}

// DeleteAddress 删除收件地址
func (s *MemberAddressService) DeleteAddress(ctx context.Context, userId int64, id int64) error {
	exists, err := s.exists(ctx, userId, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.NewBizError(1004003005, "收件地址不存在")
	}

	u := s.q.MemberAddress
	_, err = u.WithContext(ctx).Where(u.ID.Eq(id)).Delete()
	return err
}

// GetAddress 获得收件地址
func (s *MemberAddressService) GetAddress(ctx context.Context, userId int64, id int64) (*member2.AppAddressResp, error) {
	u := s.q.MemberAddress
	address, err := u.WithContext(ctx).Where(u.UserID.Eq(userId), u.ID.Eq(id)).First()
	if err != nil {
		return nil, nil // Return nil if not found or error? Java throws exception if validated. Here simple get returns nil or error.
	}
	return s.convertResp(address), nil
}

// GetDefaultUserAddress 获得默认收件地址
func (s *MemberAddressService) GetDefaultUserAddress(ctx context.Context, userId int64) (*member2.AppAddressResp, error) {
	u := s.q.MemberAddress
	address, err := u.WithContext(ctx).Where(u.UserID.Eq(userId), u.DefaultStatus.Eq(model.NewBitBool(true))).First()
	if err != nil {
		return nil, nil
	}
	return s.convertResp(address), nil
}

// GetDefaultAddress 获得默认收件地址（Alias for GetDefaultUserAddress）
func (s *MemberAddressService) GetDefaultAddress(ctx context.Context, userId int64) (*member2.AppAddressResp, error) {
	return s.GetDefaultUserAddress(ctx, userId)
}

// GetAddressList 获得收件地址列表
func (s *MemberAddressService) GetAddressList(ctx context.Context, userId int64) ([]*member2.AppAddressResp, error) {
	u := s.q.MemberAddress
	list, err := u.WithContext(ctx).Where(u.UserID.Eq(userId)).Order(u.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return lo.Map(list, func(item *member.MemberAddress, _ int) *member2.AppAddressResp {
		return s.convertResp(item)
	}), nil
}

func (s *MemberAddressService) exists(ctx context.Context, userId int64, id int64) (bool, error) {
	u := s.q.MemberAddress
	count, err := u.WithContext(ctx).Where(u.UserID.Eq(userId), u.ID.Eq(id)).Count()
	return count > 0, err
}

func (s *MemberAddressService) updateDefaultStatus(ctx context.Context, tx *query.Query, userId int64, excludeId int64) error {
	u := tx.MemberAddress
	// Set all others to false
	q := u.WithContext(ctx).Where(u.UserID.Eq(userId), u.DefaultStatus.Eq(model.NewBitBool(true)))
	if excludeId > 0 {
		q = q.Where(u.ID.Neq(excludeId))
	}
	_, err := q.Update(u.DefaultStatus, model.NewBitBool(false))
	return err
}

func (s *MemberAddressService) convertResp(item *member.MemberAddress) *member2.AppAddressResp {
	return &member2.AppAddressResp{
		AreaName:      area.Format(int(item.AreaID)),
		ID:            item.ID,
		Name:          item.Name,
		Mobile:        item.Mobile,
		AreaID:        item.AreaID,
		DetailAddress: item.DetailAddress,
		DefaultStatus: bool(item.DefaultStatus),
		CreateTime:    types.ToJsonDateTime(item.CreateTime),
	}
}
