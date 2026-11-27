package models

import "time"

type TrainingSession struct {
	SessionID     int               `gorm:"column:session_id;primaryKey;autoIncrement" json:"session_id"`
	Title         string            `gorm:"column:title" json:"title"`
	SessionDate   string            `gorm:"column:session_date" json:"session_date"`
	StartTime     *string           `gorm:"column:start_time" json:"start_time"`
	EndTime       *string           `gorm:"column:end_time" json:"end_time"`
	Venue         string            `gorm:"column:venue" json:"venue"`
	AgeCategoryID *int              `gorm:"column:age_category_id" json:"age_category_id"`
	CoachUserID   *int              `gorm:"column:coach_user_id" json:"coach_user_id"`
	CreatedAt     time.Time         `gorm:"column:created_at" json:"created_at"`
	AgeCategory   *GenericReference `gorm:"foreignKey:AgeCategoryID;references:Seqnum" json:"age_category,omitempty"`
	Coach         *User             `gorm:"foreignKey:CoachUserID;references:UserID" json:"coach,omitempty"`
}

func (TrainingSession) TableName() string {
	return "training_sessions"
}

type PlayerAttendance struct {
	AttendanceID int               `gorm:"column:attendance_id;primaryKey;autoIncrement" json:"attendance_id"`
	SessionID    int               `gorm:"column:session_id" json:"session_id"`
	PlayerID     int               `gorm:"column:player_id" json:"player_id"`
	StatusID     *int              `gorm:"column:status_id" json:"status_id"`
	CheckInTime  *time.Time        `gorm:"column:check_in_time" json:"check_in_time"`
	Remarks      string            `gorm:"column:remarks" json:"remarks"`
	RecordedBy   *int              `gorm:"column:recorded_by" json:"recorded_by"`
	CreatedAt    time.Time         `gorm:"column:created_at" json:"created_at"`
	Session      *TrainingSession  `gorm:"foreignKey:SessionID;references:SessionID" json:"session,omitempty"`
	Player       *Player           `gorm:"foreignKey:PlayerID;references:PlayerID" json:"player,omitempty"`
	Status       *GenericReference `gorm:"foreignKey:StatusID;references:Seqnum" json:"status,omitempty"`
}

func (PlayerAttendance) TableName() string {
	return "player_attendance"
}
