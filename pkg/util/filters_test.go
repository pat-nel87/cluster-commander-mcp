package util

import "testing"

func TestNamespaceOrAll(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"default", "default"},
		{"all", ""},
		{"*", ""},
		{"", ""},
		{"kube-system", "kube-system"},
	}
	for _, tt := range tests {
		result := NamespaceOrAll(tt.input)
		if result != tt.expected {
			t.Errorf("NamespaceOrAll(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestListOptions(t *testing.T) {
	opts := ListOptions("app=nginx", "status.phase=Running")
	if opts.LabelSelector != "app=nginx" {
		t.Errorf("LabelSelector = %q, want 'app=nginx'", opts.LabelSelector)
	}
	if opts.FieldSelector != "status.phase=Running" {
		t.Errorf("FieldSelector = %q, want 'status.phase=Running'", opts.FieldSelector)
	}

	empty := ListOptions("", "")
	if empty.LabelSelector != "" || empty.FieldSelector != "" {
		t.Error("empty selectors should produce empty options")
	}
}
