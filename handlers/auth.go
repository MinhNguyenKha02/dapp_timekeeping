package handlers

import (
	"dapp_timekeeping/config"
	"dapp_timekeeping/models"
	"dapp_timekeeping/types"
	"dapp_timekeeping/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type RegisterRequest struct {
	Username      string `json:"username" validate:"required"`
	Password      string `json:"password" validate:"required,min=6"`
	WalletAddress string `json:"wallet_address" validate:"required"`
	ReferralCode  string `json:"referral_code" validate:"required"`
}

func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		utils.Logger.Error("Failed to parse login request")
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrInvalidInput,
		})
	}

	var user models.User
	if err := DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return c.Status(401).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		utils.Logger.Error("Failed to generate token", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   "Could not generate token",
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Login successful",
		Data: fiber.Map{
			"token": tokenString,
			"user": fiber.Map{
				"id":       user.ID,
				"username": user.Username,
				"role":     user.Role,
			},
		},
	})
}

func Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		utils.Logger.Error("Failed to parse register request")
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrInvalidInput,
		})
	}

	// Check if username exists
	var existingUser models.User
	if err := DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return c.Status(400).JSON(types.APIResponse{
			Success: false,
			Error:   "Username already exists",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Logger.Error("Failed to hash password", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   "Failed to process registration",
		})
	}

	// Create user
	user := models.User{
		ID:            uuid.New(),
		Username:      req.Username,
		PasswordHash:  string(hashedPassword),
		WalletAddress: req.WalletAddress,
		Role:          "employee",
		Status:        "pending",
		LeaveBalance:  20,                      // Default leave balance
		ReferralCode:  uuid.New().String()[:8], // Generate referral code
	}

	if err := DB.Create(&user).Error; err != nil {
		utils.Logger.Error("Failed to create user", zap.Error(err))
		return c.Status(500).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrDatabaseError,
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Registration successful, waiting for approval",
		Data:    user,
	})
}

func Logout(c *fiber.Ctx) error {
	// Since we're using JWT, we just need to tell the client to remove the token
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

func VerifyToken(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(401).JSON(types.APIResponse{
			Success: false,
			Error:   types.ErrUnauthorized,
		})
	}

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		return c.Status(401).JSON(types.APIResponse{
			Success: false,
			Error:   "Invalid or expired token",
		})
	}

	var user models.User
	userID, _ := uuid.Parse(claims["user_id"].(string))
	if err := DB.First(&user, userID).Error; err != nil {
		return c.Status(401).JSON(types.APIResponse{
			Success: false,
			Error:   "User not found",
		})
	}

	return c.JSON(types.APIResponse{
		Success: true,
		Data: fiber.Map{
			"user_id":  user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}
