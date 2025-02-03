package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Username      string    `gorm:"unique" json:"username"`
	PasswordHash  string    `json:"-"`
	Role          string    `json:"role"` // root, hr_manager, accountant, employee
	ReferralCode  string    `json:"referral_code,omitempty"`
	WalletAddress string    `json:"wallet_address"`
	Salary        float64   `json:"salary"`
	LeaveBalance  int       `json:"leave_balance"`
	Status        string    `json:"status"` // active, inactive, left_company
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Attendance struct {
	ID            uint      `gorm:"primaryKey"`
	UserID        uuid.UUID `json:"user_id"`
	CheckInTime   time.Time `json:"check_in_time"`
	CheckOutTime  time.Time `json:"check_out_time"`
	Status        string    `json:"status"` // on_time, late, absent
	ViolationType string    `json:"violation_type,omitempty"`
	LeaveType     string    `json:"leave_type,omitempty"`
	User          User      `gorm:"foreignKey:UserID"`
}

type LeaveRequest struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uuid.UUID `json:"user_id"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Status    string    `json:"status"` // pending, approved, rejected
	LeaveType string    `json:"leave_type"`
	Reason    string    `json:"reason"`
	User      User      `gorm:"foreignKey:UserID"`
}

type Violation struct {
	ID              uint      `gorm:"primaryKey"`
	UserID          uuid.UUID `json:"user_id"`
	Type            string    `json:"type"`
	Date            time.Time `json:"date"`
	Details         string    `json:"details"`
	DeductionAmount float64   `json:"deduction_amount"`
	User            User      `gorm:"foreignKey:UserID"`
}

type Report struct {
	ID      uint      `gorm:"primaryKey"`
	UserID  uuid.UUID `json:"user_id"`
	Type    string    `json:"type"`
	Date    time.Time `json:"date"`
	FileURL string    `json:"file_url"`
	User    User      `gorm:"foreignKey:UserID"`
}
