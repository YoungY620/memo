//go:build testing

package analyzer

// Export internal functions for testing.
// This file is only compiled with: go test -tags testing
// It allows external test packages (tests/analyzer) to access internal functions.

var (
	// Analyser exports
	GenerateSessionID = generateSessionID
	ToRelativePaths   = toRelativePaths
	SplitIntoBatches  = splitIntoBatches
	LoadPrompt        = loadPrompt

	// Banner exports
	GetGreeting  = getGreeting
	RuneWidth    = runeWidth
	TruncatePath = truncatePath
)
