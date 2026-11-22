package models

const (
	RefRoleAdmin  = 5
	RefRoleCoach  = 6
	RefRoleParent = 7
	RefRolePlayer = 8

	RefAccountActive = 10

	RefSOAUnpaid  = 42
	RefSOAPartial = 43
	RefSOAPaid    = 44
)

type GenericReference struct {
	Seqnum   int    `gorm:"column:seqnum;primaryKey;autoIncrement" json:"seqnum"`
	ParentID int    `gorm:"column:parent_id" json:"parent_id"`
	Details  string `gorm:"column:details" json:"details"`
}

func (GenericReference) TableName() string {
	return "genericreference"
}
