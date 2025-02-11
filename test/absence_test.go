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
		Status:            "active",
	}
	if err := db.Create(&rootUser).Error; err != nil {
		t.Fatalf("Failed to create root user: %v", err)
	}

	// Create test user
	employee1 := models.User{
		ID:                uuid.New().String(),
		FullName:          "Test Employee 1",
		Email:             "emp1@company.com",
		PhoneNumber:       "+9876543210",
		Address:           "456 Test St",
		DateOfBirth:       time.Now().AddDate(-25, 0, 0),
		Gender:            "female",
		TaxID:             "TAX456",
		HealthInsuranceID: "HI456",
		SocialInsuranceID: "SI456",
		Position:          "Developer",
		Location:          "Branch A",
		OnboardDate:       time.Now(),
		Role:              "employee",
		Department:        "IT",
		WalletAddress:     "0x456...",
		Salary:            5000,
		Status:            "active",
	}

	// Create and verify in transaction
	tx := db.Begin()
	if err := tx.Create(&employee1).Error; err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create absence with root user as processor
	absence := models.Absence{
		ID:     uuid.New().String(),
		UserID: employee1.ID,
		Date:   time.Now(),
		Type:   "without_permission",
		Reason: "Personal emergency",
		Status: "pending",
		// Don't set ProcessedBy for pending status
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
				assert.Equal(t, employee1.FullName, firstAbsence["full_name"])
				assert.Equal(t, employee1.Department, firstAbsence["department"])
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
	db.Unscoped().Delete(&employee1)
	db.Unscoped().Delete(&rootUser)

	// Verify cleanup
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.Absence{}).Count(&absenceCount)
	t.Logf("After cleanup - Users: %d, Absences: %d", userCount, absenceCount)
}

func TestGetEmployeeStatistics(t *testing.T) {
	app, db := SetupTest(t)
	app.Get("/statistics", handlers.GetEmployeeStatistics)

	// Create test users with transactions
	tx := db.Begin()

	// Create root user
	rootUser := models.User{
		ID:          uuid.New().String(),
		FullName:    "Root Admin",
		Email:       "root@company.com",
		PhoneNumber: "+1234567890",
		Role:        "root",
		Department:  "Management",
		Status:      "active",
	}
	if err := tx.Create(&rootUser).Error; err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create root user: %v", err)
	}

	// Create HR manager
	hrManager := models.User{
		ID:          uuid.New().String(),
		FullName:    "HR Manager",
		Email:       "hr@company.com",
		PhoneNumber: "+1234567891",
		Role:        "hr_manager",
		Department:  "HR",
		Status:      "active",
	}
	if err := tx.Create(&hrManager).Error; err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create HR manager: %v", err)
	}

	// Create test employee
	employee1 := models.User{
		ID:          uuid.New().String(),
		FullName:    "Test Employee 1",
		Email:       "emp1@company.com",
		PhoneNumber: "+9876543210",
		Role:        "employee",
		Department:  "IT",
		Status:      "active",
	}
	if err := tx.Create(&employee1).Error; err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create employee: %v", err)
	}

	// Create test absences with correct data
	absences := []models.Absence{
		{
			ID:          uuid.New().String(),
			UserID:      employee1.ID,
			Date:        time.Now(),
			Type:        "with_permission",
			Reason:      "Annual leave",
			Status:      "approved",
			ProcessedBy: strPtr(hrManager.ID),
			ProcessedAt: ptr(time.Now()),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:        uuid.New().String(),
			UserID:    employee1.ID,
			Date:      time.Now(),
			Type:      "without_permission",
			Reason:    "Family emergency",
			Status:    "pending",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New().String(),
			UserID:    employee1.ID,
			Date:      time.Now(),
			Type:      "resign",
			Reason:    "Career change",
			Status:    "pending",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// After creating the data, let's verify it was saved correctly
	for _, absence := range absences {
		if err := tx.Create(&absence).Error; err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create absence: %v", err)
		}
		// Verify the data was saved
		var saved models.Absence
		if err := tx.First(&saved, "id = ?", absence.ID).Error; err != nil {
			tx.Rollback()
			t.Fatalf("Failed to verify absence: %v", err)
		}
		t.Logf("Created absence - ID: %s, Type: %s, Status: %s", saved.ID, saved.Type, saved.Status)
	}

	// Add a query to check all absences before committing
	var count int64
	tx.Model(&models.Absence{}).Count(&count)
	t.Logf("Total absences before commit: %d", count)

	// Create test attendances with precise time differences
	baseTime := time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC)
	attendances := []models.Attendance{
		{
			ID:           uuid.New().String(),
			UserID:       employee1.ID,
			CheckInTime:  baseTime.Add(15 * time.Minute),
			ExpectedTime: baseTime,
			// 15 minutes late
		},
		{
			ID:           uuid.New().String(),
			UserID:       employee1.ID,
			CheckInTime:  baseTime.Add(30 * time.Minute),
			ExpectedTime: baseTime,
			// 30 minutes late
		},
	}

	// Create attendances and verify
	for _, attendance := range attendances {
		if err := tx.Create(&attendance).Error; err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create attendance: %v", err)
		}
	}

	tx.Commit()

	// After commit, verify the data
	var absenceList []models.Absence
	db.Find(&absenceList)
	t.Logf("All absences after commit:")
	for _, a := range absenceList {
		t.Logf("ID: %s, Type: %s, Status: %s, ProcessedBy: %v", a.ID, a.Type, a.Status, a.ProcessedBy)
	}

	// Debug the actual SQL query
	var debugStats struct {
		WithPermission    int `gorm:"column:with_permission"`
		WithoutPermission int `gorm:"column:without_permission"`
		PendingLeaves     int `gorm:"column:pending_leaves"`
	}
	db.Model(&models.Absence{}).
		Select(`
			COUNT(CASE WHEN type = 'with_permission' AND status = 'approved' THEN 1 END) as with_permission,
			COUNT(CASE WHEN type = 'without_permission' THEN 1 END) as without_permission,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_leaves
		`).
		Scan(&debugStats)
	t.Logf("Debug stats: %+v", debugStats)

	// Generate auth token for root user
	token := createTestToken(rootUser.ID, "root")

	// Test the endpoint
	req := httptest.NewRequest("GET", "/statistics", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.Nil(t, err)
	assert.True(t, response.Success)

	// Verify statistics with updated expectations
	stats, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)

	// Debug output
	t.Logf("Leave Stats: %+v", stats["leave_stats"])
	t.Logf("Resign Stats: %+v", stats["resign_stats"])
	t.Logf("Late Stats: %+v", stats["late_stats"])

	// Check leave statistics
	leaveStats := stats["leave_stats"].(map[string]interface{})
	assert.Equal(t, float64(1), leaveStats["with_permission"], "Should have 1 approved with_permission leave")
	assert.Equal(t, float64(1), leaveStats["without_permission"], "Should have 1 without_permission leave")
	assert.Equal(t, float64(2), leaveStats["pending_leaves"], "Should have 2 pending leaves")

	// Check resign statistics
	resignStats := stats["resign_stats"].(map[string]interface{})
	assert.Equal(t, float64(0), resignStats["approved"], "Should have 0 approved resignations")
	assert.Equal(t, float64(1), resignStats["pending"], "Should have 1 pending resignation")
	assert.Equal(t, float64(1), resignStats["total"], "Should have 1 total resignation")

	// Check late statistics
	lateStats := stats["late_stats"].(map[string]interface{})
	assert.Equal(t, float64(2), lateStats["total_incidents"], "Should have 2 late incidents")
	assert.Equal(t, float64(1), lateStats["unique_employees"], "Should have 1 employee being late")
	assert.InDelta(t, float64(22.5), lateStats["average_minutes"], 0.1, "Average minutes late should be 22.5")

	// Add validation test
	t.Run("Validate ProcessedBy requirement", func(t *testing.T) {
		// Try to create an approved absence without ProcessedBy
		invalidAbsence := models.Absence{
			ID:     uuid.New().String(),
			UserID: employee1.ID,
			Date:   time.Now(),
			Type:   "with_permission",
			Status: "approved", // Approved but no ProcessedBy - should fail
		}

		err := db.Create(&invalidAbsence).Error
		assert.Error(t, err, "Should not allow approved absence without ProcessedBy")
	})

	t.Run("Validate Status Transitions", func(t *testing.T) {
		// Test pending status
		pendingAbsence := models.Absence{
			ID:     uuid.New().String(),
			UserID: employee1.ID,
			Date:   time.Now(),
			Type:   "with_permission",
			Reason: "Test reason",
			Status: "pending",
			// Don't set ProcessedBy for pending
		}

		err := db.Create(&pendingAbsence).Error
		assert.NoError(t, err)

		// Verify saved data
		var saved models.Absence
		err = db.First(&saved, "id = ?", pendingAbsence.ID).Error
		assert.NoError(t, err, "Should find the created absence")
		assert.Nil(t, saved.ProcessedBy)
		assert.Nil(t, saved.ProcessedAt)

		// Test approved status with ProcessedBy (should succeed)
		validApprovedAbsence := models.Absence{
			ID:          uuid.New().String(),
			UserID:      employee1.ID,
			Date:        time.Now(),
			Type:        "with_permission",
			Reason:      "Approved reason",
			Status:      "approved",
			ProcessedBy: strPtr(hrManager.ID),
		}
		err = db.Create(&validApprovedAbsence).Error
		assert.NoError(t, err)

		// Verify ProcessedAt was automatically set
		var savedApproved models.Absence
		err = db.First(&savedApproved, "id = ?", validApprovedAbsence.ID).Error
		assert.NoError(t, err, "Should find the approved absence")
		assert.NotNil(t, savedApproved.ProcessedBy)
		assert.NotNil(t, savedApproved.ProcessedAt)

		// Cleanup
		db.Unscoped().Delete(&models.Absence{}, "id IN (?)", []string{
			pendingAbsence.ID,
			validApprovedAbsence.ID,
		})
	})

	// Cleanup
	db.Unscoped().Delete(&models.Attendance{}, "user_id = ?", employee1.ID)
	db.Unscoped().Delete(&models.Absence{}, "user_id = ?", employee1.ID)
	db.Unscoped().Delete(&employee1)
	db.Unscoped().Delete(&hrManager)
	db.Unscoped().Delete(&rootUser)
}

// Helper function to create pointer to time.Time
func ptr(t time.Time) *time.Time {
	return &t
}

// Helper function to get string pointer
func strPtr(s string) *string {
	return &s
}
