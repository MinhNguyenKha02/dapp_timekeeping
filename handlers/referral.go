package handlers

import (
	"dapp_timekeeping/types"

	"github.com/gofiber/fiber/v2"
)

// Referral Management
func GenerateReferralCode(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}

func ListReferralCodes(c *fiber.Ctx) error {
	return c.JSON(types.APIResponse{
		Success: true,
		Message: "Not implemented",
	})
}