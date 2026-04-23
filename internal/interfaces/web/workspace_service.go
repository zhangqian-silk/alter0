package web

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	workspaceServiceRegistryFilename = "workspace-services.json"
	defaultWorkspaceServiceID        = "web"
	workspaceServiceTypeFrontendDist = "frontend_dist"
	workspaceServiceTypeHTTP         = "http"
)

type workspaceServiceRegistration struct {
	SessionID      string    `json:"session_id"`
	ServiceID      string    `json:"service_id"`
	ServiceType    string    `json:"service_type"`
	ShortHash      string    `json:"short_hash"`
	Host           string    `json:"host"`
	URL            string    `json:"url"`
	RepositoryPath string    `json:"repository_path,omitempty"`
	DistPath       string    `json:"dist_path,omitempty"`
	UpstreamURL    string    `json:"upstream_url,omitempty"`
	StartCommand   string    `json:"start_command,omitempty"`
	Workdir        string    `json:"workdir,omitempty"`
	HealthPath     string    `json:"health_path,omitempty"`
	Port           int       `json:"port,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type workspaceServiceRegistrationInput struct {
	SessionID      string
	ServiceID      string
	ServiceType    string
	RepositoryPath string
	UpstreamURL    string
	StartCommand   string
	Workdir        string
	HealthPath     string
	Port           int
}

type workspaceServiceRegistry struct {
	path       string
	baseDomain string

	mu      sync.RWMutex
	entries map[string]workspaceServiceRegistration
}

type workspaceServiceRegistrationRequest struct {
	ServiceType    string `json:"service_type"`
	RepositoryPath string `json:"repository_path,omitempty"`
	UpstreamURL    string `json:"upstream_url,omitempty"`
	StartCommand   string `json:"start_command,omitempty"`
	Workdir        string `json:"workdir,omitempty"`
	HealthPath     string `json:"health_path,omitempty"`
	Port           int    `json:"port,omitempty"`
}

type workspaceServiceResponse struct {
	workspaceServiceRegistration
	RuntimeDir string `json:"runtime_dir,omitempty"`
	LogPath    string `json:"log_path,omitempty"`
	PID        int    `json:"pid,omitempty"`
	Status     string `json:"status,omitempty"`
}

func newFileWorkspaceServiceRegistry(path string, baseDomain string) (*workspaceServiceRegistry, error) {
	registry := &workspaceServiceRegistry{
		path:       strings.TrimSpace(path),
		baseDomain: normalizePreviewBaseDomain(baseDomain),
		entries:    map[string]workspaceServiceRegistration{},
	}
	if registry.path == "" {
		return registry, nil
	}
	if err := registry.load(); err != nil {
		return nil, err
	}
	return registry, nil
}

func (r *workspaceServiceRegistry) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	raw, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		r.entries = map[string]workspaceServiceRegistration{}
		return nil
	}
	if err != nil {
		return err
	}

	var payload struct {
		Items []workspaceServiceRegistration `json:"items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}

	entries := make(map[string]workspaceServiceRegistration, len(payload.Items))
	for _, item := range payload.Items {
		sessionID := sanitizeWorkspaceSegment(item.SessionID)
		serviceID := normalizeWorkspaceServiceID(item.ServiceID)
		if sessionID == "" || serviceID == "" {
			continue
		}
		item.SessionID = sessionID
		item.ServiceID = serviceID
		entries[workspaceServiceKey(sessionID, serviceID)] = item
	}
	r.entries = entries
	return nil
}

func (r *workspaceServiceRegistry) List() []workspaceServiceRegistration {
	if r == nil {
		return []workspaceServiceRegistration{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]workspaceServiceRegistration, 0, len(r.entries))
	for _, item := range r.entries {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			if items[i].SessionID == items[j].SessionID {
				return items[i].ServiceID < items[j].ServiceID
			}
			return items[i].SessionID < items[j].SessionID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items
}

func (r *workspaceServiceRegistry) ResolveHost(host string) (workspaceServiceRegistration, bool) {
	if r == nil {
		return workspaceServiceRegistration{}, false
	}
	normalizedHost := normalizePreviewHost(host)
	if normalizedHost == "" {
		return workspaceServiceRegistration{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.entries {
		if normalizePreviewHost(item.Host) == normalizedHost {
			return item, true
		}
	}
	return workspaceServiceRegistration{}, false
}

func (r *workspaceServiceRegistry) ResolveService(sessionID string, serviceID string) (workspaceServiceRegistration, bool) {
	if r == nil {
		return workspaceServiceRegistration{}, false
	}
	sessionID = sanitizeWorkspaceSegment(sessionID)
	serviceID = normalizeWorkspaceServiceID(serviceID)
	if sessionID == "" || serviceID == "" {
		return workspaceServiceRegistration{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.entries[workspaceServiceKey(sessionID, serviceID)]
	return item, ok
}

func (r *workspaceServiceRegistry) Upsert(input workspaceServiceRegistrationInput) (workspaceServiceRegistration, error) {
	if r == nil {
		return workspaceServiceRegistration{}, errors.New("workspace service registry unavailable")
	}

	sessionID := sanitizeWorkspaceSegment(input.SessionID)
	if sessionID == "" {
		return workspaceServiceRegistration{}, errors.New("session_id is required")
	}
	serviceID := normalizeWorkspaceServiceID(input.ServiceID)
	if serviceID == "" {
		return workspaceServiceRegistration{}, errors.New("service_id is required")
	}
	serviceType := normalizeWorkspaceServiceType(input.ServiceType)
	if serviceType == "" {
		return workspaceServiceRegistration{}, errors.New("service_type must be frontend_dist or http")
	}

	entry := workspaceServiceRegistration{
		SessionID:   sessionID,
		ServiceID:   serviceID,
		ServiceType: serviceType,
		ShortHash:   shortSessionPreviewHash(sessionID),
		UpdatedAt:   time.Now().UTC(),
	}
	if entry.ShortHash == "" {
		return workspaceServiceRegistration{}, errors.New("failed to derive session short hash")
	}

	switch serviceType {
	case workspaceServiceTypeFrontendDist:
		repositoryPath, distPath, err := resolvePreviewRepositoryPaths(input.RepositoryPath)
		if err != nil {
			return workspaceServiceRegistration{}, err
		}
		entry.RepositoryPath = repositoryPath
		entry.DistPath = distPath
	case workspaceServiceTypeHTTP:
		if strings.TrimSpace(input.StartCommand) != "" {
			workdir, err := normalizeWorkspaceServiceWorkdir(input.Workdir)
			if err != nil {
				return workspaceServiceRegistration{}, err
			}
			port, err := normalizeWorkspaceServicePort(input.Port)
			if err != nil {
				return workspaceServiceRegistration{}, err
			}
			entry.StartCommand = strings.TrimSpace(input.StartCommand)
			entry.Workdir = workdir
			entry.Port = port
			entry.HealthPath = normalizeWorkspaceServiceHealthPath(input.HealthPath)
			entry.UpstreamURL = fmt.Sprintf("http://127.0.0.1:%d", port)
		} else {
			upstreamURL, err := normalizeWorkspaceServiceUpstreamURL(input.UpstreamURL)
			if err != nil {
				return workspaceServiceRegistration{}, err
			}
			entry.UpstreamURL = upstreamURL
		}
	default:
		return workspaceServiceRegistration{}, errors.New("unsupported workspace service type")
	}

	entry.Host = buildWorkspaceServiceHost(entry.ShortHash, serviceID, r.baseDomain)
	entry.URL = "https://" + entry.Host

	r.mu.Lock()
	defer r.mu.Unlock()
	key := workspaceServiceKey(sessionID, serviceID)
	previous, hadPrevious := r.entries[key]
	r.entries[key] = entry
	if err := r.persistLocked(); err != nil {
		if hadPrevious {
			r.entries[key] = previous
		} else {
			delete(r.entries, key)
		}
		return workspaceServiceRegistration{}, err
	}
	return entry, nil
}

func (r *workspaceServiceRegistry) Delete(sessionID string, serviceID string) (bool, error) {
	if r == nil {
		return false, errors.New("workspace service registry unavailable")
	}
	sessionID = sanitizeWorkspaceSegment(sessionID)
	serviceID = normalizeWorkspaceServiceID(serviceID)
	if sessionID == "" {
		return false, errors.New("session_id is required")
	}
	if serviceID == "" {
		return false, errors.New("service_id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	key := workspaceServiceKey(sessionID, serviceID)
	if _, ok := r.entries[key]; !ok {
		return false, nil
	}
	delete(r.entries, key)
	if err := r.persistLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func workspaceServiceRegistrationToInput(entry workspaceServiceRegistration) workspaceServiceRegistrationInput {
	return workspaceServiceRegistrationInput{
		SessionID:      entry.SessionID,
		ServiceID:      entry.ServiceID,
		ServiceType:    entry.ServiceType,
		RepositoryPath: entry.RepositoryPath,
		UpstreamURL:    entry.UpstreamURL,
		StartCommand:   entry.StartCommand,
		Workdir:        entry.Workdir,
		HealthPath:     entry.HealthPath,
		Port:           entry.Port,
	}
}

func (r *workspaceServiceRegistry) persistLocked() error {
	if strings.TrimSpace(r.path) == "" {
		return nil
	}
	items := make([]workspaceServiceRegistration, 0, len(r.entries))
	for _, item := range r.entries {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SessionID == items[j].SessionID {
			return items[i].ServiceID < items[j].ServiceID
		}
		return items[i].SessionID < items[j].SessionID
	})
	payload, err := json.MarshalIndent(struct {
		Items []workspaceServiceRegistration `json:"items"`
	}{Items: items}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.path, payload, 0o644)
}

func workspaceServiceKey(sessionID string, serviceID string) string {
	return sessionID + "\x00" + serviceID
}

func normalizeWorkspaceServiceID(value string) string {
	trimmed := sanitizeWorkspaceSegment(value)
	if trimmed == "" {
		return defaultWorkspaceServiceID
	}
	return trimmed
}

func normalizeWorkspaceServiceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case workspaceServiceTypeFrontendDist:
		return workspaceServiceTypeFrontendDist
	case workspaceServiceTypeHTTP:
		return workspaceServiceTypeHTTP
	default:
		return ""
	}
}

func normalizeWorkspaceServiceUpstreamURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("upstream_url is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid upstream_url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("upstream_url must use http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", errors.New("upstream_url host is required")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizeWorkspaceServiceWorkdir(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("workdir is required when start_command is set")
	}
	absolutePath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	normalizedPath := filepath.Clean(absolutePath)
	info, err := os.Stat(normalizedPath)
	if err != nil {
		return "", errors.New("workdir does not exist")
	}
	if !info.IsDir() {
		return "", errors.New("workdir must be a directory")
	}
	return filepath.ToSlash(normalizedPath), nil
}

func normalizeWorkspaceServicePort(value int) (int, error) {
	if value <= 0 || value > 65535 {
		return 0, errors.New("port must be between 1 and 65535 when start_command is set")
	}
	return value, nil
}

func normalizeWorkspaceServiceHealthPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}

func buildWorkspaceServiceHost(shortHash string, serviceID string, baseDomain string) string {
	if shortHash == "" {
		return ""
	}
	if serviceID == "" || serviceID == defaultWorkspaceServiceID {
		return shortHash + "." + baseDomain
	}
	return serviceID + "." + shortHash + "." + baseDomain
}

func shortSessionPreviewHash(sessionID string) string {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return ""
	}
	sum := sha1.Sum([]byte(trimmed))
	return hex.EncodeToString(sum[:])[:8]
}

func resolvePreviewRepositoryPaths(rawRepositoryPath string) (string, string, error) {
	repositoryPath := strings.TrimSpace(rawRepositoryPath)
	if repositoryPath == "" {
		return "", "", errors.New("repository_path is required")
	}
	absoluteRepositoryPath, err := filepath.Abs(repositoryPath)
	if err != nil {
		return "", "", err
	}
	repositoryPath = filepath.Clean(absoluteRepositoryPath)

	gitMarkerPath := filepath.Join(repositoryPath, ".git")
	if info, statErr := os.Stat(gitMarkerPath); statErr != nil {
		return "", "", errors.New("repository_path must point to a git repository workspace")
	} else if !info.IsDir() && !info.Mode().IsRegular() {
		return "", "", errors.New("repository_path must point to a git repository workspace")
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

func workspaceServiceResourceID(path string) (string, string, bool) {
	const prefix = "/api/control/workspace-services/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if trimmed == "" {
		return "", "", false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		sessionID := strings.TrimSpace(parts[0])
		if sessionID == "" {
			return "", "", false
		}
		return sessionID, defaultWorkspaceServiceID, true
	}
	if len(parts) == 2 {
		sessionID := strings.TrimSpace(parts[0])
		serviceID := strings.TrimSpace(parts[1])
		if sessionID == "" || serviceID == "" {
			return "", "", false
		}
		return sessionID, serviceID, true
	}
	return "", "", false
}

func (s *Server) withWorkspaceServiceGateway(next http.Handler) http.Handler {
	if s == nil || s.workspaceService == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entry, ok := s.workspaceService.ResolveHost(r.Host)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/login" || r.URL.Path == "/logout" {
			next.ServeHTTP(w, r)
			return
		}

		switch entry.ServiceType {
		case workspaceServiceTypeFrontendDist:
			if s.serveWorkspaceFrontendService(w, r, entry) {
				return
			}
			next.ServeHTTP(w, r)
		case workspaceServiceTypeHTTP:
			effectiveEntry := entry
			if s.workspaceRuntime != nil && isManagedWorkspaceService(entry) {
				managedEntry, _, err := s.workspaceRuntime.EnsureStarted(entry)
				if err != nil {
					if s.logger != nil {
						s.logger.Error("workspace service startup failed",
							"session_id", entry.SessionID,
							"service_id", entry.ServiceID,
							"error", err.Error(),
						)
					}
					http.Error(w, "workspace service unavailable", http.StatusBadGateway)
					return
				}
				effectiveEntry = managedEntry
			}
			s.serveWorkspaceHTTPService(w, r, effectiveEntry)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func (s *Server) serveWorkspaceFrontendService(w http.ResponseWriter, r *http.Request, entry workspaceServiceRegistration) bool {
	switch {
	case r.Method == http.MethodGet && (r.URL.Path == "/" || r.URL.Path == "/chat"):
		s.serveWorkspaceFrontendPage(w, entry)
		return true
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets/"):
		s.serveWorkspaceFrontendStaticFile(w, r, entry, "/assets/", filepath.Join(filepath.FromSlash(entry.DistPath), "assets"), immutableStaticAssetCacheControl)
		return true
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/legacy/"):
		s.serveWorkspaceFrontendStaticFile(w, r, entry, "/legacy/", filepath.Join(filepath.FromSlash(entry.DistPath), "legacy"), bridgeStaticAssetCacheControl)
		return true
	default:
		return false
	}
}

func (s *Server) serveWorkspaceFrontendPage(w http.ResponseWriter, entry workspaceServiceRegistration) {
	indexPath := filepath.Join(filepath.FromSlash(entry.DistPath), "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		if s != nil && s.logger != nil {
			s.logger.Error("workspace frontend page unavailable", "session_id", entry.SessionID, "service_id", entry.ServiceID, "error", err.Error())
		}
		http.Error(w, "workspace frontend page unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", webPageCacheControl)
	w.Header().Set("X-Alter0-Workspace-Service", entry.ServiceID)
	w.Header().Set("X-Alter0-Workspace-Session", entry.SessionID)
	_, _ = w.Write(content)
}

func (s *Server) serveWorkspaceFrontendStaticFile(
	w http.ResponseWriter,
	r *http.Request,
	entry workspaceServiceRegistration,
	prefix string,
	rootPath string,
	cacheControl string,
) {
	filePath, ok := resolveWorkspaceServiceStaticPath(rootPath, r.URL.Path, prefix)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	}
	w.Header().Set("X-Alter0-Workspace-Service", entry.ServiceID)
	w.Header().Set("X-Alter0-Workspace-Session", entry.SessionID)
	http.ServeFile(w, r, filePath)
}

func resolveWorkspaceServiceStaticPath(rootPath string, requestPath string, prefix string) (string, bool) {
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

func (s *Server) serveWorkspaceHTTPService(w http.ResponseWriter, r *http.Request, entry workspaceServiceRegistration) {
	target, err := url.Parse(entry.UpstreamURL)
	if err != nil {
		http.Error(w, "workspace service unavailable", http.StatusBadGateway)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		if s != nil && s.logger != nil {
			s.logger.Error("workspace service proxy failed",
				"session_id", entry.SessionID,
				"service_id", entry.ServiceID,
				"upstream_url", entry.UpstreamURL,
				"error", proxyErr.Error(),
			)
		}
		http.Error(w, "workspace service unavailable", http.StatusBadGateway)
	}
	w.Header().Set("X-Alter0-Workspace-Service", entry.ServiceID)
	w.Header().Set("X-Alter0-Workspace-Session", entry.SessionID)
	proxy.ServeHTTP(w, r)
}

func (s *Server) workspaceServiceCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s == nil || s.workspaceService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "workspace service registry unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.workspaceService.List()})
}

func (s *Server) workspaceServiceItemHandler(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.workspaceService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "workspace service registry unavailable"})
		return
	}
	sessionID, serviceID, ok := workspaceServiceResourceID(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspace service path"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, found := s.workspaceService.ResolveService(sessionID, serviceID)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "workspace service registration not found"})
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		defer r.Body.Close()
		var req workspaceServiceRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		previous, hadPrevious := s.workspaceService.ResolveService(sessionID, serviceID)
		item, err := s.workspaceService.Upsert(workspaceServiceRegistrationInput{
			SessionID:      sessionID,
			ServiceID:      serviceID,
			ServiceType:    req.ServiceType,
			RepositoryPath: req.RepositoryPath,
			UpstreamURL:    req.UpstreamURL,
			StartCommand:   req.StartCommand,
			Workdir:        req.Workdir,
			HealthPath:     req.HealthPath,
			Port:           req.Port,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		response := workspaceServiceResponse{workspaceServiceRegistration: item}
		if s.workspaceRuntime != nil && isManagedWorkspaceService(item) {
			managedEntry, runtimeStatus, runtimeErr := s.workspaceRuntime.EnsureStarted(item)
			if runtimeErr != nil {
				if hadPrevious {
					_, _ = s.workspaceService.Upsert(workspaceServiceRegistrationToInput(previous))
				} else {
					_, _ = s.workspaceService.Delete(sessionID, serviceID)
				}
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": runtimeErr.Error()})
				return
			}
			response.workspaceServiceRegistration = managedEntry
			response.RuntimeDir = runtimeStatus.RuntimeDir
			response.LogPath = runtimeStatus.LogPath
			response.PID = runtimeStatus.PID
			response.Status = runtimeStatus.Status
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodDelete:
		item, found := s.workspaceService.ResolveService(sessionID, serviceID)
		if found && s.workspaceRuntime != nil {
			if err := s.workspaceRuntime.Stop(item); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		deleted, err := s.workspaceService.Delete(sessionID, serviceID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if !deleted {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "workspace service registration not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
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

func normalizePreviewBaseDomain(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "alter0.cn"
	}
	return strings.Trim(trimmed, ".")
}
