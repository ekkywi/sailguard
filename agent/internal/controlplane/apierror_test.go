package controlplane

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ekkywi/sailguard/pkgs/agentcontract"
)

func TestAPIErrorMapsCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		code   string
		want   error
	}{
		{"device disabled", http.StatusForbidden, "device_disabled", ErrDeviceDisabled},
		{"device unauthorized", http.StatusUnauthorized, "unauthorized", ErrUnauthorized},
		{"enrollment invalid", http.StatusUnauthorized, "enrollment_invalid", ErrEnrollmentInvalid},
		{"bare 401", http.StatusUnauthorized, "", ErrUnauthorized},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var ae *agentcontract.Error
			if tc.code != "" {
				ae = &agentcontract.Error{Code: tc.code, Message: "x"}
			}
			err := apiError(tc.status, ae, nil, "op")
			if !errors.Is(err, tc.want) {
				t.Fatalf("apiError(...) = %v, want wrap %v", err, tc.want)
			}
		})
	}
}
