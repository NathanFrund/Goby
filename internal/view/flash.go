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
)

// GetFlashData retrieves success and error messages from the session.
// It returns a partials.FlashData struct expected by the Base layout component.
// CRITICAL: This function consumes the flash messages (they are deleted after retrieval).
func GetFlashData(c echo.Context) partials.FlashData {
	// 1. FIX: Get the session store using session.Get(name, context)
	sess, err := session.Get(flashSessionName, c)
	if err != nil {
		c.Logger().Errorf("Failed to get session for flash messages: %v", err)
		return partials.FlashData{}
	}

	flash := partials.FlashData{}
	needsSave := false

	// 2. Retrieve Success messages and cast them to the correct type (string).
	// sess.Flashes() retrieves the messages and simultaneously clears them from the session map.
	successVal := sess.Flashes(flashSuccessKey)
	if len(successVal) > 0 {
		for _, val := range successVal {
			if s, ok := val.(string); ok {
				flash.Success = append(flash.Success, s)
			}
		}
		needsSave = true
	}

	// 3. Retrieve Error messages and cast them to the correct type (string).
	errorVal := sess.Flashes(flashErrorKey)
	if len(errorVal) > 0 {
		for _, val := range errorVal {
			if s, ok := val.(string); ok {
				flash.Error = append(flash.Error, s)
			}
		}
		needsSave = true
	}

	// 4. Save the session to commit the clearing of flashes.
	if needsSave {
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			c.Logger().Errorf("Failed to save session after consuming flashes: %v", err)
		}
	}

	return flash
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

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		c.Logger().Errorf("Failed to save session after setting success flash: %v", err)
	}
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

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		c.Logger().Errorf("Failed to save session after setting error flash: %v", err)
	}
}
