package main

import (
	"dapp_timekeeping/models"
	"dapp_timekeeping/services"
	"log"
	"os"

	"github.com/dfinity/agent-go/identity"
	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Database connection
var DB *gorm.DB
var blockchainService *services.BlockchainService

func initServices() error {
	// Initialize SQLite
	var err error
	DB, err = gorm.Open(sqlite.Open("company.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	// Auto-migrate models
	DB.AutoMigrate(&models.User{}, &models.Attendance{}, &models.LeaveRequest{}, &models.Violation{}, &models.Report{})

	// Initialize ICP connection
	pemFile := os.Getenv("ICP_IDENTITY_PEM")
	identity, err := identity.NewIdentityFromPEM(pemFile)
	if err != nil {
		return err
	}

	canisterID := os.Getenv("COMPANY_REGISTRY_CANISTER_ID")
	blockchainService, err = services.NewBlockchainService(identity, canisterID)
	if err != nil {
		return err
	}

	return nil
}

// Example of a handler that uses both SQLite and ICP
func addEmployee(c *fiber.Ctx) error {
	var employee struct {
		Username      string  `json:"username"`
		WalletAddress string  `json:"wallet_address"`
		Salary        float64 `json:"salary"`
	}

	if err := c.BodyParser(&employee); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Start database transaction
	tx := DB.Begin()

	// Create user in SQLite
	user := models.User{
		Username:      employee.Username,
		WalletAddress: employee.WalletAddress,
		Salary:        employee.Salary,
		Status:        "active",
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}

	// Add to blockchain
	ctx := c.Context()
	err := blockchainService.AddEmployee(ctx, employee.WalletAddress, uint64(employee.Salary))
	if err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": "Blockchain error"})
	}

	// Commit transaction
	tx.Commit()

	return c.JSON(fiber.Map{
		"message": "Employee added successfully",
		"user":    user,
	})
}

func main() {
	if err := initServices(); err != nil {
		log.Fatal("Failed to initialize services:", err)
	}

	app := fiber.New()

	// Routes
	app.Post("/employees", addEmployee)
	// Add more routes...

	log.Fatal(app.Listen(":3000"))
}
