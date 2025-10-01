package auth

// LoginData is a View Model (DTO) used specifically for the login template.
// It simplifies data passed from the handler, such as a previously submitted email.
type LoginData struct {
	Email string
}

// ForgotPasswordData is used to transfer data (like a pre-filled email) to the forgot password template.
type ForgotPasswordData struct {
	Email string
}

// ResetPasswordData is used to transfer the necessary token to the password reset form.
type ResetPasswordData struct {
	Token string
}

// RegisterData is used to transfer data (like the pre-filled email) to the registration template.
type RegisterData struct {
	Email string
}
