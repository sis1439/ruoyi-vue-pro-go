package system

import (
	"context"
	"encoding/json"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"log"
	"time"
)

type TenantInspection struct {
	TenantID     int64 `json:"tenantId"`
	ProductCount int64 `json:"productCount"`
	OrderCount   int64 `json:"orderCount"`
}

// InspectTenant exposes only two aggregate counts. No caller receives the target context or a generic bypass.
func (s *AuthService) InspectTenant(ctx context.Context, target int64) (result *TenantInspection, err error) {
	actor := pkgContext.GetLoginUserFromContext(ctx)
	event := map[string]any{"operation": "platform.tenant.inspect", "targetTenantId": target, "outcome": "denied"}
	if actor != nil {
		event["actorUserId"] = actor.UserID
		event["sourceTenantId"] = actor.TenantID
	}
	defer func() {
		if err == nil {
			event["outcome"] = "success"
		}
		encoded, _ := json.Marshal(event)
		log.Printf("security_audit %s", encoded)
	}()
	if actor == nil || actor.UserID <= 0 || actor.TenantID <= 0 || actor.UserType != consts.UserTypeAdmin || target <= 0 {
		return nil, errors.NewBizError(403, "平台权限不足")
	}
	if err = s.permSvc.ValidateAdminScope(ctx, actor.UserID); err != nil {
		return nil, err
	}
	u := s.repo.SystemUser
	if _, err = u.WithContext(ctx).Where(u.ID.Eq(actor.UserID), u.TenantID.Eq(actor.TenantID), u.Status.Eq(0)).First(); err != nil {
		return nil, errors.NewBizError(403, "平台权限不足")
	}
	ids, err := s.permSvc.GetUserRoleIdListByUserId(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errors.NewBizError(403, "平台权限不足")
	}
	role := s.repo.SystemRole
	count, err := role.WithContext(ctx).Where(role.ID.In(ids...), role.TenantID.Eq(actor.TenantID), role.Code.Eq(PlatformAdminRole), role.Type.Eq(consts.RoleTypeSystem), role.Status.Eq(0), role.DataScope.Eq(consts.DataScopeAll)).Count()
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, errors.NewBizError(403, "平台权限不足")
	}
	tenants := s.repo.SystemTenant
	source, err := tenants.WithContext(ctx).Where(tenants.ID.Eq(actor.TenantID), tenants.Status.Eq(0)).First()
	if err != nil || (!source.ExpireDate.IsZero() && !source.ExpireDate.After(time.Now())) {
		return nil, errors.NewBizError(403, "平台来源租户不可用")
	}
	if _, err = tenants.WithContext(ctx).Where(tenants.ID.Eq(target)).First(); err != nil {
		return nil, err
	}
	// Fresh context prevents the actor/Gin identity from overriding the explicit target tenant.
	scoped, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := context.AfterFunc(ctx, cancel)
	defer stop()
	scoped = pkgContext.WithTenant(scoped, target)
	result = &TenantInspection{TenantID: target}
	result.ProductCount, err = s.repo.ProductSpu.WithContext(scoped).Count()
	if err != nil {
		return nil, err
	}
	result.OrderCount, err = s.repo.TradeOrder.WithContext(scoped).Count()
	if err != nil {
		return nil, err
	}
	return result, nil
}
