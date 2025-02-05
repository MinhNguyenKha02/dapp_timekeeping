package test

import (
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type AbsenceResponse struct {
	ID          string     `json:"id"`
	FullName    string     `json:"full_name"`
	Department  string     `json:"department"`
	Date        time.Time  `json:"date"`
	Type        string     `json:"type"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"`
	ProcessedBy *string    `json:"processed_by,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

func TestGetAbsences(t *testing.T) {
	app, db := SetupTest(t)
	t.Log("Test setup completed")

	// Create root user first
	rootUser := models.User{
		ID:                uuid.New().String(),
		FullName:          "Root Admin",
		Email:             "root@company.com",
		PhoneNumber:       "+1234567890",
		Address:           "123 Admin St",
		DateOfBirth:       time.Now().AddDate(-30, 0, 0),
		Gender:            "male",
		TaxID:             "TAX123",
		HealthInsuranceID: "HI123",
		SocialInsuranceID: "SI123",
		Position:          "System Administrator",
		Location:          "HQ",
		OnboardDate:       time.Now(),
		Role:              "root",
		Department:        "Management",
		WalletAddress:     "0x123...",
		Salary:            10000,
		LeaveBalance:      30,
		Status:            "active",
	}
	if err := db.Create(&rootUser).Error; err != nil {
		t.Fatalf("Failed to create root user: %v", err)
	}

	// Create test user
	user := models.User{
		ID:                uuid.New().String(),
		FullName:          "Test Employee",
		Email:             "test@company.com",
		PhoneNumber:       "+9876543210",
		Address:           "456 Test St",
		DateOfBirth:       time.Now().AddDate(-25, 0, 0),
		Gender:            "female",
		TaxID:             "TAX456",
		HealthInsuranceID: "HI456",
		SocialInsuranceID: "SI456",
		Position:          "Software Engineer",
		Location:          "Branch A",
		OnboardDate:       time.Now(),
		Role:              "employee",
		Department:        "IT",
		WalletAddress:     "0x456...",
		Salary:            5000,
		LeaveBalance:      20,
		Status:            "active",
	}

	// Create and verify in transaction
	tx := db.Begin()
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create absence with root user as processor
	absence := models.Absence{
		ID:          uuid.New().String(),
		UserID:      user.ID,
		Date:        time.Now(),
		Type:        "without_permission",
		Reason:      "Personal emergency",
		Status:      "pending",
		ProcessedBy: rootUser.ID, // Set root user as processor
	}
	if err := tx.Create(&absence).Error; err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create absence: %v", err)
	}

	tx.Commit()
	t.Logf("Created test absence - ID: %s, Type: %s, Status: %s", absence.ID, absence.Type, absence.Status)

	// Verify data in DB
	var userCount, absenceCount int64
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.Absence{}).Count(&absenceCount)
	t.Logf("After setup - Users: %d, Absences: %d", userCount, absenceCount)

	// Generate test token
	token := createTestToken(rootUser.ID, "root")
	t.Logf("Generated auth token for root user")

	// Set up routes
	root := app.Group("/root")
	attendance := root.Group("/attendance")
	attendance.Get("/absences", handlers.GetAbsences)
	t.Log("Routes configured")

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkResponse  func(*testing.T, types.APIResponse)
	}{
		{
			name:           "Get all absences",
			queryParams:    "",
			expectedStatus: 200,
			checkResponse: func(t *testing.T, response types.APIResponse) {
				assert.True(t, response.Success)
				absences, ok := response.Data.([]interface{})
				assert.True(t, ok)
				assert.NotEmpty(t, absences)

				firstAbsence := absences[0].(map[string]interface{})
				assert.Equal(t, absence.ID, firstAbsence["id"])
				assert.Equal(t, user.FullName, firstAbsence["full_name"])
				assert.Equal(t, user.Department, firstAbsence["department"])
			},
		},
		{
			name:           "Filter by type",
			queryParams:    "?type=without_permission",
			expectedStatus: 200,
			checkResponse: func(t *testing.T, response types.APIResponse) {
				assert.True(t, response.Success)
				absences, ok := response.Data.([]interface{})
				assert.True(t, ok)
				for _, abs := range absences {
					absMap := abs.(map[string]interface{})
					assert.Equal(t, "without_permission", absMap["type"])
				}
			},
		},
		{
			name:           "Filter by department",
			queryParams:    "?department=IT",
			expectedStatus: 200,
			checkResponse: func(t *testing.T, response types.APIResponse) {
				assert.True(t, response.Success)
				absences, ok := response.Data.([]interface{})
				assert.True(t, ok)
				for _, abs := range absences {
					absMap := abs.(map[string]interface{})
					assert.Equal(t, "IT", absMap["department"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test case: %s", tt.name)
			t.Logf("Making request to: /root/attendance/absences%s", tt.queryParams)

			req := httptest.NewRequest("GET", "/root/attendance/absences"+tt.queryParams, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			t.Logf("Response status: %d", resp.StatusCode)

			var response types.APIResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)
			t.Logf("Response: %+v", response)

			tt.checkResponse(t, response)
			t.Logf("Test case completed: %s", tt.name)
		})
	}

	// Cleanup and verify
	t.Log("Starting cleanup")
	db.Unscoped().Delete(&absence)
	db.Unscoped().Delete(&user)
	db.Unscoped().Delete(&rootUser)

	// Verify cleanup
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.Absence{}).Count(&absenceCount)
	t.Logf("After cleanup - Users: %d, Absences: %d", userCount, absenceCount)
}
