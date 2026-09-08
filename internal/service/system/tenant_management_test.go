package system

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	contract "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pc "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

func TestReviewTenantAdminCannotReadOtherTenantManagement(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Create(&model.SystemTenant{ID: 2, Name: "B", ContactName: "private-B-contact", ContactMobile: "13900000002", ExpireDate: time.Now().Add(time.Hour)}).Error)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	q := query.Use(db)
	ctx := pc.WithTenant(context.Background(), 1)
	u := &model.SystemUser{Username: "review-admin", TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.WithContext(ctx).Create(u).Error)
	require.NoError(t, db.WithContext(ctx).Create(&model.SystemUserRole{UserID: u.ID, RoleID: 1, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}).Error)
	role := NewRoleService(q)
	perm := NewPermissionService(q, role)
	allowed, err := perm.HasPermission(ctx, u.ID, "system:tenant:query")
	require.NoError(t, err)
	require.True(t, allowed, "seeded ordinary tenant_admin passes current route permission")
	other, err := NewTenantService(q, role, perm).GetTenant(ctx, 2)
	t.Logf("ordinary tenant_admin route permission=%v; tenant B detail=%+v err=%v", allowed, other, err)
	require.Error(t, err, "ordinary tenant admin must not access another tenant management detail without platform authorization/audit")
}

func TestTenantManagementOwnScopeAndPublicDiscovery(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Create(&model.SystemTenant{ID: 2, Name: "private B", ContactName: "B contact", ContactMobile: "13900000002", Websites: model.StringListFromCSV{"b.example.test"}, ExpireDate: time.Now().Add(time.Hour)}).Error)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	q := query.Use(db)
	roles := NewRoleService(q)
	svc := NewTenantService(q, roles, NewPermissionService(q, roles))
	for _, tenantID := range []int64{1, 2} {
		ctx := pc.WithTenant(context.Background(), tenantID)
		own, err := svc.GetTenant(ctx, tenantID)
		require.NoError(t, err)
		require.Equal(t, tenantID, own.ID)
		page, err := svc.GetTenantPage(ctx, &contract.TenantPageReq{PageParam: pagination.PageParam{PageNo: 1, PageSize: 20}})
		require.NoError(t, err)
		require.EqualValues(t, 1, page.Total)
		require.Len(t, page.List, 1)
		require.Equal(t, tenantID, page.List[0].ID)
		rows, err := svc.GetTenantList(ctx, &contract.TenantExportReq{})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, tenantID, rows[0].ID)
	}
	missing := context.Background()
	_, err := svc.GetTenant(missing, 1)
	require.Error(t, err)
	_, err = svc.GetTenantPage(missing, &contract.TenantPageReq{})
	require.Error(t, err)
	_, err = svc.GetTenantList(missing, &contract.TenantExportReq{})
	require.Error(t, err)
	// Filtering by another tenant's name must not widen either page or export scope.
	ctxA := pc.WithTenant(context.Background(), 1)
	page, err := svc.GetTenantPage(ctxA, &contract.TenantPageReq{Name: "private B", PageParam: pagination.PageParam{PageNo: 1, PageSize: 20}})
	require.NoError(t, err)
	require.Zero(t, page.Total)
	rows, err := svc.GetTenantList(ctxA, &contract.TenantExportReq{Name: "private B"})
	require.NoError(t, err)
	require.Empty(t, rows)
	public, err := svc.GetTenantByWebsite(context.Background(), "b.example.test")
	require.NoError(t, err)
	encoded, err := json.Marshal(public)
	require.NoError(t, err)
	var data map[string]any
	require.NoError(t, json.Unmarshal(encoded, &data))
	require.Len(t, data, 2)
	require.Contains(t, data, "id")
	require.Contains(t, data, "name")
}
