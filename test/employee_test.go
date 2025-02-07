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
		{
			name:          "Filter by salary range",
			filter:        "salary_from=4000&salary_to=5000",
			expectedCount: 2,
		},
		{
			name:          "Filter by onboard date range",
			filter:        "onboard_from=2023-01-01&onboard_to=2024-12-31",
			expectedCount: 2,
		},
		{
			name:          "Filter by department and salary",
			filter:        "department=IT&salary_from=4500",
			expectedCount: 1,
		},
		{
			name:          "Filter by multiple criteria",
			filter:        "department=IT&salary_from=4000&onboard_from=2023-01-01",
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

func TestGetEmployeeTimeStats(t *testing.T) {
	app, db := SetupTest(t)
	app.Get("/employee-stats", handlers.GetEmployeeTimeStats)

	// Create test employees in different departments
	employees := []models.User{
		{
			ID:                uuid.New().String(),
			FullName:          "IT Employee 1",
			Email:             "it1@company.com",
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
			WalletAddress:     "0xit1...",
			Salary:            5000,
			Role:              "employee",
			Status:            "active",
			OnboardDate:       time.Now().AddDate(0, -6, 0),
			LeaveBalance:      20,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                uuid.New().String(),
			FullName:          "IT Employee 2",
			Email:             "it2@company.com",
			PhoneNumber:       "+1234567891",
			Address:           "124 IT St",
			DateOfBirth:       time.Now().AddDate(-28, 0, 0),
			Gender:            "female",
			TaxID:             "TAX124",
			HealthInsuranceID: "HI124",
			SocialInsuranceID: "SI124",
			Position:          "Senior Developer",
			Location:          "HQ",
			Department:        "IT",
			WalletAddress:     "0xit2...",
			Salary:            6000,
			Role:              "employee",
			Status:            "active",
			OnboardDate:       time.Now().AddDate(0, -3, 0),
			LeaveBalance:      20,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                uuid.New().String(),
			FullName:          "HR Employee",
			Email:             "hr@company.com",
			PhoneNumber:       "+1234567892",
			Address:           "125 HR St",
			DateOfBirth:       time.Now().AddDate(-35, 0, 0),
			Gender:            "female",
			TaxID:             "TAX125",
			HealthInsuranceID: "HI125",
			SocialInsuranceID: "SI125",
			Position:          "HR Manager",
			Location:          "HQ",
			Department:        "HR",
			WalletAddress:     "0xhr...",
			Salary:            5500,
			Role:              "hr",
			Status:            "active",
			OnboardDate:       time.Now().AddDate(-1, 0, 0),
			LeaveBalance:      20,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
	}
	for _, emp := range employees {
		db.Create(&emp)
	}

	// Use UTC time for consistency with precise seconds
	baseTime := time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC) // 09:00:00 exactly

	attendances := []models.Attendance{
		// IT Employee 1 - varying check-in/out times
		{
			ID:           uuid.New().String(),
			UserID:       employees[0].ID,
			CheckInTime:  baseTime.Add(-15*time.Minute - 30*time.Second), // 08:44:30
			CheckOutTime: baseTime.Add(8*time.Hour + 15*time.Second),     // 17:00:15
			ExpectedTime: baseTime,
			IsLate:       false,
		},
		{
			ID:           uuid.New().String(),
			UserID:       employees[0].ID,
			CheckInTime:  baseTime.Add(15*time.Minute + 45*time.Second), // 09:15:45
			CheckOutTime: baseTime.Add(9*time.Hour + 30*time.Second),    // 18:00:30
			ExpectedTime: baseTime,
			IsLate:       true,
		},
		{
			ID:           uuid.New().String(),
			UserID:       employees[0].ID,
			CheckInTime:  baseTime.Add(20 * time.Second),                              // 09:00:20
			CheckOutTime: baseTime.Add(8*time.Hour + 30*time.Minute + 15*time.Second), // 17:30:15
			ExpectedTime: baseTime,
			IsLate:       false,
		},

		// IT Employee 2 - consistent but late
		{
			ID:           uuid.New().String(),
			UserID:       employees[1].ID,
			CheckInTime:  baseTime.Add(30*time.Minute + 15*time.Second), // 09:30:15
			CheckOutTime: baseTime.Add(9*time.Hour + 45*time.Second),    // 18:00:45
			ExpectedTime: baseTime,
			IsLate:       true,
		},
		{
			ID:           uuid.New().String(),
			UserID:       employees[1].ID,
			CheckInTime:  baseTime.Add(25*time.Minute + 45*time.Second),               // 09:25:45
			CheckOutTime: baseTime.Add(8*time.Hour + 45*time.Minute + 30*time.Second), // 17:45:30
			ExpectedTime: baseTime,
			IsLate:       true,
		},

		// HR Employee - early bird
		{
			ID:           uuid.New().String(),
			UserID:       employees[2].ID,
			CheckInTime:  baseTime.Add(-45*time.Minute - 15*time.Second), // 08:14:45
			CheckOutTime: baseTime.Add(7*time.Hour + 20*time.Second),     // 16:00:20
			ExpectedTime: baseTime,
			IsLate:       false,
		},
		{
			ID:           uuid.New().String(),
			UserID:       employees[2].ID,
			CheckInTime:  baseTime.Add(-30*time.Minute - 45*time.Second),              // 08:29:15
			CheckOutTime: baseTime.Add(7*time.Hour + 30*time.Minute + 40*time.Second), // 16:30:40
			ExpectedTime: baseTime,
			IsLate:       false,
		},
	}
	for _, att := range attendances {
		db.Create(&att)
	}

	// Log initial test data
	t.Log("=== Test Setup ===")
	for _, emp := range employees {
		t.Logf("Created employee: ID=%s, Name=%s, Dept=%s",
			emp.ID, emp.FullName, emp.Department)
	}

	t.Log("\n=== Attendance Records ===")
	for _, att := range attendances {
		t.Logf("Created attendance: UserID=%s, CheckIn=%s, CheckOut=%s, IsLate=%v",
			att.UserID, att.CheckInTime, att.CheckOutTime, att.IsLate)
	}

	// Test the stats endpoint
	req := httptest.NewRequest("GET", "/employee-stats", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)

	stats := response.Data.([]interface{})

	// Log the stats results in a clear format
	t.Log("\n=== Department Average Time Stats ===")

	// Group stats by department for better readability
	departments := make(map[string][]map[string]interface{})
	for _, stat := range stats {
		s := stat.(map[string]interface{})
		dept := s["department"].(string)
		departments[dept] = append(departments[dept], s)
	}

	// Print stats grouped by department
	for dept, empStats := range departments {
		t.Logf("\nDepartment: %s", dept)
		for _, s := range empStats {
			t.Logf("  Employee: %s", s["employee_name"])
			t.Logf("    Avg Check-in:  %s", s["avg_check_in"])
			t.Logf("    Avg Check-out: %s", s["avg_check_out"])
		}
	}

	// Basic validation
	assert.Greater(t, len(stats), 0, "Should have at least one stat record")
	for _, stat := range stats {
		s := stat.(map[string]interface{})
		// Verify required fields exist
		assert.NotEmpty(t, s["department"], "Department should not be empty")
		assert.NotEmpty(t, s["employee_name"], "Employee name should not be empty")
		assert.NotEmpty(t, s["avg_check_in"], "Average check-in time should not be empty")
		assert.NotEmpty(t, s["avg_check_out"], "Average check-out time should not be empty")
	}

	// Cleanup
	for _, att := range attendances {
		db.Unscoped().Delete(&att)
	}
	for _, emp := range employees {
		db.Unscoped().Delete(&emp)
	}
}

func TestGetEmployeeWorkHoursRanking(t *testing.T) {
	app, db := SetupTest(t)
	app.Get("/employee-work-hours", handlers.GetEmployeeWorkHoursRanking)

	// Create test employees with complete details
	employees := []models.User{
			{
					ID:                uuid.New().String(),
					FullName:          "IT Employee 1",
					Email:             "it1@company.com",
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
					WalletAddress:     "0xit1...",
					Salary:            5000,
					Role:              "employee",
					Status:            "active",
					OnboardDate:       time.Now().AddDate(0, -6, 0),
					LeaveBalance:      20,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
			},
			{
					ID:                uuid.New().String(),
					FullName:          "IT Employee 2",
					Email:             "it2@company.com",
					PhoneNumber:       "+1234567891",
					Address:           "124 IT St",
					DateOfBirth:       time.Now().AddDate(-28, 0, 0),
					Gender:            "female",
					TaxID:             "TAX124",
					HealthInsuranceID: "HI124",
					SocialInsuranceID: "SI124",
					Position:          "Senior Developer",
					Location:          "HQ",
					Department:        "IT",
					WalletAddress:     "0xit2...",
					Salary:            6000,
					Role:              "employee",
					Status:            "active",
					OnboardDate:       time.Now().AddDate(0, -3, 0),
					LeaveBalance:      20,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
			},
			{
					ID:                uuid.New().String(),
					FullName:          "HR Employee",
					Email:             "hr@company.com",
					PhoneNumber:       "+1234567892",
					Address:           "125 HR St",
					DateOfBirth:       time.Now().AddDate(-35, 0, 0),
					Gender:            "female",
					TaxID:             "TAX125",
					HealthInsuranceID: "HI125",
					SocialInsuranceID: "SI125",
					Position:          "HR Manager",
					Location:          "HQ",
					Department:        "HR",
					WalletAddress:     "0xhr...",
					Salary:            5500,
					Role:              "hr",
					Status:            "active",
					OnboardDate:       time.Now().AddDate(-1, 0, 0),
					LeaveBalance:      20,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
			},
	}
	for _, emp := range employees {
			db.Create(&emp)
	}

	// Base time for consistent testing
	baseTime := time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC) // Expected start: 09:00:00

	attendances := []models.Attendance{
			// IT Employee 1 - Regular hours, no late
			{
					ID:           uuid.New().String(),
					UserID:       employees[0].ID,
					CheckInTime:  baseTime,                                    // 09:00:00
					CheckOutTime: baseTime.Add(9 * time.Hour),                // 18:00:00
					ExpectedTime: baseTime,
					IsLate:       false,
					// Work duration = 9h
					// No late penalty
					// Effective = 9h
			},
			{
					ID:           uuid.New().String(),
					UserID:       employees[0].ID,
					CheckInTime:  baseTime.Add(5 * time.Minute),              // 09:05:00
					CheckOutTime: baseTime.Add(9*time.Hour + 5*time.Minute),  // 18:05:00
					ExpectedTime: baseTime,
					IsLate:       true,
					// Work duration = 9h
					// Late penalty = 5m
					// Effective = 8h 55m
			},

			// IT Employee 2 - Late but works extra
			{
					ID:           uuid.New().String(),
					UserID:       employees[1].ID,
					CheckInTime:  baseTime.Add(30 * time.Minute),             // 09:30:00
					CheckOutTime: baseTime.Add(10 * time.Hour),               // 19:00:00
					ExpectedTime: baseTime,
					IsLate:       true,
					// Work duration = 9h 30m
					// Late penalty = 30m
					// Effective = 9h
			},
			{
					ID:           uuid.New().String(),
					UserID:       employees[1].ID,
					CheckInTime:  baseTime.Add(15 * time.Minute),             // 09:15:00
					CheckOutTime: baseTime.Add(9*time.Hour + 45*time.Minute), // 18:45:00
					ExpectedTime: baseTime,
					IsLate:       true,
					// Work duration = 9h 30m
					// Late penalty = 15m
					// Effective = 9h 15m
			},

			// HR Employee - Early bird
			{
					ID:           uuid.New().String(),
					UserID:       employees[2].ID,
					CheckInTime:  baseTime.Add(-30 * time.Minute),            // 08:30:00
					CheckOutTime: baseTime.Add(8 * time.Hour),                // 17:00:00
					ExpectedTime: baseTime,
					IsLate:       false,
					// Work duration = 8h 30m
					// No late penalty
					// Effective = 8h 30m
			},
			{
					ID:           uuid.New().String(),
					UserID:       employees[2].ID,
					CheckInTime:  baseTime.Add(-15 * time.Minute),            // 08:45:00
					CheckOutTime: baseTime.Add(8*time.Hour + 15*time.Minute), // 17:15:00
					ExpectedTime: baseTime,
					IsLate:       false,
					// Work duration = 8h 30m
					// No late penalty
					// Effective = 8h 30m
			},
	}
	for _, att := range attendances {
			db.Create(&att)
	}

	// Test the endpoint
	req := httptest.NewRequest("GET", "/employee-work-hours", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)

	stats := response.Data.([]interface{})

	// Log results for debugging
	t.Log("\n=== Employee Work Hours Ranking ===")
	for _, stat := range stats {
			s := stat.(map[string]interface{})
			t.Logf("Employee: %s (Dept: %s)", s["employee_name"], s["department"])
			t.Logf("  Work Hours: %s", s["work_hours"])
	}

	// Expected total work hours:
	// IT Employee 2:
	// Day 1: 9h 30m - 30m penalty = 9h
	// Day 2: 9h 30m - 15m penalty = 9h 15m
	// Total = 18h 15m

	// IT Employee 1:
	// Day 1: 9h - 0m penalty = 9h
	// Day 2: 9h - 5m penalty = 8h 55m
	// Total = 17h 55m

	// HR Employee:
	// Day 1: 8h 30m - 0m penalty = 8h 30m
	// Day 2: 8h 30m - 0m penalty = 8h 30m
	// Total = 17h 00m

	assert.Equal(t, 3, len(stats), "Should have 3 employees")

	firstEmployee := stats[0].(map[string]interface{})
	assert.Equal(t, "IT Employee 2", firstEmployee["employee_name"], "IT Employee 2 should have most hours")
	assert.Equal(t, "18:15:00", firstEmployee["work_hours"], "Total should be 18h 15m")

	secondEmployee := stats[1].(map[string]interface{})
	assert.Equal(t, "IT Employee 1", secondEmployee["employee_name"])
	assert.Equal(t, "17:55:00", secondEmployee["work_hours"], "Total should be 17h 55m")

	thirdEmployee := stats[2].(map[string]interface{})
	assert.Equal(t, "HR Employee", thirdEmployee["employee_name"])
	assert.Equal(t, "17:00:00", thirdEmployee["work_hours"], "Total should be 17h 00m")

	// Cleanup
	for _, att := range attendances {
			db.Unscoped().Delete(&att)
	}
	for _, emp := range employees {
			db.Unscoped().Delete(&emp)
	}
}
