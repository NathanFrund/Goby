package email

import (
	"fmt"

	"github.com/nfrund/goby/internal/config"
)

// NewEmailService creates and returns an email sender based on the configuration.
func NewEmailService(cfg *config.Config) (EmailSender, error) {
	switch cfg.EmailProvider {
	case "log":
		return &LogSender{senderAddress: cfg.EmailSender}, nil
	case "resend":
		if cfg.EmailAPIKey == "" {
			return nil, fmt.Errorf("email provider is 'resend' but EMAIL_API_KEY is not set")
		}
		return &ResendSender{apiKey: cfg.EmailAPIKey, senderAddress: cfg.EmailSender}, nil
	default:
		return nil, fmt.Errorf("unknown email provider: %s", cfg.EmailProvider)
	}
}
