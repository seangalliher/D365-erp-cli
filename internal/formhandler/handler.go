// Package formhandler implements the daemon.Handler interface using a
// headless browser (Playwright/Chromium) to drive D365 F&O form sessions.
// It launches a browser, completes the OIDC login flow, then uses
// Playwright's automation API to interact with D365 forms.
package formhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/seangalliher/d365-erp-cli/internal/daemon"
)

// Handler implements daemon.Handler by driving a headless Chromium browser
// session against the D365 F&O web client via Playwright.
type Handler struct {
	mu      sync.Mutex
	session *browserSession
	envURL  string
	company string
	logger  *log.Logger
	visible bool
}

// Config holds browser handler configuration.
type Config struct {
	EnvURL  string
	Company string
	Visible bool
	Logger  *log.Logger
}

// New creates a new Playwright-based form handler.
func New(cfg Config) (*Handler, error) {
	if cfg.EnvURL == "" {
		return nil, fmt.Errorf("environment URL is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stderr, "[formhandler] ", log.LstdFlags|log.Lmsgprefix)
	}
	return &Handler{
		envURL:  cfg.EnvURL,
		company: cfg.Company,
		visible: cfg.Visible,
		logger:  cfg.Logger,
	}, nil
}

// Handle processes a daemon command by driving the browser.
func (h *Handler) Handle(ctx context.Context, command string, args json.RawMessage) (json.RawMessage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.ensureReady(ctx); err != nil {
		return nil, fmt.Errorf("browser session not ready: %w", err)
	}

	switch command {
	case daemon.CmdFormFindMenu:
		return h.findMenu(ctx, args)
	case daemon.CmdFormOpen:
		return h.openForm(ctx, args)
	case daemon.CmdFormClose:
		return h.closeForm(ctx)
	case daemon.CmdFormSave:
		return h.saveForm(ctx)
	case daemon.CmdFormState:
		return h.getFormState(ctx)
	case daemon.CmdFormClick:
		return h.clickControl(ctx, args)
	case daemon.CmdFormSetValues:
		return h.setValues(ctx, args)
	case daemon.CmdFormOpenLookup:
		return h.openLookup(ctx, args)
	case daemon.CmdFormOpenTab:
		return h.openTab(ctx, args)
	case daemon.CmdFormFilter:
		return h.filterForm(ctx, args)
	case daemon.CmdFormFilterGrid:
		return h.filterGrid(ctx, args)
	case daemon.CmdFormSelectRow:
		return h.selectRow(ctx, args)
	case daemon.CmdFormSortGrid:
		return h.sortGrid(ctx, args)
	case daemon.CmdFormFind:
		return h.findControls(ctx, args)
	default:
		return nil, fmt.Errorf("unsupported command: %s", command)
	}
}

// Close shuts down the browser session.
func (h *Handler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.session != nil {
		h.logger.Println("closing browser session")
		err := h.session.close()
		h.session = nil
		return err
	}
	return nil
}

// WarmUp eagerly starts the browser and completes login.
func (h *Handler) WarmUp(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.ensureReady(ctx)
}

func (h *Handler) ensureReady(ctx context.Context) error {
	if h.session != nil && h.session.isAlive() {
		return nil
	}
	h.logger.Println("starting browser session...")
	sess, err := newBrowserSession(ctx, browserConfig{
		envURL:  h.envURL,
		company: h.company,
		visible: h.visible,
		logger:  h.logger,
	})
	if err != nil {
		return err
	}
	h.session = sess
	return nil
}

func marshal(v interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal response: %w", err)
	}
	return data, nil
}
