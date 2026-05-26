package runtimeconfig

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

const runtimeSkillFilesDir = ".alter0/codex-runtime/skills"

type FileBackedSkillReference struct {
	Key      string
	ID       string
	FilePath string
}

type MaterializedFileBackedSkills struct {
	ManagedFiles []ManagedFile
	FilePaths    map[string]string
}

func MaterializeFileBackedSkillReferences(refs []FileBackedSkillReference) (MaterializedFileBackedSkills, error) {
	result := MaterializedFileBackedSkills{
		FilePaths: map[string]string{},
	}
	written := map[string]struct{}{}
	for _, ref := range refs {
		key := strings.TrimSpace(ref.Key)
		if key == "" {
			continue
		}
		originalPath := strings.TrimSpace(ref.FilePath)
		if originalPath == "" || isRuntimeSkillFilePath(originalPath) {
			continue
		}
		sourcePath, found, err := resolveReadableFileBackedSkillPath(originalPath)
		if err != nil {
			return result, err
		}
		if !found {
			continue
		}
		sourceRoot := resolveFileBackedSkillRoot(sourcePath, ref.ID)
		sourceRel, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return result, fmt.Errorf("resolve skill file relative path %s: %w", sourcePath, err)
		}
		targetRoot := path.Join(runtimeSkillFilesDir, safeRuntimeSkillID(ref.ID, originalPath))
		targetPath := path.Join(targetRoot, filepath.ToSlash(sourceRel))
		result.FilePaths[key] = targetPath

		files, err := managedFilesForSkillDirectory(sourceRoot, targetRoot, written)
		if err != nil {
			return result, err
		}
		result.ManagedFiles = append(result.ManagedFiles, files...)
	}
	return result, nil
}

func isRuntimeSkillFilePath(value string) bool {
	normalized := path.Clean(filepath.ToSlash(strings.TrimSpace(value)))
	return normalized == runtimeSkillFilesDir || strings.HasPrefix(normalized, runtimeSkillFilesDir+"/")
}

func resolveReadableFileBackedSkillPath(rawPath string) (string, bool, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", false, nil
	}
	if filepath.IsAbs(trimmed) {
		return statReadableSkillFile(filepath.Clean(trimmed))
	}
	relativePath := filepath.FromSlash(trimmed)
	wd, err := os.Getwd()
	if err != nil {
		return "", false, fmt.Errorf("resolve working directory for skill file %s: %w", rawPath, err)
	}
	current := wd
	for {
		candidate := filepath.Join(current, relativePath)
		if resolved, found, err := statReadableSkillFile(candidate); err != nil || found {
			return resolved, found, err
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", false, nil
}

func statReadableSkillFile(candidate string) (string, bool, error) {
	info, err := os.Stat(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("stat skill file %s: %w", candidate, err)
	}
	if info.IsDir() {
		return "", false, nil
	}
	absolute, err := filepath.Abs(candidate)
	if err != nil {
		return "", false, fmt.Errorf("resolve absolute skill file path %s: %w", candidate, err)
	}
	return absolute, true, nil
}

func resolveFileBackedSkillRoot(sourcePath string, skillID string) string {
	cleanSource := filepath.Clean(sourcePath)
	normalizedID := strings.TrimSpace(skillID)
	if normalizedID == "" {
		return filepath.Dir(cleanSource)
	}
	sourceSlash := filepath.ToSlash(cleanSource)
	for _, marker := range []string{
		path.Join("docs", "skills", normalizedID),
		path.Join("docs", "agents", normalizedID),
	} {
		if root, ok := sourceRootThroughMarker(sourceSlash, marker); ok {
			return filepath.FromSlash(root)
		}
	}
	return filepath.Dir(cleanSource)
}

func sourceRootThroughMarker(sourceSlash string, marker string) (string, bool) {
	for _, needle := range []string{"/" + marker + "/", "/" + marker} {
		index := strings.Index(sourceSlash, needle)
		if index < 0 {
			continue
		}
		end := index + len(needle)
		if strings.HasSuffix(needle, "/") {
			end--
		}
		return sourceSlash[:end], true
	}
	return "", false
}

func managedFilesForSkillDirectory(sourceRoot string, targetRoot string, written map[string]struct{}) ([]ManagedFile, error) {
	files := []ManagedFile{}
	err := filepath.WalkDir(sourceRoot, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat skill runtime file %s: %w", filePath, err)
		}
		if info.Mode()&os.ModeType != 0 {
			return nil
		}
		relativePath, err := filepath.Rel(sourceRoot, filePath)
		if err != nil {
			return fmt.Errorf("resolve skill runtime file path %s: %w", filePath, err)
		}
		targetPath := path.Join(targetRoot, filepath.ToSlash(relativePath))
		if _, ok := written[targetPath]; ok {
			return nil
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read skill runtime file %s: %w", filePath, err)
		}
		written[targetPath] = struct{}{}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		files = append(files, ManagedFile{
			RelativePath: targetPath,
			Content:      string(content),
			Mode:         mode,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("materialize skill directory %s: %w", sourceRoot, err)
	}
	return files, nil
}

func safeRuntimeSkillID(skillID string, fallback string) string {
	trimmed := strings.TrimSpace(skillID)
	if trimmed == "" {
		trimmed = strings.TrimSpace(filepath.Base(filepath.Dir(filepath.FromSlash(fallback))))
	}
	var builder strings.Builder
	for _, r := range strings.ToLower(trimmed) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(r)
		case r == '.', r == '_', r == '-':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}
	value := strings.Trim(builder.String(), ".-_")
	if value != "" {
		return value
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(fallback))
	return fmt.Sprintf("skill-%08x", hash.Sum32())
}
