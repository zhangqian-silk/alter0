package infrastructure

import (
	"context"
	"strconv"
	"strings"
	"time"

	orchdomain "alter0/internal/orchestration/domain"
)

type CommandCatalog interface {
	List() []string
}

type HelpCommandHandler struct {
	catalog CommandCatalog
}

func NewHelpCommandHandler(catalog CommandCatalog) *HelpCommandHandler {
	return &HelpCommandHandler{catalog: catalog}
}

func (h *HelpCommandHandler) Name() string {
	return "help"
}

func (h *HelpCommandHandler) Aliases() []string {
	return []string{"h"}
}

func (h *HelpCommandHandler) Execute(_ context.Context, _ orchdomain.CommandRequest) (orchdomain.CommandResult, error) {
	commands := h.catalog.List()
	items := make([]string, 0, len(commands))
	for _, command := range commands {
		items = append(items, "/"+command)
	}
	return orchdomain.CommandResult{
		Output: "available commands: " + strings.Join(items, ", "),
		Metadata: map[string]string{
			"command_count": strconv.Itoa(len(commands)),
		},
	}, nil
}

type EchoCommandHandler struct{}

func NewEchoCommandHandler() *EchoCommandHandler {
	return &EchoCommandHandler{}
}

func (e *EchoCommandHandler) Name() string {
	return "echo"
}

func (e *EchoCommandHandler) Aliases() []string {
	return []string{}
}

func (e *EchoCommandHandler) Execute(_ context.Context, req orchdomain.CommandRequest) (orchdomain.CommandResult, error) {
	return orchdomain.CommandResult{
		Output: strings.Join(req.Args, " "),
	}, nil
}

type TimeCommandHandler struct{}

func NewTimeCommandHandler() *TimeCommandHandler {
	return &TimeCommandHandler{}
}

func (t *TimeCommandHandler) Name() string {
	return "time"
}

func (t *TimeCommandHandler) Aliases() []string {
	return []string{"now"}
}

func (t *TimeCommandHandler) Execute(_ context.Context, _ orchdomain.CommandRequest) (orchdomain.CommandResult, error) {
	return orchdomain.CommandResult{
		Output: time.Now().UTC().Format(time.RFC3339),
	}, nil
}
