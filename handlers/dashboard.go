package handlers

import (
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"

	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Dashboard & Overview
type DashboardStats struct {
	TotalEmployees     int64            `json:"total_employees"`
	LeaveRequests      int64            `json:"leave_requests"`
	ApprovedLeaves     int64            `json:"approved_leaves"`
	AverageCheckInTime string           `json:"average_check_in_time"`
	AverageWorkHours   float64          `json:"average_work_hours"`
	DepartmentStats    []DepartmentStat `json:"department_stats"`
	TopWorkers         []WorkerRanking  `json:"top_workers"`
}

type DepartmentStat struct {
	Name               string  `json:"name"`
	EmployeeCount      int     `json:"employee_count"`
	AverageCheckInTime string  `json:"average_check_in_time"`
	AverageWorkHours   float64 `json:"average_work_hours"`
}

type WorkerRanking struct {
	UserID     uuid.UUID `json:"user_id"`
	Username   string    `json:"username"`
	Department string    `json:"department"`
	TotalHours float64   `json:"total_hours"`
	Rank       int       `json:"rank"`
}

func GetRootDashboard(c *fiber.Ctx) error {
	// Get date range for statistics
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var stats DashboardStats

	// Get total employees
	DB.Model(&models.User{}).Where("status = ?", "active").Count(&stats.TotalEmployees)

	// Get leave requests
	DB.Model(&models.LeaveRequest{}).
		Where("start_date >= ?", startOfMonth).
		Count(&stats.LeaveRequests)

	// Get approved leaves
	DB.Model(&models.LeaveRequest{}).
		Where("start_date >= ? AND status = ?", startOfMonth, "approved").
		Count(&stats.ApprovedLeaves)

	// Calculate average check-in time
	var avgCheckIn time.Time
	DB.Model(&models.Attendance{}).
		Where("check_in_time >= ?", startOfMonth).
		Select("AVG(check_in_time)").
		Scan(&avgCheckIn)
	stats.AverageCheckInTime = avgCheckIn.Format("15:04")

	// Get department statistics
	DB.Raw(`
		SELECT 
			d.name,
			COUNT(DISTINCT u.id) as employee_count,
			TIME_FORMAT(AVG(a.check_in_time), '%H:%i') as average_check_in_time,
			AVG(TIMESTAMPDIFF(HOUR, a.check_in_time, a.check_out_time)) as average_work_hours
		FROM departments d
		LEFT JOIN users u ON u.department = d.name
		LEFT JOIN attendances a ON a.user_id = u.id
		WHERE a.check_in_time >= ?
		GROUP BY d.name
	`, startOfMonth).Scan(&stats.DepartmentStats)

	// Get top workers
	DB.Raw(`
		SELECT 
			u.id as user_id,
			u.username,
			u.department,
			SUM(TIMESTAMPDIFF(HOUR, a.check_in_time, a.check_out_time)) as total_hours,
			RANK() OVER (ORDER BY SUM(TIMESTAMPDIFF(HOUR, a.check_in_time, a.check_out_time)) DESC) as rank
		FROM users u
		JOIN attendances a ON a.user_id = u.id
		WHERE a.check_in_time >= ?
		GROUP BY u.id
		ORDER BY total_hours DESC
		LIMIT 10
	`, startOfMonth).Scan(&stats.TopWorkers)

	return c.JSON(types.APIResponse{
		Success: true,
		Data:    stats,
	})
}
