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
	t.Logf("Created test user with ID: %s", user.ID)

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
	t.Logf("Created test absence with ID: %s", absence.ID)

	// Create root user for authentication
	rootUser := models.User{
		ID:       uuid.New(),
		Username: "root_user",
		Role:     "root",
		Status:   "active",
	}
	result = db.Create(&rootUser)
	assert.NoError(t, result.Error)
	t.Logf("Created root user with ID: %s", rootUser.ID)

	// Generate test token
	token := createTestToken(rootUser.ID.String(), "root")

	// Set up routes for this test
	root := app.Group("/root")
	attendance := root.Group("/attendance")
	attendance.Get("/absences", handlers.GetAbsences)

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
			req := httptest.NewRequest("GET", "/root/attendance/absences"+tt.queryParams, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response types.APIResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			tt.checkResponse(t, response)
		})
	}

	// Cleanup
	db.Unscoped().Delete(&absence)
	db.Unscoped().Delete(&user)
	db.Unscoped().Delete(&rootUser)
}
