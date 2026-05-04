package formhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/seangalliher/d365-erp-cli/internal/config"
)

// browserConfig holds configuration for a browser session.
type browserConfig struct {
	envURL  string
	company string
	visible bool
	logger  *log.Logger
}

// browserSession manages a Chromium browser connected to a D365 environment.
type browserSession struct {
	pw       *playwright.Playwright
	browser  playwright.Browser
	context  playwright.BrowserContext
	page     playwright.Page
	envURL   string
	company  string
	logger   *log.Logger
}

// userDataDir returns a persistent browser profile directory so that
// AAD SSO cookies survive across daemon restarts.
func userDataDir() (string, error) {
	dir, err := config.ConfigDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "browser-profile"), nil
}

// newBrowserSession launches Chromium via Playwright, navigates to D365,
// and waits for login to complete.
func newBrowserSession(ctx context.Context, cfg browserConfig) (*browserSession, error) {
	dataDir, err := userDataDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine browser profile path: %w", err)
	}

	cfg.logger.Printf("launching browser (visible: %v, profile: %s)", cfg.visible, dataDir)

	// Install Playwright browsers if not already present.
	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}}); err != nil {
		cfg.logger.Printf("playwright install warning: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("cannot start playwright: %w", err)
	}

	// Launch persistent context (keeps cookies/session across restarts).
	context, err := pw.Chromium.LaunchPersistentContext(dataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(!cfg.visible),
		Args: []string{
			"--disable-gpu",
			"--no-first-run",
			"--no-default-browser-check",
		},
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("cannot launch browser: %w", err)
	}

	// Get the default page or create one.
	pages := context.Pages()
	var page playwright.Page
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = context.NewPage()
		if err != nil {
			context.Close()
			pw.Stop()
			return nil, fmt.Errorf("cannot create page: %w", err)
		}
	}

	s := &browserSession{
		pw:      pw,
		context: context,
		page:    page,
		envURL:  cfg.envURL,
		company: cfg.company,
		logger:  cfg.logger,
	}

	// Restore cached cookies (supplements the persistent context).
	if err := s.restoreCookies(); err != nil {
		cfg.logger.Printf("no cached cookies to restore: %v", err)
	} else {
		cfg.logger.Println("restored cached session cookies")
	}

	if err := s.login(ctx); err != nil {
		_ = s.close()
		return nil, fmt.Errorf("login failed: %w", err)
	}

	// Cache cookies for next time.
	if err := s.saveCookies(); err != nil {
		cfg.logger.Printf("warning: could not cache cookies: %v", err)
	} else {
		cfg.logger.Println("session cookies cached for next startup")
	}

	return s, nil
}

// login navigates to D365 and waits for the app to load.
func (s *browserSession) login(ctx context.Context) error {
	s.logger.Println("navigating to D365...")

	if _, err := s.page.Goto(s.envURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60000),
	}); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	s.logger.Println("waiting for D365 to load (complete login if prompted)...")

	// Wait for D365 shell to appear (up to 5 minutes for login).
	deadline := time.After(5 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("login timeout — D365 did not load within 5 minutes")
		case <-ticker.C:
			if s.isD365Loaded() {
				s.logger.Println("D365 loaded successfully")
				time.Sleep(2 * time.Second)
				return nil
			}
		}
	}
}

// isD365Loaded checks whether D365 has finished loading.
func (s *browserSession) isD365Loaded() bool {
	url := s.page.URL()

	// Still on login page?
	if strings.Contains(url, "login.microsoftonline.com") ||
		strings.Contains(url, "login.windows.net") ||
		strings.Contains(url, "login.live.com") {
		return false
	}

	if !strings.HasPrefix(url, s.envURL) {
		return false
	}

	// Check for D365 shell elements.
	result, err := s.page.Evaluate(`() => {
		var indicators = [
			'[data-dyn-role="Shell"]',
			'[data-dyn-role="Navigation"]',
			'[data-dyn-role="Form"]',
			'.modulesFlyout',
			'.dashboard-wrapper'
		];
		for (var i = 0; i < indicators.length; i++) {
			if (document.querySelector(indicators[i])) return true;
		}
		return false;
	}`)
	if err != nil {
		return false
	}
	loaded, ok := result.(bool)
	return ok && loaded
}

// isAlive checks whether the browser session is still usable.
func (s *browserSession) isAlive() bool {
	if s == nil || s.page == nil {
		return false
	}
	_, err := s.page.Evaluate(`() => document.readyState`)
	return err == nil
}

// navigate opens a URL and waits for D365 to stabilize.
func (s *browserSession) navigate(ctx context.Context, targetURL string) error {
	// Dismiss any blocking dialogs first.
	s.dismissDialogs()

	if _, err := s.page.Goto(targetURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	}); err != nil {
		s.logger.Printf("navigation warning: %v", err)
		s.dismissDialogs()
	}

	// Wait for page to settle.
	time.Sleep(2 * time.Second)

	return nil
}

// dismissDialogs clicks Yes/OK/Close on any visible D365 dialogs.
func (s *browserSession) dismissDialogs() {
	selectors := []string{
		"[data-dyn-controlname=\"Yes_Button\"] button",
		"[data-dyn-controlname=\"YesButton\"] button",
		"[data-dyn-controlname=\"OKButton\"] button",
		"[data-dyn-controlname=\"Ok\"] button",
		"button[title=\"Yes\"]",
		"button[title=\"OK\"]",
		"button[title=\"Close\"]",
	}
	for _, sel := range selectors {
		loc := s.page.Locator(sel).First()
		if visible, _ := loc.IsVisible(); visible {
			_ = loc.Click()
			return
		}
	}
}

// eval executes JavaScript in the page and returns the result.
func (s *browserSession) eval(js string) (interface{}, error) {
	return s.page.Evaluate(js)
}

// evalString executes JS and returns the string result.
func (s *browserSession) evalString(js string) (string, error) {
	result, err := s.page.Evaluate(js)
	if err != nil {
		return "", err
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	// Marshal non-string results.
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// close shuts down the browser.
func (s *browserSession) close() error {
	_ = s.saveCookies()

	var firstErr error
	if s.context != nil {
		if err := s.context.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.pw != nil {
		if err := s.pw.Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ---------------------------------------------------------------------------
// Cookie persistence
// ---------------------------------------------------------------------------

func cookieCachePath() (string, error) {
	dir, err := config.ConfigDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session-cookies.json"), nil
}

func (s *browserSession) saveCookies() error {
	cookies, err := s.context.Cookies()
	if err != nil {
		return fmt.Errorf("cannot get cookies: %w", err)
	}

	// Filter to relevant domains.
	var relevant []playwright.Cookie
	for _, c := range cookies {
		domain := c.Domain
		if strings.HasPrefix(domain, ".") {
			domain = domain[1:]
		}
		if strings.Contains(domain, "dynamics.com") ||
			strings.Contains(domain, "microsoftonline.com") ||
			strings.Contains(domain, "login.windows.net") ||
			strings.Contains(domain, "login.live.com") {
			relevant = append(relevant, c)
		}
	}

	data, err := json.Marshal(relevant)
	if err != nil {
		return fmt.Errorf("cannot marshal cookies: %w", err)
	}

	path, err := cookieCachePath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("cannot write cookie cache: %w", err)
	}

	s.logger.Printf("saved %d session cookies to cache", len(relevant))
	return nil
}

func (s *browserSession) restoreCookies() error {
	path, err := cookieCachePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("no cookie cache: %w", err)
	}

	var cookies []playwright.OptionalCookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return fmt.Errorf("invalid cookie cache: %w", err)
	}

	if len(cookies) == 0 {
		return fmt.Errorf("cookie cache is empty")
	}

	if err := s.context.AddCookies(cookies); err != nil {
		return fmt.Errorf("cannot inject cookies: %w", err)
	}

	s.logger.Printf("restored %d cookies from cache", len(cookies))
	return nil
}
