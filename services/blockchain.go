package services

import (
	"context"

	"github.com/dfinity/agent-go/agent"
	"github.com/dfinity/agent-go/candid"
	"github.com/dfinity/agent-go/principal"
)

type BlockchainService struct {
	agent    *agent.Agent
	canister principal.Principal
}

func NewBlockchainService(identity agent.Identity, canisterID string) (*BlockchainService, error) {
	// Create agent configuration
	config := agent.Config{
		Identity: identity,
		Host:     "https://ic0.app", // IC mainnet
	}

	// Create new agent
	agent, err := agent.New(config)
	if err != nil {
		return nil, err
	}

	// Parse canister ID
	canister, err := principal.Decode(canisterID)
	if err != nil {
		return nil, err
	}

	return &BlockchainService{
		agent:    agent,
		canister: canister,
	}, nil
}

// AddEmployee adds a new employee to the blockchain
func (s *BlockchainService) AddEmployee(ctx context.Context, walletAddress string, salary uint64) error {
	args := candid.Encode(
		walletAddress,
		salary,
	)

	_, err := s.agent.Call(ctx, s.canister, "add_employee", args)
	return err
}

// PaySalary records a salary payment on the blockchain
func (s *BlockchainService) PaySalary(ctx context.Context, employee string, amount, deductions, bonus uint64) error {
	args := candid.Encode(
		employee,
		amount,
		deductions,
		bonus,
	)

	_, err := s.agent.Call(ctx, s.canister, "pay_salary", args)
	return err
}

// UpdateCompanyRule updates a company rule on the blockchain
func (s *BlockchainService) UpdateCompanyRule(ctx context.Context, ruleID, details string) error {
	args := candid.Encode(
		ruleID,
		details,
	)

	_, err := s.agent.Call(ctx, s.canister, "update_company_rule", args)
	return err
}
