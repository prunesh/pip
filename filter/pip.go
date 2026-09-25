// Package filter implements the prunesh/pip filter logic.
//
// Contract:
//   - id:      prunesh/pip
//   - command: pip
//
// Rewrite: injects --progress-bar=off into install/download/wheel when no
// quieting flag is present, suppressing multi-line download progress output.
//
// FilterOutput: strips Collecting/Downloading/Using cached/building noise from
// install output; compacts show fields; caps list output; passes freeze through.
package filter

import (
	"fmt"
	"strings"
)

const (
	// ID is the full filter identity following the author/<name> rule.
	ID = "prunesh/pip"

	// Command is the argv0 intercepted by this module.
	Command = "pip"

	maxListLines  = 50
	maxErrorLines = 40

	// keepSatisfied caps "Requirement already satisfied" lines to avoid
	// drowning the output when many deps are pre-installed.
	keepSatisfied = 3
)

// showFields are the pip show metadata fields worth keeping.
var showFields = map[string]bool{
	"Name":        true,
	"Version":     true,
	"Location":    true,
	"Requires":    true,
	"Required-by": true,
}

// Rewrite adds --progress-bar=off to install/download/wheel invocations that
// do not already silence output. Returns false when no rewrite is needed.
func Rewrite(args []string) ([]string, bool) {
	if len(args) == 0 {
		return nil, false
	}
	switch args[0] {
	case "install", "download", "wheel":
	default:
		return nil, false
	}
	for _, a := range args[1:] {
		switch a {
		case "-q", "--quiet", "-qq", "--no-input":
			return nil, false
		}
		if strings.HasPrefix(a, "--progress-bar") {
			return nil, false
		}
	}
	out := make([]string, len(args)+1)
	copy(out, args)
	out[len(args)] = "--progress-bar=off"
	return out, true
}

// FilterOutput dispatches to the appropriate filter by pip subcommand.
func FilterOutput(args []string, output string, exitCode int) string {
	if len(args) == 0 || output == "" {
		return output
	}
	switch args[0] {
	case "install", "download", "wheel":
		return filterInstall(output, exitCode)
	case "uninstall":
		return filterUninstall(output, exitCode)
	case "show":
		return filterShow(output)
	case "list":
		return filterList(output)
	default:
		return output
	}
}

func filterInstall(output string, exitCode int) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if exitCode != 0 {
		return filterInstallError(lines)
	}

	var kept []string
	satisfied := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isInstallNoise(trimmed) {
			continue
		}
		if strings.HasPrefix(trimmed, "Requirement already satisfied:") {
			if satisfied < keepSatisfied {
				kept = append(kept, trimmed)
				satisfied++
			} else if satisfied == keepSatisfied {
				kept = append(kept, "... (further already-satisfied requirements omitted)")
				satisfied++
			}
			continue
		}
		kept = append(kept, trimmed)
	}

	if len(kept) == 0 {
		return "ok\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

func filterInstallError(lines []string) string {
	var sb strings.Builder
	kept := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isInstallNoise(trimmed) && !isErrorLine(trimmed) {
			continue
		}
		if kept >= maxErrorLines {
			sb.WriteString(fmt.Sprintf("... (%d more lines)\n", len(lines)-i))
			break
		}
		sb.WriteString(trimmed)
		sb.WriteByte('\n')
		kept++
	}
	result := sb.String()
	if result == "" {
		return output(lines)
	}
	return result
}

func filterUninstall(out string, exitCode int) string {
	if exitCode != 0 {
		return out
	}
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "successfully uninstalled") {
			return trimmed + "\n"
		}
	}
	if strings.TrimSpace(out) == "" {
		return "ok\n"
	}
	return out
}

func filterShow(out string) string {
	var sb strings.Builder
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "---" {
			sb.WriteByte('\n')
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		idx := strings.Index(line, ": ")
		if idx < 0 {
			continue
		}
		if showFields[line[:idx]] {
			sb.WriteString(trimmed)
			sb.WriteByte('\n')
		}
	}
	result := sb.String()
	if strings.TrimSpace(result) == "" {
		return out
	}
	return result
}

func filterList(out string) string {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) <= maxListLines {
		return out
	}

	var header, packages []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "Package") || strings.HasPrefix(trimmed, "---") {
			header = append(header, line)
		} else {
			packages = append(packages, line)
		}
	}

	var sb strings.Builder
	for _, h := range header {
		sb.WriteString(h)
		sb.WriteByte('\n')
	}
	shown := packages
	if len(packages) > maxListLines {
		shown = packages[:maxListLines]
	}
	for _, p := range shown {
		sb.WriteString(p)
		sb.WriteByte('\n')
	}
	if len(packages) > maxListLines {
		sb.WriteString(fmt.Sprintf("... +%d more packages\n", len(packages)-maxListLines))
	}
	return sb.String()
}

// isInstallNoise reports whether a trimmed line is download/resolution noise.
func isInstallNoise(trimmed string) bool {
	switch {
	case strings.HasPrefix(trimmed, "Collecting "):
		return true
	case strings.HasPrefix(trimmed, "Downloading "):
		return true
	case strings.HasPrefix(trimmed, "Using cached "):
		return true
	case strings.HasPrefix(trimmed, "Obtaining "):
		return true
	case strings.HasPrefix(trimmed, "Building wheels"):
		return true
	case strings.HasPrefix(trimmed, "Building wheel"):
		return true
	case strings.HasPrefix(trimmed, "Created wheel"):
		return true
	case strings.HasPrefix(trimmed, "Stored in directory"):
		return true
	case strings.HasPrefix(trimmed, "Preparing metadata"):
		return true
	case containsProgressBar(trimmed):
		return true
	}
	return false
}

func isErrorLine(trimmed string) bool {
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "error:") ||
		strings.HasPrefix(lower, "warning:") ||
		strings.HasPrefix(lower, "could not") ||
		strings.HasPrefix(lower, "no matching") ||
		strings.HasPrefix(lower, "failed")
}

// containsProgressBar detects unicode box-drawing progress bars left after
// ANSI stripping (the proxy strips ANSI codes before calling FilterOutput).
func containsProgressBar(s string) bool {
	return strings.ContainsRune(s, '━') ||
		strings.ContainsRune(s, '▉') ||
		strings.ContainsRune(s, '█') ||
		(strings.Contains(s, "eta ") && strings.Contains(s, "/"))
}

func output(lines []string) string {
	return strings.Join(lines, "\n") + "\n"
}
