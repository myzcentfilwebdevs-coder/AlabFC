package models

import "time"

type User struct {
	UserID       int          `gorm:"column:user_id;primaryKey;autoIncrement" json:"user_id"`
	Username     string       `gorm:"column:username" json:"username"`
	Email        string       `gorm:"column:email" json:"email"`
	PasswordHash string       `gorm:"column:password_hash" json:"-"`
	RoleID       int          `gorm:"column:role_id" json:"role_id"`
	StatusID     int          `gorm:"column:status_id" json:"status_id"`
	IsVerified   bool         `gorm:"column:is_verified" json:"is_verified"`
	LastLoginAt  *time.Time   `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time    `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time    `gorm:"column:updated_at" json:"updated_at"`
	Profile      *UserProfile `gorm:"foreignKey:UserID;references:UserID" json:"profile,omitempty"`
}

func (User) TableName() string {
	return "users"
}

type UserProfile struct {
	ProfileID     int       `gorm:"column:profile_id;primaryKey;autoIncrement" json:"profile_id"`
	UserID        int       `gorm:"column:user_id" json:"user_id"`
	FirstName     string    `gorm:"column:first_name" json:"first_name"`
	MiddleName    string    `gorm:"column:middle_name" json:"middle_name"`
	LastName      string    `gorm:"column:last_name" json:"last_name"`
	Birthdate     *string   `gorm:"column:birthdate" json:"birthdate"`
	GenderID      *int      `gorm:"column:gender_id" json:"gender_id"`
	AgeCategoryID *int      `gorm:"column:age_category_id" json:"age_category_id"`
	PositionID    *int      `gorm:"column:position_id" json:"position_id"`
	JerseyNumber  *int      `gorm:"column:jersey_number" json:"jersey_number"`
	PhoneNumber   string    `gorm:"column:phone_number" json:"phone_number"`
	Address       string    `gorm:"column:address" json:"address"`
	AvatarURL     string    `gorm:"column:avatar_url" json:"avatar_url"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (UserProfile) TableName() string {
	return "user_profiles"
}

type ParentPlayer struct {
	ID             int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ParentUserID   int       `gorm:"column:parent_user_id" json:"parent_user_id"`
	PlayerUserID   int       `gorm:"column:player_user_id" json:"player_user_id"`
	RelationshipID *int      `gorm:"column:relationship_id" json:"relationship_id"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	Player         *User     `gorm:"foreignKey:PlayerUserID;references:UserID" json:"player,omitempty"`
}

func (ParentPlayer) TableName() string {
	return "parent_players"
}

type OTPVerification struct {
	OTPID     int        `gorm:"column:otp_id;primaryKey;autoIncrement" json:"otp_id"`
	UserID    int        `gorm:"column:user_id" json:"user_id"`
	OTPCode   string     `gorm:"column:otp_code" json:"otp_code"`
	Purpose   string     `gorm:"column:purpose" json:"purpose"`
	ExpiresAt time.Time  `gorm:"column:expires_at" json:"expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at" json:"used_at"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (OTPVerification) TableName() string {
	return "otp_verifications"
}
