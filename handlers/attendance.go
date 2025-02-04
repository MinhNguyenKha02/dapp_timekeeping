package handlers

import (
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"dapp_timekeeping/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CheckInRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

func CheckIn(c *fiber.Ctx) error {
	var req CheckInRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrInvalidInput,
		})
	}

	// Get current time
	now := time.Now()

	// Check if already checked in today
	var existingAttendance models.Attendance
	if err := DB.Where("user_id = ? AND DATE(check_in_time) = DATE(?)",
		req.UserID, now).First(&existingAttendance).Error; err == nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "Already checked in today",
		})
	}

	// Get company rules for check-in time
	var rules models.CompanyRule
	if err := DB.Where("rule_name = ?", "check_in_time").First(&rules).Error; err != nil {
		utils.Logger.Error("Failed to get company rules", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   "Failed to validate check-in time",
		})
	}

	// Create attendance record
	attendance := models.Attendance{
		UserID:      req.UserID,
		CheckInTime: now,
		Status:      "on_time",
	}

	// Check for late arrival
	checkInLimit, err := time.Parse("15:04", rules.Details)
	if err != nil {
		utils.Logger.Error("Failed to parse check-in time", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid check-in time rule",
		})
	}
	if now.After(checkInLimit) {
		attendance.Status = "late"
		attendance.ViolationType = "late_arrival"

		// Create violation record
		violation := models.Violation{
			UserID:          req.UserID,
			Type:            "late_arrival",
			Date:            now,
			Details:         "Late check-in",
			DeductionAmount: calculateLateDeduction(now.Sub(checkInLimit)),
		}

		if err := DB.Create(&violation).Error; err != nil {
			utils.Logger.Error("Failed to create violation record", zap.Error(err))
		}
	}

	// Save attendance record
	if err := DB.Create(&attendance).Error; err != nil {
		utils.Logger.Error("Failed to create attendance record", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Check-in recorded successfully",
		Data:    attendance,
	})
}

func CheckOut(c *fiber.Ctx) error {
	var req CheckInRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrInvalidInput,
		})
	}

	// Get today's attendance record
	var attendance models.Attendance
	if err := DB.Where("user_id = ? AND DATE(check_in_time) = DATE(?)",
		req.UserID, time.Now()).First(&attendance).Error; err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "No check-in record found for today",
		})
	}

	// Update check-out time
	now := time.Now()
	attendance.CheckOutTime = now

	// Check for early leave
	var rules models.CompanyRule
	if err := DB.Where("rule_name = ?", "check_out_time").First(&rules).Error; err == nil {
		checkOutTime, err := time.Parse("15:04", rules.Details)
		if err != nil {
			utils.Logger.Error("Failed to parse check-out time", zap.Error(err))
			return c.Status(500).JSON(types.APIResponse{
				Success: false,
				Error:   "Invalid check-out time rule",
			})
		}
		if now.Before(checkOutTime) {
			attendance.ViolationType = "early_leave"

			// Create violation record
			violation := models.Violation{
				UserID:          req.UserID,
				Type:            "early_leave",
				Date:            now,
				Details:         "Early check-out",
				DeductionAmount: calculateEarlyLeaveDeduction(checkOutTime.Sub(now)),
			}

			if err := DB.Create(&violation).Error; err != nil {
				utils.Logger.Error("Failed to create violation record", zap.Error(err))
			}
		}
	}

	// Save updated attendance record
	if err := DB.Save(&attendance).Error; err != nil {
		utils.Logger.Error("Failed to update attendance record", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Check-out recorded successfully",
		Data:    attendance,
	})
}

// Helper functions for calculating deductions
func calculateLateDeduction(lateDuration time.Duration) float64 {
	// Example: 5% deduction per hour late
	hours := lateDuration.Hours()
	return hours * 0.05
}

func calculateEarlyLeaveDeduction(earlyDuration time.Duration) float64 {
	// Example: 5% deduction per hour early
	hours := earlyDuration.Hours()
	return hours * 0.05
}

// Add the CompanyRule model to models/models.go
type CompanyRule struct {
	ID        uint      `gorm:"primaryKey"`
	RuleName  string    `json:"rule_name"`
	Details   string    `json:"details"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func RecordAttendance(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Not implemented",
	})
}

func RequestLeave(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Not implemented",
	})
}

func GetLeaveRequests(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Not implemented",
	})
}
