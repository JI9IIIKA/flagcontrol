package flagsdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLastKnownConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tenant":"t","key":"feature","environment":"prod","version":1,"enabled":true,"percentage":100,"allow_users":[]}`))
	}))
	c := Client{BaseURL: srv.URL, Token: "token"}
	if c.Evaluate("feature", "prod", "u", false) {
		t.Fatal("missing flag ignored fallback")
	}
	if e := c.Refresh(context.Background(), "feature", "prod"); e != nil {
		t.Fatal(e)
	}
	srv.Close()
	if e := c.Refresh(context.Background(), "feature", "prod"); e == nil {
		t.Fatal("refresh unexpectedly succeeded")
	}
	if !c.Evaluate("feature", "prod", "u", false) {
		t.Fatal("lost last-known config")
	}
}
