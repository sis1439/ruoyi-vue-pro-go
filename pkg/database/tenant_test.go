package database

import (
	"context"
	"fmt"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
	"time"
)

type tenantFixture struct {
	ID       int64 `gorm:"primaryKey"`
	TenantID int64
	Name     string
}

func (tenantFixture) TableName() string { return "tenant_fixture" }
func tenantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN required")
	}
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlRoot, _ := root.DB()
	t.Cleanup(func() { sqlRoot.Close() })
	name := fmt.Sprintf("security_%d", time.Now().UnixNano())
	if err = root.Exec("CREATE SCHEMA " + name).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Exec("DROP SCHEMA " + name + " CASCADE") })
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+name), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err = db.AutoMigrate(&tenantFixture{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&[]tenantFixture{{ID: 1, TenantID: 1, Name: "A"}, {ID: 2, TenantID: 2, Name: "B"}}).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Use(&AuditPlugin{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Use(&TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func TestTenantDatabaseBoundaries(t *testing.T) {
	db := tenantTestDB(t)
	a := db.WithContext(pkgContext.WithTenant(context.Background(), 1))
	b := db.WithContext(pkgContext.WithTenant(context.Background(), 2))
	var rows []tenantFixture
	if err := db.Find(&rows).Error; err == nil {
		t.Fatal("missing tenant query accepted")
	}
	if err := a.Find(&rows).Error; err != nil || len(rows) != 1 || rows[0].Name != "A" {
		t.Fatalf("tenant A query: %v %v", rows, err)
	}
	var count int64
	if err := a.Model(&tenantFixture{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("count %d %v", count, err)
	}
	if err := a.First(&tenantFixture{}, 2).Error; err != gorm.ErrRecordNotFound {
		t.Fatalf("cross tenant ID: %v", err)
	}
	result := a.Model(&tenantFixture{}).Where("id = ?", 2).Update("name", "stolen")
	if result.Error != nil || result.RowsAffected != 0 {
		t.Fatalf("cross update: %v rows %d", result.Error, result.RowsAffected)
	}
	result = a.Delete(&tenantFixture{}, 2)
	if result.Error != nil || result.RowsAffected != 0 {
		t.Fatalf("cross delete: %v rows %d", result.Error, result.RowsAffected)
	}
	for name, run := range map[string]func() error{
		"raw read":       func() error { return a.Raw("SELECT * FROM tenant_fixture").Scan(&rows).Error },
		"raw write":      func() error { return a.Exec("DELETE FROM tenant_fixture").Error },
		"table override": func() error { return a.Table("tenant_fixture x").Find(&rows).Error },
		"join":           func() error { return a.Joins("JOIN tenant_fixture b ON true").Find(&rows).Error },
		"subquery": func() error {
			return a.Select("(SELECT name FROM tenant_fixture WHERE id=2) AS name").Find(&rows).Error
		},
		"tenant mutation":   func() error { return a.Model(&tenantFixture{}).Where("id=1").Update("tenant_id", 2).Error },
		"cross insert":      func() error { return a.Create(&tenantFixture{ID: 3, TenantID: 2}).Error },
		"no context insert": func() error { return db.Create(&tenantFixture{ID: 3}).Error },
		"save cross tenant": func() error { return a.Save(&tenantFixture{ID: 2, TenantID: 1, Name: "stolen"}).Error },
	} {
		t.Run(name, func(t *testing.T) {
			if err := run(); err == nil {
				t.Fatal("unsafe operation accepted")
			}
		})
	}
	if err := a.Save(&tenantFixture{ID: 1, TenantID: 1, Name: "updated"}).Error; err != nil {
		t.Fatalf("own Save rejected: %v", err)
	}
	if err := a.Model(&tenantFixture{}).Where("id = ?", 1).Select("*").Updates(&tenantFixture{ID: 1, Name: "zero tenant omitted"}).Error; err != nil {
		t.Fatal(err)
	}
	var retained tenantFixture
	if err := a.First(&retained, 1).Error; err != nil || retained.TenantID != 1 {
		t.Fatalf("zero tenant moved row: %v %v", retained, err)
	}
	batch := []tenantFixture{{ID: 3, Name: "A3"}, {ID: 4, Name: "A4"}}
	if err := a.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range batch {
		if row.TenantID != 1 {
			t.Fatal("tenant not stamped")
		}
	}
	var other tenantFixture
	if err := b.First(&other, 2).Error; err != nil || other.Name != "B" {
		t.Fatalf("tenant B was changed: %v %v", other, err)
	}
	userCtx := pkgContext.WithLoginUser(context.Background(), &pkgContext.LoginUser{UserID: 3, UserType: 1, TenantID: 2})
	if err := db.WithContext(userCtx).Create(&tenantFixture{ID: 5, Name: "job"}).Error; err != nil {
		t.Fatalf("job context: %v", err)
	}
}
