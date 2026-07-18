package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateClientName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"Acme Corp", true},
		{"a", true},
		{"", false},
		{"   ", false},
		{"\t\n", false},
		{strings.Repeat("a", 255), true},
		{strings.Repeat("a", 256), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClientName(tt.name)
			if tt.valid && err != nil {
				t.Errorf("expected nil, got %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Errorf("error must wrap ErrValidation, got %v", err)
			}
		})
	}
}

func TestValidateClientStatus(t *testing.T) {
	tests := []struct {
		status ClientStatus
		valid  bool
	}{
		{ClientStatusActive, true},
		{ClientStatusInactive, true},
		{"pending", false},
		{"", false},
		{"ACTIVE", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			err := ValidateClientStatus(tt.status)
			if tt.valid && err != nil {
				t.Errorf("expected nil, got %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Errorf("error must wrap ErrValidation, got %v", err)
			}
		})
	}
}

func TestValidateAPIIdentifier(t *testing.T) {
	tests := []struct {
		api   string
		valid bool
	}{
		{"openai", true},
		{"gpt-4", true},
		{"my_api_1", true},
		{"a", true},
		{strings.Repeat("a", 64), true},
		{"", false},
		{"Open AI", false},
		{"has space", false},
		{"UPPERCASE", false},
		{strings.Repeat("a", 65), false},
		{"-no-leading-dash", false},
		{"_no_leading_underscore", false},
	}

	for _, tt := range tests {
		t.Run(tt.api, func(t *testing.T) {
			err := ValidateAPIIdentifier(tt.api)
			if tt.valid && err != nil {
				t.Errorf("expected nil, got %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Errorf("error must wrap ErrValidation, got %v", err)
			}
		})
	}
}

func TestValidateRateRule(t *testing.T) {
	validRule := RateRule{
		ClientID:        uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		API:             "openai",
		RequestsAllowed: 100,
		WindowSeconds:   60,
	}

	t.Run("valid rule returns nil", func(t *testing.T) {
		if err := ValidateRateRule(validRule); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("requests_allowed <= 0", func(t *testing.T) {
		r := validRule
		r.RequestsAllowed = 0
		if err := ValidateRateRule(r); err == nil {
			t.Error("expected error for requests_allowed=0")
		}
		r.RequestsAllowed = -1
		if err := ValidateRateRule(r); err == nil {
			t.Error("expected error for requests_allowed=-1")
		}
	})

	t.Run("window_seconds <= 0", func(t *testing.T) {
		r := validRule
		r.WindowSeconds = 0
		if err := ValidateRateRule(r); err == nil {
			t.Error("expected error for window_seconds=0")
		}
		r.WindowSeconds = -1
		if err := ValidateRateRule(r); err == nil {
			t.Error("expected error for window_seconds=-1")
		}
	})

	t.Run("window_seconds > 86400", func(t *testing.T) {
		r := validRule
		r.WindowSeconds = 86401
		if err := ValidateRateRule(r); err == nil {
			t.Error("expected error for window_seconds=86401")
		}
	})

	t.Run("api empty", func(t *testing.T) {
		r := validRule
		r.API = ""
		if err := ValidateRateRule(r); err == nil {
			t.Error("expected error for empty api")
		}
	})

	t.Run("api fails slug regex", func(t *testing.T) {
		r := validRule
		r.API = "Open AI"
		if err := ValidateRateRule(r); err == nil {
			t.Error("expected error for invalid api slug")
		}
	})

	t.Run("client_id is zero", func(t *testing.T) {
		r := validRule
		r.ClientID = uuid.Nil
		if err := ValidateRateRule(r); err == nil {
			t.Error("expected error for zero client_id")
		}
	})

	t.Run("all fields invalid returns joined error mentioning every field", func(t *testing.T) {
		r := RateRule{
			RequestsAllowed: -5,
			WindowSeconds:   99999,
			API:             "",
			ClientID:        uuid.Nil,
		}
		err := ValidateRateRule(r)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, ErrValidation) {
			t.Errorf("must wrap ErrValidation")
		}

		msg := err.Error()
		checks := []string{
			"requests_allowed",
			"window_seconds",
			"api",
			"client_id",
		}
		for _, c := range checks {
			if !strings.Contains(msg, c) {
				t.Errorf("joined error should mention %q, got: %s", c, msg)
			}
		}
	})
}

func TestClientStatusIsValid(t *testing.T) {
	if ClientStatus("pending").IsValid() {
		t.Error("expected 'pending' to be invalid")
	}
	if !ClientStatusActive.IsValid() {
		t.Error("expected 'active' to be valid")
	}
	if !ClientStatusInactive.IsValid() {
		t.Error("expected 'inactive' to be valid")
	}
	if ClientStatus("").IsValid() {
		t.Error("expected empty to be invalid")
	}
}
