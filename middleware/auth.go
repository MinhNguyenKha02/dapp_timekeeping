package middleware

import (
	"strings"

	"dapp_timekeeping/config"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func extractToken(c *fiber.Ctx) (string, error) {
	auth := c.Get("Authorization")
	if auth == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "No token provided")
	}

	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Invalid token format")
	}

	return parts[1], nil
}

func RequireAuth(c *fiber.Ctx) error {
	token, err := extractToken(c)
	if err != nil {
		return err
	}

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	// Add claims to context for use in handlers
	c.Locals("user_id", claims["user_id"])
	c.Locals("role", claims["role"])

	return c.Next()
}

func RequireRoot(c *fiber.Ctx) error {
	if err := RequireAuth(c); err != nil {
		return err
	}

	if c.Locals("role") != "root" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Root access required",
		})
	}

	return c.Next()
}

func RequireHR(c *fiber.Ctx) error {
	if err := RequireAuth(c); err != nil {
		return err
	}

	role := c.Locals("role").(string)
	if role != "hr_manager" && role != "root" {
		return c.Status(403).JSON(fiber.Map{
			"error": "HR access required",
		})
	}

	return c.Next()
}
