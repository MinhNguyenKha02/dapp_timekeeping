package handlers

import (
	"github.com/gofiber/fiber/v2"
)

func UpdateCompanyRule(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Not implemented",
	})
}

func RecordViolation(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Not implemented",
	})
}

func GenerateReports(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Not implemented",
	})
}
