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

	// Set up routes with middleware
	app.Get("/auth/active-code", func(c *fiber.Ctx) error {
		c.Locals("claims", jwt.MapClaims{
			"role": "root",
			"id":   rootUser.ID,
		})
		return handlers.GetActiveCode(c)
	})
	app.Post("/auth/login-with-code", handlers.LoginWithCode)

	// Generate root token for auth
	rootToken := createTestToken(rootUser.ID, "root")

	// Root gets initial active code
	req := httptest.NewRequest("GET", "/auth/active-code", nil)
	req.Header.Set("Authorization", "Bearer "+rootToken)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var codeResponse types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&codeResponse)
	assert.NoError(t, err)
	assert.True(t, codeResponse.Success)

	initialCodeData := codeResponse.Data.(map[string]interface{})
	code := initialCodeData["code"].(string)
	t.Logf("Initial active code generated: %s", code)
	t.Logf("Root sees initial code details: %+v", initialCodeData)

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

	// Get the new code from login response
	responseData := loginResponse.Data.(map[string]interface{})
	assert.NotEmpty(t, responseData["token"])
	newCodeData := responseData["newCode"].(map[string]interface{})
	assert.NotEmpty(t, newCodeData["code"])

	// After login, verify root sees the same new code
	req = httptest.NewRequest("GET", "/auth/active-code", nil)
	req.Header.Set("Authorization", "Bearer "+rootToken)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var activeCodeResp types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&activeCodeResp)
	assert.NoError(t, err)
	assert.True(t, activeCodeResp.Success)

	// Compare the code from login response with what root sees
	activeCode := activeCodeResp.Data.(map[string]interface{})
	assert.Equal(t, newCodeData["code"], activeCode["code"], "Root should see the same code as login response")
	t.Logf("Root can see active code: %v", activeCode)

	// Cleanup
	db.Unscoped().Delete(&rootUser)
}
