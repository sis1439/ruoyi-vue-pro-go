package infra

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"testing"
)

func TestPostgresFileMasterBitBoolUpdates(t *testing.T) {
	db := testutil.PostgreSQL(t)
	first := model.InfraFileConfig{Name: "first", Storage: 10, Master: true, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	second := model.InfraFileConfig{Name: "second", Storage: 10, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	other := model.InfraFileConfig{Name: "other tenant", Storage: 10, Master: true, TenantBaseDO: model.TenantBaseDO{TenantID: 2}}
	for _, item := range []*model.InfraFileConfig{&first, &second, &other} {
		require.NoError(t, db.Create(item).Error)
	}
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	svc := NewFileConfigService(query.Use(db))
	ctx := pkgContext.WithTenant(context.Background(), 1)
	require.NoError(t, svc.UpdateFileConfigMaster(ctx, second.ID))
	got, err := svc.GetMasterFileConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, second.ID, got.ID)
	var old model.InfraFileConfig
	require.NoError(t, db.WithContext(ctx).First(&old, first.ID).Error)
	require.False(t, bool(old.Master))
	var untouched model.InfraFileConfig
	require.NoError(t, db.WithContext(pkgContext.WithTenant(context.Background(), 2)).First(&untouched, other.ID).Error)
	require.True(t, bool(untouched.Master))
}
