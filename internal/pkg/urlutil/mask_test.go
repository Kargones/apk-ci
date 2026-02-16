package urlutil

import "testing"

func TestMaskURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "https://hooks.slack.com/services/XXX/YYY/ZZZ",
			expected: "https://hooks.slack.com/***",
		},
		{
			input:    "http://internal.webhook.local:8080/alert?token=secret",
			expected: "http://internal.webhook.local:8080/***",
		},
		{
			input:    "https://api.pagerduty.com/v2/enqueue",
			expected: "https://api.pagerduty.com/***",
		},
		{
			input:    "not-a-valid-url",
			expected: "***invalid-url***",
		},
		{
			input:    "http://pushgateway:9091/metrics",
			expected: "http://pushgateway:9091/***",
		},
		{
			input:    "",
			expected: "***invalid-url***",
		},
		{
			input:    "http://user:pass@host:9091/path",
			expected: "http://host:9091/***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MaskURL(tt.input)
			if got != tt.expected {
				t.Errorf("MaskURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
