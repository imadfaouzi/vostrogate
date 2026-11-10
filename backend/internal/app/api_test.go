package app

import (
	"net/http/httptest"
	"testing"
)

func TestSetupNeedsToken(t *testing.T) {
	t.Setenv("VOSTROGATE_SETUP_TOKEN", "abcdefghijklmnopqrstuvwxyz0123456789")
	req := httptest.NewRequest("POST", "/api/setup/verify", nil)
	if validateSetupToken(req) {
		t.Fatal("setup accepted missing token")
	}
	req.Header.Set("X-Setup-Token", "abcdefghijklmnopqrstuvwxyz0123456789")
	if !validateSetupToken(req) {
		t.Fatal("setup rejected valid token")
	}
}
