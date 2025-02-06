package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                 string       `gorm:"type:text;primary_key" json:"id"`
	FullName           string       `gorm:"type:text;not null;default:''" json:"full_name"`
	Email              string       `gorm:"type:text;unique;not null;default:''" json:"email"`
	PhoneNumber        string       `gorm:"type:text;default:''" json:"phone_number"`
	Address            string       `gorm:"type:text;default:''" json:"address"`
	DateOfBirth        time.Time    `json:"date_of_birth"`
	Gender             string       `gorm:"type:text;default:''" json:"gender"`
	TaxID              string       `gorm:"type:text;default:''" json:"tax_id"`
	HealthInsuranceID  string       `gorm:"type:text;default:''" json:"health_insurance_id"`
	SocialInsuranceID  string       `gorm:"type:text;default:''" json:"social_insurance_id"`
	NumberOfDependents int          `json:"number_of_dependents"`
	Position           string       `gorm:"type:text;default:''" json:"position"`
	Location           string       `gorm:"type:text;default:''" json:"location"`
	OnboardDate        time.Time    `json:"onboard_date"`
	Role               string       `gorm:"type:text;not null;default:'employee'" json:"role"`
	Department         string       `gorm:"type:text;default:''" json:"department"`
	ReferralCode       string       `json:"referral_code,omitempty"`
	WalletAddress      string       `gorm:"type:text;default:''" json:"wallet_address"`
	Salary             float64      `gorm:"default:0" json:"salary"`
	LeaveBalance       int          `gorm:"default:0" json:"leave_balance"`
	Status             string       `gorm:"type:text;not null;default:'active'" json:"status"`
	Permissions        []Permission `gorm:"many2many:user_permissions;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"permissions"`
	CreatedAt          time.Time    `gorm:"not null" json:"created_at"`
	UpdatedAt          time.Time    `gorm:"not null" json:"updated_at"`
}

type Permission struct {
	ID          string `gorm:"type:text;primary_key" json:"id"`
	Name        string `gorm:"unique;not null" json:"name"` // attendance_approval, salary_management, etc.
	Description string `json:"description"`
}

type Department struct {
	ID        string    `gorm:"type:text;primary_key" json:"id"`
	Name      string    `gorm:"unique;not null" json:"name"`
	ManagerID string    `gorm:"type:text;not null" json:"manager_id"`
	Manager   User      `gorm:"foreignKey:ManagerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

// For tracking delegated permissions
type PermissionGrant struct {
	ID           string    `gorm:"type:text;primary_key" json:"id"`
	GrantedBy    string    `gorm:"type:text" json:"granted_by"`
	GrantedTo    string    `gorm:"type:text" json:"granted_to"`
	PermissionID uint      `json:"permission_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// For salary approval workflow
type SalaryApproval struct {
	ID          string    `gorm:"type:text;primary_key" json:"id"`
	UserID      string    `gorm:"type:text;not null" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ApprovedBy  string    `gorm:"type:text" json:"approved_by"`
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
	ID           string    `gorm:"type:text;primary_key" json:"id"`
	UserID       string    `gorm:"type:text" json:"user_id"`
	User         User      `gorm:"foreignKey:UserID"`
	CheckInTime  time.Time `json:"check_in_time"`
	CheckOutTime time.Time `json:"check_out_time"`
	ExpectedTime time.Time `json:"expected_time"`
}

type Absence struct {
	ID          string     `gorm:"type:text;primary_key" json:"id"`
	UserID      string     `gorm:"type:text;references:users(id);not null" json:"user_id"`
	User        User       `gorm:"foreignKey:UserID;references:ID"`
	Date        time.Time  `gorm:"not null" json:"date"`
	Type        string     `gorm:"type:text;not null;check:type IN ('with_permission','without_permission','resign')" json:"type"`
	Reason      string     `gorm:"type:text;not null" json:"reason"`
	Status      string     `gorm:"type:text;not null;default:'pending';check:status IN ('pending','approved','rejected')" json:"status"`
	ProcessedBy *string    `gorm:"type:text;references:users(id)" json:"processed_by"`
	Processor   User       `gorm:"foreignKey:ProcessedBy;references:ID"`
	ProcessedAt *time.Time `json:"processed_at"`
	CreatedAt   time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null" json:"updated_at"`
}

// BeforeSave hook to validate ProcessedBy and ProcessedAt
func (a *Absence) BeforeSave(tx *gorm.DB) error {
	// Validate required fields
	if a.UserID == "" {
		return errors.New("user ID is required")
	}
	if a.Type == "" {
		return errors.New("type is required")
	}
	if a.Reason == "" {
		return errors.New("reason is required")
	}

	// Validate ProcessedBy for approved/rejected status
	if a.Status == "approved" || a.Status == "rejected" {
		if a.ProcessedBy == nil {
			return errors.New("processed_by is required for approved/rejected absences")
		}
		if a.ProcessedAt == nil {
			now := time.Now()
			a.ProcessedAt = &now
		}
	}

	// Clear processor fields for pending status
	if a.Status == "pending" {
		a.ProcessedBy = nil
		a.ProcessedAt = nil
	}

	return nil
}

type UserPermission struct {
	ID           string `gorm:"type:text;primary_key" json:"id"`
	UserID       string `gorm:"type:text;primary_key" json:"user_id"`
	User         User   `gorm:"foreignKey:UserID"`
	PermissionID uint
	GrantedBy    uuid.UUID `gorm:"type:uuid" json:"granted_by"`
	Granter      User      `gorm:"foreignKey:GrantedBy"`
	ExpiresAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
