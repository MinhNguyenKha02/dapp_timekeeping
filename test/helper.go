package test

import (
	"dapp_timekeeping/config"
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/models"
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
	// Use file-based SQLite for tests with foreign key support only
	testDB, err = gorm.Open(sqlite.Open("company.db?_foreign_keys=on"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		log.Fatal("Failed to connect to test database:", err)
	}

	// Initialize handlers with test DB
	handlers.InitHandlers(testDB)

	// Create new Fiber app for each test
	testApp = fiber.New()
}

func SetupTest(t *testing.T) (*fiber.App, *gorm.DB) {
	// Drop existing tables first
	testDB.Migrator().DropTable(
		&models.UserPermission{},
		&models.Absence{},
		&models.Attendance{},
		&models.PermissionGrant{},
		&models.SalaryApproval{},
		&models.Department{},
		&models.Permission{},
		&models.User{},
	)

	// Then create tables in correct order
	err := testDB.AutoMigrate(
		&models.User{},       // Users first
		&models.Permission{}, // Independent tables
		&models.Department{},
		&models.Absence{}, // Tables with foreign keys
		&models.Attendance{},
		&models.UserPermission{},
		&models.PermissionGrant{},
		&models.SalaryApproval{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Print schema for debugging
	schema, _ := testDB.Migrator().GetTables()
	t.Logf("Database schema: %v", schema)

	// Reset database and clear cache
	ResetTestDB()
	testDB.Exec("PRAGMA foreign_keys = ON")
	testDB.Exec("VACUUM") // Clear SQLite cache

	// Create fresh app instance
	testApp = fiber.New()
	handlers.InitHandlers(testDB)

	return testApp, testDB
}

func ResetTestDB() {
	// Clear all data
	testDB.Exec("DELETE FROM attendances")
	testDB.Exec("DELETE FROM users")

	// Only reset sequence if table exists
	var count int64
	testDB.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='sqlite_sequence'").Count(&count)
	if count > 0 {
		testDB.Exec("UPDATE sqlite_sequence SET seq = 0")
	}
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
