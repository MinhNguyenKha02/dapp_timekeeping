package test

import (
	"bytes"
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func dumpUsers(t *testing.T, label string) {
	var users []models.User
	if err := GetTestDB().Find(&users).Error; err != nil {
		t.Logf("%s - Error getting users: %v", label, err)
		return
	}
	t.Logf("%s - Found %d users:", label, len(users))
	for _, u := range users {
		t.Logf("  User: ID=%s, FullName=%s, Status=%s", u.ID, u.FullName, u.Status)
	}
}

func TestGetAllEmployees(t *testing.T) {
	app := GetTestApp()
	app.Get("/employees", handlers.GetAllEmployees)

	// Test successful fetch
	req := httptest.NewRequest("GET", "/employees", nil)
	resp, err := app.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.Nil(t, err)
	assert.True(t, response.Success)
}

func TestAddEmployee(t *testing.T) {
	app, db := SetupTest(t)
	app.Post("/employees", handlers.AddEmployee)

	// Log initial state
	t.Log("Initial state:")
	dumpUsers(t, "Before adding employee")

	// Test successful creation with HR role
	employee := handlers.AddEmployeeRequest{
		FullName:      "Test Employee",
		Email:         "test@company.com",
		PhoneNumber:   "+1234567890",
		Address:       "123 Test St",
		DateOfBirth:   time.Now().AddDate(-25, 0, 0),
		Gender:        "male",
		TaxID:         "TAX123",
		Position:      "HR Staff",
		Location:      "HQ",
		Department:    "HR",
		WalletAddress: "0x123...",
		Salary:        5000,
		Role:          "hr",
	}

	body, _ := json.Marshal(employee)
	req := httptest.NewRequest("POST", "/employees", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Log response
	var response types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.Nil(t, err)
	t.Logf("Response: %+v", response)

	// Verify database storage and log result
	var storedEmployee models.User
	err = db.Where("email = ?", employee.Email).First(&storedEmployee).Error
	assert.Nil(t, err)
	t.Logf("Stored employee: %+v", storedEmployee)

	// Log final state
	dumpUsers(t, "After adding employee")

	// Verify specific fields
	assert.Equal(t, employee.FullName, storedEmployee.FullName)
	assert.Equal(t, employee.Email, storedEmployee.Email)
	assert.Equal(t, employee.Department, storedEmployee.Department)
	assert.Equal(t, "hr", storedEmployee.Role)
	assert.Equal(t, "active", storedEmployee.Status)
}

func TestUpdateEmployee(t *testing.T) {
	app, db := SetupTest(t)

	// Set up route with JWT middleware
	app.Patch("/employees/:id", func(c *fiber.Ctx) error {
		// Set claims for the request
		c.Locals("claims", jwt.MapClaims{
			"role": c.Get("X-Test-Role", ""), // Will be set in test requests
			"id":   c.Get("X-Test-ID", ""),
		})
		return handlers.UpdateEmployee(c)
	})

	// Create test employee
	employee := models.User{
		ID:                uuid.New().String(),
		FullName:          "Update Test",
		Email:             "update@company.com",
		PhoneNumber:       "+1234567890",
		Address:           "456 Test St",
		DateOfBirth:       time.Now().AddDate(-30, 0, 0),
		Gender:            "female",
		TaxID:             "TAX456",
		HealthInsuranceID: "HI456",
		SocialInsuranceID: "SI456",
		Position:          "Developer",
		Location:          "Branch A",
		Department:        "IT",
		WalletAddress:     "0x456...",
		Salary:            6000,
		Role:              "employee",
		Status:            "active",
		OnboardDate:       time.Now(),
		LeaveBalance:      20,
	}

	result := db.Create(&employee)
	assert.Nil(t, result.Error)

	// Test update non-salary fields (should succeed)
	updateData := map[string]interface{}{
		"position":   "Senior Developer",
		"location":   "Branch B",
		"department": "Engineering",
	}

	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Role", "employee")
	req.Header.Set("X-Test-ID", employee.ID)

	resp, err := app.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify non-salary updates
	var updatedEmployee models.User
	err = db.First(&updatedEmployee, "id = ?", employee.ID).Error
	assert.Nil(t, err)
	assert.Equal(t, "Senior Developer", updatedEmployee.Position)
	assert.Equal(t, "Branch B", updatedEmployee.Location)
	assert.Equal(t, "Engineering", updatedEmployee.Department)
	assert.Equal(t, float64(6000), updatedEmployee.Salary) // Salary should remain unchanged

	// Test salary update as non-root (should fail)
	updateData = map[string]interface{}{
		"salary": 7000,
	}
	body, _ = json.Marshal(updateData)
	req = httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Role", "employee")
	req.Header.Set("X-Test-ID", employee.ID)

	resp, err = app.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, 403, resp.StatusCode) // Should be forbidden

	// Test salary update as root (should succeed)
	req = httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Role", "root")
	req.Header.Set("X-Test-ID", uuid.New().String()) // Root user ID

	resp, err = app.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify salary was updated by root
	err = db.First(&updatedEmployee, "id = ?", employee.ID).Error
	assert.Nil(t, err)
	assert.Equal(t, float64(7000), updatedEmployee.Salary)
	t.Log("Root user successfully updated salary")
}

func TestDeleteEmployee(t *testing.T) {
	app, db := SetupTest(t)
	app.Delete("/employees/:id", handlers.DeleteEmployee)

	// Create test employee
	employee := models.User{
		ID:                uuid.New().String(),
		FullName:          "Delete Test",
		Email:             "delete@company.com",
		PhoneNumber:       "+1234567890",
		Address:           "789 Test St",
		DateOfBirth:       time.Now().AddDate(-28, 0, 0),
		Gender:            "male",
		TaxID:             "TAX789",
		HealthInsuranceID: "HI789",
		SocialInsuranceID: "SI789",
		Position:          "Analyst",
		Location:          "Branch C",
		Department:        "Finance",
		WalletAddress:     "0x789...",
		Salary:            5500,
		Role:              "employee",
		Status:            "active",
		OnboardDate:       time.Now(),
		LeaveBalance:      20,
	}

	result := db.Create(&employee)
	assert.Nil(t, result.Error)

	// Test deletion
	req := httptest.NewRequest("DELETE", "/employees/"+employee.ID, nil)
	resp, err := app.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify soft delete
	var deletedEmployee models.User
	err = db.First(&deletedEmployee, "id = ?", employee.ID).Error
	assert.Nil(t, err)
	assert.Equal(t, "left_company", deletedEmployee.Status)
}

func TestRootUpdateEmployeeSalary(t *testing.T) {
	app, db := SetupTest(t)

	// Set up route with JWT middleware
	app.Patch("/employees/:id", func(c *fiber.Ctx) error {
		// Set claims for the request
		c.Locals("claims", jwt.MapClaims{
			"role": c.Get("X-Test-Role", ""), // Will be set in test requests
			"id":   c.Get("X-Test-ID", ""),
		})
		return handlers.UpdateEmployee(c)
	})

	// Create root user with complete details
	rootUser := models.User{
		ID:                uuid.New().String(),
		FullName:          "Root Admin",
		Email:             "root@company.com",
		PhoneNumber:       "+1234567890",
		Address:           "123 Admin St",
		DateOfBirth:       time.Now().AddDate(-35, 0, 0),
		Gender:            "male",
		TaxID:             "ROOT123",
		HealthInsuranceID: "HI-ROOT123",
		SocialInsuranceID: "SI-ROOT123",
		Position:          "System Admin",
		Location:          "HQ",
		Department:        "IT",
		WalletAddress:     "0xroot...",
		Role:              "root",
		Status:            "active",
		OnboardDate:       time.Now().AddDate(-2, 0, 0),
	}
	db.Create(&rootUser)
	t.Logf("Created root user: %+v", rootUser)

	// Create test employee with complete details
	employee := models.User{
		ID:                uuid.New().String(),
		FullName:          "Test Employee",
		Email:             "employee@company.com",
		PhoneNumber:       "+9876543210",
		Address:           "456 Staff St",
		DateOfBirth:       time.Now().AddDate(-25, 0, 0),
		Gender:            "female",
		TaxID:             "EMP456",
		HealthInsuranceID: "HI-EMP456",
		SocialInsuranceID: "SI-EMP456",
		Position:          "Staff",
		Location:          "Branch A",
		Department:        "Operations",
		WalletAddress:     "0xemp...",
		Salary:            5000,
		Role:              "employee",
		Status:            "active",
		OnboardDate:       time.Now().AddDate(0, -6, 0),
		LeaveBalance:      20,
	}
	db.Create(&employee)
	t.Logf("Initial employee details: %+v", employee)

	// Root updates employee salary
	rootToken := createTestToken(rootUser.ID, "root")
	updateReq := handlers.UpdateEmployeeRequest{
		Salary: 6000,
	}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+rootToken)
	req.Header.Set("X-Test-Role", "root") // Set test role
	req.Header.Set("X-Test-ID", rootUser.ID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify salary was updated
	var updatedEmployee models.User
	err = db.First(&updatedEmployee, "id = ?", employee.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, float64(6000), updatedEmployee.Salary)
	t.Logf("Updated employee salary: %v", updatedEmployee.Salary)

	// Try updating salary with non-root user (should fail)
	employeeToken := createTestToken(employee.ID, "employee")
	req = httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+employeeToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
	t.Log("Non-root user cannot update salary")

	// Cleanup
	db.Unscoped().Delete(&employee)
	db.Unscoped().Delete(&rootUser)
}

func TestGetEmployeesWithFilters(t *testing.T) {
	app, db := SetupTest(t)
	app.Get("/employees", handlers.GetAllEmployees)

	// Create test employees with complete details
	employees := []models.User{
		{
			ID:                uuid.New().String(),
			FullName:          "IT Employee",
			Email:             "it@company.com",
			PhoneNumber:       "+1234567890",
			Address:           "123 IT St",
			DateOfBirth:       time.Now().AddDate(-30, 0, 0),
			Gender:            "male",
			TaxID:             "TAX123",
			HealthInsuranceID: "HI123",
			SocialInsuranceID: "SI123",
			Position:          "Developer",
			Location:          "HQ",
			Department:        "IT",
			WalletAddress:     "0xit...",
			Salary:            5000,
			Role:              "employee",
			Status:            "active",
			OnboardDate:       time.Now().AddDate(0, -6, 0),
			LeaveBalance:      20,
		},
		{
			ID:                uuid.New().String(),
			FullName:          "HR Employee",
			Email:             "hr@company.com",
			PhoneNumber:       "+9876543210",
			Address:           "456 HR St",
			DateOfBirth:       time.Now().AddDate(-28, 0, 0),
			Gender:            "female",
			TaxID:             "TAX456",
			HealthInsuranceID: "HI456",
			SocialInsuranceID: "SI456",
			Position:          "HR Staff",
			Location:          "Branch A",
			Department:        "HR",
			WalletAddress:     "0xhr...",
			Salary:            4500,
			Role:              "hr",
			Status:            "left_company", // resigned
			OnboardDate:       time.Now().AddDate(-1, 0, 0),
			LeaveBalance:      15,
		},
	}
	for _, emp := range employees {
		db.Create(&emp)
	}

	// Create absence with permission
	approvedAbsence := models.Absence{
		ID:          uuid.New().String(),
		UserID:      employees[0].ID,
		Date:        time.Now(),
		Type:        "with_permission", // Must match model enum
		Reason:      "Annual leave",
		Status:      "approved", // Must match model enum
		StartDate:   time.Now(),
		EndDate:     time.Now().AddDate(0, 0, 1),
		ProcessedBy: &employees[1].ID, // Required for approved status
		ProcessedAt: &time.Time{},     // Required for approved status
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	db.Create(&approvedAbsence)

	// Create late attendance
	lateAttendance := models.Attendance{
		ID:           uuid.New().String(),
		UserID:       employees[0].ID,
		CheckInTime:  time.Now(),
		CheckOutTime: time.Now().Add(8 * time.Hour),
		ExpectedTime: time.Now().Add(-15 * time.Minute),
		IsLate:       true,
	}
	db.Create(&lateAttendance)

	// Test cases
	testCases := []struct {
		name           string
		filter         string
		expectedCount  int
		expectedDept   string
		expectedStatus string
	}{
		{
			name:          "Filter by IT department",
			filter:        "department=IT",
			expectedCount: 1,
			expectedDept:  "IT",
		},
		{
			name:           "Filter resigned employees",
			filter:         "status=resign",
			expectedCount:  1,
			expectedStatus: "left_company",
		},
		{
			name:          "Filter late employees",
			filter:        "status=late",
			expectedCount: 1,
		},
		{
			name:          "Filter employees on approved leave",
			filter:        "status=leave_with_permission",
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/employees?"+tc.filter, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)

			var response types.APIResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			employees := response.Data.([]interface{})
			assert.Equal(t, tc.expectedCount, len(employees))

			if len(employees) > 0 {
				emp := employees[0].(map[string]interface{})
				if tc.expectedDept != "" {
					assert.Equal(t, tc.expectedDept, emp["department"])
				}
				if tc.expectedStatus != "" {
					assert.Equal(t, tc.expectedStatus, emp["status"])
				}
			}

			t.Logf("Filter '%s' returned %d employees", tc.filter, len(employees))
		})
	}

	// Cleanup in correct order
	db.Unscoped().Delete(&approvedAbsence) // Delete absences first
	db.Unscoped().Delete(&lateAttendance)  // Delete attendances next
	for _, emp := range employees {        // Delete employees last
		db.Unscoped().Delete(&emp)
	}
}
