package util

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// ErrorResult returns an MCP error result with the given message.
func ErrorResult(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf(format, args...)},
		},
		IsError: true,
	}
}

// SuccessResult returns an MCP success result with the given text.
func SuccessResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// HandleK8sError converts a Kubernetes API error into a user-friendly MCP error result.
func HandleK8sError(action string, err error) *mcp.CallToolResult {
	if apierrors.IsNotFound(err) {
		return ErrorResult("Not found: %s", action)
	}
	if apierrors.IsForbidden(err) {
		return ErrorResult("Permission denied: %s. Check RBAC permissions.", action)
	}
	if apierrors.IsUnauthorized(err) {
		return ErrorResult("Unauthorized: %s. Check cluster credentials.", action)
	}
	if apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) {
		return ErrorResult("Timeout: %s. The cluster may be unreachable.", action)
	}
	return ErrorResult("Error %s: %v", action, err)
}
