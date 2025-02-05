package test

import (
	"dapp_timekeeping/config"
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/utils"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	testApp *fiber.App
	testDB  *gorm.DB
)

func init() {
	// Find and load .env file from project root
	projectRoot, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, ".env")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			log.Fatal("Could not find .env file")
		}
		projectRoot = parent
	}
	os.Chdir(projectRoot)

	config.LoadConfig()
	utils.InitLogger()

	var err error
	// Use in-memory SQLite for tests
	testDB, err = gorm.Open(sqlite.Open("company.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to test database:", err)
	}

	// Initialize handlers with test DB
	handlers.InitHandlers(testDB)

	// Create new Fiber app for each test
	testApp = fiber.New()
}

func SetupTest(t *testing.T) (*fiber.App, *gorm.DB) {
	// Reset database
	ResetTestDB()

	// Create fresh app instance
	testApp = fiber.New()
	handlers.InitHandlers(testDB)

	return testApp, testDB
}

func ResetTestDB() {
	testDB.Exec("DELETE FROM absences")
	testDB.Exec("DELETE FROM users")
	testDB.Exec("DELETE FROM attendances")
	testDB.Exec("DELETE FROM leave_requests")
	testDB.Exec("DELETE FROM company_rules")
}

func GetTestDB() *gorm.DB {
	return testDB
}

func GetTestApp() *fiber.App {
	return testApp
}

// Helper function to create test JWT token
func createTestToken(userID string, role string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		log.Printf("Error creating test token: %v", err)
		return ""
	}
	return tokenString
}
