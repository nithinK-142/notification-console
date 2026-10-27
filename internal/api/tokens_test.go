package api

import (
	"strings"
	"testing"
)

func TestValidateTokenRequest(t *testing.T) {
	tests := []struct {
		name    string
		request tokenRequest
		valid   bool
	}{
		{name: "android", request: tokenRequest{ExternalID: "user-1", Token: "fcm-token", Platform: "android"}, valid: true},
		{name: "ios", request: tokenRequest{ExternalID: "user-1", Token: "fcm-token", Platform: "ios"}, valid: true},
		{name: "web", request: tokenRequest{ExternalID: "user-1", Token: "fcm-token", Platform: "web"}, valid: true},
		{name: "missing external ID", request: tokenRequest{Token: "fcm-token", Platform: "android"}},
		{name: "missing token", request: tokenRequest{ExternalID: "user-1", Platform: "android"}},
		{name: "unsupported platform", request: tokenRequest{ExternalID: "user-1", Token: "fcm-token", Platform: "desktop"}},
		{name: "external ID too long", request: tokenRequest{ExternalID: strings.Repeat("u", 256), Token: "fcm-token", Platform: "android"}},
		{name: "token too long", request: tokenRequest{ExternalID: "user-1", Token: strings.Repeat("t", 4097), Platform: "android"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateTokenRequest(test.request)
			if test.valid && err != nil {
				t.Fatalf("validateTokenRequest() error = %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("validateTokenRequest() error = nil, want validation error")
			}
		})
	}
}
