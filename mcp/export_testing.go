//go:build testing

package mcp

// Export internal functions and types for testing.
// This file is only compiled with: go test -tags testing

// GetStatusFromServer exports the getStatus method for testing
func (s *Server) GetStatusFromServer() Status {
	return s.getStatus()
}
