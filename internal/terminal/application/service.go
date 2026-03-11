package application

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	sharedapp "alter0/internal/shared/application"
	terminaldomain "alter0/internal/terminal/domain"
)

const (
	defaultCodexCommand            = "codex"
	defaultCodexSandbox            = "danger-full-access"
	defaultLinuxSandboxBwrapFeature = "use_linux_sandbox_bwrap"
	maxEntryPageLimit              = 200
)

var (
	ErrSessionOwnerRequired = errors.New("terminal session owner is required")
	ErrSessionNotFound      = errors.New("terminal session not found")
	ErrSessionNotRunning    = errors.New("terminal session is not running")
	ErrSessionLimitReached  = errors.New("terminal session limit reached")
	ErrSessionBusy          = errors.New("terminal session is processing another turn")
	ErrSessionInputRequired = errors.New("terminal input is required")
	ErrTurnNotFound         = errors.New("terminal turn not found")
	ErrStepNotFound         = errors.New("terminal step not found")
)

type Options struct {
	MaxSessions   int
	WorkingDir    string
	Shell         string
	ShellArgs     []string
	ShellArgsLine string
}

type CreateRequest struct {
	OwnerID string
	Title   string
}

type EntryPage struct {
	Items      []terminaldomain.Entry `json:"items"`
	Cursor     int                    `json:"cursor"`
	NextCursor int                    `json:"next_cursor"`
	HasMore    bool                   `json:"has_more"`
}

type commandRunner func(ctx context.Context, name string, args ...string) *exec.Cmd

type Service struct {
	rootCtx     context.Context
	idGenerator sharedapp.IDGenerator
	logger      *slog.Logger
	options     Options
	runner      commandRunner

	mu       sync.RWMutex
	sessions map[string]*runtimeSession
}

type codexCommand struct {
	path       string
	globalArgs []string
	label      string
}

type codexExecEvent struct {
	Type     string         `json:"type"`
	ThreadID string         `json:"thread_id,omitempty"`
	Item     *codexExecItem `json:"item,omitempty"`
}

type codexExecItem struct {
	ID               string `json:"id,omitempty"`
	Type             string `json:"type,omitempty"`
	Text             string `json:"text,omitempty"`
	Delta            string `json:"delta,omitempty"`
	Command          string `json:"command,omitempty"`
	AggregatedOutput string `json:"aggregated_output,omitempty"`
	Status           string `json:"status,omitempty"`
	ExitCode         *int   `json:"exit_code,omitempty"`
}