// bootstrap provisions an explicit administrator without storing default credentials in migrations.
package main

import (
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	dsn := os.Getenv("RUOYI_DATABASE_DSN")
	username := os.Getenv("RUOYI_BOOTSTRAP_USERNAME")
	password := os.Getenv("RUOYI_BOOTSTRAP_PASSWORD")
	if dsn == "" || username == "" || len(password) < 16 || len(password) > 72 {
		return fmt.Errorf("RUOYI_DATABASE_DSN, RUOYI_BOOTSTRAP_USERNAME and a 16..72-byte RUOYI_BOOTSTRAP_PASSWORD are required")
	}
	platformFlag := os.Getenv("RUOYI_BOOTSTRAP_PLATFORM_ADMIN")
	if platformFlag != "" && platformFlag != "false" && platformFlag != "true" {
		return fmt.Errorf("RUOYI_BOOTSTRAP_PLATFORM_ADMIN must be true or false")
	}
	platformAdmin := platformFlag == "true"
	website := strings.ToLower(strings.TrimSpace(os.Getenv("RUOYI_BOOTSTRAP_WEBSITE")))
	if website != "" && !validHostname(website) {
		return fmt.Errorf("RUOYI_BOOTSTRAP_WEBSITE must be a hostname without scheme, path or port")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	return db.Transaction(func(tx *gorm.DB) error {
		var tenant model.SystemTenant
		if err := tx.First(&tenant, 1).Error; err != nil {
			return fmt.Errorf("apply migrations first: %w", err)
		}
		var count int64
		if err := tx.Model(&model.SystemUser{}).Where("tenant_id = ? AND username = ?", 1, username).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("administrator already exists; bootstrap never resets passwords")
		}
		user := model.SystemUser{Username: username, Password: string(hash), Nickname: username, Status: 0, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.SystemUserRole{UserID: user.ID, RoleID: 1, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}).Error; err != nil {
			return err
		}
		// Reserved platform access is offline-only, explicit, and never granted by seed/default.
		if platformAdmin {
			var role model.SystemRole
			err := tx.Where("tenant_id = ? AND code = ?", 1, "platform_admin").First(&role).Error
			if err == gorm.ErrRecordNotFound {
				role = model.SystemRole{Name: "Platform read administrator", Code: "platform_admin", Type: 1, Status: 0, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
				if err := tx.Create(&role).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else if role.Type != 1 || role.Status != 0 {
				return fmt.Errorf("reserved platform_admin role is not an enabled system role")
			}
			if err := tx.Create(&model.SystemUserRole{UserID: user.ID, RoleID: role.ID, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}).Error; err != nil {
				return err
			}
		}
		updates := map[string]any{"contact_user_id": user.ID}
		if website != "" {
			updates["website"] = website
		}
		return tx.Model(&model.SystemTenant{}).Where("id = ?", 1).Updates(updates).Error
	})
}

func validHostname(host string) bool {
	if len(host) > 253 || strings.ContainsAny(host, "/:@, ") {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	label := regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	for _, part := range strings.Split(host, ".") {
		if !label.MatchString(part) {
			return false
		}
	}
	return true
}
