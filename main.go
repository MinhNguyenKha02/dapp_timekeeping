package main

import (
	"dapp_timekeeping/config"
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/middleware"
	"dapp_timekeeping/models"
	"log"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Database connection
var DB *gorm.DB

func initServices() error {
	var err error
	DB, err = gorm.Open(sqlite.Open(config.AppConfig.DBPath), &gorm.Config{})
	if err != nil {
		return err
	}

	// Auto-migrate models
	DB.AutoMigrate(
		&models.User{},
		&models.Permission{},
		&models.Department{},
		&models.SalaryApproval{},
		&models.Attendance{},
		&models.LeaveRequest{},
		&models.Violation{},
		&models.Report{},
		&models.CompanyRule{},
		&models.Absence{},
		&models.UserPermission{},
		&models.ReferralCode{},
		&models.PayrollApproval{},
	)

	return nil
}

func setupRoutes(app *fiber.App) {
	// Public routes
	app.Post("/login", handlers.Login)
	app.Post("/register", handlers.Register)
	app.Post("/logout", handlers.Logout)

	// Root only routes
	root := app.Group("/root", middleware.RequireRoot)
	root.Post("/employees", handlers.AddEmployee)
	root.Post("/rules", handlers.UpdateCompanyRule)
	root.Get("/reports", handlers.GenerateReports)

	// HR routes
	hr := app.Group("/hr", middleware.RequireHR)
	hr.Post("/attendance", handlers.RecordAttendance)
	hr.Post("/violations", handlers.RecordViolation)
	hr.Get("/leave-requests", handlers.GetLeaveRequests)

	// Employee routes
	emp := app.Group("/employee", middleware.RequireAuth)
	emp.Post("/check-in", handlers.CheckIn)
	emp.Post("/check-out", handlers.CheckOut)
	emp.Post("/leave-request", handlers.RequestLeave)
	emp.Get("/salary", handlers.GetSalaryInfo)
}

func setupRootRoutes(app *fiber.App) {
	root := app.Group("/root", middleware.RequireRoot)

	// // Dashboard Statistics
	// root.Get("/dashboard", handlers.GetDashboardStats)
	// // Returns:
	// // - late_employees_count
	// // - absent_with_permission_count
	// // - absent_without_permission_count
	// // - unprocessed_absences_count
	// // - resigned_employees_count
	// // - average_check_in_time
	// // - average_check_out_time
	// // - department_stats
	// // - top_workers_ranking

	// Attendance Management
	attendance := root.Group("/attendance")
	// attendance.Get("/late", handlers.GetLateEmployees)
	attendance.Get("/absences", handlers.GetAbsences)
	// attendance.Get("/unprocessed", handlers.GetUnprocessedAbsences)
	// attendance.Post("/process/:id", handlers.ProcessAbsence)
	// attendance.Get("/department/:id", handlers.GetDepartmentAttendance)

	// // Leave Management
	// leaves := root.Group("/leaves")
	// leaves.Get("/pending", handlers.GetPendingLeaves)
	// leaves.Get("/approved", handlers.GetApprovedLeaves)
	// leaves.Post("/:id/approve", handlers.ApproveLeave)
	// leaves.Post("/:id/reject", handlers.RejectLeave)

	// Employee Management
	employees := root.Group("/employees")
	employees.Get("/", handlers.GetAllEmployees)
	employees.Post("/", handlers.AddEmployee)
	employees.Patch("/:id", handlers.UpdateEmployee)
	employees.Delete("/:id", handlers.DeleteEmployee)
	employees.Put("/:id/salary", handlers.UpdateSalary)

	// // Permission Management
	// permissions := root.Group("/permissions")
	// permissions.Post("/grant", handlers.GrantPermission)
	// permissions.Delete("/revoke", handlers.RevokePermission)
	// permissions.Get("/user/:id", handlers.GetUserPermissions)

	// // Referral Management
	// referrals := root.Group("/referrals")
	// referrals.Post("/generate", handlers.GenerateReferralCode)
	// referrals.Get("/", handlers.ListReferralCodes)
	// referrals.Delete("/:code", handlers.DeleteReferralCode)

	// // Payroll Management
	// payroll := root.Group("/payroll")
	// payroll.Get("/pending", handlers.GetPendingPayroll)
	// payroll.Post("/approve", handlers.ApprovePayroll)
	// payroll.Get("/history", handlers.GetPayrollHistory)

	// // Reports
	// reports := root.Group("/reports")
	// reports.Get("/attendance", handlers.ExportAttendanceReport)
	// reports.Get("/salary", handlers.ExportSalaryReport)
	// reports.Get("/leave", handlers.ExportLeaveReport)
	// reports.Get("/violations", handlers.ExportViolationsReport)
}

func main() {
	// Load configuration
	config.LoadConfig()

	if err := initServices(); err != nil {
		log.Fatal("Failed to initialize services:", err)
	}

	app := fiber.New()
	// setupRoutes(app)
	setupRootRoutes(app)
	log.Fatal(app.Listen(":" + config.AppConfig.Port))
}
