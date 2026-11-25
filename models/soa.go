package models

import "time"

type StatementOfAccount struct {
	SOAID         int               `gorm:"column:soa_id;primaryKey;autoIncrement" json:"soa_id"`
	SOANumber     string            `gorm:"column:soa_number" json:"soa_number"`
	PlayerUserID  int               `gorm:"column:player_user_id" json:"player_user_id"`
	ParentUserID  *int              `gorm:"column:parent_user_id" json:"parent_user_id"`
	BillingPeriod string            `gorm:"column:billing_period" json:"billing_period"`
	DueDate       string            `gorm:"column:due_date" json:"due_date"`
	TotalAmount   float64           `gorm:"column:total_amount" json:"total_amount"`
	AmountPaid    float64           `gorm:"column:amount_paid" json:"amount_paid"`
	Balance       float64           `gorm:"column:balance" json:"balance"`
	StatusID      *int              `gorm:"column:status_id" json:"status_id"`
	Notes         string            `gorm:"column:notes" json:"notes"`
	CreatedAt     time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"column:updated_at" json:"updated_at"`
	Player        *User             `gorm:"foreignKey:PlayerUserID;references:UserID" json:"player,omitempty"`
	Parent        *User             `gorm:"foreignKey:ParentUserID;references:UserID" json:"parent,omitempty"`
	Status        *GenericReference `gorm:"foreignKey:StatusID;references:Seqnum" json:"status,omitempty"`
	Items         []SOAItem         `gorm:"foreignKey:SOAID;references:SOAID" json:"items,omitempty"`
	Payments      []SOAPayment      `gorm:"foreignKey:SOAID;references:SOAID" json:"payments,omitempty"`
}

func (StatementOfAccount) TableName() string {
	return "statements_of_account"
}

type SOAItem struct {
	ItemID      int               `gorm:"column:item_id;primaryKey;autoIncrement" json:"item_id"`
	SOAID       int               `gorm:"column:soa_id" json:"soa_id"`
	FeeTypeID   *int              `gorm:"column:fee_type_id" json:"fee_type_id"`
	Description string            `gorm:"column:description" json:"description"`
	Amount      float64           `gorm:"column:amount" json:"amount"`
	CreatedAt   time.Time         `gorm:"column:created_at" json:"created_at"`
	FeeType     *GenericReference `gorm:"foreignKey:FeeTypeID;references:Seqnum" json:"fee_type,omitempty"`
}

func (SOAItem) TableName() string {
	return "soa_items"
}

type SOAPayment struct {
	PaymentID       int               `gorm:"column:payment_id;primaryKey;autoIncrement" json:"payment_id"`
	SOAID           int               `gorm:"column:soa_id" json:"soa_id"`
	AmountPaid      float64           `gorm:"column:amount_paid" json:"amount_paid"`
	PaymentMethodID *int              `gorm:"column:payment_method_id" json:"payment_method_id"`
	ReferenceNumber string            `gorm:"column:reference_number" json:"reference_number"`
	ReceiptURL      string            `gorm:"column:receipt_url" json:"receipt_url"`
	VerifiedBy      *int              `gorm:"column:verified_by" json:"verified_by"`
	PaidAt          time.Time         `gorm:"column:paid_at" json:"paid_at"`
	PaymentMethod   *GenericReference `gorm:"foreignKey:PaymentMethodID;references:Seqnum" json:"payment_method,omitempty"`
}

func (SOAPayment) TableName() string {
	return "soa_payments"
}
