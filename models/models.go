package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID    `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Username      string       `gorm:"unique;not null" json:"username"`
	PasswordHash  string       `json:"-"`
	Role          string       `gorm:"not null" json:"role"` // root, hr_manager, accountant, employee
	Department    string       `json:"department"`           // IT, HR, Finance, etc.
	ReferralCode  string       `json:"referral_code,omitempty"`
	WalletAddress string       `json:"wallet_address"`
	Salary        float64      `json:"salary"`
	LeaveBalance  int          `json:"leave_balance"`
	Status        string       `gorm:"not null;default:'active'" json:"status"` // active, inactive, left_company
	Permissions   []Permission `gorm:"many2many:user_permissions;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"permissions"`
	CreatedAt     time.Time    `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time    `gorm:"not null" json:"updated_at"`
}

type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name        string    `gorm:"unique;not null" json:"name"` // attendance_approval, salary_management, etc.
	Description string    `json:"description"`
}

type Department struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"unique;not null" json:"name"`
	ManagerID uuid.UUID `gorm:"type:uuid;not null" json:"manager_id"`
	Manager   User      `gorm:"foreignKey:ManagerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

// For tracking delegated permissions
type PermissionGrant struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	GrantedBy    uuid.UUID `json:"granted_by"`
	GrantedTo    uuid.UUID `json:"granted_to"`
	PermissionID uint      `json:"permission_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// For salary approval workflow
type SalaryApproval struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ApprovedBy  uuid.UUID `gorm:"type:uuid" json:"approved_by"`
	Approver    User      `gorm:"foreignKey:ApprovedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Month       time.Time `gorm:"not null" json:"month"`
	BaseSalary  float64   `json:"base_salary"`
	Deductions  float64   `json:"deductions"`
	Bonus       float64   `json:"bonus"`
	FinalSalary float64   `json:"final_salary"`
	Status      string    `gorm:"not null;default:'pending'" json:"status"` // pending, approved, rejected
	ApprovedAt  time.Time `json:"approved_at"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
}

type Attendance struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID        uuid.UUID `gorm:"type:uuid" json:"user_id"`
	User          User      `gorm:"foreignKey:UserID"`
	CheckInTime   time.Time `json:"check_in_time"`
	CheckOutTime  time.Time `json:"check_out_time"`
	Status        string    `json:"status"` // on_time, late, absent
	ViolationType string    `json:"violation_type,omitempty"`
	LeaveType     string    `json:"leave_type,omitempty"`
}

type LeaveRequest struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID     uuid.UUID `gorm:"type:uuid"`
	User       User      `gorm:"foreignKey:UserID"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Type       string    `json:"type"` // resignation, vacation, sick, etc.
	Reason     string    `json:"reason"`
	Status     string    // pending, approved, rejected
	ApprovedBy uuid.UUID `gorm:"type:uuid"`
	Approver   User      `gorm:"foreignKey:ApprovedBy"`
	ApprovedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Violation struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID          uuid.UUID `json:"user_id"`
	Type            string    `json:"type"`
	Date            time.Time `json:"date"`
	Details         string    `json:"details"`
	DeductionAmount float64   `json:"deduction_amount"`
	User            User      `gorm:"foreignKey:UserID"`
}

type Report struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID  uuid.UUID `gorm:"type:uuid" json:"user_id"`
	User    User      `gorm:"foreignKey:UserID"`
	Type    string    `json:"type"`
	Date    time.Time `json:"date"`
	FileURL string    `json:"file_url"`
}

type CompanyRule struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	RuleName  string    `json:"rule_name"`
	Details   string    `json:"details"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Absence struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID      uuid.UUID `gorm:"type:uuid" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID"`
	Date        time.Time
	Type        string // with_permission, without_permission
	Reason      string
	Status      string    // pending, processed
	ProcessedBy uuid.UUID `gorm:"type:uuid" json:"processed_by"`
	Processor   User      `gorm:"foreignKey:ProcessedBy"`
	ProcessedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type UserPermission struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID       uuid.UUID `gorm:"type:uuid" json:"user_id"`
	User         User      `gorm:"foreignKey:UserID"`
	PermissionID uint
	GrantedBy    uuid.UUID `gorm:"type:uuid" json:"granted_by"`
	Granter      User      `gorm:"foreignKey:GrantedBy"`
	ExpiresAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ReferralCode struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	Code      string    `gorm:"unique"`
	CreatedBy uuid.UUID `gorm:"type:uuid"`
	UsedBy    uuid.UUID `gorm:"type:uuid"`
	Creator   User      `gorm:"foreignKey:CreatedBy"`
	User      User      `gorm:"foreignKey:UsedBy"`
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PayrollApproval struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;onDelete:CASCADE"`
	Month       time.Time
	Department  string
	TotalAmount float64
	Status      string // pending, approved
	ApprovedBy  uuid.UUID
	ApprovedAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
