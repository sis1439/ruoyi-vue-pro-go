package model

import (
	"time"
)

// BaseDO 基础实体对象
type BaseDO struct {
	Creator    string    `gorm:"column:creator;size:64;default:'';comment:创建者" json:"creator"`
	Updater    string    `gorm:"column:updater;size:64;default:'';comment:更新者" json:"updater"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime;comment:创建时间" json:"createTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime;comment:更新时间" json:"updateTime"`
	Deleted    BitBool   `gorm:"column:deleted;softDelete:flag;default:(0);comment:是否删除" json:"deleted"`
}

// TenantBaseDO 包含租户编号的基础实体对象
type TenantBaseDO struct {
	BaseDO
	TenantID int64 `gorm:"column:tenant_id;default:0;comment:租户编号" json:"tenantId"`
}
