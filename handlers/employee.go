package handlers

import (
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"dapp_timekeeping/utils"

	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AddEmployeeRequest struct {
	FullName      string    `json:"full_name" validate:"required"`
	Email         string    `json:"email" validate:"required,email"`
	PhoneNumber   string    `json:"phone_number" validate:"required"`
	Address       string    `json:"address" validate:"required"`
	DateOfBirth   time.Time `json:"date_of_birth" validate:"required"`
	Gender        string    `json:"gender" validate:"required,oneof=male female other"`
	TaxID         string    `json:"tax_id" validate:"required"`
	Position      string    `json:"position" validate:"required"`
	Location      string    `json:"location" validate:"required"`
	Department    string    `json:"department" validate:"required"`
	WalletAddress string    `json:"wallet_address" validate:"required"`
	Salary        float64   `json:"salary" validate:"required,gt=0"`
	Role          string    `json:"role" validate:"required,oneof=employee hr hr_manager accountant"`
}

type UpdateEmployeeRequest struct {
	FullName    string  `json:"full_name"`
	Email       string  `json:"email"`
	PhoneNumber string  `json:"phone_number"`
	Position    string  `json:"position"`
	Department  string  `json:"department"`
	Salary      float64 `json:"salary"`
}

// EmployeeFilters represents the available filter options
type EmployeeFilters struct {
	Department  string  `query:"department"`
	Status      string  `query:"status"` // leave_with_permission, leave_without_permission, late, resign
	SalaryFrom  float64 `query:"salary_from"`
	SalaryTo    float64 `query:"salary_to"`
	OnboardFrom string  `query:"onboard_from"` // Format: YYYY-MM-DD
	OnboardTo   string  `query:"onboard_to"`   // Format: YYYY-MM-DD
}

func GetAllEmployees(c *fiber.Ctx) error {
	var filters EmployeeFilters
	if err := c.QueryParser(&filters); err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid filter parameters",
		})
	}

	query := DB.Model(&models.User{})

	// Apply department filter
	if filters.Department != "" {
		query = query.Where("department = ?", filters.Department)
	}

	// Apply salary range filter
	if filters.SalaryFrom > 0 {
		query = query.Where("salary >= ?", filters.SalaryFrom)
	}
	if filters.SalaryTo > 0 {
		query = query.Where("salary <= ?", filters.SalaryTo)
	}

	// Apply onboard date range filter
	if filters.OnboardFrom != "" {
		query = query.Where("DATE(onboard_date) >= ?", filters.OnboardFrom)
	}
	if filters.OnboardTo != "" {
		query = query.Where("DATE(onboard_date) <= ?", filters.OnboardTo)
	}

	// Apply status filter
	if filters.Status != "" {
		today := time.Now().Format("2006-01-02")
		switch filters.Status {
		case "leave_with_permission":
			query = query.Joins("JOIN absences ON users.id = absences.user_id").
				Where("absences.type = 'with_permission' AND absences.status = 'approved' AND ? BETWEEN DATE(absences.start_date) AND DATE(absences.end_date)", today)
		case "leave_without_permission":
			query = query.Joins("JOIN absences ON users.id = absences.user_id").
				Where("absences.type = 'without_permission' AND ? BETWEEN DATE(absences.start_date) AND DATE(absences.end_date)", today)
		case "late":
			query = query.Joins("JOIN attendances ON users.id = attendances.user_id").
				Where("DATE(attendances.check_in_time) = ? AND attendances.is_late = true", today)
		case "resign":
			query = query.Where("users.status = 'left_company'")
		}
	}

	var employees []models.User
	if err := query.Find(&employees).Error; err != nil {
		utils.Logger.Error("Failed to fetch employees", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Data:    employees,
	})
}

func AddEmployee(c *fiber.Ctx) error {
	var req AddEmployeeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrInvalidInput,
		})
	}

	// Create new employee
	employee := models.User{
		ID:            uuid.New().String(),
		FullName:      req.FullName,
		Email:         req.Email,
		PhoneNumber:   req.PhoneNumber,
		Address:       req.Address,
		DateOfBirth:   req.DateOfBirth,
		Gender:        req.Gender,
		TaxID:         req.TaxID,
		Position:      req.Position,
		Location:      req.Location,
		Department:    req.Department,
		WalletAddress: req.WalletAddress,
		Salary:        req.Salary,
		Role:          req.Role,
		Status:        "active",
		LeaveBalance:  20, // Default leave balance
		OnboardDate:   time.Now(),
	}

	// Start transaction
	tx := DB.Begin()

	if err := tx.Create(&employee).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to create employee", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	tx.Commit()

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Employee added successfully",
		Data:    employee,
	})
}

func GetSalaryInfo(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: false,
		Error:   "Not implemented",
	})
}

func UpdateEmployee(c *fiber.Ctx) error {
	// Get claims from context
	claims := c.Locals("claims").(jwt.MapClaims)
	userRole := claims["role"].(string)

	var req UpdateEmployeeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Only root can update salary
	if req.Salary != 0 && userRole != "root" {
		return c.Status(403).JSON(types.APIResponse{
			Success: false,
			Error:   "Only root can update salary",
		})
	}

	id := c.Params("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
	}

	var updateData map[string]interface{}
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrInvalidInput,
		})
	}

	// Start transaction
	tx := DB.Begin()

	// Find employee first
	var employee models.User
	if err := tx.First(&employee, "id = ?", userID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(types.APIResponse{
				Success: false,
				Error:   "Employee not found",
			})
		}
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	// Update employee
	if err := tx.Model(&employee).Updates(updateData).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to update employee", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	tx.Commit()

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Employee updated successfully",
		Data:    employee,
	})
}

func DeleteEmployee(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
	}

	var employee models.User
	if err := DB.First(&employee, "id = ?", userID).Error; err != nil {
		return c.Status(404).JSON(types.APIResponse{
			Success: false,
			Error:   "Employee not found",
		})
	}

	// Soft delete by updating status
	if err := DB.Model(&employee).Update("status", "left_company").Error; err != nil {
		utils.Logger.Error("Failed to delete employee", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Employee deleted successfully",
	})
}
