package view

// Data is a View Model (DTO) used specifically for the dashboard template.
// It simplifies the data received from the domain layer into simple string types
// for safe and easy rendering in the template.
type Data struct {
	ID                string
	Email             string
	ProfilePictureURL string // URL to the user's profile picture, if available.
}
