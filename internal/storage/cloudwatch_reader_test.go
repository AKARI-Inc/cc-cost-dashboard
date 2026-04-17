package storage

import "testing"

func TestBuildFilterPattern(t *testing.T) {
	tests := []struct {
		name string
		opts *ReadOptions
		want string
	}{
		{"nil opts", nil, ""},
		{"empty opts", &ReadOptions{}, ""},
		{"event_name only", &ReadOptions{EventName: "claude_code.api_request"}, `{ $.event_name = "claude_code.api_request" }`},
		{"user_email only", &ReadOptions{UserEmail: "alice@example.com"}, `{ $.user_email = "alice@example.com" }`},
		{
			"both filters",
			&ReadOptions{EventName: "claude_code.api_request", UserEmail: "alice@example.com"},
			`{ $.event_name = "claude_code.api_request" && $.user_email = "alice@example.com" }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFilterPattern(tt.opts)
			if got != tt.want {
				t.Errorf("buildFilterPattern() = %q, want %q", got, tt.want)
			}
		})
	}
}
