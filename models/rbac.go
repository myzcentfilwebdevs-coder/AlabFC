package models

type Module struct {
	Seqnum     int    `gorm:"column:seqnum;primaryKey;autoIncrement" json:"seqnum"`
	ParentID   int    `gorm:"column:parent_id" json:"parent_id"`
	ModuleName string `gorm:"column:module_name" json:"module_name"`
	Code       string `gorm:"column:code" json:"code"`
}

func (Module) TableName() string {
	return "modules"
}

type RoleAccess struct {
	Seqnum    int    `gorm:"column:seqnum;primaryKey;autoIncrement" json:"seqnum"`
	RoleID    int    `gorm:"column:roleid;uniqueIndex" json:"roleid"`
	ModuleIDs string `gorm:"column:module_ids" json:"module_ids"`
}

func (RoleAccess) TableName() string {
	return "role_access"
}
