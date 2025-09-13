package view

import (
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

const (
	flashSessionName = "flash-session"
	flashKeySuccess  = "success"
	flashKeyError    = "error"
)

// setFlash sets a flash message in the session.
func setFlash(c echo.Context, key, message string) {
	sess, _ := session.Get(flashSessionName, c)
	sess.AddFlash(message, key)
	sess.Save(c.Request(), c.Response())
}

// SetFlashSuccess sets a success flash message.
func SetFlashSuccess(c echo.Context, message string) {
	setFlash(c, flashKeySuccess, message)
}

// SetFlashError sets an error flash message.
func SetFlashError(c echo.Context, message string) {
	setFlash(c, flashKeyError, message)
}

// GetFlashes retrieves and clears flash messages from the session.
func GetFlashes(c echo.Context) map[string][]interface{} {
	// The map we will return.
	flashes := make(map[string][]interface{})

	sess, _ := session.Get(flashSessionName, c)

	// Get flashes for both success and error keys.
	// The Flashes() method retrieves and then clears the flashes from the session.
	successFlashes := sess.Flashes(flashKeySuccess)
	errorFlashes := sess.Flashes(flashKeyError)

	// If we have flashes, save the session to persist the clearing of flashes.
	if len(successFlashes) > 0 || len(errorFlashes) > 0 {
		flashes[flashKeySuccess] = successFlashes
		flashes[flashKeyError] = errorFlashes
		_ = sess.Save(c.Request(), c.Response())
	}
	return flashes
}
