package model

import (
	"encoding/json"
)

// InfraFileConfig 文件配置表
type InfraFileConfig struct {
	ID      int64           `gorm:"primaryKey;autoIncrement;comment:配置编号" json:"id"`
	Name    string          `gorm:"size:63;not null;comment:配置名" json:"name"`
	Storage int32           `gorm:"not null;comment:存储器" json:"storage"` // 参见 FileStorageEnum
	Master  BitBool         `gorm:"default:(0);comment:是否为主配置" json:"master"`
	Config  json.RawMessage `gorm:"type:json;serializer:json;comment:支付渠道配置" json:"config"`
	Remark  string          `gorm:"size:255;comment:备注" json:"remark"`
	TenantBaseDO
}

func (InfraFileConfig) TableName() string {
	return "infra_file_config"
}

// InfraFile 文件表
type InfraFile struct {
	ID       int64  `gorm:"primaryKey;autoIncrement;comment:文件编号" json:"id"`
	ConfigId int64  `gorm:"not null;comment:配置编号" json:"configId"`
	Name     string `gorm:"size:255;comment:原文件名" json:"name"`
	Path     string `gorm:"size:255;comment:路径" json:"path"`
	Url      string `gorm:"size:1024;comment:访问地址" json:"url"`
	Type     string `gorm:"size:63;comment:文件类型" json:"type"`
	Size     int    `gorm:"comment:文件大小" json:"size"`
	TenantBaseDO
}

func (InfraFile) TableName() string {
	return "infra_file"
}
