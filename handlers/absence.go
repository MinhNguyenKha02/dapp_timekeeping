package handlers

import (
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"dapp_timekeeping/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AbsenceResponse struct {
	ID          string     `json:"id"`
	FullName    string     `json:"full_name"`
	Date        time.Time  `json:"date"`
	Type        string     `json:"type"` // with_permission, without_permission
	Reason      string     `json:"reason"`
	Status      string     `json:"status"` // pending, processed
	Department  string     `json:"department"`
	ProcessedBy *string    `json:"processed_by,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// Statistics response structure
type EmployeeStatistics struct {
	LeaveStats struct {
		WithPermission    int `json:"with_permission"`
		WithoutPermission int `json:"without_permission"`
		PendingLeaves     int `json:"pending_leaves"`
	} `json:"leave_stats"`
	ResignStats struct {
		Approved int `json:"approved"`
		Pending  int `json:"pending"`
		Total    int `json:"total"`
	} `json:"resign_stats"`
	LateStats struct {
		TotalIncidents  int     `json:"total_incidents"`
		UniqueEmployees int     `json:"unique_employees"`
		AverageMinutes  float64 `json:"average_minutes"`
	} `json:"late_stats"`
}

func GetAbsences(c *fiber.Ctx) error {
	// Get query parameters
	absenceType := c.Query("type")      // with_permission, without_permission, or empty for all
	status := c.Query("status")         // pending, processed, or empty for all
	department := c.Query("department") // department filter
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Parse dates if provided
	var start, end time.Time
	var err error
	if startDate != "" {
		start, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			return c.Status(400).JSON(types.APIResponse{
				Success: false,
				Error:   "Invalid start date format. Use YYYY-MM-DD",
			})
		}
	}
	if endDate != "" {
		end, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return c.Status(400).JSON(types.APIResponse{
				Success: false,
				Error:   "Invalid end date format. Use YYYY-MM-DD",
			})
		}
	}

	// Build the query
	query := DB.Table("absences").
		Select("absences.*, users.full_name, users.department, processors.full_name as processor_name").
		Joins("LEFT JOIN users ON users.id = absences.user_id").
		Joins("LEFT JOIN users processors ON processors.id = absences.processed_by")

	// Apply filters
	if absenceType != "" {
		query = query.Where("absences.type = ?", absenceType)
	}
	if status != "" {
		query = query.Where("absences.status = ?", status)
	}
	if department != "" {
		query = query.Where("users.department = ?", department)
	}
	if !start.IsZero() {
		query = query.Where("absences.date >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("absences.date <= ?", end)
	}

	// Execute query
	var absences []struct {
		ID            string
		FullName      string
		Date          time.Time
		Type          string
		Reason        string
		Status        string
		Department    string
		ProcessorName *string
		ProcessedAt   *time.Time
	}

	if err := query.Find(&absences).Error; err != nil {
		utils.Logger.Error("Failed to fetch absences", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	// Transform to response format
	response := make([]AbsenceResponse, len(absences))
	for i, abs := range absences {
		response[i] = AbsenceResponse{
			ID:          abs.ID,
			FullName:    abs.FullName,
			Date:        abs.Date,
			Type:        abs.Type,
			Reason:      abs.Reason,
			Status:      abs.Status,
			Department:  abs.Department,
			ProcessedBy: abs.ProcessorName,
			ProcessedAt: abs.ProcessedAt,
		}
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetEmployeeStatistics returns statistics for root user
func GetEmployeeStatistics(c *fiber.Ctx) error {
	var stats EmployeeStatistics

	// Get leave statistics from absence table
	DB.Model(&models.Absence{}).
		Select(`
			COUNT(CASE WHEN type = 'with_permission' AND status = 'approved' THEN 1 END) as with_permission,
			COUNT(CASE WHEN type = 'without_permission' THEN 1 END) as without_permission,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_leaves
		`).
		Scan(&stats.LeaveStats)

	// Get resign statistics from absence table
	DB.Table("absences").
		Select(`
			COUNT(CASE WHEN status = 'approved' THEN 1 END) as approved,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(*) as total
		`).
		Where("type = 'resign'").
		Scan(&stats.ResignStats)

	// Get late statistics from attendance table
	DB.Table("attendances").
		Select(`
			COUNT(*) as total_incidents,
			COUNT(DISTINCT user_id) as unique_employees,
			AVG((julianday(check_in_time) - julianday(expected_time)) * 24 * 60) as average_minutes
		`).
		Where("check_in_time > expected_time").
		Scan(&stats.LateStats)

	return c.JSON(types.APIResponse{
		Success: true,
		Data:    stats,
	})
}
