package auth

type LoginRequest struct {
	Secret string `json:"secret" binding:"required"`
}

type LoginResponse struct {
	Token   string `json:"token"`
	Message string `json:"message"`
}
