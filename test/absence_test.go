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
	Username    string     `json:"username"`
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

	// Log initial DB state
	var userCount, absenceCount int64
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.Absence{}).Count(&absenceCount)
	t.Logf("Initial DB state - Users: %d, Absences: %d", userCount, absenceCount)

	// Create test user
	user := models.User{
		ID:         uuid.New(),
		Username:   "testuser",
		Department: "IT",
		Role:       "employee",
		Status:     "active",
	}
	result := db.Create(&user)
	assert.NoError(t, result.Error)
	t.Logf("Created test user - ID: %s, Username: %s, Department: %s", user.ID, user.Username, user.Department)

	// Create test absence
	absence := models.Absence{
		ID:     uuid.New(),
		UserID: user.ID,
		Date:   time.Now(),
		Type:   "without_permission",
		Reason: "Personal emergency",
		Status: "pending",
	}
	result = db.Create(&absence)
	assert.NoError(t, result.Error)
	t.Logf("Created test absence - ID: %s, Type: %s, Status: %s", absence.ID, absence.Type, absence.Status)

	// Create root user
	rootUser := models.User{
		ID:       uuid.New(),
		Username: "root_user",
		Role:     "root",
		Status:   "active",
	}
	result = db.Create(&rootUser)
	assert.NoError(t, result.Error)
	t.Logf("Created root user - ID: %s, Username: %s, Role: %s", rootUser.ID, rootUser.Username, rootUser.Role)

	// Verify data in DB
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.Absence{}).Count(&absenceCount)
	t.Logf("After setup - Users: %d, Absences: %d", userCount, absenceCount)

	// Generate test token
	token := createTestToken(rootUser.ID.String(), "root")
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

				// Convert first absence to map for checking
				firstAbsence := absences[0].(map[string]interface{})
				assert.Equal(t, absence.ID.String(), firstAbsence["id"])
				assert.Equal(t, user.Username, firstAbsence["username"])
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
