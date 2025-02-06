package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"sync"

	"dapp_timekeeping/types"
	"dapp_timekeeping/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// In-memory cache for login codes
var (
	loginCodes = make(map[string]string) // code -> random unique string
	codeMutex  sync.RWMutex
)

// GenerateLoginCode generates a random code for employee login
func GenerateLoginCode(c *fiber.Ctx) error {
	var claims jwt.MapClaims
	if c.Locals("claims") != nil {
		claims = c.Locals("claims").(jwt.MapClaims)
	} else {
		// For testing purposes
		claims = jwt.MapClaims{
			"role": c.Get("X-Test-Role", ""),
		}
	}

	if claims["role"] != "root" {
		return c.Status(403).JSON(types.APIResponse{
			Success: false,
			Error:   "Only root can generate login codes",
		})
	}

	// Generate random 8-character code
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		utils.Logger.Error("Failed to generate random bytes", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
	}
	code := base64.URLEncoding.EncodeToString(bytes)[:8]

	// Store in memory cache
	codeMutex.Lock()
	loginCodes[code] = "active"
	codeMutex.Unlock()

	return c.JSON(types.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"code": code,
		},
	})
}

// LoginWithCode handles employee login using generated code
func LoginWithCode(c *fiber.Ctx) error {
	var req struct {
		Code string `json:"code" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Validate code
	codeMutex.RLock()
	_, valid := loginCodes[req.Code]
	codeMutex.RUnlock()

	if !valid {
		return c.Status(401).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid login code",
		})
	}

	// Generate new code for next login
	newBytes := make([]byte, 6)
	if _, err := rand.Read(newBytes); err != nil {
		utils.Logger.Error("Failed to generate new code", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
	}
	newCode := base64.URLEncoding.EncodeToString(newBytes)[:8]

	// Update code in cache
	codeMutex.Lock()
	delete(loginCodes, req.Code)
	loginCodes[newCode] = "active"
	codeMutex.Unlock()

	// Generate JWT token for the employee
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["role"] = "employee"
	// Add other necessary claims

	t, err := token.SignedString([]byte(utils.Config.JWTSecret))
	if err != nil {
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"token":   t,
			"newCode": newCode,
		},
	})
}
