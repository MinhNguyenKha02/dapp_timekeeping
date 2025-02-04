package handlers

import (
	"dapp_timekeeping/types"

	"github.com/gofiber/fiber/v2"
)

// Rankings
func GetWorkHoursRanking(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}
