package application

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	nativeSkillManagedPrefix = "alter0-"
	nativeSkillMarkerFile    = ".alter0-managed.json"
)

type NativeSkillSource struct {
	ID       string
	Enabled  bool
	Public   bool
	FilePath string
}

type NativeSkillReconcileError struct {
	SkillID string `json:"skill_id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type NativeSkillReconcileResult struct {
	Installed []string                    `json:"installed"`
	Removed   []string                    `json:"removed"`
	Errors    []NativeSkillReconcileError `json:"errors"`
}

type NativeSkillReconciler struct {
	destinationRoot string
}

type nativeSkillMarker struct {
	ManagedBy string `json:"managed_by"`
	SkillID   string `json:"skill_id"`
}

func NewNativeSkillReconciler(destinationRoot string) *NativeSkillReconciler {
	return &NativeSkillReconciler{destinationRoot: filepath.Clean(strings.TrimSpace(destinationRoot))}
}

func ResolveNativeSkillRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(home, ".agents", "skills"), nil
}

func (r *NativeSkillReconciler) Reconcile(sources []NativeSkillSource) NativeSkillReconcileResult {
	result := NativeSkillReconcileResult{
		Installed: []string{},
		Removed:   []string{},
		Errors:    []NativeSkillReconcileError{},
	}
	if r == nil || strings.TrimSpace(r.destinationRoot) == "" || r.destinationRoot == "." {
		result.Errors = append(result.Errors, nativeSkillError("", "destination_unavailable", "native skill destination is not configured"))
		return result
	}
	if err := os.MkdirAll(r.destinationRoot, 0o755); err != nil {
		result.Errors = append(result.Errors, nativeSkillError("", "destination_unavailable", err.Error()))
		return result
	}

	desired := map[string]struct{}{}
	for _, source := range sources {
		id := normalizeNativeSkillID(source.ID)
		if id == "" {
			if source.Enabled && source.Public {
				result.Errors = append(result.Errors, nativeSkillError(source.ID, "invalid_id", "skill id is invalid"))
			}
			continue
		}
		if !source.Enabled || !source.Public {
			continue
		}
		desired[id] = struct{}{}
		if err := r.install(id, source.FilePath); err != nil {
			code := "install_failed"
			if strings.Contains(err.Error(), "destination is unmanaged") {
				code = "destination_collision"
			} else if strings.Contains(err.Error(), "SKILL.md") {
				code = "invalid_source"
			}
			result.Errors = append(result.Errors, nativeSkillError(id, code, err.Error()))
			continue
		}
		result.Installed = append(result.Installed, id)
	}

	entries, err := os.ReadDir(r.destinationRoot)
	if err != nil {
		result.Errors = append(result.Errors, nativeSkillError("", "destination_unavailable", err.Error()))
		return result
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), nativeSkillManagedPrefix) {
			continue
		}
		dir := filepath.Join(r.destinationRoot, entry.Name())
		marker, ok := readNativeSkillMarker(dir)
		if !ok {
			continue
		}
		id := normalizeNativeSkillID(marker.SkillID)
		if _, keep := desired[id]; keep {
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			result.Errors = append(result.Errors, nativeSkillError(id, "remove_failed", err.Error()))
			continue
		}
		result.Removed = append(result.Removed, id)
	}
	sort.Strings(result.Installed)
	sort.Strings(result.Removed)
	sort.Slice(result.Errors, func(i, j int) bool {
		if result.Errors[i].SkillID == result.Errors[j].SkillID {
			return result.Errors[i].Code < result.Errors[j].Code
		}
		return result.Errors[i].SkillID < result.Errors[j].SkillID
	})
	return result
}

func (r *NativeSkillReconciler) install(id string, sourceFile string) error {
	sourceFile = filepath.Clean(strings.TrimSpace(sourceFile))
	if filepath.Base(sourceFile) != "SKILL.md" {
		return fmt.Errorf("skill %s source must be a SKILL.md file", id)
	}
	if err := validateNativeSkillFile(sourceFile); err != nil {
		return err
	}
	destination := filepath.Join(r.destinationRoot, nativeSkillManagedPrefix+id)
	if _, err := os.Stat(destination); err == nil {
		if _, managed := readNativeSkillMarker(destination); !managed {
			return fmt.Errorf("skill %s destination is unmanaged", id)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	staging := filepath.Join(r.destinationRoot, fmt.Sprintf(".%s%s.staging-%d", nativeSkillManagedPrefix, id, time.Now().UnixNano()))
	_ = os.RemoveAll(staging)
	if err := copyNativeSkillDirectory(filepath.Dir(sourceFile), staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	marker, _ := json.Marshal(nativeSkillMarker{ManagedBy: "alter0", SkillID: id})
	if err := os.WriteFile(filepath.Join(staging, nativeSkillMarkerFile), append(marker, '\n'), 0o644); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	backup := destination + fmt.Sprintf(".backup-%d", time.Now().UnixNano())
	hadDestination := false
	if _, err := os.Stat(destination); err == nil {
		if err := os.Rename(destination, backup); err != nil {
			_ = os.RemoveAll(staging)
			return err
		}
		hadDestination = true
	}
	if err := os.Rename(staging, destination); err != nil {
		if hadDestination {
			_ = os.Rename(backup, destination)
		}
		_ = os.RemoveAll(staging)
		return err
	}
	if hadDestination {
		_ = os.RemoveAll(backup)
	}
	return nil
}

func validateNativeSkillFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read SKILL.md: %w", err)
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) < 4 || strings.TrimSpace(lines[0]) != "---" {
		return fmt.Errorf("SKILL.md must start with YAML frontmatter")
	}
	name := ""
	description := ""
	descriptionDeclared := false
	for index := 1; index < len(lines); index++ {
		rawLine := lines[index]
		line := strings.TrimSpace(rawLine)
		if line == "---" {
			break
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			if descriptionDeclared && description == "" && strings.TrimSpace(rawLine) != "" {
				description = strings.TrimSpace(rawLine)
			}
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			name = strings.Trim(strings.TrimSpace(value), "\"")
		case "description":
			descriptionDeclared = true
			description = strings.TrimSpace(value)
		}
	}
	if name == "" || description == "" {
		return fmt.Errorf("SKILL.md frontmatter requires name and description")
	}
	return nil
}

func copyNativeSkillDirectory(sourceDir string, destinationDir string) error {
	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("skill source contains unsupported symlink: %s", path)
		}
		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destinationDir, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
			return os.Chmod(target, info.Mode().Perm())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, content, info.Mode().Perm()); err != nil {
			return err
		}
		return os.Chmod(target, info.Mode().Perm())
	})
}

func readNativeSkillMarker(dir string) (nativeSkillMarker, bool) {
	raw, err := os.ReadFile(filepath.Join(dir, nativeSkillMarkerFile))
	if err != nil {
		return nativeSkillMarker{}, false
	}
	marker := nativeSkillMarker{}
	if err := json.Unmarshal(raw, &marker); err != nil || marker.ManagedBy != "alter0" || normalizeNativeSkillID(marker.SkillID) == "" {
		return nativeSkillMarker{}, false
	}
	return marker, true
}

func normalizeNativeSkillID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return ""
	}
	return value
}

func nativeSkillError(skillID string, code string, message string) NativeSkillReconcileError {
	return NativeSkillReconcileError{SkillID: strings.TrimSpace(skillID), Code: code, Message: strings.TrimSpace(message)}
}
