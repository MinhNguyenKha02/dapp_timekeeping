package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func LoadTestConfig() error {
	// Find project root (where .env.test is located)
	projectRoot, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, ".env.test")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			return fmt.Errorf("could not find .env.test file")
		}
		projectRoot = parent
	}

	// Change to project root and load .env.test
	os.Chdir(projectRoot)
	if err := godotenv.Load(".env.test"); err != nil {
		return fmt.Errorf("error loading .env.test: %w", err)
	}

	// Verify required variables
	if os.Getenv("TEST_CANISTER_ID") == "" {
		return fmt.Errorf("TEST_CANISTER_ID not set in .env.test")
	}
	if os.Getenv("IC_HOST") == "" {
		return fmt.Errorf("IC_HOST not set in .env.test")
	}

	return nil
}
