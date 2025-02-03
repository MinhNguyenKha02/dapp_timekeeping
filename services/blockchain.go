package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type BlockchainService struct {
	client   *http.Client
	endpoint string
	canister string
}

func NewBlockchainService(canisterID string) *BlockchainService {
	return &BlockchainService{
		client:   &http.Client{},
		endpoint: "https://ic0.app", // ICP mainnet
		canister: canisterID,
	}
}

// AddEmployee adds a new employee to the blockchain
func (s *BlockchainService) AddEmployee(ctx context.Context, walletAddress string, salary uint64) error {
	payload := map[string]interface{}{
		"method": "add_employee",
		"args": map[string]interface{}{
			"wallet_address": walletAddress,
			"salary":         salary,
		},
	}

	return s.callCanister(ctx, payload)
}

// PaySalary records a salary payment on the blockchain
func (s *BlockchainService) PaySalary(ctx context.Context, employee string, amount, deductions, bonus uint64) error {
	payload := map[string]interface{}{
		"method": "pay_salary",
		"args": map[string]interface{}{
			"employee":   employee,
			"amount":     amount,
			"deductions": deductions,
			"bonus":      bonus,
		},
	}

	return s.callCanister(ctx, payload)
}

func (s *BlockchainService) callCanister(ctx context.Context, payload map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v2/canister/%s/call", s.endpoint, s.canister)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("canister call failed with status: %d", resp.StatusCode)
	}

	return nil
}

// UpdateCompanyRule updates a company rule on the blockchain
func (s *BlockchainService) UpdateCompanyRule(ctx context.Context, ruleID, details string) error {
	payload := map[string]interface{}{
		"method": "update_company_rule",
		"args": map[string]interface{}{
			"rule_id": ruleID,
			"details": details,
		},
	}

	return s.callCanister(ctx, payload)
}
