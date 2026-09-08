package model

// SystemConfig 参数配置表
type SystemConfig struct {
	ID        int64   `gorm:"primaryKey;autoIncrement;comment:参数主键" json:"id"`
	Category  string  `gorm:"size:50;not null;comment:参数分类" json:"category"`
	Name      string  `gorm:"size:100;not null;comment:参数名称" json:"name"`
	ConfigKey string  `gorm:"size:100;not null;comment:参数键名" json:"configKey"`
	Value     string  `gorm:"size:500;not null;comment:参数键值" json:"value"`
	Type      int32   `gorm:"size:4;not null;default:1;comment:参数类型" json:"type"`
	Visible   BitBool `gorm:"not null;default:(1);comment:是否可见" json:"visible"`
	Remark    string  `gorm:"size:500;comment:备注" json:"remark"`
	TenantBaseDO
}

func (SystemConfig) TableName() string {
	return "infra_config"
}
