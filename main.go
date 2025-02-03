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
	DB.AutoMigrate(&models.User{}, &models.Attendance{}, &models.LeaveRequest{}, &models.Violation{}, &models.Report{})

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
	log.Fatal(app.Listen(":" + config.AppConfig.Port))
}
