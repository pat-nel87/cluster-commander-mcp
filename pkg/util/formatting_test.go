package util

import (
	"strings"
	"testing"
	"time"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		contains string
	}{
		{"zero time", time.Time{}, "<unknown>"},
		{"seconds ago", time.Now().Add(-30 * time.Second), "s"},
		{"minutes ago", time.Now().Add(-5 * time.Minute), "m"},
		{"hours ago", time.Now().Add(-3 * time.Hour), "h"},
		{"days ago", time.Now().Add(-48 * time.Hour), "d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAge(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("FormatAge() = %q, want substring %q", result, tt.contains)
			}
		})
	}
}

func TestFormatTable(t *testing.T) {
	headers := []string{"NAME", "STATUS"}
	rows := [][]string{
		{"pod-1", "Running"},
		{"pod-2", "Pending"},
	}
	result := FormatTable(headers, rows)
	if !strings.Contains(result, "NAME") {
		t.Error("table should contain header NAME")
	}
	if !strings.Contains(result, "pod-1") {
		t.Error("table should contain pod-1")
	}
	if !strings.Contains(result, "Pending") {
		t.Error("table should contain Pending")
	}
}

func TestFormatTableEmpty(t *testing.T) {
	result := FormatTable([]string{"NAME"}, nil)
	if result != "(none)" {
		t.Errorf("empty table should return '(none)', got %q", result)
	}
}

func TestFormatLabels(t *testing.T) {
	result := FormatLabels(nil)
	if result != "<none>" {
		t.Errorf("nil labels should return '<none>', got %q", result)
	}

	result = FormatLabels(map[string]string{"app": "nginx"})
	if !strings.Contains(result, "app=nginx") {
		t.Errorf("labels should contain 'app=nginx', got %q", result)
	}
}

func TestTruncateString(t *testing.T) {
	short := "hello"
	if TruncateString(short, 10) != short {
		t.Error("short string should not be truncated")
	}

	long := strings.Repeat("x", 100)
	result := TruncateString(long, 50)
	if !strings.Contains(result, "truncated") {
		t.Error("long string should contain truncation notice")
	}
	if len(result) <= 50 {
		t.Error("truncated result should include notice text")
	}
}

func TestFormatHeader(t *testing.T) {
	result := FormatHeader("Test")
	if result != "=== Test ===" {
		t.Errorf("FormatHeader() = %q, want '=== Test ==='", result)
	}
}

func TestJoinNonEmpty(t *testing.T) {
	result := JoinNonEmpty(",", "a", "", "b", "", "c")
	if result != "a,b,c" {
		t.Errorf("JoinNonEmpty() = %q, want 'a,b,c'", result)
	}
}
