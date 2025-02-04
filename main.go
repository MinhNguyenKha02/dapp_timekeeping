package main

import (
	"dapp_timekeeping/config"
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/middleware"
	"dapp_timekeeping/models"
	"dapp_timekeeping/services"
	"log"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Database connection
var DB *gorm.DB
var blockchainService *services.BlockchainService

func initServices() error {
	var err error
	DB, err = gorm.Open(sqlite.Open(config.AppConfig.DBPath), &gorm.Config{})
	if err != nil {
		return err
	}

	// Auto-migrate models
	DB.AutoMigrate(&models.User{}, &models.Attendance{}, &models.LeaveRequest{}, &models.Violation{}, &models.Report{}, &models.CompanyRule{})

	// Initialize ICP connection
	blockchainService = services.NewBlockchainService(config.AppConfig.CanisterID)

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

	// Employee Management
	root.Get("/dashboard", handlers.GetRootDashboard)
	root.Get("/employees", handlers.GetAllEmployees)
	root.Post("/employees", handlers.AddEmployee)
	root.Patch("/employees/:id", handlers.UpdateEmployee)
	root.Delete("/employees/:id", handlers.DeleteEmployee)

	// Department Statistics
	root.Get("/departments/stats", handlers.GetDepartmentStats)
	root.Get("/departments/:id/attendance", handlers.GetDepartmentAttendance)

	// Working Hours Rankings
	root.Get("/rankings/work-hours", handlers.GetWorkHoursRanking)

	// Permission Management
	root.Post("/permissions/grant", handlers.GrantPermission)
	root.Delete("/permissions/revoke", handlers.RevokePermission)

	// Referral Management
	root.Post("/referrals", handlers.GenerateReferralCode)
	root.Get("/referrals", handlers.ListReferralCodes)

	// Salary Management
	root.Patch("/employees/:id/salary", handlers.UpdateSalary)
	root.Get("/salary-approvals", handlers.GetPendingSalaryApprovals)
	root.Post("/salary-approvals/:id/approve", handlers.ApproveSalary)

	// Reports
	root.Get("/reports/attendance", handlers.GenerateAttendanceReport)
	root.Get("/reports/salary", handlers.GenerateSalaryReport)
	root.Get("/reports/leave", handlers.GenerateLeaveReport)
}

func main() {
	// Load configuration
	config.LoadConfig()

	if err := initServices(); err != nil {
		log.Fatal("Failed to initialize services:", err)
	}

	// Initialize handlers with dependencies
	handlers.InitHandlers(DB, blockchainService)

	app := fiber.New()
	setupRoutes(app)
	setupRootRoutes(app)
	log.Fatal(app.Listen(":" + config.AppConfig.Port))
}
