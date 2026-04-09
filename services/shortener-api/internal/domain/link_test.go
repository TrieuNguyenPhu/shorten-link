package domain

import (
	"errors"
	"testing"
	"time"
)

func TestLinkStatusAndResolution(t *testing.T) {
	now := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
	expiredAt := now
	future := now.Add(time.Hour)

	tests := []struct {
		name       string
		link       Link
		wantStatus LinkStatus
		wantError  error
	}{
		{
			name:       "active without expiration",
			link:       Link{Enabled: true},
			wantStatus: LinkStatusActive,
		},
		{
			name:       "active before expiration",
			link:       Link{Enabled: true, ExpiresAt: &future},
			wantStatus: LinkStatusActive,
		},
		{
			name:       "expired at boundary",
			link:       Link{Enabled: true, ExpiresAt: &expiredAt},
			wantStatus: LinkStatusExpired,
			wantError:  ErrLinkExpired,
		},
		{
			name:       "disabled takes precedence",
			link:       Link{Enabled: false, ExpiresAt: &expiredAt},
			wantStatus: LinkStatusDisabled,
			wantError:  ErrLinkDisabled,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.link.StatusAt(now); got != test.wantStatus {
				t.Fatalf("StatusAt() = %q, want %q", got, test.wantStatus)
			}
			if err := test.link.EnsureResolvableAt(now); !errors.Is(err, test.wantError) {
				t.Fatalf("EnsureResolvableAt() error = %v, want %v", err, test.wantError)
			}
		})
	}
}
