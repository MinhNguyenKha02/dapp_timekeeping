package handlers

import (
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
