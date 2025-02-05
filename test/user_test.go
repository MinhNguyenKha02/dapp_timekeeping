package test

import (
	"testing"
	"time"

	"dapp_timekeeping/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserModel(t *testing.T) {
	app, db := SetupTest(t)
	defer app.Shutdown()

	user := models.User{
		ID:                uuid.New().String(),
		FullName:          "Test User",
		Email:             "test@company.com",
		PhoneNumber:       "+1234567890",
		Address:           "789 Test St",
		DateOfBirth:       time.Now().AddDate(-25, 0, 0),
		Gender:            "male",
		TaxID:             "TAX789",
		HealthInsuranceID: "HI789",
		SocialInsuranceID: "SI789",
		Position:          "Developer",
		Location:          "Branch B",
		OnboardDate:       time.Now(),
		Role:              "employee",
		Department:        "IT",
		WalletAddress:     "0x789...",
		Salary:            6000,
		LeaveBalance:      20,
		Status:            "active",
	}

	err := db.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Logf("Created user with ID: %v", user.ID)

	// Verify user was created
	var found models.User
	err = db.First(&found, "id = ?", user.ID).Error
	if err != nil {
		t.Fatalf("Failed to find created user: %v", err)
	}

	// Assert all fields match
	assert.Equal(t, user.FullName, found.FullName)
	assert.Equal(t, user.Email, found.Email)
	assert.Equal(t, user.PhoneNumber, found.PhoneNumber)
	assert.Equal(t, user.Address, found.Address)
	assert.Equal(t, user.Gender, found.Gender)
	assert.Equal(t, user.TaxID, found.TaxID)
	assert.Equal(t, user.Position, found.Position)
	assert.Equal(t, user.Department, found.Department)
	assert.Equal(t, user.Role, found.Role)
	assert.Equal(t, user.Status, found.Status)
}
