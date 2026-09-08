package system

import (
	"bytes"
	"context"
	"fmt"
	contract "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	productModel "github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"log"
	"strings"
	"testing"
)

func TestPlatformInspectionAndReservedRoleGuards(t *testing.T) {
	db := testutil.PostgreSQL(t)
	for _, sql := range []string{
		`INSERT INTO system_tenant(id,name,status,expire_time,package_id) VALUES(860001,'Security A',0,'2099-01-01',0),(860002,'Security B',0,'2099-01-01',1)`,
		`INSERT INTO system_users(id,username,password,nickname,status,tenant_id) VALUES(860001,'platform-security','unchanged','Platform',0,860001),(860002,'ordinary-security','unchanged','Normal',0,860001)`,
		`INSERT INTO system_role(id,name,code,status,type,tenant_id) VALUES(860001,'Platform','platform_admin',0,1,860001),(860002,'Normal','normal_security',0,2,860001)`,
		`INSERT INTO system_user_role(id,user_id,role_id,tenant_id) VALUES(860001,860001,860001,860001),(860002,860002,860002,860001)`,
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i, tenant := range []int64{860001, 860002, 860002} {
		if err := db.Create(&productModel.ProductSpu{ID: int64(860001 + i), Name: "inspection fixture", CategoryID: 1, TenantBaseDO: model.TenantBaseDO{TenantID: tenant}}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i, tenant := range []int64{860001, 860002} {
		if err := db.Create(&tradeModel.TradeOrder{ID: int64(860001 + i), No: fmt.Sprintf("inspection-%d", i), UserID: tenant, TenantBaseDO: model.TenantBaseDO{TenantID: tenant}}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	q := query.Use(db)
	roles := NewRoleService(q)
	permissions := NewPermissionService(q, roles)
	users := NewUserService(q, nil)
	svc := &AuthService{repo: q, permSvc: permissions}
	actor := pkgContext.WithLoginUser(context.Background(), &pkgContext.LoginUser{UserID: 860001, UserType: 2, TenantID: 860001})
	normal := pkgContext.WithLoginUser(context.Background(), &pkgContext.LoginUser{UserID: 860002, UserType: 2, TenantID: 860001})
	var audit bytes.Buffer
	old := log.Writer()
	log.SetOutput(&audit)
	t.Cleanup(func() { log.SetOutput(old) })
	for target, want := range map[int64]int64{860001: 1, 860002: 2} {
		result, err := svc.InspectTenant(actor, target)
		if err != nil || result.ProductCount != want || result.OrderCount != 1 {
			t.Fatalf("target %d: %v %v", target, result, err)
		}
	}
	if _, err := svc.InspectTenant(normal, 860002); err == nil {
		t.Fatal("ordinary admin crossed tenants")
	}
	if id, _ := pkgContext.TenantID(actor); id != 860001 {
		t.Fatal("inspection mutated actor context")
	}
	for _, expected := range []string{`"actorUserId":860001`, `"sourceTenantId":860001`, `"targetTenantId":860002`, `"operation":"platform.tenant.inspect"`, `"outcome":"success"`, `"outcome":"denied"`} {
		if !strings.Contains(audit.String(), expected) {
			t.Fatalf("audit missing %s: %s", expected, audit.String())
		}
	}
	for name, run := range map[string]func() error{
		"unsupported departmental scope": func() error { return roles.UpdateRoleDataScope(normal, 860002, 2, []int64{1}) },
		"no default password": func() error {
			_, err := users.CreateUser(normal, &contract.UserSaveReq{Username: "no-password-security"})
			return err
		},
		"weak update password": func() error {
			return users.UpdateUserPassword(normal, &contract.UserUpdatePasswordReq{ID: 860002, Password: "short"})
		},
		"weak reset password": func() error {
			return users.ResetUserPassword(normal, &contract.UserResetPasswordReq{ID: 860002, Password: "short"})
		},
		"create platform role": func() error {
			_, err := roles.CreateRole(normal, &contract.RoleSaveReq{Code: PlatformAdminRole})
			return err
		},
		"create super role": func() error {
			_, err := roles.CreateRole(normal, &contract.RoleSaveReq{Code: "super_admin"})
			return err
		},
		"promote role": func() error {
			return roles.UpdateRole(normal, &contract.RoleSaveReq{ID: 860002, Code: PlatformAdminRole})
		},
		"rename reserved role":    func() error { return roles.UpdateRole(normal, &contract.RoleSaveReq{ID: 860001, Code: "renamed"}) },
		"assign reserved role":    func() error { return permissions.AssignUserRole(normal, 860002, []int64{860001}) },
		"remove reserved binding": func() error { return permissions.AssignUserRole(normal, 860001, nil) },
		"reserved menu":           func() error { return permissions.AssignRoleMenu(normal, 860001, nil) },
		"create user role alternate": func() error {
			_, err := users.CreateUser(normal, &contract.UserSaveReq{Username: "illegal", RoleIDs: []int64{860001}})
			return err
		},
		"update user role alternate": func() error {
			return users.UpdateUser(normal, &contract.UserSaveReq{ID: 860002, RoleIDs: []int64{860001}})
		},
		"take over platform password": func() error {
			return users.UpdateUserPassword(normal, &contract.UserUpdatePasswordReq{ID: 860001, Password: "replacement"})
		},
		"reset platform password": func() error {
			return users.ResetUserPassword(normal, &contract.UserResetPasswordReq{ID: 860001, Password: "replacement"})
		},
		"delete platform actor": func() error { return users.DeleteUser(normal, 860001) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := run(); err == nil {
				t.Fatal("reserved role escalation accepted")
			}
		})
	}
	if err := roles.UpdateRoleDataScope(normal, 860002, 1, nil); err != nil {
		t.Fatalf("supported All scope unwritable: %v", err)
	}
	normalRole, err := q.SystemRole.WithContext(normal).Where(q.SystemRole.ID.Eq(860002)).First()
	if err != nil || normalRole.DataScope != 1 || len(normalRole.DataScopeDeptIds) != 0 {
		t.Fatalf("rejected scope mutation changed role: %v %v", normalRole, err)
	}
	role, err := q.SystemRole.WithContext(normal).Where(q.SystemRole.ID.Eq(860001)).First()
	if err != nil || role.Code != PlatformAdminRole {
		t.Fatalf("reserved role altered: %v %v", role, err)
	}
	user, err := q.SystemUser.WithContext(normal).Where(q.SystemUser.ID.Eq(860001)).First()
	if err != nil || user.Password != "unchanged" {
		t.Fatalf("platform actor altered: %v %v", user, err)
	}
}

func TestAdminPasswordLengthBoundary(t *testing.T) {
	for _, length := range []int{0, 7, 8, 72, 73} {
		err := validateAdminPassword(strings.Repeat("a", length))
		if (err == nil) != (length >= 8 && length <= 72) {
			t.Fatalf("length %d err %v", length, err)
		}
	}
}
