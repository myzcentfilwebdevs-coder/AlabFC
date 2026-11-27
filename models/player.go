package models

import "time"

type Player struct {
	PlayerID      int       `gorm:"column:player_id;primaryKey;autoIncrement" json:"player_id"`
	FirstName     string    `gorm:"column:first_name" json:"first_name"`
	MiddleName    string    `gorm:"column:middle_name" json:"middle_name"`
	LastName      string    `gorm:"column:last_name" json:"last_name"`
	Birthdate     *string   `gorm:"column:birthdate" json:"birthdate"`
	GenderID      *int      `gorm:"column:gender_id" json:"gender_id"`
	AgeCategoryID *int      `gorm:"column:age_category_id" json:"age_category_id"`
	PositionID    *int      `gorm:"column:position_id" json:"position_id"`
	JerseyNumber  *int      `gorm:"column:jersey_number" json:"jersey_number"`
	AvatarURL     string    `gorm:"column:avatar_url" json:"avatar_url"`
	StatusID      int       `gorm:"column:status_id" json:"status_id"`
	CreatedBy     *int      `gorm:"column:created_by" json:"created_by"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
	CreatedByUser *User     `gorm:"foreignKey:CreatedBy;references:UserID" json:"created_by_user,omitempty"`
}

func (Player) TableName() string {
	return "players"
}
