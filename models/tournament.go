package models

import "time"

type Tournament struct {
	TournamentID    int       `gorm:"column:tournament_id;primaryKey;autoIncrement" json:"tournament_id"`
	Title           string    `gorm:"column:title" json:"title"`
	Description     string    `gorm:"column:description" json:"description"`
	Venue           string    `gorm:"column:venue" json:"venue"`
	StartDate       string    `gorm:"column:start_date" json:"start_date"`
	EndDate         *string   `gorm:"column:end_date" json:"end_date"`
	AgeCategoryID   *int      `gorm:"column:age_category_id" json:"age_category_id"`
	StatusID        *int      `gorm:"column:status_id" json:"status_id"`
	RegistrationFee float64   `gorm:"column:registration_fee" json:"registration_fee"`
	BannerURL       string    `gorm:"column:banner_url" json:"banner_url"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Tournament) TableName() string {
	return "tournaments"
}

type TournamentRoster struct {
	RosterID      int       `gorm:"column:roster_id;primaryKey;autoIncrement" json:"roster_id"`
	TournamentID  int       `gorm:"column:tournament_id" json:"tournament_id"`
	PlayerUserID  int       `gorm:"column:player_user_id" json:"player_user_id"`
	ParentConsent bool      `gorm:"column:parent_consent" json:"parent_consent"`
	Remarks       string    `gorm:"column:remarks" json:"remarks"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	Player        *User     `gorm:"foreignKey:PlayerUserID;references:UserID" json:"player,omitempty"`
}

func (TournamentRoster) TableName() string {
	return "tournament_rosters"
}
