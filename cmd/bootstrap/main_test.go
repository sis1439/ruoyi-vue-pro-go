package main

import (
	"github.com/stretchr/testify/require"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	"golang.org/x/crypto/bcrypt"
	"os"
	"testing"
)

func TestHostnameValidation(t *testing.T) {
	for _, host := range []string{"localhost", "mall.example.com", "127.0.0.1"} {
		require.True(t, validHostname(host), host)
	}
	for _, host := range []string{"", "https://mall.example.com", "mall.example.com:48080", "mall.example.com/path", "a,b", "-bad.example", "bad..example"} {
		require.False(t, validHostname(host), host)
	}
}
func TestPostgresBootstrapNonDefaultCredentialsAndHost(t *testing.T) {
	db := testutil.PostgreSQL(t)
	var name string
	require.NoError(t, db.Raw("SELECT current_schema()").Scan(&name).Error)
	t.Setenv("RUOYI_DATABASE_DSN", os.Getenv("TEST_POSTGRES_DSN")+" search_path="+name)
	t.Setenv("RUOYI_BOOTSTRAP_PLATFORM_ADMIN", "")
	t.Setenv("RUOYI_BOOTSTRAP_USERNAME", "integration-admin")
	t.Setenv("RUOYI_BOOTSTRAP_PASSWORD", "Synthetic-Only-Test-Password-439!")
	t.Setenv("RUOYI_BOOTSTRAP_WEBSITE", "mall.test")
	require.NoError(t, run())
	var user model.SystemUser
	require.NoError(t, db.Where("username=?", "integration-admin").First(&user).Error)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(os.Getenv("RUOYI_BOOTSTRAP_PASSWORD"))))
	var tenant model.SystemTenant
	require.NoError(t, db.First(&tenant, 1).Error)
	require.Equal(t, model.StringListFromCSV{"mall.test"}, tenant.Websites)
	require.Equal(t, user.ID, tenant.ContactUserID)
	var count int64
	require.NoError(t, db.Model(&model.SystemUserRole{}).Where("tenant_id=1 AND user_id=? AND role_id=1", user.ID).Count(&count).Error)
	require.Equal(t, int64(1), count)
	require.ErrorContains(t, run(), "already exists")
	var platformCount int64
	require.NoError(t, db.Model(&model.SystemRole{}).Where("code=?", "platform_admin").Count(&platformCount).Error)
	require.Zero(t, platformCount, "normal bootstrap never grants platform access")
	t.Setenv("RUOYI_BOOTSTRAP_USERNAME", "platform-read-fixture")
	t.Setenv("RUOYI_BOOTSTRAP_PLATFORM_ADMIN", "true")
	require.NoError(t, run())
	var platformUser model.SystemUser
	require.NoError(t, db.Where("username=?", "platform-read-fixture").First(&platformUser).Error)
	var role model.SystemRole
	require.NoError(t, db.Where("code=? AND tenant_id=1", "platform_admin").First(&role).Error)
	require.Equal(t, int32(1), role.Type)
	require.Zero(t, role.Status)
	require.NoError(t, db.Model(&model.SystemUserRole{}).Where("tenant_id=1 AND user_id=? AND role_id=?", platformUser.ID, role.ID).Count(&platformCount).Error)
	require.Equal(t, int64(1), platformCount)
	t.Setenv("RUOYI_BOOTSTRAP_PLATFORM_ADMIN", "yes")
	require.ErrorContains(t, run(), "must be true or false")

}
