package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"dapp_timekeeping/types"
	"dapp_timekeeping/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// ActiveCode represents the current active login code
type ActiveCode struct {
	Code      string `json:"code"`
	CreatedAt int64  `json:"created_at"`
}

// In-memory cache for login code
var (
	activeCode ActiveCode
	codeMutex  sync.RWMutex
)

// generateNewCode creates a new random code and updates the active code
func generateNewCode() error {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}
	code := base64.URLEncoding.EncodeToString(bytes)[:8]

	codeMutex.Lock()
	activeCode = ActiveCode{
		Code:      code,
		CreatedAt: time.Now().Unix(),
	}
	codeMutex.Unlock()
	return nil
}

// GetActiveCode returns or generates the active code (root only)
func GetActiveCode(c *fiber.Ctx) error {
	if c.Locals("claims").(jwt.MapClaims)["role"] != "root" {
		return c.Status(403).JSON(types.APIResponse{
			Success: false,
			Error:   "Only root can view active code",
		})
	}

	codeMutex.RLock()
	code := activeCode
	codeMutex.RUnlock()

	if code.Code == "" {
		if err := generateNewCode(); err != nil {
			return c.Status(500).JSON(types.APIResponse{
				Success: false,
				Error:   "internal server error",
			})
		}
		code = activeCode
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Data:    code,
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

	codeMutex.RLock()
	valid := req.Code == activeCode.Code
	codeMutex.RUnlock()

	if !valid {
		return c.Status(401).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid login code",
		})
	}

	// Generate JWT token
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["role"] = "employee"
	t, err := token.SignedString([]byte(utils.Config.JWTSecret))
	if err != nil {
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   "internal server error",
		})
	}

	// Generate new code after successful login
	if err := generateNewCode(); err != nil {
		utils.Logger.Error("Failed to generate new code", zap.Error(err))
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"token": t,
			"newCode": map[string]interface{}{
				"code":      activeCode.Code,
				"createdAt": activeCode.CreatedAt,
			},
		},
	})
}
