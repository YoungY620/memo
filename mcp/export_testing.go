//go:build testing

package mcp

// Export internal functions and types for testing.
// This file is only compiled with: go test -tags testing

// GetStatusFromServer exports the getStatus method for testing
func (s *Server) GetStatusFromServer() Status {
	return s.getStatus()
}

// HandleRequestForTest exports the handleRequest method for testing
func (s *Server) HandleRequestForTest(req *Request) *Response {
	return s.handleRequest(req)
}

// IndexExistsForTest exports the indexExists method for testing
func (s *Server) IndexExistsForTest() bool {
	return s.indexExists()
}
