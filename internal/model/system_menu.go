package model

// SystemMenu 菜单权限表
type SystemMenu struct {
	ID            int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name          string  `gorm:"column:name;not null" json:"name"`
	Permission    string  `gorm:"column:permission;default:''" json:"permission"`
	Type          int32   `gorm:"column:type;not null" json:"type"` // 1:目录, 2:菜单, 3:按钮
	Sort          int32   `gorm:"column:sort;not null;default:0" json:"sort"`
	ParentID      int64   `gorm:"column:parent_id;not null;default:0" json:"parentId"`
	Path          string  `gorm:"column:path;default:''" json:"path"`
	Icon          string  `gorm:"column:icon;default:''" json:"icon"`
	Component     string  `gorm:"column:component;default:''" json:"component"`
	ComponentName string  `gorm:"column:component_name;default:''" json:"componentName"`
	Status        int32   `gorm:"column:status;not null;default:0" json:"status"` // 0:开启, 1:禁用
	Visible       BitBool `gorm:"column:visible;not null;default:(1)" json:"visible"`
	KeepAlive     BitBool `gorm:"column:keep_alive;not null;default:(1)" json:"keepAlive"`
	AlwaysShow    BitBool `gorm:"column:always_show;not null;default:(1)" json:"alwaysShow"`
	BaseDO
}

func (SystemMenu) TableName() string {
	return "system_menu"
}
