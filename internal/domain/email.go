package domain

// EmailSender defines the interface for sending emails. This allows for
// different implementations (e.g., for logging, Resend, Mailgun).
type EmailSender interface {
	Send(to, subject, htmlBody string) error
}
