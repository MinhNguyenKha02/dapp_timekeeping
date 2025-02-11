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
	app, db := SetupTest(t)
	app.Get("/employees", handlers.GetAllEmployees)

	t.Run("Get Employees When Empty", func(t *testing.T) {
		// Ensure no employees exist
		result := db.Exec("DELETE FROM users")
		assert.NoError(t, result.Error)
		t.Log("Cleared all users from database")

		req := httptest.NewRequest("GET", "/employees", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response types.APIResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.True(t, response.Success)

		// Verify empty list is returned
		employeeList := response.Data.([]interface{})
		assert.Equal(t, 0, len(employeeList))
		t.Log("Successfully retrieved empty employee list")
	})

	// Create test employees with minimal info (as root would do)
	employees := []models.User{
		{
			ID:          uuid.New().String(),
			Nickname:    "test1",
			Role:        "employee",
			Status:      "pending", // Initial status is pending
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			OnboardDate: time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Nickname:    "test2",
			Role:        "hr",
			Status:      "pending", // Initial status is pending
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			OnboardDate: time.Now(),
		},
	}

	t.Logf("Creating %d employees with minimal info", len(employees))
	for _, emp := range employees {
		result := db.Create(&emp)
		assert.NoError(t, result.Error)
		t.Logf("Created employee: ID=%s, Nickname=%s, Role=%s, Status=%s",
			emp.ID, emp.Nickname, emp.Role, emp.Status)
	}

	// Update employee profiles (as employees would do)
	updates := []map[string]interface{}{
		{
			"full_name":    "Test Employee 1",
			"email":        "test1@company.com",
			"phone_number": "+1234567890",
			"department":   "IT",
			"position":     "Developer",
			"status":       "active",
		},
		{
			"full_name":    "Test Employee 2",
			"email":        "test2@company.com",
			"phone_number": "+0987654321",
			"department":   "HR",
			"position":     "HR Staff",
			"status":       "active",
		},
	}

	t.Log("Updating employee profiles")
	for i, emp := range employees {
		result := db.Model(&emp).Updates(updates[i])
		assert.NoError(t, result.Error)
		t.Logf("Updated employee %s: Department=%s, Position=%s, Status=%s",
			emp.Nickname, updates[i]["department"], updates[i]["position"], updates[i]["status"])
	}

	t.Run("Get All Employees Without Filters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/employees", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response types.APIResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.True(t, response.Success)

		// Verify all employees are returned
		employeeList := response.Data.([]interface{})
		assert.Equal(t, len(employees), len(employeeList))
		t.Logf("Retrieved %d employees without filters", len(employeeList))

		// Log each employee's details
		for _, e := range employeeList {
			emp := e.(map[string]interface{})
			t.Logf("Employee: Nickname=%v, Department=%v, Role=%v, Status=%v",
				emp["nickname"], emp["department"], emp["role"], emp["status"])
		}
	})

	// Cleanup
	t.Log("Cleaning up test data")
	for _, emp := range employees {
		result := db.Unscoped().Delete(&emp)
		assert.NoError(t, result.Error)
		t.Logf("Deleted employee: ID=%s", emp.ID)
	}
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

func TestGetEmployeeReport(t *testing.T) {
	app, db := SetupTest(t)
	app.Get("/employee-report", handlers.GetEmployeeReport)

	// Create test employees with minimal info (as root would do)
	employees := []models.User{
		{
			ID:          uuid.New().String(),
			Nickname:    "emp1",
			Role:        "employee",
			Status:      "pending",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			OnboardDate: time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Nickname:    "emp2",
			Role:        "employee",
			Status:      "pending",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			OnboardDate: time.Now(),
		},
	}

	t.Logf("Creating %d employees with minimal info", len(employees))
	for _, emp := range employees {
		result := db.Create(&emp)
		assert.NoError(t, result.Error)
		t.Logf("Created employee: ID=%s, Nickname=%s, Status=%s",
			emp.ID, emp.Nickname, emp.Status)
	}

	// Update employee profiles
	updates := []map[string]interface{}{
		{
			"full_name":  "Top Performer",
			"department": "IT",
			"position":   "Developer",
			"status":     "active",
		},
		{
			"full_name":  "Average Worker",
			"department": "HR",
			"position":   "HR Staff",
			"status":     "active",
		},
	}

	t.Log("Updating employee profiles")
	for i, emp := range employees {
		result := db.Model(&emp).Updates(updates[i])
		assert.NoError(t, result.Error)
		t.Logf("Updated employee %s: Name=%s, Department=%s, Position=%s",
			emp.Nickname, updates[i]["full_name"], updates[i]["department"], updates[i]["position"])
	}

	// Create attendance records with consistent times
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	attendances := []models.Attendance{
		// Today's records
		{
			ID:           uuid.New().String(),
			UserID:       employees[0].ID,
			CheckInTime:  today.Add(9 * time.Hour),  // 09:00:00 today
			CheckOutTime: today.Add(18 * time.Hour), // 18:00:00 today
			ExpectedTime: today.Add(9 * time.Hour),
			OnTime:       true,
			CreatedAt:    today,
			UpdatedAt:    today,
		},
		{
			ID:           uuid.New().String(),
			UserID:       employees[1].ID,
			CheckInTime:  today.Add(9*time.Hour + 30*time.Minute), // 09:30:00 today
			CheckOutTime: today.Add(17 * time.Hour),               // 17:00:00 today
			ExpectedTime: today.Add(9 * time.Hour),
			OnTime:       false,
			CreatedAt:    today,
			UpdatedAt:    today,
		},
		// Yesterday's records
		{
			ID:           uuid.New().String(),
			UserID:       employees[0].ID,
			CheckInTime:  yesterday.Add(8*time.Hour + 45*time.Minute),  // 08:45:00 yesterday
			CheckOutTime: yesterday.Add(17*time.Hour + 30*time.Minute), // 17:30:00 yesterday
			ExpectedTime: yesterday.Add(9 * time.Hour),
			OnTime:       true,
			CreatedAt:    yesterday,
			UpdatedAt:    yesterday,
		},
		{
			ID:           uuid.New().String(),
			UserID:       employees[1].ID,
			CheckInTime:  yesterday.Add(9*time.Hour + 15*time.Minute),  // 09:15:00 yesterday
			CheckOutTime: yesterday.Add(16*time.Hour + 45*time.Minute), // 16:45:00 yesterday
			ExpectedTime: yesterday.Add(9 * time.Hour),
			OnTime:       false,
			CreatedAt:    yesterday,
			UpdatedAt:    yesterday,
		},
	}

	t.Log("Creating attendance records")
	for _, att := range attendances {
		result := db.Create(&att)
		assert.NoError(t, result.Error)
		t.Logf("Created attendance: UserID=%s, Date=%s, CheckIn=%s, CheckOut=%s",
			att.UserID,
			att.CheckInTime.Format("2006-01-02"),
			att.CheckInTime.Format("15:04:05"),
			att.CheckOutTime.Format("15:04:05"))
	}

	timeRanges := []string{"week", "month", "year"}
	for _, timeRange := range timeRanges {
		t.Run("Get Report for "+timeRange, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/employee-report?time_range="+timeRange, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)

			var response types.APIResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)
			assert.True(t, response.Success)

			report := response.Data.(map[string]interface{})

			// Log company stats
			companyStats := report["company_stats"].(map[string]interface{})
			t.Logf("\nCompany Stats (%s):", timeRange)
			t.Logf("  Time Range: %s to %s", companyStats["start_date"], companyStats["end_date"])
			t.Logf("  Total Work Hours: %.2f", companyStats["total_work_hours"])
			t.Logf("  Avg Check-in: %s", companyStats["avg_check_in"])
			t.Logf("  Avg Check-out: %s", companyStats["avg_check_out"])

			// Verify the averages
			assert.NotEmpty(t, companyStats["avg_check_in"], "Average check-in time should not be empty")
			assert.NotEmpty(t, companyStats["avg_check_out"], "Average check-out time should not be empty")
			assert.Greater(t, companyStats["total_work_hours"], float64(0), "Total work hours should be positive")

			// Log top employees
			topEmployees := report["top_employees"].([]interface{})
			t.Logf("\nTop Employees:")
			for i, emp := range topEmployees {
				e := emp.(map[string]interface{})
				t.Logf("  %d. %s (%s) - %.2f hours",
					i+1,
					e["full_name"],
					e["position"],
					e["total_work_hours"])
				t.Logf("     Avg Check-in: %s, Avg Check-out: %s",
					e["avg_check_in"],
					e["avg_check_out"])
			}
		})
	}

	// Cleanup
	t.Log("Cleaning up test data")
	for _, att := range attendances {
		db.Unscoped().Delete(&att)
	}
	for _, emp := range employees {
		db.Unscoped().Delete(&emp)
	}
}
