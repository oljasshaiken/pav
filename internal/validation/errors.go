package validation

import "errors"

// ErrValidationFailed is returned when pre- or post-transform validation fails.
var ErrValidationFailed = errors.New("validation failed")
