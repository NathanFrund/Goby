package view

import (
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/web/src/templates/partials"

	// FIX: Import the session package to use the session.Get() helper
	"github.com/labstack/echo-contrib/session"
)

// These constants define the keys used to store flash messages in the session.
const (
	// Using a generic name for the session store, assuming middleware setup
	flashSessionName = "flash-session"
	flashSuccessKey  = "flash_success"
	flashErrorKey    = "flash_error"
	flashFormEmail   = "form_email"
)

// FlashData holds all flash messages for the view.
type FlashData struct {
	Messages  partials.FlashData
	FormEmail string
}

// GetFlashData retrieves all flash messages (success, error, and form data) from the session.
// It returns a single FlashData struct containing all retrieved messages.
// CRITICAL: This function consumes the flash messages (they are deleted after retrieval).
func GetFlashData(c echo.Context) FlashData {
	// 1. FIX: Get the session store using session.Get(name, context)
	sess, err := session.Get(flashSessionName, c)
	if err != nil {
		c.Logger().Errorf("Failed to get session for flash messages: %v", err)
		return FlashData{}
	}

	data := FlashData{}
	needsSave := false

	// 2. Retrieve Success messages and cast them to the correct type (string).
	// sess.Flashes() retrieves the messages and simultaneously clears them from the session map.
	successVal := sess.Flashes(flashSuccessKey)
	if len(successVal) > 0 {
		for _, val := range successVal {
			if s, ok := val.(string); ok {
				data.Messages.Success = append(data.Messages.Success, s)
			}
		}
		needsSave = true
	}

	// 3. Retrieve Error messages and cast them to the correct type (string).
	errorVal := sess.Flashes(flashErrorKey)
	if len(errorVal) > 0 {
		for _, val := range errorVal {
			if s, ok := val.(string); ok {
				data.Messages.Error = append(data.Messages.Error, s)
			}
		}
		needsSave = true
	}

	// 3. Retrieve form-specific flashes like pre-filled email.
	formEmailVal := sess.Flashes(flashFormEmail)
	if len(formEmailVal) > 0 {
		if s, ok := formEmailVal[0].(string); ok {
			data.FormEmail = s
		}
		needsSave = true
	}

	// 4. CRITICAL: Save the session only once to commit the clearing of all flashes.
	if needsSave {
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			c.Logger().Errorf("Failed to save session after consuming flashes: %v", err)
		}
	}

	return data
}

// SetFlashSuccess adds a success message to the session for the next request.
func SetFlashSuccess(c echo.Context, message string) {
	// FIX: Use session.Get()
	sess, err := session.Get(flashSessionName, c)
	if err != nil {
		c.Logger().Errorf("Failed to get session for flash success: %v", err)
		return
	}

	sess.AddFlash(message, flashSuccessKey)
}

// SetFlashError adds an error message to the session for the next request.
func SetFlashError(c echo.Context, message string) {
	// FIX: Use session.Get()
	sess, err := session.Get(flashSessionName, c)
	if err != nil {
		c.Logger().Errorf("Failed to get session for flash error: %v", err)
		return
	}

	sess.AddFlash(message, flashErrorKey)
}

// SetFlashFormData adds form-related data to the session without saving it.
// This is useful when you need to set multiple flash values before a single save
// operation, typically right before a redirect.
func SetFlashFormData(c echo.Context, key string, value string) {
	sess, err := session.Get(flashSessionName, c)
	if err != nil {
		c.Logger().Errorf("Failed to get session for flash form data: %v", err)
		return
	}

	sess.AddFlash(value, key)
}

// SaveFlashes commits all pending flash messages to the session.
// This should be called once in a handler after all flash messages have been set,
// typically right before a redirect.
func SaveFlashes(c echo.Context) error {
	sess, err := session.Get(flashSessionName, c)
	if err != nil {
		c.Logger().Errorf("Failed to get session to save flashes: %v", err)
		return err
	}

	return sess.Save(c.Request(), c.Response())
}
