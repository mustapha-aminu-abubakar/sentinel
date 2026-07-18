package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var apiIdentifierRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func ValidateClientName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: client name must not be empty", ErrValidation)
	}
	if len(name) > 255 {
		return fmt.Errorf("%w: client name must not exceed 255 characters", ErrValidation)
	}
	return nil
}

func ValidateClientStatus(s ClientStatus) error {
	if !s.IsValid() {
		return fmt.Errorf("%w: invalid client status %q; must be 'active' or 'inactive'", ErrValidation, string(s))
	}
	return nil
}

func ValidateAPIIdentifier(api string) error {
	if api == "" {
		return fmt.Errorf("%w: api identifier must not be empty", ErrValidation)
	}
	if !apiIdentifierRegex.MatchString(api) {
		return fmt.Errorf("%w: api identifier %q must match %s", ErrValidation, api, apiIdentifierRegex.String())
	}
	return nil
}

func ValidateRateRule(r RateRule) error {
	var errs []error

	if r.RequestsAllowed <= 0 {
		errs = append(errs, fmt.Errorf("%w: requests_allowed must be > 0, got %d", ErrValidation, r.RequestsAllowed))
	}

	if r.WindowSeconds <= 0 {
		errs = append(errs, fmt.Errorf("%w: window_seconds must be > 0, got %d", ErrValidation, r.WindowSeconds))
	} else if r.WindowSeconds > 86400 {
		errs = append(errs, fmt.Errorf("%w: window_seconds must not exceed 86400, got %d", ErrValidation, r.WindowSeconds))
	}

	if r.API == "" {
		errs = append(errs, fmt.Errorf("%w: api must not be empty", ErrValidation))
	} else if !apiIdentifierRegex.MatchString(r.API) {
		errs = append(errs, fmt.Errorf("%w: api %q must match %s", ErrValidation, r.API, apiIdentifierRegex.String()))
	}

	if r.ClientID == uuid.Nil {
		errs = append(errs, fmt.Errorf("%w: client_id must not be zero", ErrValidation))
	}

	return errors.Join(errs...)
}
