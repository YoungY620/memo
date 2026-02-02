package analyzer

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// ANSI color codes
const (
	colorYellow = "\033[38;5;178m" // Muted gold
	colorDim    = "\033[38;5;136m" // Dark olive/brown for borders
	colorCyan   = "\033[38;5;80m"  // Cyan for update notice
	colorReset  = "\033[0m"
)

// ASCII art for "memo" (width ~47 characters)
var bannerArt = []string{
	"███╗   ███╗███████╗███╗   ███╗ ██████╗ ",
	"████╗ ████║██╔════╝████╗ ████║██╔═══██╗",
	"██╔████╔██║█████╗  ██╔████╔██║██║   ██║",
	"██║╚██╔╝██║██╔══╝  ██║╚██╔╝██║██║   ██║",
	"██║ ╚═╝ ██║███████╗██║ ╚═╝ ██║╚██████╔╝",
	"╚═╝     ╚═╝╚══════╝╚═╝     ╚═╝ ╚═════╝ ",
}

// BannerOptions contains the information to display in the banner
type BannerOptions struct {
	WorkDir    string
	Version    string
	UpdateInfo *UpdateInfo // Optional: update information to display
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	LatestVersion string
	UpdateCommand string
}

// PrintBanner prints the startup banner with width adaptation
func PrintBanner(opts BannerOptions) {
	width := getTermWidth()
	greeting := getGreeting()

	if width >= 60 {
		printFullBanner(opts, greeting, width)
	} else if width >= 40 {
		printCompactBanner(opts, greeting)
	} else {
		printMinimalBanner(opts, greeting)
	}
}

// getTermWidth returns the terminal width, defaults to 80 if unavailable
func getTermWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return w
}

// getGreeting returns a greeting message based on current time
func getGreeting() string {
	now := time.Now()
	month, day := now.Month(), now.Day()
	hour := now.Hour()

	// New Year greeting (Jan 1 - Feb 1)
	if month == time.January || (month == time.February && day == 1) {
		return fmt.Sprintf("Welcome to %d! Happy New Year!", now.Year())
	}

	// Late night care (2:00-5:00)
	if hour >= 2 && hour < 5 {
		return "It's late, take care of yourself."
	}

	return ""
}

// ============== Full Banner (>= 60) ==============

func printFullBanner(opts BannerOptions, greeting string, termWidth int) {
	// Box width: min(termWidth, 60), but at least 55 for ASCII art
	boxWidth := termWidth
	if boxWidth > 60 {
		boxWidth = 60
	}
	if boxWidth < 55 {
		boxWidth = 55
	}
	innerWidth := boxWidth - 2 // subtract left/right borders

	// Helper: generate a line with borders and right padding
	// content is the visible content (no color codes)
	// colored is the same content with color codes for display
	line := func(content, colored string) string {
		contentWidth := runeWidth(content)
		padding := innerWidth - contentWidth
		if padding < 0 {
			padding = 0
		}
		return colorDim + "│" + colorReset + colored + strings.Repeat(" ", padding) + colorDim + "│" + colorReset
	}

	// Simple line without color in content
	simpleLine := func(content string) string {
		return line(content, content)
	}

	// Print banner
	fmt.Println()
	fmt.Println(colorDim + "╭" + strings.Repeat("─", innerWidth) + "╮" + colorReset)
	fmt.Println(simpleLine(""))
	for _, art := range bannerArt {
		plain := "  " + art
		colored := "  " + colorYellow + art + colorReset
		fmt.Println(line(plain, colored))
	}
	fmt.Println(simpleLine(""))
	fmt.Println(simpleLine("  " + truncatePath(opts.WorkDir, innerWidth-4)))
	fmt.Println(simpleLine("  " + opts.Version))
	if greeting != "" {
		fmt.Println(simpleLine(""))
		plain := "  ✨ " + greeting
		colored := "  " + colorYellow + "✨ " + greeting + colorReset
		fmt.Println(line(plain, colored))
	}
	// Update notice
	if opts.UpdateInfo != nil {
		fmt.Println(simpleLine(""))
		plain1 := "  ⬆ New version " + opts.UpdateInfo.LatestVersion + " available"
		colored1 := "  " + colorCyan + "⬆ New version " + opts.UpdateInfo.LatestVersion + " available" + colorReset
		fmt.Println(line(plain1, colored1))
		plain2 := "    " + opts.UpdateInfo.UpdateCommand
		colored2 := "    " + colorDim + opts.UpdateInfo.UpdateCommand + colorReset
		fmt.Println(line(plain2, colored2))
	}
	fmt.Println(simpleLine(""))
	fmt.Println(colorDim + "╰" + strings.Repeat("─", innerWidth) + "╯" + colorReset)
	fmt.Println()
}

// ============== Compact Banner (40-59) ==============

func printCompactBanner(opts BannerOptions, greeting string) {
	fmt.Println()
	for _, art := range bannerArt {
		fmt.Println("  " + colorYellow + art + colorReset)
	}
	fmt.Println()
	fmt.Println("  " + opts.WorkDir)
	fmt.Println("  " + opts.Version)
	if greeting != "" {
		fmt.Println("  " + colorYellow + greeting + colorReset)
	}
	// Update notice
	if opts.UpdateInfo != nil {
		fmt.Println()
		fmt.Println("  " + colorCyan + "⬆ New version " + opts.UpdateInfo.LatestVersion + " available" + colorReset)
		fmt.Println("    " + colorDim + opts.UpdateInfo.UpdateCommand + colorReset)
	}
	fmt.Println()
}

// ============== Minimal Banner (< 40) ==============

func printMinimalBanner(opts BannerOptions, greeting string) {
	fmt.Printf("%smemo%s %s\n", colorYellow, colorReset, opts.Version)
	if greeting != "" {
		fmt.Println(colorYellow + greeting + colorReset)
	}
	// Update notice
	if opts.UpdateInfo != nil {
		fmt.Println(colorCyan + "⬆ " + opts.UpdateInfo.LatestVersion + " available" + colorReset)
		fmt.Println(colorDim + opts.UpdateInfo.UpdateCommand + colorReset)
	}
}

// ============== Helper Functions ==============

// runeWidth calculates the display width of a string
// Box-drawing and block characters are treated specially
func runeWidth(s string) int {
	width := 0
	for _, r := range s {
		switch {
		case r >= 0x2500 && r <= 0x257F: // Box-drawing characters
			width += 1
		case r >= 0x2580 && r <= 0x259F: // Block elements (█, ▀, ▄, etc.)
			width += 1
		case r >= 0x2550 && r <= 0x256C: // Double-line box-drawing
			width += 1
		case r == '╔' || r == '╗' || r == '╚' || r == '╝' || r == '║' || r == '═':
			width += 1
		case r > 127:
			width += 2 // CJK/other wide characters
		default:
			width += 1
		}
	}
	return width
}

// truncatePath truncates a path if it exceeds maxWidth
// Shows "...suffix" format
func truncatePath(s string, maxWidth int) string {
	if runeWidth(s) <= maxWidth {
		return s
	}
	// Keep "..." prefix and as much of the tail as possible
	for i := len(s) - 1; i >= 0; i-- {
		sub := "..." + s[i:]
		if runeWidth(sub) <= maxWidth {
			return sub
		}
	}
	return "..."
}
