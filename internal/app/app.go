// Package app wires together services, coordinates agents, and manages
// application lifecycle.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/fantasy"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/agent"
	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/internal/agentstatus"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/format"
	"github.com/charmbracelet/crush/internal/history"
	"github.com/charmbracelet/crush/internal/home"
	"github.com/charmbracelet/crush/internal/log"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/shell"
	"github.com/charmbracelet/crush/internal/skills"
	"github.com/charmbracelet/crush/internal/subagent"
	"github.com/charmbracelet/crush/internal/tui/components/anim"
	"github.com/charmbracelet/crush/internal/tui/styles"
	"github.com/charmbracelet/crush/internal/update"
	"github.com/charmbracelet/crush/internal/version"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/charmbracelet/x/term"
)

type App struct {
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service

	AgentCoordinator agent.Coordinator
	SubagentRegistry *subagent.Registry
	StatusReporter   *agentstatus.Reporter

	LSPClients *csync.Map[string, *lsp.Client]

	config *config.Config

	serviceEventsWG *sync.WaitGroup
	eventsCtx       context.Context
	events          chan tea.Msg
	mcpInitDone     chan struct{}
	tuiWG           *sync.WaitGroup

	// global context and cleanup functions
	globalCtx    context.Context
	cleanupFuncs []func() error
}

// New initializes a new application instance.
func New(ctx context.Context, conn *sql.DB, cfg *config.Config) (*App, error) {
	q := db.New(conn)
	sessions := session.NewService(q)
	messages := message.NewService(q)
	files := history.NewService(q, conn)
	skipPermissionsRequests := cfg.Permissions != nil && cfg.Permissions.SkipRequests
	var allowedTools []string
	if cfg.Permissions != nil && cfg.Permissions.AllowedTools != nil {
		allowedTools = cfg.Permissions.AllowedTools
	}

	app := &App{
		Sessions:    sessions,
		Messages:    messages,
		History:     files,
		Permissions: permission.NewPermissionService(cfg.WorkingDir(), skipPermissionsRequests, allowedTools),
		LSPClients:  csync.NewMap[string, *lsp.Client](),

		globalCtx: ctx,

		config: cfg,

		events:          make(chan tea.Msg, 100),
		serviceEventsWG: &sync.WaitGroup{},
		tuiWG:           &sync.WaitGroup{},
	}

	app.setupEvents()

	// Initialize LSP clients in the background.
	app.initLSPClients(ctx)

	// Check for updates in the background.
	go app.checkForUpdates(ctx)

	app.mcpInitDone = make(chan struct{})
	go func() {
		slog.Info("Initializing MCP clients")
		mcp.Initialize(ctx, app.Permissions, cfg)
		close(app.mcpInitDone)
	}()

	// cleanup database upon app shutdown
	app.cleanupFuncs = append(app.cleanupFuncs, conn.Close, mcp.Close)

	// Initialize agent status reporter if configured.
	if err := app.initStatusReporter(cfg); err != nil {
		slog.Warn("Failed to initialize agent status reporter", "error", err)
		// Non-fatal: continue without status reporting.
	}

	// TODO: remove the concept of agent config, most likely.
	if !cfg.IsConfigured() {
		slog.Warn("No agent configuration found")
		return app, nil
	}
	if err := app.InitCoderAgent(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize coder agent: %w", err)
	}
	return app, nil
}

// Config returns the application configuration.
func (app *App) Config() *config.Config {
	return app.config
}

// RunNonInteractive runs the application in non-interactive mode with the
// given prompt, printing to stdout.
func (app *App) RunNonInteractive(ctx context.Context, output io.Writer, prompt string, quiet bool) error {
	slog.Info("Running in non-interactive mode")

	// Wait for MCP clients to initialize before starting the run
	select {
	case <-app.mcpInitDone:
		// MCP initialization complete
	case <-ctx.Done():
		return ctx.Err()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		spinner   *format.Spinner
		stdoutTTY bool
		stderrTTY bool
		stdinTTY  bool
	)

	if f, ok := output.(*os.File); ok {
		stdoutTTY = term.IsTerminal(f.Fd())
	}
	stderrTTY = term.IsTerminal(os.Stderr.Fd())
	stdinTTY = term.IsTerminal(os.Stdin.Fd())

	if !quiet && stderrTTY {
		t := styles.CurrentTheme()

		// Detect background color to set the appropriate color for the
		// spinner's 'Generating...' text. Without this, that text would be
		// unreadable in light terminals.
		hasDarkBG := true
		if f, ok := output.(*os.File); ok && stdinTTY && stdoutTTY {
			hasDarkBG = lipgloss.HasDarkBackground(os.Stdin, f)
		}
		defaultFG := lipgloss.LightDark(hasDarkBG)(charmtone.Pepper, t.FgBase)

		spinner = format.NewSpinner(ctx, cancel, anim.Settings{
			Size:        10,
			Label:       "Generating",
			LabelColor:  defaultFG,
			GradColorA:  t.Primary,
			GradColorB:  t.Secondary,
			CycleColors: true,
		})
		spinner.Start()
	}

	// Helper function to stop spinner once.
	stopSpinner := func() {
		if !quiet && spinner != nil {
			spinner.Stop()
			spinner = nil
		}
	}
	defer stopSpinner()

	const maxPromptLengthForTitle = 100
	const titlePrefix = "Non-interactive: "
	var titleSuffix string

	if len(prompt) > maxPromptLengthForTitle {
		titleSuffix = prompt[:maxPromptLengthForTitle] + "..."
	} else {
		titleSuffix = prompt
	}
	title := titlePrefix + titleSuffix

	sess, err := app.Sessions.Create(ctx, title)
	if err != nil {
		return fmt.Errorf("failed to create session for non-interactive mode: %w", err)
	}
	slog.Info("Created session for non-interactive run", "session_id", sess.ID)

	// Automatically approve all permission requests for this non-interactive
	// session.
	app.Permissions.AutoApproveSession(sess.ID)

	type response struct {
		result *fantasy.AgentResult
		err    error
	}
	done := make(chan response, 1)

	go func(ctx context.Context, sessionID, prompt string) {
		result, err := app.AgentCoordinator.Run(ctx, sess.ID, prompt)
		if err != nil {
			done <- response{
				err: fmt.Errorf("failed to start agent processing stream: %w", err),
			}
		}
		done <- response{
			result: result,
		}
	}(ctx, sess.ID, prompt)

	messageEvents := app.Messages.Subscribe(ctx)
	messageReadBytes := make(map[string]int)

	defer func() {
		if stderrTTY {
			_, _ = fmt.Fprintf(os.Stderr, ansi.ResetProgressBar)
		}

		// Always print a newline at the end. If output is a TTY this will
		// prevent the prompt from overwriting the last line of output.
		_, _ = fmt.Fprintln(output)
	}()

	for {
		if stderrTTY {
			// HACK: Reinitialize the terminal progress bar on every iteration
			// so it doesn't get hidden by the terminal due to inactivity.
			_, _ = fmt.Fprintf(os.Stderr, ansi.SetIndeterminateProgressBar)
		}

		select {
		case result := <-done:
			stopSpinner()
			if result.err != nil {
				if errors.Is(result.err, context.Canceled) || errors.Is(result.err, agent.ErrRequestCancelled) {
					slog.Info("Non-interactive: agent processing cancelled", "session_id", sess.ID)
					return nil
				}
				return fmt.Errorf("agent processing failed: %w", result.err)
			}
			return nil

		case event := <-messageEvents:
			msg := event.Payload
			if msg.SessionID == sess.ID && msg.Role == message.Assistant && len(msg.Parts) > 0 {
				stopSpinner()

				content := msg.Content().String()
				readBytes := messageReadBytes[msg.ID]

				if len(content) < readBytes {
					slog.Error("Non-interactive: message content is shorter than read bytes", "message_length", len(content), "read_bytes", readBytes)
					return fmt.Errorf("message content is shorter than read bytes: %d < %d", len(content), readBytes)
				}

				part := content[readBytes:]
				// Trim leading whitespace. Sometimes the LLM includes leading
				// formatting and intentation, which we don't want here.
				if readBytes == 0 {
					part = strings.TrimLeft(part, " \t")
				}
				fmt.Fprint(output, part)
				messageReadBytes[msg.ID] = len(content)
			}

		case <-ctx.Done():
			stopSpinner()
			return ctx.Err()
		}
	}
}

func (app *App) UpdateAgentModel(ctx context.Context) error {
	if app.AgentCoordinator == nil {
		return fmt.Errorf("agent configuration is missing")
	}
	return app.AgentCoordinator.UpdateModels(ctx)
}

func (app *App) setupEvents() {
	ctx, cancel := context.WithCancel(app.globalCtx)
	app.eventsCtx = ctx
	setupSubscriber(ctx, app.serviceEventsWG, "sessions", app.Sessions.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "messages", app.Messages.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "permissions", app.Permissions.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "permissions-notifications", app.Permissions.SubscribeNotifications, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "history", app.History.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "mcp", mcp.SubscribeEvents, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "lsp", SubscribeLSPEvents, app.events)
	cleanupFunc := func() error {
		cancel()
		app.serviceEventsWG.Wait()
		return nil
	}
	app.cleanupFuncs = append(app.cleanupFuncs, cleanupFunc)
}

func setupSubscriber[T any](
	ctx context.Context,
	wg *sync.WaitGroup,
	name string,
	subscriber func(context.Context) <-chan pubsub.Event[T],
	outputCh chan<- tea.Msg,
) {
	wg.Go(func() {
		subCh := subscriber(ctx)
		for {
			select {
			case event, ok := <-subCh:
				if !ok {
					slog.Debug("subscription channel closed", "name", name)
					return
				}
				var msg tea.Msg = event
				select {
				case outputCh <- msg:
				case <-time.After(2 * time.Second):
					slog.Warn("message dropped due to slow consumer", "name", name)
				case <-ctx.Done():
					slog.Debug("subscription cancelled", "name", name)
					return
				}
			case <-ctx.Done():
				slog.Debug("subscription cancelled", "name", name)
				return
			}
		}
	})
}

func (app *App) InitCoderAgent(ctx context.Context) error {
	coderAgentCfg := app.config.Agents[config.AgentCoder]
	if coderAgentCfg.ID == "" {
		return fmt.Errorf("coder agent configuration is missing")
	}
	var err error
	app.AgentCoordinator, err = agent.NewCoordinator(
		ctx,
		app.config,
		app.Sessions,
		app.Messages,
		app.Permissions,
		app.History,
		app.LSPClients,
	)
	if err != nil {
		slog.Error("Failed to create coder agent", "err", err)
		return err
	}

	// Set status reporter on the coordinator if available.
	if app.StatusReporter != nil {
		app.AgentCoordinator.SetStatusReporter(app.StatusReporter)
	}

	// Initialize subagent registry.
	if err := app.initSubagentRegistry(ctx); err != nil {
		slog.Warn("Failed to initialize subagent registry", "error", err)
		// Non-fatal: continue without subagents.
	}

	return nil
}

// initSubagentRegistry initializes the subagent registry and starts file watching.
func (app *App) initSubagentRegistry(ctx context.Context) error {
	// Build watch paths.
	// Use directory of global config as user config dir.
	userConfigDir := filepath.Dir(config.GlobalConfig())
	watchPaths := subagent.DiscoverPaths(app.config.WorkingDir(), userConfigDir)

	app.SubagentRegistry = subagent.NewRegistry(watchPaths)

	// Start the registry (initial discovery + file watching).
	if err := app.SubagentRegistry.Start(ctx); err != nil {
		return fmt.Errorf("starting subagent registry: %w", err)
	}

	// Connect registry to coordinator.
	app.AgentCoordinator.SetSubagentRegistry(app.SubagentRegistry)

	// Add cleanup function.
	app.cleanupFuncs = append(app.cleanupFuncs, func() error {
		return app.SubagentRegistry.Stop()
	})

	// Forward registry events to the TUI events channel.
	go app.forwardSubagentEvents(ctx)

	slog.Info("Subagent registry initialized", "paths", watchPaths, "count", app.SubagentRegistry.Count())
	return nil
}

// forwardSubagentEvents forwards subagent registry events to the TUI.
func (app *App) forwardSubagentEvents(ctx context.Context) {
	if app.SubagentRegistry == nil {
		return
	}

	events := app.SubagentRegistry.Subscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			// Convert to pubsub message.
			var msg pubsub.SubagentEventMsg
			msg.Type = string(event.Type)
			if event.Subagent != nil {
				msg.Name = event.Subagent.Name
			}
			if event.Error != nil {
				msg.Err = event.Error
			}

			// Send to TUI events channel.
			select {
			case app.events <- msg:
			default:
				slog.Debug("Dropping subagent event, channel full")
			}

			// Also invalidate coordinator cache on updates/removes.
			if event.Type == subagent.EventUpdated || event.Type == subagent.EventRemoved {
				slog.Debug("Subagent changed, coordinator cache will be invalidated on next use",
					"name", event.Subagent.Name, "type", event.Type)
			}
		}
	}
}

// ReloadSubagents triggers a reload of all subagent definitions.
func (app *App) ReloadSubagents() error {
	if app.AgentCoordinator == nil {
		return errors.New("agent coordinator not initialized")
	}
	return app.AgentCoordinator.ReloadSubagents()
}

// initStatusReporter initializes the agent status reporter if configured.
func (app *App) initStatusReporter(cfg *config.Config) error {
	// Check for AGENT_STATUS_DISABLE env var.
	if os.Getenv("AGENT_STATUS_DISABLE") == "1" {
		slog.Debug("Agent status reporting disabled via AGENT_STATUS_DISABLE")
		return nil
	}

	// Determine the status directory: config > env var > disabled.
	statusDir := ""
	if cfg.Options != nil && cfg.Options.AgentStatusDir != "" {
		statusDir = cfg.Options.AgentStatusDir
	} else if envDir := os.Getenv("AGENT_STATUS_DIR"); envDir != "" {
		statusDir = envDir
	}

	if statusDir == "" {
		slog.Debug("Agent status reporting disabled (no directory configured)")
		return nil
	}

	reporter, err := agentstatus.NewReporter(statusDir)
	if err != nil {
		return fmt.Errorf("creating status reporter: %w", err)
	}

	app.StatusReporter = reporter

	// Set initial project info.
	projectName := filepath.Base(cfg.WorkingDir())
	reporter.SetProject(projectName, cfg.WorkingDir())

	// Set up the permission service to report waiting status.
	app.Permissions.SetStatusCallback(func(waiting permission.WaitingState) {
		if waiting == permission.WaitingForInput {
			reporter.SetStatus(agentstatus.StatusWaiting)
		} else {
			reporter.SetStatus(agentstatus.StatusWorking)
		}
	})

	// Add cleanup on exit.
	app.cleanupFuncs = append(app.cleanupFuncs, func() error {
		slog.Debug("Closing agent status reporter")
		return reporter.Close()
	})

	slog.Info("Agent status reporter initialized", "dir", statusDir)
	return nil
}

// ListSubagents returns all registered subagents.
func (app *App) ListSubagents() []*subagent.Subagent {
	if app.AgentCoordinator == nil {
		return nil
	}
	return app.AgentCoordinator.ListSubagents()
}

// ListSkills returns all discovered skills from configured paths.
func (app *App) ListSkills() []*skills.Skill {
	cfg := app.Config()
	if len(cfg.Options.SkillsPaths) == 0 {
		return nil
	}
	expandedPaths := make([]string, 0, len(cfg.Options.SkillsPaths))
	for _, pth := range cfg.Options.SkillsPaths {
		expandedPaths = append(expandedPaths, home.Long(pth))
	}
	return skills.Discover(expandedPaths)
}

// Subscribe sends events to the TUI as tea.Msgs.
func (app *App) Subscribe(program *tea.Program) {
	defer log.RecoverPanic("app.Subscribe", func() {
		slog.Info("TUI subscription panic: attempting graceful shutdown")
		program.Quit()
	})

	app.tuiWG.Add(1)
	tuiCtx, tuiCancel := context.WithCancel(app.globalCtx)
	app.cleanupFuncs = append(app.cleanupFuncs, func() error {
		slog.Debug("Cancelling TUI message handler")
		tuiCancel()
		app.tuiWG.Wait()
		return nil
	})
	defer app.tuiWG.Done()

	for {
		select {
		case <-tuiCtx.Done():
			slog.Debug("TUI message handler shutting down")
			return
		case msg, ok := <-app.events:
			if !ok {
				slog.Debug("TUI message channel closed")
				return
			}
			program.Send(msg)
		}
	}
}

// Shutdown performs a graceful shutdown of the application.
func (app *App) Shutdown() {
	start := time.Now()
	defer func() { slog.Info("Shutdown took " + time.Since(start).String()) }()

	// First, cancel all agents and wait for them to finish. This must complete
	// before closing the DB so agents can finish writing their state.
	if app.AgentCoordinator != nil {
		app.AgentCoordinator.CancelAll()
	}

	// Now run remaining cleanup tasks in parallel.
	var wg sync.WaitGroup

	// Kill all background shells.
	wg.Go(func() {
		shell.GetBackgroundShellManager().KillAll()
	})

	// Shutdown all LSP clients.
	shutdownCtx, cancel := context.WithTimeout(app.globalCtx, 5*time.Second)
	defer cancel()
	for name, client := range app.LSPClients.Seq2() {
		wg.Go(func() {
			if err := client.Close(shutdownCtx); err != nil &&
				!errors.Is(err, io.EOF) &&
				!errors.Is(err, context.Canceled) &&
				err.Error() != "signal: killed" {
				slog.Warn("Failed to shutdown LSP client", "name", name, "error", err)
			}
		})
	}

	// Call all cleanup functions.
	for _, cleanup := range app.cleanupFuncs {
		if cleanup != nil {
			wg.Go(func() {
				if err := cleanup(); err != nil {
					slog.Error("Failed to cleanup app properly on shutdown", "error", err)
				}
			})
		}
	}
	wg.Wait()
}

// checkForUpdates checks for available updates.
func (app *App) checkForUpdates(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	info, err := update.Check(checkCtx, version.Version, update.Default)
	if err != nil || !info.Available() {
		return
	}
	app.events <- pubsub.UpdateAvailableMsg{
		CurrentVersion: info.Current,
		LatestVersion:  info.Latest,
		IsDevelopment:  info.IsDevelopment(),
	}
}
