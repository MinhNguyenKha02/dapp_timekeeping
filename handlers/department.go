package handlers

import (
	"dapp_timekeeping/types"

	"github.com/gofiber/fiber/v2"
)

// Department Management
func GetDepartmentStats(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}

func GetDepartmentAttendance(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}