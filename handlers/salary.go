package handlers

import (
	"dapp_timekeeping/types"

	"github.com/gofiber/fiber/v2"
)

// Salary Management
func UpdateSalary(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}

func GetPendingSalaryApprovals(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}

func ApproveSalary(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}