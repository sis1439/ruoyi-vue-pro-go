package permission

import (
	"context"
	modelDO "github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	systemSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"testing"
)

func TestPolicyStartupTenantIsolation(t *testing.T) {
	db := testutil.PostgreSQL(t)
	statements := []string{
		`INSERT INTO system_users(id,username,password,nickname,status,tenant_id) VALUES(810001,'security-a','unused','A',0,1),(810002,'security-b','unused','B',0,2)`,
		`INSERT INTO system_role(id,name,code,status,tenant_id) VALUES(810001,'Security A','security_a',0,1),(810002,'Security B','security_b',0,2)`,
		`INSERT INTO system_menu(id,name,permission,type,sort,status) VALUES(810001,'Security permission','security:test',3,0,0)`,
		`INSERT INTO system_role_menu(id,role_id,menu_id,tenant_id) VALUES(810001,810001,810001,1),(810002,810002,810001,2)`,
		`INSERT INTO system_user_role(id,user_id,role_id,tenant_id) VALUES(810001,810001,810001,1),(810002,810002,810002,2),(810003,810001,810002,1)`,
	}
	for _, sql := range statements {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	enforcer, err := InitEnforcer(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		sub  string
		want bool
	}{{"tenant:1:user:810001", true}, {"tenant:2:user:810002", true}, {"tenant:2:user:810001", false}, {"user:810001", false}} {
		got, err := enforcer.Enforce(tc.sub, "security:test", "access")
		if err != nil || got != tc.want {
			t.Fatalf("%s got=%v err=%v", tc.sub, got, err)
		}
	}
	if ok, err := enforcer.HasGroupingPolicy("tenant:1:user:810001", "tenant:1:role:810002"); err != nil || ok {
		t.Fatal("cross tenant corrupt role assignment loaded")
	}
	q := query.Use(db)
	permissions := systemSvc.NewPermissionService(q, systemSvc.NewRoleService(q))
	a := pkgContext.WithTenant(context.Background(), 1)
	if allowed, err := permissions.HasPermission(a, 810001, "security:test"); err != nil || !allowed {
		t.Fatalf("live permission missing: %v", err)
	}
	if err := db.WithContext(a).Model(&modelDO.SystemRole{}).Where("id = ?", 810001).Update("data_scope", 2).Error; err != nil {
		t.Fatal(err)
	}
	if allowed, err := permissions.HasPermission(a, 810001, "security:test"); err != nil || allowed {
		t.Fatalf("unsupported data scope retained permission: %v", err)
	}
	if err := db.WithContext(a).Model(&modelDO.SystemRole{}).Where("id = ?", 810001).Update("data_scope", 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(a).Model(&modelDO.SystemRole{}).Where("id = ?", 810001).Update("status", 1).Error; err != nil {
		t.Fatal(err)
	}
	if allowed, err := permissions.HasPermission(a, 810001, "security:test"); err != nil || allowed {
		t.Fatalf("disabled role retains live permission: %v", err)
	}
	if err := db.WithContext(database.PolicyReadContext(context.Background())).Model(&modelDO.SystemMenu{}).Where("id = ?", 810001).Update("name", "unauthorized platform write").Error; err == nil {
		t.Fatal("read-only platform scope allowed write")
	}

}
