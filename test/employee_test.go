package test

import (
	"bytes"
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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

func TestAddEmployeeByRoot(t *testing.T) {
	app, db := SetupTest(t)

	// Set up route with JWT middleware
	app.Post("/employees", func(c *fiber.Ctx) error {
		// Get role from request header
		role := c.Get("X-Test-Role", "root")         // Default to root if not set
		t.Logf("Request role from header: %s", role) // Log the role

		claims := jwt.MapClaims{
			"role": role,
			"id":   uuid.New().String(),
		}
		t.Logf("Setting claims: %+v", claims) // Log the claims
		c.Locals("claims", claims)

		return handlers.AddEmployee(c)
	})

	t.Run("Root Creates Employee", func(t *testing.T) {
		req := handlers.AddEmployeeRequest{
			Nickname: "john_doe",
			Role:     "hr",
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/employees", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Test-Role", "root")
		t.Logf("Making root request with headers: %+v", httpReq.Header)

		resp, err := app.Test(httpReq)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Log response
		var response types.APIResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		t.Logf("Response from create: %+v", response)

		// Verify created employee
		var employee models.User
		err = db.Where("nickname = ?", req.Nickname).First(&employee).Error
		assert.NoError(t, err)
		t.Logf("Created employee: %+v", employee)
		assert.Equal(t, "hr", employee.Role)
		assert.Equal(t, "pending", employee.Status)
		assert.Empty(t, employee.FullName) // Should be empty until profile completion
	})

	t.Run("Non-Root Cannot Create Employee", func(t *testing.T) {
		req := handlers.AddEmployeeRequest{
			Nickname: "jane_doe",
			Role:     "employee",
		}
		t.Logf("Attempting to create employee with request: %+v", req)

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/employees", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Test-Role", "hr")
		t.Logf("Making HR request with headers: %+v", httpReq.Header)

		resp, err := app.Test(httpReq)
		assert.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)

		var response types.APIResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		t.Logf("Response for HR request: %+v", response)

		// Verify no employee was created
		var count int64
		err = db.Model(&models.User{}).Where("nickname = ?", req.Nickname).Count(&count).Error
		assert.NoError(t, err)
		t.Logf("Number of users with nickname %s: %d", req.Nickname, count)
		assert.Equal(t, int64(0), count, "No user should be created")

		assert.False(t, response.Success)
		assert.Equal(t, "Only root can create initial employee records", response.Error)
	})

	// Cleanup
	db.Exec("DELETE FROM users")
}

func TestUpdateEmployeeByNonRoot(t *testing.T) {
	app, db := SetupTest(t)

	// Set up route with JWT middleware
	app.Patch("/employees/:id", func(c *fiber.Ctx) error {
		c.Locals("claims", jwt.MapClaims{
			"role": c.Get("X-Test-Role", ""),
			"id":   c.Get("X-Test-ID", ""),
		})
		return handlers.UpdateEmployee(c)
	})

	// Create test employee with minimal info (as root would do)
	employee := models.User{
		ID:          uuid.New().String(),
		Nickname:    "testuser",
		Role:        "employee",
		Status:      "pending", // Initial status is pending
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		OnboardDate: time.Now().AddDate(-1, 0, 0),
	}
	result := db.Create(&employee)
	assert.NoError(t, result.Error)
	t.Logf("Created test employee with minimal info: %+v", employee)

	t.Run("Employee Completes Employee Profile", func(t *testing.T) {
		updateData := map[string]interface{}{
			"full_name":    "Test Employee",
			"email":        "test@company.com",
			"phone_number": "+1234567890",
			"address":      "123 Test St",
			"position":     "Developer",
			"department":   "IT",
			"location":     "HQ",
			"status":       "active",
		}
		body, _ := json.Marshal(updateData)

		req := httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Role", "hr")
		req.Header.Set("X-Test-ID", uuid.New().String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Verify updates
		var updatedEmployee models.User
		err = db.First(&updatedEmployee, "id = ?", employee.ID).Error
		assert.NoError(t, err)

		// Verify all fields were updated correctly
		assert.Equal(t, "Test Employee", updatedEmployee.FullName)
		assert.Equal(t, "test@company.com", updatedEmployee.Email)
		assert.Equal(t, "+1234567890", updatedEmployee.PhoneNumber)
		assert.Equal(t, "123 Test St", updatedEmployee.Address)
		assert.Equal(t, "Developer", updatedEmployee.Position)
		assert.Equal(t, "IT", updatedEmployee.Department)
		assert.Equal(t, "HQ", updatedEmployee.Location)

		// Verify protected fields remain unchanged
		assert.Equal(t, "testuser", updatedEmployee.Nickname)
		assert.Equal(t, "employee", updatedEmployee.Role)
		assert.Equal(t, float64(0), updatedEmployee.Salary)

		// Verify status changed to active after profile completion
		assert.Equal(t, "active", updatedEmployee.Status)

		t.Logf("Updated employee profile: %+v", updatedEmployee)
	})
	t.Run("Employee Cannot Update Protected Fields", func(t *testing.T) {
		protectedUpdates := map[string]interface{}{
			"nickname": "newname",
			"role":     "hr",
			"salary":   5000,
		}
		body, _ := json.Marshal(protectedUpdates)

		req := httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Role", "hr")
		req.Header.Set("X-Test-ID", uuid.New().String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)

		// Verify protected fields remain unchanged
		var unchangedEmployee models.User
		err = db.First(&unchangedEmployee, "id = ?", employee.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "testuser", unchangedEmployee.Nickname)
		assert.Equal(t, "employee", unchangedEmployee.Role)
		assert.Equal(t, float64(0), unchangedEmployee.Salary)
		t.Log("Protected fields update correctly rejected for HR role")
	})

	// Cleanup
	db.Unscoped().Delete(&employee)
}

func TestRootUpdateEmployeeSalary(t *testing.T) {
	app, db := SetupTest(t)

	// Set up route with JWT middleware
	app.Patch("/employees/:id", func(c *fiber.Ctx) error {
		c.Locals("claims", jwt.MapClaims{
			"role": c.Get("X-Test-Role", ""),
		})
		return handlers.UpdateEmployee(c)
	})

	// Create test employee with minimal info (as root would do)
	employee := models.User{
		ID:       uuid.New().String(),
		Nickname: "testuser",
		Role:     "employee",
		Status:   "pending", // Initial status is pending
	}
	result := db.Create(&employee)
	assert.NoError(t, result.Error)
	t.Logf("Created employee with minimal info: %+v", employee)

	// HR updates employee details
	updateProfileData := map[string]interface{}{
		"full_name":    "Test Employee",
		"email":        "test@company.com",
		"phone_number": "+1234567890",
		"address":      "123 Test St",
		"position":     "Developer",
		"department":   "IT",
		"location":     "HQ",
		"status":       "active",
	}
	body, _ := json.Marshal(updateProfileData)

	// Update profile as HR
	req := httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Role", "employee")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify profile was updated
	var updatedEmployee models.User
	err = db.First(&updatedEmployee, "id = ?", employee.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test Employee", updatedEmployee.FullName)
	assert.Equal(t, float64(0), updatedEmployee.Salary) // Salary should still be 0
	t.Logf("Updated employee profile: %+v", updatedEmployee)

	// Now test root updating salary
	salaryUpdate := map[string]interface{}{
		"salary": 6000,
	}
	body, _ = json.Marshal(salaryUpdate)

	// Update salary as root
	req = httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Role", "root")

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify salary was updated
	err = db.First(&updatedEmployee, "id = ?", employee.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, float64(6000), updatedEmployee.Salary)
	t.Logf("Root updated employee salary: %v", updatedEmployee.Salary)

	// Test non-root user trying to update salary
	req = httptest.NewRequest("PATCH", "/employees/"+employee.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Role", "hr")

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
	t.Log("Non-root user cannot update salary")

	// Cleanup
	db.Unscoped().Delete(&employee)
}

func TestGetEmployeesWithFilters(t *testing.T) {
	app, db := SetupTest(t)

	// Set up routes
	app.Get("/employees", handlers.GetAllEmployees)
	app.Patch("/employees/:id", func(c *fiber.Ctx) error {
		c.Locals("claims", jwt.MapClaims{
			"role": c.Get("X-Test-Role", ""),
		})
		return handlers.UpdateEmployee(c)
	})

	now := time.Now()
	baseTime := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())

	// Create test employees with minimal info (as root)
	employees := []models.User{
		{
			ID:          uuid.New().String(),
			Nickname:    "it_user",
			Role:        "employee",
			Status:      "pending",
			CreatedAt:   now,
			UpdatedAt:   now,
			OnboardDate: baseTime.AddDate(0, -6, 0),
		},
		{
			ID:          uuid.New().String(),
			Nickname:    "hr_user",
			Role:        "hr",
			Status:      "pending",
			CreatedAt:   now,
			UpdatedAt:   now,
			OnboardDate: baseTime.AddDate(-1, 0, 0),
		},
	}
	for _, emp := range employees {
		result := db.Create(&emp)
		assert.NoError(t, result.Error)
	}
	t.Log("Created employees with minimal info")

	// Update employee profiles
	updateData := []map[string]interface{}{
		{
			"full_name":    "IT Employee",
			"email":        "it@company.com",
			"phone_number": "+1234567890",
			"address":      "123 IT St",
			"position":     "Developer",
			"department":   "IT",
			"location":     "HQ",
			"status":       "active",
		},
		{
			"full_name":    "HR Employee",
			"email":        "hr@company.com",
			"phone_number": "+9876543210",
			"address":      "456 HR St",
			"position":     "HR Staff",
			"department":   "HR",
			"location":     "Branch A",
			"status":       "active",
		},
	}

	// Update profiles
	for i, emp := range employees {
		body, _ := json.Marshal(updateData[i])
		req := httptest.NewRequest("PATCH", "/employees/"+emp.ID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Role", "employee")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Verify update
		var updatedEmp models.User
		err = db.First(&updatedEmp, "id = ?", emp.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "active", updatedEmp.Status)
	}
	t.Log("Updated employee profiles")

	// Create attendance records that will trigger auto-creation of absences
	// 1. Late without permission (30 minutes late)
	lateAttendance := models.Attendance{
		ID:           uuid.New().String(),
		UserID:       employees[0].ID,
		CheckInTime:  baseTime.Add(30 * time.Minute),
		CheckOutTime: baseTime.Add(9 * time.Hour),
		ExpectedTime: baseTime,
		OnTime:       false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	result := db.Create(&lateAttendance)
	assert.NoError(t, result.Error)

	// Auto-create late without permission absence
	lateWithoutPermission := models.Absence{
		ID:          uuid.New().String(),
		UserID:      employees[0].ID,
		Date:        now,
		Type:        "late_without_permission",
		Reason:      "Late check-in without prior permission",
		Status:      "approved",
		StartDate:   lateAttendance.CheckInTime,
		EndDate:     lateAttendance.CheckInTime,
		ProcessedBy: &employees[1].ID, // HR processes the absence
		ProcessedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	result = db.Create(&lateWithoutPermission)
	assert.NoError(t, result.Error)

	// 2. Leave without permission (left 2 hours early)
	earlyLeaveAttendance := models.Attendance{
		ID:           uuid.New().String(),
		UserID:       employees[0].ID,
		CheckInTime:  baseTime,
		CheckOutTime: baseTime.Add(6 * time.Hour),
		ExpectedTime: baseTime,
		OnTime:       false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	result = db.Create(&earlyLeaveAttendance)
	assert.NoError(t, result.Error)

	// Auto-create leave without permission absence
	leaveWithoutPermission := models.Absence{
		ID:          uuid.New().String(),
		UserID:      employees[0].ID,
		Date:        now,
		Type:        "leave_without_permission",
		Reason:      "Early leave without prior permission",
		Status:      "approved",
		StartDate:   earlyLeaveAttendance.CheckOutTime,
		EndDate:     baseTime.Add(8 * time.Hour),
		ProcessedBy: &employees[1].ID, // HR processes the absence
		ProcessedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	result = db.Create(&leaveWithoutPermission)
	assert.NoError(t, result.Error)

	// 3. Create leave with permission (approved)
	leaveWithPermission := models.Absence{
		ID:          uuid.New().String(),
		UserID:      employees[1].ID,
		Date:        now,
		Type:        "leave_with_permission",
		Reason:      "Doctor appointment",
		Status:      "approved",
		StartDate:   baseTime.AddDate(0, 0, 1),
		EndDate:     baseTime.AddDate(0, 0, 1).Add(4 * time.Hour),
		ProcessedBy: &employees[1].ID,
		ProcessedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	result = db.Create(&leaveWithPermission)
	assert.NoError(t, result.Error)

	// 4. Create late with permission request (approved)
	lateWithPermission := models.Absence{
		ID:          uuid.New().String(),
		UserID:      employees[1].ID,
		Date:        now,
		Type:        "late_with_permission",
		Reason:      "Traffic accident",
		Status:      "approved",
		StartDate:   baseTime,
		EndDate:     baseTime.Add(1 * time.Hour),
		ProcessedBy: &employees[1].ID, // HR processes the absence
		ProcessedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	result = db.Create(&lateWithPermission)
	assert.NoError(t, result.Error)

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
			name:           "Filter by late without permission",
			filter:         "status=late_without_permission",
			expectedCount:  1,
			expectedStatus: "active",
		},
		{
			name:           "Filter by leave without permission",
			filter:         "status=leave_without_permission",
			expectedCount:  1,
			expectedStatus: "active",
		},
		{
			name:           "Filter by leave with permission",
			filter:         "status=leave_with_permission",
			expectedCount:  1,
			expectedStatus: "active",
		},
		{
			name:           "Filter by late with permission",
			filter:         "status=late_with_permission",
			expectedCount:  1,
			expectedStatus: "active",
		},
		{
			name: "Filter by onboard date range",
			filter: fmt.Sprintf("onboard_from=%s&onboard_to=%s",
				time.Now().AddDate(-2, 0, 0).Format("2006-01-02"),
				time.Now().Format("2006-01-02")),
			expectedCount: 2,
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
	db.Unscoped().Delete(&lateWithPermission)
	db.Unscoped().Delete(&leaveWithPermission)
	db.Unscoped().Delete(&leaveWithoutPermission)
	db.Unscoped().Delete(&lateWithoutPermission)
	db.Unscoped().Delete(&earlyLeaveAttendance)
	db.Unscoped().Delete(&lateAttendance)
	for _, emp := range employees {
		db.Unscoped().Delete(&emp)
	}
}

// func TestGetEmployeeTimeStats(t *testing.T) {
// 	app, db := SetupTest(t)
// 	app.Get("/employee-stats", handlers.GetEmployeeTimeStats)

// 	// Create test employees in different departments
// 	employees := []models.User{
// 		{
// 			ID:                uuid.New().String(),
// 			FullName:          "IT Employee 1",
// 			Email:             "it1@company.com",
// 			PhoneNumber:       "+1234567890",
// 			Address:           "123 IT St",
// 			DateOfBirth:       time.Now().AddDate(-30, 0, 0),
// 			Gender:            "male",
// 			TaxID:             "TAX123",
// 			HealthInsuranceID: "HI123",
// 			SocialInsuranceID: "SI123",
// 			Position:          "Developer",
// 			Location:          "HQ",
// 			Department:        "IT",
// 			WalletAddress:     "0xit1...",
// 			Salary:            5000,
// 			Role:              "employee",
// 			Status:            "active",
// 			OnboardDate:       time.Now().AddDate(0, -6, 0),
// 			CreatedAt:         time.Now(),
// 			UpdatedAt:         time.Now(),
// 		},
// 		{
// 			ID:                uuid.New().String(),
// 			FullName:          "IT Employee 2",
// 			Email:             "it2@company.com",
// 			PhoneNumber:       "+1234567891",
// 			Address:           "124 IT St",
// 			DateOfBirth:       time.Now().AddDate(-28, 0, 0),
// 			Gender:            "female",
// 			TaxID:             "TAX124",
// 			HealthInsuranceID: "HI124",
// 			SocialInsuranceID: "SI124",
// 			Position:          "Senior Developer",
// 			Location:          "HQ",
// 			Department:        "IT",
// 			WalletAddress:     "0xit2...",
// 			Salary:            6000,
// 			Role:              "employee",
// 			Status:            "active",
// 			OnboardDate:       time.Now().AddDate(0, -3, 0),
// 			CreatedAt:         time.Now(),
// 			UpdatedAt:         time.Now(),
// 		},
// 		{
// 			ID:                uuid.New().String(),
// 			FullName:          "HR Employee",
// 			Email:             "hr@company.com",
// 			PhoneNumber:       "+1234567892",
// 			Address:           "125 HR St",
// 			DateOfBirth:       time.Now().AddDate(-35, 0, 0),
// 			Gender:            "female",
// 			TaxID:             "TAX125",
// 			HealthInsuranceID: "HI125",
// 			SocialInsuranceID: "SI125",
// 			Position:          "HR Manager",
// 			Location:          "HQ",
// 			Department:        "HR",
// 			WalletAddress:     "0xhr...",
// 			Salary:            5500,
// 			Role:              "hr",
// 			Status:            "active",
// 			OnboardDate:       time.Now().AddDate(-1, 0, 0),
// 			CreatedAt:         time.Now(),
// 			UpdatedAt:         time.Now(),
// 		},
// 	}
// 	for _, emp := range employees {
// 		db.Create(&emp)
// 	}

// 	// Use UTC time for consistency with precise seconds
// 	baseTime := time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC) // 09:00:00 exactly

// 	attendances := []models.Attendance{
// 		// IT Employee 1 - varying check-in/out times
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[0].ID,
// 			CheckInTime:  baseTime.Add(-15*time.Minute - 30*time.Second), // 08:44:30
// 			CheckOutTime: baseTime.Add(8*time.Hour + 15*time.Second),     // 17:00:15
// 			ExpectedTime: baseTime,
// 			IsLate:       false,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 		},
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[0].ID,
// 			CheckInTime:  baseTime.Add(15*time.Minute + 45*time.Second), // 09:15:45
// 			CheckOutTime: baseTime.Add(9*time.Hour + 30*time.Second),    // 18:00:30
// 			ExpectedTime: baseTime,
// 			IsLate:       true,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 		},
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[0].ID,
// 			CheckInTime:  baseTime.Add(20 * time.Second),                              // 09:00:20
// 			CheckOutTime: baseTime.Add(8*time.Hour + 30*time.Minute + 15*time.Second), // 17:30:15
// 			ExpectedTime: baseTime,
// 			IsLate:       false,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 		},

// 		// IT Employee 2 - consistent but late
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[1].ID,
// 			CheckInTime:  baseTime.Add(30*time.Minute + 15*time.Second), // 09:30:15
// 			CheckOutTime: baseTime.Add(9*time.Hour + 45*time.Second),    // 18:00:45
// 			ExpectedTime: baseTime,
// 			IsLate:       true,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 		},
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[1].ID,
// 			CheckInTime:  baseTime.Add(25*time.Minute + 45*time.Second),               // 09:25:45
// 			CheckOutTime: baseTime.Add(8*time.Hour + 45*time.Minute + 30*time.Second), // 17:45:30
// 			ExpectedTime: baseTime,
// 			IsLate:       true,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 		},

// 		// HR Employee - early bird
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[2].ID,
// 			CheckInTime:  baseTime.Add(-45*time.Minute - 15*time.Second), // 08:14:45
// 			CheckOutTime: baseTime.Add(7*time.Hour + 20*time.Second),     // 16:00:20
// 			ExpectedTime: baseTime,
// 			IsLate:       false,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 		},
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[2].ID,
// 			CheckInTime:  baseTime.Add(-30*time.Minute - 45*time.Second),              // 08:29:15
// 			CheckOutTime: baseTime.Add(7*time.Hour + 30*time.Minute + 40*time.Second), // 16:30:40
// 			ExpectedTime: baseTime,
// 			IsLate:       false,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 		},
// 	}
// 	for _, att := range attendances {
// 		db.Create(&att)
// 	}

// 	// Log initial test data
// 	t.Log("=== Test Setup ===")
// 	for _, emp := range employees {
// 		t.Logf("Created employee: ID=%s, Name=%s, Dept=%s",
// 			emp.ID, emp.FullName, emp.Department)
// 	}

// 	t.Log("\n=== Attendance Records ===")
// 	for _, att := range attendances {
// 		t.Logf("Created attendance: UserID=%s, CheckIn=%s, CheckOut=%s, IsLate=%v",
// 			att.UserID, att.CheckInTime, att.CheckOutTime, att.IsLate)
// 	}

// 	// Test the stats endpoint
// 	req := httptest.NewRequest("GET", "/employee-stats", nil)
// 	resp, err := app.Test(req)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 200, resp.StatusCode)

// 	var response types.APIResponse
// 	err = json.NewDecoder(resp.Body).Decode(&response)
// 	assert.NoError(t, err)

// 	stats := response.Data.([]interface{})

// 	// Log the stats results in a clear format
// 	t.Log("\n=== Department Average Time Stats ===")

// 	// Group stats by department for better readability
// 	departments := make(map[string][]map[string]interface{})
// 	for _, stat := range stats {
// 		s := stat.(map[string]interface{})
// 		dept := s["department"].(string)
// 		departments[dept] = append(departments[dept], s)
// 	}

// 	// Print stats grouped by department
// 	for dept, empStats := range departments {
// 		t.Logf("\nDepartment: %s", dept)
// 		for _, s := range empStats {
// 			t.Logf("  Employee: %s", s["employee_name"])
// 			t.Logf("    Avg Check-in:  %s", s["avg_check_in"])
// 			t.Logf("    Avg Check-out: %s", s["avg_check_out"])
// 		}
// 	}

// 	// Basic validation
// 	assert.Greater(t, len(stats), 0, "Should have at least one stat record")
// 	for _, stat := range stats {
// 		s := stat.(map[string]interface{})
// 		// Verify required fields exist
// 		assert.NotEmpty(t, s["department"], "Department should not be empty")
// 		assert.NotEmpty(t, s["employee_name"], "Employee name should not be empty")
// 		assert.NotEmpty(t, s["avg_check_in"], "Average check-in time should not be empty")
// 		assert.NotEmpty(t, s["avg_check_out"], "Average check-out time should not be empty")
// 	}

// 	// Cleanup
// 	for _, att := range attendances {
// 		db.Unscoped().Delete(&att)
// 	}
// 	for _, emp := range employees {
// 		db.Unscoped().Delete(&emp)
// 	}
// }

// func TestGetEmployeeWorkHoursRanking(t *testing.T) {
// 	app, db := SetupTest(t)
// 	app.Get("/employee-work-hours", handlers.GetEmployeeWorkHoursRanking)

// 	// Create test employees with complete details
// 	employees := []models.User{
// 		{
// 			ID:                uuid.New().String(),
// 			FullName:          "IT Employee 1",
// 			Email:             "it1@company.com",
// 			PhoneNumber:       "+1234567890",
// 			Address:           "123 IT St",
// 			DateOfBirth:       time.Now().AddDate(-30, 0, 0),
// 			Gender:            "male",
// 			TaxID:             "TAX123",
// 			HealthInsuranceID: "HI123",
// 			SocialInsuranceID: "SI123",
// 			Position:          "Developer",
// 			Location:          "HQ",
// 			Department:        "IT",
// 			WalletAddress:     "0xit1...",
// 			Salary:            5000,
// 			Role:              "employee",
// 			Status:            "active",
// 			OnboardDate:       time.Now().AddDate(0, -6, 0),
// 			CreatedAt:         time.Now(),
// 			UpdatedAt:         time.Now(),
// 		},
// 		{
// 			ID:                uuid.New().String(),
// 			FullName:          "IT Employee 2",
// 			Email:             "it2@company.com",
// 			PhoneNumber:       "+1234567891",
// 			Address:           "124 IT St",
// 			DateOfBirth:       time.Now().AddDate(-28, 0, 0),
// 			Gender:            "female",
// 			TaxID:             "TAX124",
// 			HealthInsuranceID: "HI124",
// 			SocialInsuranceID: "SI124",
// 			Position:          "Senior Developer",
// 			Location:          "HQ",
// 			Department:        "IT",
// 			WalletAddress:     "0xit2...",
// 			Salary:            6000,
// 			Role:              "employee",
// 			Status:            "active",
// 			OnboardDate:       time.Now().AddDate(0, -3, 0),
// 			CreatedAt:         time.Now(),
// 			UpdatedAt:         time.Now(),
// 		},
// 		{
// 			ID:                uuid.New().String(),
// 			FullName:          "HR Employee",
// 			Email:             "hr@company.com",
// 			PhoneNumber:       "+1234567892",
// 			Address:           "125 HR St",
// 			DateOfBirth:       time.Now().AddDate(-35, 0, 0),
// 			Gender:            "female",
// 			TaxID:             "TAX125",
// 			HealthInsuranceID: "HI125",
// 			SocialInsuranceID: "SI125",
// 			Position:          "HR Manager",
// 			Location:          "HQ",
// 			Department:        "HR",
// 			WalletAddress:     "0xhr...",
// 			Salary:            5500,
// 			Role:              "hr",
// 			Status:            "active",
// 			OnboardDate:       time.Now().AddDate(-1, 0, 0),
// 			CreatedAt:         time.Now(),
// 			UpdatedAt:         time.Now(),
// 		},
// 	}
// 	for _, emp := range employees {
// 		db.Create(&emp)
// 	}

// 	// Base time for consistent testing
// 	baseTime := time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC) // Expected start: 09:00:00

// 	attendances := []models.Attendance{
// 		// IT Employee 1 - Regular hours, no late
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[0].ID,
// 			CheckInTime:  baseTime,                    // 09:00:00
// 			CheckOutTime: baseTime.Add(9 * time.Hour), // 18:00:00
// 			ExpectedTime: baseTime,
// 			IsLate:       false,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 			// Work duration = 9h
// 			// No late penalty
// 			// Effective = 9h
// 		},
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[0].ID,
// 			CheckInTime:  baseTime.Add(5 * time.Minute),             // 09:05:00
// 			CheckOutTime: baseTime.Add(9*time.Hour + 5*time.Minute), // 18:05:00
// 			ExpectedTime: baseTime,
// 			IsLate:       true,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 			// Work duration = 9h
// 			// Late penalty = 5m
// 			// Effective = 8h 55m
// 		},

// 		// IT Employee 2 - Late but works extra
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[1].ID,
// 			CheckInTime:  baseTime.Add(30 * time.Minute), // 09:30:00
// 			CheckOutTime: baseTime.Add(10 * time.Hour),   // 19:00:00
// 			ExpectedTime: baseTime,
// 			IsLate:       true,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 			// Work duration = 9h 30m
// 			// Late penalty = 30m
// 			// Effective = 9h
// 		},
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[1].ID,
// 			CheckInTime:  baseTime.Add(15 * time.Minute),             // 09:15:00
// 			CheckOutTime: baseTime.Add(9*time.Hour + 45*time.Minute), // 18:45:00
// 			ExpectedTime: baseTime,
// 			IsLate:       true,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 			// Work duration = 9h 30m
// 			// Late penalty = 15m
// 			// Effective = 9h 15m
// 		},

// 		// HR Employee - Early bird
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[2].ID,
// 			CheckInTime:  baseTime.Add(-30 * time.Minute), // 08:30:00
// 			CheckOutTime: baseTime.Add(8 * time.Hour),     // 17:00:00
// 			ExpectedTime: baseTime,
// 			IsLate:       false,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 			// Work duration = 8h 30m
// 			// No late penalty
// 			// Effective = 8h 30m
// 		},
// 		{
// 			ID:           uuid.New().String(),
// 			UserID:       employees[2].ID,
// 			CheckInTime:  baseTime.Add(-15 * time.Minute),            // 08:45:00
// 			CheckOutTime: baseTime.Add(8*time.Hour + 15*time.Minute), // 17:15:00
// 			ExpectedTime: baseTime,
// 			IsLate:       false,
// 			CreatedAt:    time.Now(),
// 			UpdatedAt:    time.Now(),
// 			// Work duration = 8h 30m
// 			// No late penalty
// 			// Effective = 8h 30m
// 		},
// 	}
// 	for _, att := range attendances {
// 		db.Create(&att)
// 	}

// 	// Test the endpoint
// 	req := httptest.NewRequest("GET", "/employee-work-hours", nil)
// 	resp, err := app.Test(req)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 200, resp.StatusCode)

// 	var response types.APIResponse
// 	err = json.NewDecoder(resp.Body).Decode(&response)
// 	assert.NoError(t, err)

// 	stats := response.Data.([]interface{})

// 	// Log results for debugging
// 	t.Log("\n=== Employee Work Hours Ranking ===")
// 	for _, stat := range stats {
// 		s := stat.(map[string]interface{})
// 		t.Logf("Employee: %s (Dept: %s)", s["employee_name"], s["department"])
// 		t.Logf("  Work Hours: %s", s["work_hours"])
// 	}

// 	// Expected total work hours:
// 	// IT Employee 2:
// 	// Day 1: 9h 30m - 30m penalty = 9h
// 	// Day 2: 9h 30m - 15m penalty = 9h 15m
// 	// Total = 18h 15m

// 	// IT Employee 1:
// 	// Day 1: 9h - 0m penalty = 9h
// 	// Day 2: 9h - 5m penalty = 8h 55m
// 	// Total = 17h 55m

// 	// HR Employee:
// 	// Day 1: 8h 30m - 0m penalty = 8h 30m
// 	// Day 2: 8h 30m - 0m penalty = 8h 30m
// 	// Total = 17h 00m

// 	assert.Equal(t, 3, len(stats), "Should have 3 employees")

// 	firstEmployee := stats[0].(map[string]interface{})
// 	assert.Equal(t, "IT Employee 2", firstEmployee["employee_name"], "IT Employee 2 should have most hours")
// 	assert.Equal(t, "18:15:00", firstEmployee["work_hours"], "Total should be 18h 15m")

// 	secondEmployee := stats[1].(map[string]interface{})
// 	assert.Equal(t, "IT Employee 1", secondEmployee["employee_name"])
// 	assert.Equal(t, "17:55:00", secondEmployee["work_hours"], "Total should be 17h 55m")

// 	thirdEmployee := stats[2].(map[string]interface{})
// 	assert.Equal(t, "HR Employee", thirdEmployee["employee_name"])
// 	assert.Equal(t, "17:00:00", thirdEmployee["work_hours"], "Total should be 17h 00m")

// 	// Cleanup
// 	for _, att := range attendances {
// 		db.Unscoped().Delete(&att)
// 	}
// 	for _, emp := range employees {
// 		db.Unscoped().Delete(&emp)
// 	}
// }
// func TestCheckInAndCheckOut(t *testing.T) {
// 	app, db := SetupTest(t)
// 	app.Post("/check-in", handlers.CheckIn)
// 	app.Post("/check-out", handlers.CheckOut)

// 	// Create test employee
// 	employee := models.User{
// 		ID:                uuid.New().String(),
// 		FullName:          "Test Employee",
// 		Email:             "test@company.com",
// 		PhoneNumber:       "+1234567890",
// 		Address:           "123 Test St",
// 		DateOfBirth:       time.Now().AddDate(-30, 0, 0),
// 		Gender:            "male",
// 		TaxID:             "TAX123",
// 		HealthInsuranceID: "HI123",
// 		SocialInsuranceID: "SI123",
// 		Position:          "Developer",
// 		Location:          "HQ",
// 		Department:        "IT",
// 		WalletAddress:     "0x123...",
// 		Salary:            5000,
// 		Role:              "employee",
// 		Status:            "active",
// 		OnboardDate:       time.Now(),
// 		CreatedAt:         time.Now(),
// 		UpdatedAt:         time.Now(),
// 	}
// 	db.Create(&employee)

// 	t.Run("Successful Check-in", func(t *testing.T) {
// 		t.Logf("Testing check-in for employee ID: %s", employee.ID)

// 		checkInReq := handlers.CheckInRequest{
// 			UserID: employee.ID,
// 		}
// 		body, _ := json.Marshal(checkInReq)
// 		req := httptest.NewRequest("POST", "/check-in", bytes.NewBuffer(body))
// 		req.Header.Set("Content-Type", "application/json")

// 		resp, err := app.Test(req)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 200, resp.StatusCode)

// 		// Log response
// 		var response types.APIResponse
// 		err = json.NewDecoder(resp.Body).Decode(&response)
// 		assert.NoError(t, err)
// 		t.Logf("Check-in response: %+v", response)

// 		// Verify attendance record
// 		var attendance models.Attendance
// 		err = db.Where("user_id = ?", employee.ID).First(&attendance).Error
// 		assert.NoError(t, err)
// 		t.Logf("Created attendance record: %+v", attendance)
// 		assert.False(t, attendance.CheckInTime.IsZero(), "Check-in time should be set")
// 		assert.True(t, attendance.CheckOutTime.IsZero(), "Check-out time should not be set")
// 	})

// 	t.Run("Duplicate Check-in", func(t *testing.T) {
// 		t.Logf("Testing duplicate check-in for employee ID: %s", employee.ID)

// 		// Verify existing attendance
// 		var existingAttendance models.Attendance
// 		err := db.Where("user_id = ?", employee.ID).First(&existingAttendance).Error
// 		assert.NoError(t, err)
// 		t.Logf("Existing attendance before duplicate check-in: %+v", existingAttendance)

// 		// Try duplicate check-in
// 		checkInReq := handlers.CheckInRequest{
// 			UserID: employee.ID,
// 		}
// 		body, _ := json.Marshal(checkInReq)
// 		req := httptest.NewRequest("POST", "/check-in", bytes.NewBuffer(body))
// 		req.Header.Set("Content-Type", "application/json")

// 		resp, err := app.Test(req)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 400, resp.StatusCode)

// 		var response types.APIResponse
// 		err = json.NewDecoder(resp.Body).Decode(&response)
// 		assert.NoError(t, err)
// 		t.Logf("Duplicate check-in response: %+v", response)
// 	})

// 	t.Run("Successful Check-out", func(t *testing.T) {
// 		t.Logf("Testing check-out for employee ID: %s", employee.ID)

// 		// Verify attendance before check-out
// 		var attendanceBeforeCheckout models.Attendance
// 		err := db.Where("user_id = ?", employee.ID).First(&attendanceBeforeCheckout).Error
// 		assert.NoError(t, err)
// 		t.Logf("Attendance before check-out: %+v", attendanceBeforeCheckout)

// 		checkOutReq := handlers.CheckOutRequest{
// 			UserID: employee.ID,
// 		}
// 		body, _ := json.Marshal(checkOutReq)
// 		req := httptest.NewRequest("POST", "/check-out", bytes.NewBuffer(body))
// 		req.Header.Set("Content-Type", "application/json")

// 		resp, err := app.Test(req)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 200, resp.StatusCode)

// 		// Verify updated attendance
// 		var attendanceAfterCheckout models.Attendance
// 		err = db.Where("user_id = ?", employee.ID).First(&attendanceAfterCheckout).Error
// 		assert.NoError(t, err)
// 		t.Logf("Attendance after check-out: %+v", attendanceAfterCheckout)
// 		assert.False(t, attendanceAfterCheckout.CheckOutTime.IsZero(), "Check-out time should be set")
// 	})

// 	t.Run("Check-out Without Check-in", func(t *testing.T) {
// 		// Create another employee
// 		employee2 := employee
// 		employee2.ID = uuid.New().String()
// 		employee2.Email = "test2@company.com"
// 		db.Create(&employee2)
// 		t.Logf("Testing check-out without check-in for employee ID: %s", employee2.ID)

// 		// Verify no existing attendance - use Count instead of First to avoid log
// 		var count int64
// 		err := db.Model(&models.Attendance{}).Where("user_id = ?", employee2.ID).Count(&count).Error
// 		assert.NoError(t, err)
// 		assert.Equal(t, int64(0), count, "Should not have any attendance records")
// 		t.Log("Verified no existing attendance record")

// 		checkOutReq := handlers.CheckOutRequest{
// 			UserID: employee2.ID,
// 		}
// 		body, _ := json.Marshal(checkOutReq)
// 		req := httptest.NewRequest("POST", "/check-out", bytes.NewBuffer(body))
// 		req.Header.Set("Content-Type", "application/json")

// 		resp, err := app.Test(req)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 400, resp.StatusCode)

// 		var response types.APIResponse
// 		err = json.NewDecoder(resp.Body).Decode(&response)
// 		assert.NoError(t, err)
// 		t.Logf("Check-out without check-in response: %+v", response)

// 		// Cleanup
// 		db.Unscoped().Delete(&employee2)
// 	})

// 	// Cleanup
// 	var attendances []models.Attendance
// 	db.Where("user_id = ?", employee.ID).Find(&attendances)
// 	for _, att := range attendances {
// 		t.Logf("Cleaning up attendance record: %+v", att)
// 		db.Unscoped().Delete(&att)
// 	}
// 	db.Unscoped().Delete(&employee)
// }
