package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	codexCommandModeEnvKey = "ALTER0_CODEX_COMMAND_MODE"
	codexCommandModeAuto   = "auto"
	codexCommandModePinned = "pinned"
)

var codexVersionPattern = regexp.MustCompile(`\bv?([0-9]+(?:\.[0-9]+){1,3}(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?)\b`)

type codexVersion struct {
	core       [4]int
	preRelease []string
}

func resolveConfiguredCodexCommand(raw string) string {
	configured := strings.TrimSpace(raw)
	if normalizeCodexCommandMode(os.Getenv(codexCommandModeEnvKey)) == codexCommandModePinned && configured != "" {
		return configured
	}

	if selected := selectLatestCodexCommand(discoverCodexCommandCandidates(configured)); selected != "" {
		return selected
	}
	if configured != "" {
		return configured
	}
	if isExecutableFile(defaultPublicCodexCommand) {
		return defaultPublicCodexCommand
	}
	return "codex"
}

func normalizeCodexCommandMode(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), codexCommandModePinned) {
		return codexCommandModePinned
	}
	return codexCommandModeAuto
}

func discoverCodexCommandCandidates(configured string) []string {
	home := strings.TrimSpace(os.Getenv("HOME"))
	candidates := make([]string, 0, 12)
	appendCandidate := func(command string) {
		command = strings.TrimSpace(command)
		if command == "" {
			return
		}
		for _, existing := range candidates {
			if sameConfiguredPath(existing, command) {
				return
			}
		}
		candidates = append(candidates, command)
	}

	if home != "" {
		appendCandidate(filepath.Join(home, ".local", "bin", "codex"))
		appendCandidate(filepath.Join(home, ".nvm", "current", "bin", "codex"))
	}
	appendCandidate(configured)
	if home != "" {
		matches, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "codex"))
		for _, match := range matches {
			appendCandidate(match)
		}
	}
	appendCandidate(defaultPublicCodexCommand)
	if pathCommand, err := exec.LookPath("codex"); err == nil {
		appendCandidate(pathCommand)
	}
	return candidates
}

func selectLatestCodexCommand(candidates []string) string {
	type versionResult struct {
		command string
		version codexVersion
		valid   bool
	}
	results := make([]versionResult, len(candidates))
	var waitGroup sync.WaitGroup
	for index, candidate := range candidates {
		command := resolveCodexCandidate(candidate)
		if command == "" {
			continue
		}
		waitGroup.Add(1)
		go func(index int, command string) {
			defer waitGroup.Done()
			version, ok := queryCodexCommandVersion(command)
			results[index] = versionResult{command: command, version: version, valid: ok}
		}(index, command)
	}
	waitGroup.Wait()

	selected := ""
	selectedVersion := codexVersion{}
	for _, result := range results {
		if !result.valid {
			continue
		}
		if selected == "" || compareCodexVersions(result.version, selectedVersion) > 0 {
			selected = result.command
			selectedVersion = result.version
		}
	}
	return selected
}

func resolveCodexCandidate(raw string) string {
	command := strings.TrimSpace(raw)
	if command == "" {
		return ""
	}
	if filepath.IsAbs(command) || strings.ContainsRune(command, filepath.Separator) {
		if isExecutableFile(command) {
			return filepath.Clean(command)
		}
		return ""
	}
	resolved, err := exec.LookPath(command)
	if err != nil || !isExecutableFile(resolved) {
		return ""
	}
	return resolved
}

func queryCodexCommandVersion(command string) (codexVersion, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, command, "--version").CombinedOutput()
	if err != nil {
		return codexVersion{}, false
	}
	return parseCodexVersion(string(output))
}

func parseCodexVersion(raw string) (codexVersion, bool) {
	match := codexVersionPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) < 2 {
		return codexVersion{}, false
	}
	value := strings.TrimPrefix(match[1], "v")
	value = strings.SplitN(value, "+", 2)[0]
	parts := strings.SplitN(value, "-", 2)
	coreParts := strings.Split(parts[0], ".")
	if len(coreParts) < 2 || len(coreParts) > 4 {
		return codexVersion{}, false
	}
	version := codexVersion{}
	for index, part := range coreParts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return codexVersion{}, false
		}
		version.core[index] = number
	}
	if len(parts) == 2 {
		version.preRelease = strings.Split(parts[1], ".")
	}
	return version, true
}

func compareCodexVersions(left codexVersion, right codexVersion) int {
	for index := range left.core {
		switch {
		case left.core[index] > right.core[index]:
			return 1
		case left.core[index] < right.core[index]:
			return -1
		}
	}
	if len(left.preRelease) == 0 && len(right.preRelease) == 0 {
		return 0
	}
	if len(left.preRelease) == 0 {
		return 1
	}
	if len(right.preRelease) == 0 {
		return -1
	}
	limit := len(left.preRelease)
	if len(right.preRelease) < limit {
		limit = len(right.preRelease)
	}
	for index := 0; index < limit; index++ {
		comparison := compareCodexVersionIdentifier(left.preRelease[index], right.preRelease[index])
		if comparison != 0 {
			return comparison
		}
	}
	switch {
	case len(left.preRelease) > len(right.preRelease):
		return 1
	case len(left.preRelease) < len(right.preRelease):
		return -1
	default:
		return 0
	}
}

func compareCodexVersionIdentifier(left string, right string) int {
	leftNumber, leftNumeric := parseCodexVersionIdentifierNumber(left)
	rightNumber, rightNumeric := parseCodexVersionIdentifierNumber(right)
	switch {
	case leftNumeric && rightNumeric && leftNumber > rightNumber:
		return 1
	case leftNumeric && rightNumeric && leftNumber < rightNumber:
		return -1
	case leftNumeric && !rightNumeric:
		return -1
	case !leftNumeric && rightNumeric:
		return 1
	case left > right:
		return 1
	case left < right:
		return -1
	default:
		return 0
	}
}

func parseCodexVersionIdentifierNumber(raw string) (int, bool) {
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	return value, err == nil
}
