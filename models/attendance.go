package models

import "time"

type TrainingSession struct {
	SessionID     int       `gorm:"column:session_id;primaryKey;autoIncrement" json:"session_id"`
	Title         string    `gorm:"column:title" json:"title"`
	SessionDate   string    `gorm:"column:session_date" json:"session_date"`
	StartTime     *string   `gorm:"column:start_time" json:"start_time"`
	EndTime       *string   `gorm:"column:end_time" json:"end_time"`
	Venue         string    `gorm:"column:venue" json:"venue"`
	AgeCategoryID *int      `gorm:"column:age_category_id" json:"age_category_id"`
	CoachUserID   *int      `gorm:"column:coach_user_id" json:"coach_user_id"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (TrainingSession) TableName() string {
	return "training_sessions"
}

type PlayerAttendance struct {
	AttendanceID int        `gorm:"column:attendance_id;primaryKey;autoIncrement" json:"attendance_id"`
	SessionID    int        `gorm:"column:session_id" json:"session_id"`
	PlayerUserID int        `gorm:"column:player_user_id" json:"player_user_id"`
	StatusID     *int       `gorm:"column:status_id" json:"status_id"`
	CheckInTime  *time.Time `gorm:"column:check_in_time" json:"check_in_time"`
	Remarks      string     `gorm:"column:remarks" json:"remarks"`
	RecordedBy   *int       `gorm:"column:recorded_by" json:"recorded_by"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	Player       *User      `gorm:"foreignKey:PlayerUserID;references:UserID" json:"player,omitempty"`
}

func (PlayerAttendance) TableName() string {
	return "player_attendance"
}
