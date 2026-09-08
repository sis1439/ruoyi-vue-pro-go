package member

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	model "github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	tenant "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"testing"
)

func TestJavaSmsSceneDestinationsPostgres(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	q := query.Use(db)
	ctx := tenant.WithTenant(context.Background(), 1)
	user := &model.MemberUser{Mobile: "15500000001"}
	other := &model.MemberUser{Mobile: "15500000002"}
	require.NoError(t, db.WithContext(ctx).Create(user).Error)
	require.NoError(t, db.WithContext(ctx).Create(other).Error)
	svc := &MemberAuthService{repo: q, userSvc: NewMemberUserService(q, nil, nil, nil)}
	login := tenant.WithLoginUser(ctx, &tenant.LoginUser{UserID: user.ID, UserType: consts.UserTypeMember, TenantID: 1})
	for _, mobile := range []string{"", other.Mobile} {
		got, err := svc.smsMobile(login, mobile, 3)
		require.NoError(t, err)
		require.Equal(t, user.Mobile, got)
	}
	_, err := svc.smsMobile(ctx, "", 3)
	require.Error(t, err)
	_, err = svc.smsMobile(login, other.Mobile, 2)
	require.Error(t, err)
	_, err = svc.smsMobile(login, user.Mobile, 2)
	require.NoError(t, err)
	_, err = svc.smsMobile(ctx, "15500000003", 4)
	require.Error(t, err)
	_, err = svc.smsMobile(ctx, user.Mobile, 4)
	require.NoError(t, err)
	_, err = svc.smsMobile(tenant.WithTenant(context.Background(), 2), user.Mobile, 4)
	require.Error(t, err)
	_, err = svc.smsMobile(ctx, "", 1)
	require.Error(t, err)
	_, err = svc.smsMobile(ctx, user.Mobile, 21)
	require.Error(t, err)
}
