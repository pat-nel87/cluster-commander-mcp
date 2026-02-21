package util

import (
	"fmt"
	"strings"
	"time"
)

// FormatAge returns a human-readable age string from a timestamp.
func FormatAge(t time.Time) string {
	if t.IsZero() {
		return "<unknown>"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m > 0 {
			return fmt.Sprintf("%dh%dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	default:
		days := int(d.Hours()) / 24
		if days > 365 {
			return fmt.Sprintf("%dy%dd", days/365, days%365)
		}
		return fmt.Sprintf("%dd", days)
	}
}

// FormatHeader returns a section header line.
func FormatHeader(title string) string {
	return fmt.Sprintf("=== %s ===", title)
}

// FormatSubHeader returns a sub-section header line.
func FormatSubHeader(title string) string {
	return fmt.Sprintf("--- %s ---", title)
}

// FormatKeyValue formats a key-value pair with aligned colons.
func FormatKeyValue(key, value string) string {
	return fmt.Sprintf("%-20s %s", key+":", value)
}

// FormatTable formats rows as an aligned table with headers.
func FormatTable(headers []string, rows [][]string) string {
	if len(rows) == 0 {
		return "(none)"
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder

	// Header row
	for i, h := range headers {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(fmt.Sprintf("%-*s", widths[i], h))
	}
	sb.WriteString("\n")

	// Data rows
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if i > 0 {
				sb.WriteString("  ")
			}
			sb.WriteString(fmt.Sprintf("%-*s", widths[i], cell))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatFinding formats a diagnostic finding.
func FormatFinding(severity, message string) string {
	return fmt.Sprintf("[%s] %s", severity, message)
}

// FormatCount returns a count summary line.
func FormatCount(label string, count int) string {
	return fmt.Sprintf("Found %d %s", count, label)
}

// TruncateString truncates a string to maxLen and appends a truncation notice.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... [output truncated]"
}

// JoinNonEmpty joins non-empty strings with a separator.
func JoinNonEmpty(sep string, parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, sep)
}

// FormatLabels formats a label map as a comma-separated string.
func FormatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "<none>"
	}
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ", ")
}

// FormatResourceQuantity formats a resource quantity string, returning "<none>" if empty.
func FormatResourceQuantity(q string) string {
	if q == "" {
		return "<none>"
	}
	return q
}
