package test

import (
	"bytes"
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"encoding/json"
	"net/http/httptest"
	"testing"

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
		t.Logf("  User: ID=%s, Username=%s, Status=%s", u.ID, u.Username, u.Status)
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
	app, _ := SetupTest(t)
	dumpUsers(t, "Before Add")
	app.Post("/employees", handlers.AddEmployee)

	// Test successful creation
	employee := handlers.AddEmployeeRequest{
		Username:      "testuser",
		WalletAddress: "0x123",
		Department:    "IT",
		Salary:        5000,
		Role:          "employee",
	}

	body, _ := json.Marshal(employee)
	req := httptest.NewRequest("POST", "/employees", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify database storage
	var storedEmployee models.User
	err = GetTestDB().Where("username = ?", employee.Username).First(&storedEmployee).Error
	assert.Nil(t, err)
	assert.Equal(t, employee.Username, storedEmployee.Username)
	assert.Equal(t, employee.WalletAddress, storedEmployee.WalletAddress)
	assert.Equal(t, employee.Salary, storedEmployee.Salary)

	dumpUsers(t, "After Add")
}

func TestUpdateEmployee(t *testing.T) {
	app, _ := SetupTest(t)
	dumpUsers(t, "Before Update")
	app.Patch("/employees/:id", handlers.UpdateEmployee)

	// Create test employee with UUID
	employee := models.User{
		ID:            uuid.New(),
		Username:      "updatetest",
		WalletAddress: "0x456",
		Department:    "HR",
		Salary:        6000,
		Role:          "employee",
		Status:        "active",
	}
	t.Logf("Created employee with ID: %s", employee.ID)

	// Create employee and verify
	if err := GetTestDB().Create(&employee).Error; err != nil {
		t.Fatalf("Failed to create test employee: %v", err)
	}

	// Verify employee was created
	var createdEmployee models.User
	err := GetTestDB().First(&createdEmployee, "id = ?", employee.ID).Error
	if err != nil {
		t.Fatalf("Failed to find created employee: %v", err)
	}
	t.Logf("Verified employee exists with ID: %s", createdEmployee.ID)

	// Log the actual request URL
	requestURL := "/employees/" + employee.ID.String()
	t.Logf("Making PATCH request to: %s", requestURL)

	// Test successful update
	updateData := map[string]interface{}{
		"salary": float64(7000),
	}
	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest("PATCH", requestURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Log response for debugging
	var response types.APIResponse
	json.NewDecoder(resp.Body).Decode(&response)
	t.Logf("Update response: %+v", response)

	dumpUsers(t, "After Update")
}

func TestDeleteEmployee(t *testing.T) {
	app, _ := SetupTest(t)
	dumpUsers(t, "Before Delete")
	app.Delete("/employees/:id", handlers.DeleteEmployee)

	// Create test employee with valid UUID
	employee := models.User{
		ID:            uuid.New(), // Generate new UUID
		Username:      "deletetest",
		WalletAddress: "0x789",
		Department:    "Finance",
		Salary:        5500,
		Role:          "employee",
		Status:        "active", // Set initial status
	}

	if err := GetTestDB().Create(&employee).Error; err != nil {
		t.Fatalf("Failed to create test employee: %v", err)
	}

	// Test successful deletion
	req := httptest.NewRequest("DELETE", "/employees/"+employee.ID.String(), nil)
	resp, err := app.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify employee status is updated
	var deletedEmployee models.User
	err = GetTestDB().First(&deletedEmployee, "id = ?", employee.ID).Error
	assert.Nil(t, err)
	assert.Equal(t, "left_company", deletedEmployee.Status)

	dumpUsers(t, "After Delete")
}
