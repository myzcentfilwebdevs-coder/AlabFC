package models

import "time"

type Tournament struct {
	TournamentID    int                `gorm:"column:tournament_id;primaryKey;autoIncrement" json:"tournament_id"`
	Title           string             `gorm:"column:title" json:"title"`
	Description     string             `gorm:"column:description" json:"description"`
	Venue           string             `gorm:"column:venue" json:"venue"`
	StartDate       string             `gorm:"column:start_date" json:"start_date"`
	EndDate         *string            `gorm:"column:end_date" json:"end_date"`
	AgeCategoryID   *int               `gorm:"column:age_category_id" json:"age_category_id"`
	AgeCategoryIDs  string             `gorm:"column:age_category_ids" json:"age_category_ids"`
	StatusID        *int               `gorm:"column:status_id" json:"status_id"`
	RegistrationFee float64            `gorm:"column:registration_fee" json:"registration_fee"`
	BannerURL       string             `gorm:"column:banner_url" json:"banner_url"`
	CreatedAt       time.Time          `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time          `gorm:"column:updated_at" json:"updated_at"`
	Status          *GenericReference  `gorm:"foreignKey:StatusID;references:Seqnum" json:"status,omitempty"`
	Matches         []TournamentMatch  `gorm:"foreignKey:TournamentID;references:TournamentID" json:"matches,omitempty"`
	Rosters         []TournamentRoster `gorm:"foreignKey:TournamentID;references:TournamentID" json:"rosters,omitempty"`
}

func (Tournament) TableName() string {
	return "tournaments"
}

type TournamentMatch struct {
	MatchID       int               `gorm:"column:match_id;primaryKey;autoIncrement" json:"match_id"`
	TournamentID  int               `gorm:"column:tournament_id" json:"tournament_id"`
	AgeCategoryID *int              `gorm:"column:age_category_id" json:"age_category_id"`
	GameLabel     string            `gorm:"column:game_label" json:"game_label"`
	Opponent      string            `gorm:"column:opponent" json:"opponent"`
	MatchDate     string            `gorm:"column:match_date" json:"match_date"`
	MatchTime     *string           `gorm:"column:match_time" json:"match_time"`
	Pitch         string            `gorm:"column:pitch" json:"pitch"`
	AlabScore     *int              `gorm:"column:alab_score" json:"alab_score"`
	OpponentScore *int              `gorm:"column:opponent_score" json:"opponent_score"`
	StatusID      *int              `gorm:"column:status_id" json:"status_id"`
	Remarks       string            `gorm:"column:remarks" json:"remarks"`
	CreatedAt     time.Time         `gorm:"column:created_at" json:"created_at"`
	AgeCategory   *GenericReference `gorm:"foreignKey:AgeCategoryID;references:Seqnum" json:"age_category,omitempty"`
	Status        *GenericReference `gorm:"foreignKey:StatusID;references:Seqnum" json:"status,omitempty"`
}

func (TournamentMatch) TableName() string {
	return "tournament_matches"
}

type TournamentRoster struct {
	RosterID      int       `gorm:"column:roster_id;primaryKey;autoIncrement" json:"roster_id"`
	TournamentID  int       `gorm:"column:tournament_id" json:"tournament_id"`
	PlayerID      int       `gorm:"column:player_id" json:"player_id"`
	ParentConsent bool      `gorm:"column:parent_consent" json:"parent_consent"`
	Remarks       string    `gorm:"column:remarks" json:"remarks"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	Player        *Player   `gorm:"foreignKey:PlayerID;references:PlayerID" json:"player,omitempty"`
}

func (TournamentRoster) TableName() string {
	return "tournament_rosters"
}
