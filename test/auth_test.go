package test

import (
	"bytes"
	"dapp_timekeeping/handlers"
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLoginCodeFlow(t *testing.T) {
	app, db := SetupTest(t)

	// Create root user first
	rootUser := models.User{
		ID:       uuid.New().String(),
		FullName: "Root Admin",
		Email:    "root@company.com",
		Role:     "root",
		Status:   "active",
	}
	if err := db.Create(&rootUser).Error; err != nil {
		t.Fatalf("Failed to create root user: %v", err)
	}
	t.Logf("Created root user: %+v", rootUser)

	// Then set up routes with middleware
	app.Post("/auth/generate-code", func(c *fiber.Ctx) error {
		// Set claims for root user
		c.Locals("claims", jwt.MapClaims{
			"role": "root",
			"id":   rootUser.ID,
		})
		return handlers.GenerateLoginCode(c)
	})
	app.Post("/auth/login-with-code", handlers.LoginWithCode)

	// Generate root token for auth
	rootToken := createTestToken(rootUser.ID, "root")

	// Root generates login code
	req := httptest.NewRequest("POST", "/auth/generate-code", nil)
	req.Header.Set("Authorization", "Bearer "+rootToken)
	req.Header.Set("X-Test-Role", "root") // Add test role header
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response to get code
	var genResponse types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&genResponse)
	assert.NoError(t, err)
	assert.True(t, genResponse.Success)

	code := genResponse.Data.(map[string]interface{})["code"].(string)
	t.Logf("Generated login code: %s", code)

	// Simulate employee using the code to login
	loginReq := struct {
		Code string `json:"code"`
	}{
		Code: code,
	}
	body, _ := json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/auth/login-with-code", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify login response
	var loginResponse types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	assert.NoError(t, err)
	assert.True(t, loginResponse.Success)

	// Log response data
	t.Logf("Login response: %+v", loginResponse.Data)

	// Verify we got a token and new code
	responseData := loginResponse.Data.(map[string]interface{})
	assert.NotEmpty(t, responseData["token"])
	newCode := responseData["newCode"].(string)
	assert.NotEmpty(t, newCode)
	t.Logf("New code for next login: %s", newCode)

	// Verify new code works for login
	loginReq.Code = newCode
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/auth/login-with-code", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "New code should work for login")
	t.Log("Verified new code is valid and stored in cache")

	// Try using old code (should fail)
	loginReq.Code = code
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/auth/login-with-code", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
	t.Log("Verified old code is invalidated")

	// Cleanup
	db.Unscoped().Delete(&rootUser)
}
