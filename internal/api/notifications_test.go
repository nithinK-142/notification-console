package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateNotificationInput(t *testing.T) {
	tests := []struct {
		name  string
		title string
		body  string
		valid bool
	}{
		{name: "valid", title: "Order update", body: "Your order shipped", valid: true},
		{name: "empty title", title: "", body: "Body"},
		{name: "title too long", title: strings.Repeat("a", 121), body: "Body"},
		{name: "empty body", title: "Title", body: ""},
		{name: "body too long", title: "Title", body: strings.Repeat("a", 2001)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateNotificationInput(test.title, test.body)
			if test.valid && err != nil {
				t.Fatalf("validateNotificationInput() error = %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("validateNotificationInput() error = nil, want validation error")
			}
		})
	}
}

func TestQueryInt(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
		valid bool
	}{
		{name: "default", query: "", want: 50, valid: true},
		{name: "valid", query: "25", want: 25, valid: true},
		{name: "below minimum", query: "0"},
		{name: "above maximum", query: "101"},
		{name: "not an integer", query: "abc"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/notifications?limit="+test.query, nil)
			got, err := queryInt(r, "limit", 50, 1, 100)
			if test.valid {
				if err != nil || got != test.want {
					t.Fatalf("queryInt() = (%d, %v), want (%d, nil)", got, err, test.want)
				}
			} else if err == nil {
				t.Fatal("queryInt() error = nil, want error")
			}
		})
	}
}
