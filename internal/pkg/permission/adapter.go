package permission

import (
	"context"
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"gorm.io/gorm"
)

// YudoAdapter 自定义 Casbin 适配器
// 直接从 system_role_menu 和 system_user_role 表加载策略
type YudoAdapter struct {
	db *gorm.DB
}

// NewAdapter 创建适配器
func NewAdapter(db *gorm.DB) *YudoAdapter {
	return &YudoAdapter{db: db}
}

// LoadPolicy 从数据库加载策略
func (a *YudoAdapter) LoadPolicy(model model.Model) error {
	// 1. 加载角色-权限策略 (p, role, permission, access)
	// 对应 system_role_menu + system_menu
	if err := a.loadRolePolicy(model); err != nil {
		return err
	}

	// 2. 加载用户-角色策略 (g, userId, role)
	// 对应 system_user_role
	if err := a.loadUserRolePolicy(model); err != nil {
		return err
	}

	return nil
}

func (a *YudoAdapter) loadRolePolicy(model model.Model) error {
	// 查询 SQL: 用于获取 角色ID -> 权限标识 的映射
	// 过滤掉 permission 为空的菜单（目录等）
	type Result struct {
		TenantID   int64
		RoleID     int64
		Permission string
	}

	// 联表查询：system_role <-> system_role_menu <-> system_menu
	// 使用 DISTINCT 去重
	var results []Result
	err := a.db.WithContext(database.PolicyReadContext(context.Background())).Table("system_role_menu srm").
		Select("DISTINCT srm.tenant_id, srm.role_id, sm.permission").
		Joins("JOIN system_role sr ON sr.id = srm.role_id AND sr.tenant_id = srm.tenant_id").
		Joins("JOIN system_menu sm ON sm.id = srm.menu_id").
		Where("sm.permission != '' AND sm.deleted = 0 AND sr.deleted = 0 AND srm.deleted = 0 AND sm.status = 0 AND sr.status = 0").
		Scan(&results).Error

	if err != nil {
		return err
	}

	for _, line := range results {
		// 添加策略: p, role_id, permission, access
		// 使用 role_id 确保多租户下的唯一性
		persist.LoadPolicyLine(fmt.Sprintf("p, tenant:%d:role:%d, %s, access", line.TenantID, line.RoleID, line.Permission), model)
	}
	return nil
}

func (a *YudoAdapter) loadUserRolePolicy(model model.Model) error {
	// 查询 SQL: 获取 用户ID -> 角色ID 的映射
	type Result struct {
		TenantID int64
		UserID   int64
		RoleID   int64
	}

	var results []Result
	err := a.db.WithContext(database.PolicyReadContext(context.Background())).Table("system_user_role sur").
		Select("sur.tenant_id, sur.user_id, sur.role_id").
		Joins("JOIN system_role sr ON sr.id = sur.role_id AND sr.tenant_id = sur.tenant_id").
		Joins("JOIN system_users su ON su.id = sur.user_id AND su.tenant_id = sur.tenant_id").
		Where("sur.deleted = 0 AND sr.deleted = 0 AND su.deleted = 0 AND sr.status = 0 AND su.status = 0").
		Scan(&results).Error

	if err != nil {
		return err
	}

	for _, line := range results {
		// 添加分组策略: g, userId, role_id
		persist.LoadPolicyLine(fmt.Sprintf("g, tenant:%d:user:%d, tenant:%d:role:%d", line.TenantID, line.UserID, line.TenantID, line.RoleID), model)
	}

	return nil
}

// SavePolicy 保存策略 (只读，不需要实现)
func (a *YudoAdapter) SavePolicy(model model.Model) error {
	return nil
}

// AddPolicy 添加策略 (只读，不需要实现)
func (a *YudoAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	return nil
}

// RemovePolicy 移除策略 (只读，不需要实现)
func (a *YudoAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return nil
}

// RemoveFilteredPolicy 移除过滤后的策略 (只读，不需要实现)
func (a *YudoAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return nil
}
