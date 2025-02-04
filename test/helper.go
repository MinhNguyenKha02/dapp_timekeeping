package test

import (
	"context"
	"dapp_timekeeping/config"
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/models"
	"dapp_timekeeping/services"
	"dapp_timekeeping/utils"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	testApp *fiber.App
	testDB  *gorm.DB
)

// Ensure MockBlockchainService implements the interface
var _ services.BlockchainServiceInterface = (*MockBlockchainService)(nil)

type BlockchainCall struct {
	Method        string
	WalletAddress string
	Salary        uint64
	Timestamp     time.Time
}

type MockBlockchainService struct {
	t     *testing.T
	calls []BlockchainCall
}

// Create a constructor for MockBlockchainService
func NewMockBlockchainService(t *testing.T) *MockBlockchainService {
	return &MockBlockchainService{t: t}
}

func (m *MockBlockchainService) UpdateEmployeeSalary(ctx context.Context, walletAddress string, newSalary uint64) error {
	m.t.Logf("MockBlockchain: UpdateEmployeeSalary called with wallet=%s, newSalary=%d", walletAddress, newSalary)
	return nil
}

func (m *MockBlockchainService) GetEmployeeSalary(ctx context.Context, walletAddress string) (uint64, error) {
	m.t.Logf("MockBlockchain: GetEmployeeSalary called with wallet=%s", walletAddress)
	return 0, nil
}

func (m *MockBlockchainService) PaySalary(ctx context.Context, employee string, amount, deductions, bonus uint64) error {
	m.t.Logf("MockBlockchain: PaySalary called with employee=%s, amount=%d, deductions=%d, bonus=%d",
		employee, amount, deductions, bonus)
	return nil
}

func (m *MockBlockchainService) UpdateCompanyRule(ctx context.Context, ruleID, details string) error {
	m.t.Logf("MockBlockchain: UpdateCompanyRule called with ruleID=%s", ruleID)
	return nil
}

func (m *MockBlockchainService) AddEmployeeSalary(ctx context.Context, walletAddress string, salary uint64) error {
	m.calls = append(m.calls, BlockchainCall{
		Method:        "AddEmployeeSalary",
		WalletAddress: walletAddress,
		Salary:        salary,
		Timestamp:     time.Now(),
	})
	m.t.Logf("MockBlockchain: AddEmployeeSalary called with wallet=%s, salary=%d", walletAddress, salary)
	return nil
}

var mockBlockchain *MockBlockchainService

func GetMockBlockchain(t *testing.T) *MockBlockchainService {
	if mockBlockchain == nil {
		mockBlockchain = NewMockBlockchainService(t)
	}
	return mockBlockchain
}

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

	// Load config with .env file
	config.LoadConfig()

	// Initialize logger
	utils.InitLogger()

	var err error
	// Use a real file for test database
	testDB, err = gorm.Open(sqlite.Open("company.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to test database:", err)
	}

	// Clean up existing tables
	testDB.Migrator().DropTable(
		&models.User{},
		&models.Attendance{},
		&models.LeaveRequest{},
		&models.Department{},
		&models.Permission{},
		&models.PermissionGrant{},
		&models.SalaryApproval{},
		&models.CompanyRule{},
	)

	// Auto migrate models
	testDB.AutoMigrate(
		&models.User{},
		&models.Attendance{},
		&models.LeaveRequest{},
		&models.Department{},
		&models.Permission{},
		&models.PermissionGrant{},
		&models.SalaryApproval{},
		&models.CompanyRule{},
	)

	// Use mock blockchain instead of real one
	mockBlockchain = NewMockBlockchainService(nil)
	handlers.InitHandlers(testDB, mockBlockchain)
	testApp = fiber.New()
}

func GetTestDB() *gorm.DB {
	return testDB
}

func GetTestApp() *fiber.App {
	return testApp
}

// Helper function to create test JWT token
func createTestToken(userID string, role string) string {
	// Implementation here
	return "test-token"
}

// Add this function
func ResetTestDB() {
	// Clean up existing tables
	testDB.Migrator().DropTable(
		&models.User{},
		&models.Attendance{},
		&models.LeaveRequest{},
		&models.Department{},
		&models.Permission{},
		&models.PermissionGrant{},
		&models.SalaryApproval{},
		&models.CompanyRule{},
	)

	// Auto migrate models
	testDB.AutoMigrate(
		&models.User{},
		&models.Attendance{},
		&models.LeaveRequest{},
		&models.Department{},
		&models.Permission{},
		&models.PermissionGrant{},
		&models.SalaryApproval{},
		&models.CompanyRule{},
	)
}

// Update the test setup to use the mock with logging
func SetupTest(t *testing.T) (*fiber.App, *gorm.DB) {
	ResetTestDB()
	mockBlockchain = NewMockBlockchainService(t)
	handlers.InitHandlers(testDB, mockBlockchain)
	return testApp, testDB
}

// Helper functions to verify blockchain calls
func GetMockBlockchainCalls(t *testing.T, method string) int {
	mock := GetMockBlockchain(t)
	count := 0
	for _, call := range mock.calls {
		if call.Method == method {
			count++
		}
	}
	return count
}

func GetLastBlockchainCall(t *testing.T, method string) *BlockchainCall {
	mock := GetMockBlockchain(t)
	for i := len(mock.calls) - 1; i >= 0; i-- {
		if mock.calls[i].Method == method {
			return &mock.calls[i]
		}
	}
	return nil
}
