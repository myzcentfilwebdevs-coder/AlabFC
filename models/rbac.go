package models

type AppModule struct {
	Seqnum     int    `gorm:"column:seqnum;primaryKey;autoIncrement" json:"seqnum"`
	ParentID   int    `gorm:"column:parent_id" json:"parent_id"`
	Access     string `gorm:"column:access" json:"access"`
	AccessType string `gorm:"column:accesstype" json:"accessType"`
	RoutePath  string `gorm:"column:routepath" json:"routePath"`
}

func (AppModule) TableName() string {
	return "app_modules"
}

type RoleAccess struct {
	Seqnum    int    `gorm:"column:seqnum;primaryKey;autoIncrement" json:"seqnum"`
	RoleID    int    `gorm:"column:roleid;uniqueIndex" json:"roleid"`
	ModuleIDs string `gorm:"column:module_ids" json:"module_ids"`
}

func (RoleAccess) TableName() string {
	return "role_access"
}
