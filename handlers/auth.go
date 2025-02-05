package handlers

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type RegisterRequest struct {
	FullName      string `json:"full_name" validate:"required"`
	Email         string `json:"email" validate:"required,email"`
	PhoneNumber   string `json:"phone_number" validate:"required"`
	Gender        string `json:"gender" validate:"required,oneof=male female other"`
	Position      string `json:"position" validate:"required"`
	Department    string `json:"department" validate:"required"`
	WalletAddress string `json:"wallet_address" validate:"required"`
	Password      string `json:"password" validate:"required,min=6"`
}

