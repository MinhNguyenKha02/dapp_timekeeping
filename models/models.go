package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID    `gorm:"type:uuid;primary_key" json:"id"`
	Username      string       `gorm:"unique" json:"username"`
	PasswordHash  string       `json:"-"`
	Role          string       `json:"role"`       // root, hr_manager, accountant, employee
	Department    string       `json:"department"` // IT, HR, Finance, etc.
	ReferralCode  string       `json:"referral_code,omitempty"`
	WalletAddress string       `json:"wallet_address"`
	Salary        float64      `json:"salary"`
	LeaveBalance  int          `json:"leave_balance"`
	Status        string       `json:"status"` // active, inactive, left_company
	Permissions   []Permission `gorm:"many2many:user_permissions;" json:"permissions"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Permission struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `json:"name"` // attendance_approval, salary_management, etc.
	Description string `json:"description"`
}

type Department struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `json:"name"`
	ManagerID uuid.UUID `json:"manager_id"`
	Manager   User      `gorm:"foreignKey:ManagerID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// For tracking delegated permissions
type PermissionGrant struct {
	ID           uint      `gorm:"primaryKey"`
	GrantedBy    uuid.UUID `json:"granted_by"`
	GrantedTo    uuid.UUID `json:"granted_to"`
	PermissionID uint      `json:"permission_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// For salary approval workflow
type SalaryApproval struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uuid.UUID `json:"user_id"`
	Month       time.Time `json:"month"`
	BaseSalary  float64   `json:"base_salary"`
	Deductions  float64   `json:"deductions"`
	Bonus       float64   `json:"bonus"`
	FinalSalary float64   `json:"final_salary"`
	Status      string    `json:"status"` // pending, approved, rejected
	ApprovedBy  uuid.UUID `json:"approved_by"`
	ApprovedAt  time.Time `json:"approved_at"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
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

type CompanyRule struct {
	ID        uint      `gorm:"primaryKey"`
	RuleName  string    `json:"rule_name"`
	Details   string    `json:"details"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
