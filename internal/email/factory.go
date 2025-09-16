package email

import (
	"fmt"

	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/domain"
)

// NewEmailService creates and returns an email sender based on the configuration.
func NewEmailService(cfg config.Provider) (domain.EmailSender, error) {
	switch cfg.GetEmailProvider() {
	case "log":
		return &LogSender{senderAddress: cfg.GetEmailSender()}, nil
	case "resend":
		if cfg.GetEmailAPIKey() == "" {
			return nil, fmt.Errorf("email provider is 'resend' but EMAIL_API_KEY is not set")
		}
		return &ResendSender{apiKey: cfg.GetEmailAPIKey(), senderAddress: cfg.GetEmailSender()}, nil
	default:
		return nil, fmt.Errorf("unknown email provider: %s", cfg.GetEmailProvider())
	}
}
