package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// EmailSender defines the interface for sending emails. This allows for
// different implementations (e.g., for logging, Resend, Mailgun).
type EmailSender interface {
	Send(to, subject, htmlBody string) error
}

// --- LogSender (for development) ---

// LogSender prints emails to the console instead of sending them.
type LogSender struct {
	senderAddress string
}

// Send logs the email content to the standard output.
func (s *LogSender) Send(to, subject, htmlBody string) error {
	slog.Info("--- Email Sent (Logged) ---")
	slog.Info("From", "address", s.senderAddress)
	slog.Info("To", "address", to)
	slog.Info("Subject", "subject", subject)
	slog.Info("Body (HTML)", "body", htmlBody)
	slog.Info("---------------------------")
	return nil
}

// --- ResendSender (for production) ---

// ResendSender sends emails using the Resend API.
type ResendSender struct {
	apiKey        string
	senderAddress string
}

type resendPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

// Send dispatches an email using the Resend API.
func (s *ResendSender) Send(to, subject, htmlBody string) error {
	sender := s.senderAddress
	if sender == "" {
		sender = "Goby <onboarding@resend.dev>" // Default sender for testing with Resend
	}

	payload := resendPayload{
		From:    sender,
		To:      to,
		Subject: subject,
		HTML:    htmlBody,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal resend payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create resend request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// In a real app, you'd parse the error body here for more details.
		return fmt.Errorf("resend API returned an error: status %d", resp.StatusCode)
	}

	slog.Info("Successfully sent email via Resend", "to", to, "subject", subject)
	return nil
}
