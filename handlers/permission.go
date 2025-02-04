package handlers

import (
	"dapp_timekeeping/types"

	"github.com/gofiber/fiber/v2"
)

// Permissions
func GrantPermission(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}

func RevokePermission(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}