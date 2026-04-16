package web

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const sessionPreviewRegistryFilename = "session-previews.json"

type sessionPreviewRegistration struct {
	SessionID      string    `json:"session_id"`
	ShortHash      string    `json:"short_hash"`
	Host           string    `json:"host"`
	URL            string    `json:"url"`
	RepositoryPath string    `json:"repository_path"`
	DistPath       string    `json:"dist_path"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type sessionPreviewRegistrationInput struct {
	SessionID      string
	RepositoryPath string
}

type sessionPreviewRegistry struct {
	path       string
	baseDomain string

	mu      sync.RWMutex
	entries map[string]sessionPreviewRegistration
}

type previewRegistrationRequest struct {
	RepositoryPath string `json:"repository_path"`
}

func newFileSessionPreviewRegistry(path string, baseDomain string) (*sessionPreviewRegistry, error) {
	registry := &sessionPreviewRegistry{
		path:       strings.TrimSpace(path),
		baseDomain: normalizePreviewBaseDomain(baseDomain),
		entries:    map[string]sessionPreviewRegistration{},
	}
	if registry.path == "" {
		return registry, nil
	}
	if err := registry.load(); err != nil {
		return nil, err
	}
	return registry, nil
}

func (r *sessionPreviewRegistry) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	raw, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		r.entries = map[string]sessionPreviewRegistration{}
		return nil
	}
	if err != nil {
		return err
	}

	var payload struct {
		Items []sessionPreviewRegistration `json:"items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}

	entries := make(map[string]sessionPreviewRegistration, len(payload.Items))
	for _, item := range payload.Items {
		sessionID := sanitizeWorkspaceSegment(item.SessionID)
		if sessionID == "" {
			continue
		}
		item.SessionID = sessionID
		entries[sessionID] = item
	}
	r.entries = entries
	return nil
}

func (r *sessionPreviewRegistry) List() []sessionPreviewRegistration {
	if r == nil {
		return []sessionPreviewRegistration{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]sessionPreviewRegistration, 0, len(r.entries))
	for _, item := range r.entries {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].SessionID < items[j].SessionID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items
}

func (r *sessionPreviewRegistry) ResolveHost(host string) (sessionPreviewRegistration, bool) {
	if r == nil {
		return sessionPreviewRegistration{}, false
	}
	normalizedHost := normalizePreviewHost(host)
	if normalizedHost == "" {
		return sessionPreviewRegistration{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.entries {
		if normalizePreviewHost(item.Host) == normalizedHost {
			return item, true
		}
	}
	return sessionPreviewRegistration{}, false
}

func (r *sessionPreviewRegistry) ResolveSession(sessionID string) (sessionPreviewRegistration, bool) {
	if r == nil {
		return sessionPreviewRegistration{}, false
	}
	normalizedSessionID := sanitizeWorkspaceSegment(sessionID)
	if normalizedSessionID == "" {
		return sessionPreviewRegistration{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.entries[normalizedSessionID]
	return item, ok
}

func (r *sessionPreviewRegistry) Upsert(input sessionPreviewRegistrationInput) (sessionPreviewRegistration, error) {
	if r == nil {
		return sessionPreviewRegistration{}, errors.New("preview registry unavailable")
	}

	sessionID := sanitizeWorkspaceSegment(input.SessionID)
	if sessionID == "" {
		return sessionPreviewRegistration{}, errors.New("session_id is required")
	}

	repositoryPath, distPath, err := resolvePreviewRepositoryPaths(input.RepositoryPath)
	if err != nil {
		return sessionPreviewRegistration{}, err
	}

	entry := sessionPreviewRegistration{
		SessionID:      sessionID,
		ShortHash:      shortSessionPreviewHash(sessionID),
		RepositoryPath: repositoryPath,
		DistPath:       distPath,
		UpdatedAt:      time.Now().UTC(),
	}
	if entry.ShortHash == "" {
		return sessionPreviewRegistration{}, errors.New("failed to derive session short hash")
	}
	entry.Host = entry.ShortHash + "." + r.baseDomain
	entry.URL = "https://" + entry.Host

	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[sessionID] = entry
	if err := r.persistLocked(); err != nil {
		delete(r.entries, sessionID)
		return sessionPreviewRegistration{}, err
	}
	return entry, nil
}

func (r *sessionPreviewRegistry) Delete(sessionID string) (bool, error) {
	if r == nil {
		return false, errors.New("preview registry unavailable")
	}
	normalizedSessionID := sanitizeWorkspaceSegment(sessionID)
	if normalizedSessionID == "" {
		return false, errors.New("session_id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[normalizedSessionID]; !ok {
		return false, nil
	}
	delete(r.entries, normalizedSessionID)
	if err := r.persistLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *sessionPreviewRegistry) persistLocked() error {
	if strings.TrimSpace(r.path) == "" {
		return nil
	}
	items := make([]sessionPreviewRegistration, 0, len(r.entries))
	for _, item := range r.entries {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].SessionID < items[j].SessionID
	})
	payload, err := json.MarshalIndent(struct {
		Items []sessionPreviewRegistration `json:"items"`
	}{Items: items}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.path, payload, 0o644)
}

func normalizePreviewBaseDomain(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "alter0.cn"
	}
	return strings.Trim(trimmed, ".")
}

func normalizePreviewHost(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		trimmed = host
	}
	return strings.Trim(trimmed, ".")
}

func resolvePreviewRepositoryPaths(rawRepositoryPath string) (string, string, error) {
	repositoryPath := strings.TrimSpace(rawRepositoryPath)
	if repositoryPath == "" {
		return "", "", errors.New("repository_path is required")
	}
	absoluteRepositoryPath, err := filepath.Abs(repositoryPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository_path: %w", err)
	}
	repositoryPath = filepath.Clean(absoluteRepositoryPath)

	gitMarkerPath := filepath.Join(repositoryPath, ".git")
	if info, statErr := os.Stat(gitMarkerPath); statErr != nil {
		return "", "", errors.New("repository_path must point to a git repository workspace")
	} else if !info.IsDir() && info.Mode().IsRegular() {
		// gitdir file is allowed.
	}

	distPath := filepath.Join(repositoryPath, "internal", "interfaces", "web", "static", "dist")
	info, err := os.Stat(distPath)
	if err != nil {
		return "", "", errors.New("repository_path does not contain a built web dist")
	}
	if !info.IsDir() {
		return "", "", errors.New("repository_path dist target is not a directory")
	}
	indexPath := filepath.Join(distPath, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return "", "", errors.New("repository_path dist is missing index.html")
	}

	return filepath.ToSlash(repositoryPath), filepath.ToSlash(distPath), nil
}

func shortSessionPreviewHash(sessionID string) string {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return ""
	}
	sum := sha1.Sum([]byte(trimmed))
	return hex.EncodeToString(sum[:])[:8]
}

func (s *Server) withSessionPreview(next http.Handler) http.Handler {
	if s == nil || s.previewRegistry == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entry, ok := s.previewRegistry.ResolveHost(r.Host)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/chat":
			s.servePreviewChatPage(w, entry)
			return
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets/"):
			s.servePreviewStaticFile(w, r, entry, "/assets/", filepath.Join(filepath.FromSlash(entry.DistPath), "assets"), immutableStaticAssetCacheControl)
			return
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/legacy/"):
			s.servePreviewStaticFile(w, r, entry, "/legacy/", filepath.Join(filepath.FromSlash(entry.DistPath), "legacy"), bridgeStaticAssetCacheControl)
			return
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func (s *Server) servePreviewChatPage(w http.ResponseWriter, entry sessionPreviewRegistration) {
	indexPath := filepath.Join(filepath.FromSlash(entry.DistPath), "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		if s != nil && s.logger != nil {
			s.logger.Error("preview chat page unavailable", "session_id", entry.SessionID, "error", err.Error())
		}
		http.Error(w, "preview chat page unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", webPageCacheControl)
	w.Header().Set("X-Alter0-Preview-Session", entry.SessionID)
	_, _ = w.Write(content)
}

func (s *Server) servePreviewStaticFile(
	w http.ResponseWriter,
	r *http.Request,
	entry sessionPreviewRegistration,
	prefix string,
	rootPath string,
	cacheControl string,
) {
	filePath, ok := resolvePreviewStaticPath(rootPath, r.URL.Path, prefix)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	}
	w.Header().Set("X-Alter0-Preview-Session", entry.SessionID)
	http.ServeFile(w, r, filePath)
}

func resolvePreviewStaticPath(rootPath string, requestPath string, prefix string) (string, bool) {
	if !strings.HasPrefix(requestPath, prefix) {
		return "", false
	}
	relativePath := strings.TrimPrefix(requestPath, prefix)
	if strings.TrimSpace(relativePath) == "" {
		return "", false
	}
	cleanRelative := filepath.Clean(relativePath)
	if cleanRelative == "." || cleanRelative == "" {
		return "", false
	}
	if cleanRelative == ".." || strings.HasPrefix(cleanRelative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.Join(rootPath, cleanRelative), true
}

func (s *Server) previewCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s == nil || s.previewRegistry == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "preview registry unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.previewRegistry.List()})
}

func (s *Server) previewItemHandler(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.previewRegistry == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "preview registry unavailable"})
		return
	}
	sessionID, ok := resourceID(r.URL.Path, "/api/control/previews/")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid preview session path"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, found := s.previewRegistry.ResolveSession(sessionID)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "preview registration not found"})
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		defer r.Body.Close()
		var req previewRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		item, err := s.previewRegistry.Upsert(sessionPreviewRegistrationInput{
			SessionID:      sessionID,
			RepositoryPath: req.RepositoryPath,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		deleted, err := s.previewRegistry.Delete(sessionID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if !deleted {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "preview registration not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
