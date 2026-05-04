package formhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/seangalliher/d365-erp-cli/internal/daemon"
)

// ---------------------------------------------------------------------------
// form.find_menu — uses D365's navigation search
// ---------------------------------------------------------------------------

func (h *Handler) findMenu(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormFindMenuArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid find_menu args: %w", err)
	}

	limit := 50
	if a.ResponseSize != "" {
		if n, err := strconv.Atoi(a.ResponseSize); err == nil && n > 0 {
			limit = n
		}
	}
	h.logger.Printf("finding menu items matching %q (limit %d)", a.Filter, limit)

	result, err := h.session.evalString(fmt.Sprintf(`() => {
		return new Promise((resolve) => {
			var search = document.querySelector('[data-dyn-controlname="NavigationSearchBox"] input') ||
			             document.querySelector('.navigation-search-input input');
			if (!search) { resolve(JSON.stringify({error: "navigation search not found"})); return; }
			var nativeSet = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
			nativeSet.call(search, %q);
			search.dispatchEvent(new Event('input', {bubbles: true}));
			search.dispatchEvent(new Event('change', {bubbles: true}));
			setTimeout(function() {
				var items = [];
				document.querySelectorAll('.navigation-search-results .result-item, [data-dyn-role="NavigationSearchResult"]').forEach(function(el, idx) {
					if (idx >= %d) return;
					items.push({
						name: el.getAttribute('data-dyn-menuitemname') || el.querySelector('.result-name')?.textContent?.trim() || '',
						label: el.querySelector('.result-label, .result-name')?.textContent?.trim() || '',
						type: el.getAttribute('data-dyn-menuitemtype') || 'Display'
					});
				});
				resolve(JSON.stringify({menuItems: items}));
			}, 3000);
		});
	}`, a.Filter, limit))
	if err != nil {
		return nil, fmt.Errorf("menu search failed: %w", err)
	}
	return json.RawMessage(result), nil
}

// ---------------------------------------------------------------------------
// form.open
// ---------------------------------------------------------------------------

func (h *Handler) openForm(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormOpenArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid open args: %w", err)
	}

	company := a.Company
	if company == "" {
		company = h.company
	}

	formURL := h.envURL + "/?mi=" + a.MenuItemName
	if company != "" {
		formURL += "&cmp=" + company
	}

	h.logger.Printf("opening form: %s (type=%s, company=%s)", a.MenuItemName, a.MenuItemType, company)

	if err := h.session.navigate(ctx, formURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to form: %w", err)
	}

	// Wait for form to load.
	if err := h.waitForForm(ctx, 60*time.Second); err != nil {
		h.logger.Printf("warning: form load wait: %v (proceeding anyway)", err)
	}

	return h.getFormState(ctx)
}

func (h *Handler) waitForForm(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("form load timeout")
		case <-ticker.C:
			result, err := h.session.page.Evaluate(`() => {
				var form = document.querySelector('[data-dyn-role="Form"]');
				if (!form) return false;
				var ctrls = form.querySelectorAll('[data-dyn-controlname]');
				if (ctrls.length < 3) return false;
				var loading = document.querySelector('.loading-overlay:not(.hidden), .busy-indicator:not(.hidden)');
				if (loading && loading.offsetParent !== null) return false;
				return true;
			}`)
			if err == nil {
				if loaded, ok := result.(bool); ok && loaded {
					time.Sleep(2 * time.Second)
					return nil
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// form.close
// ---------------------------------------------------------------------------

func (h *Handler) closeForm(ctx context.Context) (json.RawMessage, error) {
	h.logger.Println("closing form")

	loc := h.session.page.Locator("[data-dyn-controlname=\"SystemDefinedCloseButton\"] button")
	if visible, _ := loc.IsVisible(); visible {
		_ = loc.Click()
	} else {
		_ = h.session.navigate(ctx, h.envURL)
	}

	time.Sleep(1 * time.Second)
	return marshal(map[string]string{"status": "closed"})
}

// ---------------------------------------------------------------------------
// form.save
// ---------------------------------------------------------------------------

func (h *Handler) saveForm(ctx context.Context) (json.RawMessage, error) {
	h.logger.Println("saving form")

	// Try multiple save approaches in order of specificity.
	saved := false

	// 1. Try the form's standard save button.
	sysBtn := h.session.page.Locator("[data-dyn-controlname=\"SystemDefinedSaveButton\"] button")
	if visible, _ := sysBtn.IsVisible(); visible {
		_ = sysBtn.Click()
		saved = true
	}

	// 2. Try by role with exact name "Save" (works in dialogs).
	if !saved {
		roleBtn := h.session.page.GetByRole("button", playwright.PageGetByRoleOptions{
			Name:  "Save",
			Exact: playwright.Bool(true),
		})
		if visible, _ := roleBtn.IsVisible(); visible {
			_ = roleBtn.Click()
			saved = true
		}
	}

	// 3. Try SaveCloseButton (dialog save-and-close).
	if !saved {
		closeBtn := h.session.page.Locator("[data-dyn-controlname=\"SaveCloseButton\"]").First()
		if visible, _ := closeBtn.IsVisible(); visible {
			_ = closeBtn.Click(playwright.LocatorClickOptions{Force: playwright.Bool(true)})
			saved = true
		}
	}

	// 4. Fallback: Ctrl+S.
	if !saved {
		_ = h.session.page.Keyboard().Press("Control+s")
	}

	// Wait for save to process. Don't use NetworkIdle — D365 has persistent
	// SignalR/telemetry connections that prevent it from ever reaching idle.
	time.Sleep(5 * time.Second)

	// Check for errors.
	result, err := h.session.evalString(jsSaveVerification)
	if err != nil {
		return nil, fmt.Errorf("save verification failed: %w", err)
	}
	return json.RawMessage(result), nil
}

// ---------------------------------------------------------------------------
// form.state
// ---------------------------------------------------------------------------

func (h *Handler) getFormState(ctx context.Context) (json.RawMessage, error) {
	result, err := h.session.evalString(jsGetFormState)
	if err != nil {
		return nil, fmt.Errorf("cannot read form state: %w", err)
	}
	return json.RawMessage(result), nil
}

// ---------------------------------------------------------------------------
// form.click — uses Playwright's native click (handles dialogs properly)
// ---------------------------------------------------------------------------

func (h *Handler) clickControl(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormClickArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid click args: %w", err)
	}

	h.logger.Printf("clicking control: %s (actionId=%s)", a.ControlName, a.ActionID)

	selector := fmt.Sprintf("[data-dyn-controlname=\"%s\"]", a.ControlName)

	if a.ActionID != "" {
		subSel := fmt.Sprintf("%s [data-dyn-actionid=\"%s\"], %s [data-dyn-optionid=\"%s\"]",
			selector, a.ActionID, selector, a.ActionID)
		loc := h.session.page.Locator(subSel).First()
		if visible, _ := loc.IsVisible(); visible {
			if err := loc.Click(); err == nil {
				goto afterClick
			}
		}
	}

	{
		// D365 action pane buttons (MenuItemButton) don't use <button> elements.
		// Use Force:true to bypass actionability checks on these controls.
		loc := h.session.page.Locator(selector).First()
		if err := loc.Click(playwright.LocatorClickOptions{
			Force:   playwright.Bool(true),
			Timeout: playwright.Float(5000),
		}); err != nil {
			return nil, fmt.Errorf("cannot click %s: %w", a.ControlName, err)
		}
	}

afterClick:
	time.Sleep(2 * time.Second)

	// Extra wait for buttons that trigger form transitions.
	if strings.Contains(strings.ToLower(a.ControlName), "new") ||
		strings.Contains(strings.ToLower(a.ControlName), "create") ||
		strings.Contains(strings.ToLower(a.ControlName), "save") {
		time.Sleep(3 * time.Second)
	}

	return h.getFormState(ctx)
}

// ---------------------------------------------------------------------------
// form.set_values — uses Playwright's Type() for D365 Knockout.js binding.
// Type() sends character-by-character keydown events which trigger
// Knockout's valueUpdate:'afterkeydown' bindings. Fill() alone doesn't.
// ---------------------------------------------------------------------------

func (h *Handler) setValues(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormSetValuesArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid set_values args: %w", err)
	}

	for _, cv := range a.Values {
		h.logger.Printf("setting %s = %s", cv.ControlName, cv.Value)

		selector := fmt.Sprintf("[data-dyn-controlname=\"%s\"] input:not([type=\"hidden\"]), [data-dyn-controlname=\"%s\"] textarea",
			cv.ControlName, cv.ControlName)

		loc := h.session.page.Locator(selector).First()

		// Click the input to focus it.
		if err := loc.Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(5000),
		}); err != nil {
			return nil, fmt.Errorf("cannot click %q: %w", cv.ControlName, err)
		}

		// Clear existing value.
		_ = loc.Fill("", playwright.LocatorFillOptions{Timeout: playwright.Float(2000)})

		// Type character-by-character with delay to trigger Knockout.js bindings.
		if err := loc.Type(cv.Value, playwright.LocatorTypeOptions{
			Delay:   playwright.Float(50),
			Timeout: playwright.Float(10000),
		}); err != nil {
			return nil, fmt.Errorf("cannot type into %q: %w", cv.ControlName, err)
		}

		// Tab out to trigger D365 validation/binding.
		_ = h.session.page.Keyboard().Press("Tab")
		time.Sleep(1 * time.Second)
	}

	return h.getFormState(ctx)
}

// ---------------------------------------------------------------------------
// form.open_lookup
// ---------------------------------------------------------------------------

func (h *Handler) openLookup(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormLookupArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid lookup args: %w", err)
	}
	h.logger.Printf("opening lookup: %s", a.LookupName)

	selector := fmt.Sprintf("[data-dyn-controlname=\"%s\"]", a.LookupName)
	btnLoc := h.session.page.Locator(fmt.Sprintf("%s .lookupButton, %s [data-dyn-role=\"LookupButton\"], %s button[title=\"Lookup\"]",
		selector, selector, selector)).First()

	if visible, _ := btnLoc.IsVisible(); visible {
		_ = btnLoc.Click()
	} else {
		// Alt+Down to open lookup.
		inputLoc := h.session.page.Locator(fmt.Sprintf("%s input", selector)).First()
		_ = inputLoc.Click()
		_ = h.session.page.Keyboard().Press("Alt+ArrowDown")
	}

	time.Sleep(1 * time.Second)
	return h.getFormState(ctx)
}

// ---------------------------------------------------------------------------
// form.open_tab
// ---------------------------------------------------------------------------

func (h *Handler) openTab(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormTabArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid tab args: %w", err)
	}
	h.logger.Printf("tab %s: %s", strings.ToLower(a.TabAction), a.TabName)

	selector := fmt.Sprintf("[data-dyn-controlname=\"%s\"]", a.TabName)
	headerLoc := h.session.page.Locator(fmt.Sprintf("%s .fasttab-header, %s .tab-header, %s [role=\"tab\"]",
		selector, selector, selector)).First()

	if visible, _ := headerLoc.IsVisible(); visible {
		_ = headerLoc.Click()
	} else {
		h.session.page.Locator(selector).First().Click()
	}

	time.Sleep(1 * time.Second)
	return h.getFormState(ctx)
}

// ---------------------------------------------------------------------------
// form.filter
// ---------------------------------------------------------------------------

func (h *Handler) filterForm(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormFilterArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid filter args: %w", err)
	}
	h.logger.Printf("filtering form: %s = %s", a.ControlName, a.FilterValue)

	selector := fmt.Sprintf("[data-dyn-controlname=\"%s\"] input", a.ControlName)
	loc := h.session.page.Locator(selector).First()
	_ = loc.Fill(a.FilterValue, playwright.LocatorFillOptions{Timeout: playwright.Float(5000)})
	_ = h.session.page.Keyboard().Press("Enter")

	time.Sleep(2 * time.Second)
	return h.getFormState(ctx)
}

// ---------------------------------------------------------------------------
// form.filter_grid
// ---------------------------------------------------------------------------

func (h *Handler) filterGrid(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormGridFilterArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid grid filter args: %w", err)
	}
	h.logger.Printf("filtering grid %s: %s = %s", a.GridName, a.GridColumnName, a.GridColumnValue)

	result, err := h.session.evalString(fmt.Sprintf(`() => {
		var grid = document.querySelector('[data-dyn-controlname="%s"]');
		if (!grid) return JSON.stringify({error: "grid not found: %s"});
		var header = grid.querySelector('[data-dyn-columnname="%s"] .column-filter input') ||
		             grid.querySelector('th[data-dyn-columnname="%s"] input');
		if (!header) return JSON.stringify({error: "column filter not found: %s"});
		var nativeSet = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
		nativeSet.call(header, %q);
		header.dispatchEvent(new Event('input', {bubbles: true}));
		header.dispatchEvent(new Event('change', {bubbles: true}));
		header.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
		return JSON.stringify({status: "filtered"});
	}`, a.GridName, a.GridName, a.GridColumnName, a.GridColumnName, a.GridColumnName, a.GridColumnValue))
	if err != nil {
		return nil, fmt.Errorf("grid filter failed: %w", err)
	}

	time.Sleep(2 * time.Second)

	var resp map[string]interface{}
	if json.Unmarshal([]byte(result), &resp) == nil {
		if errMsg, ok := resp["error"].(string); ok {
			return nil, fmt.Errorf("%s", errMsg)
		}
	}
	return h.getFormState(ctx)
}

// ---------------------------------------------------------------------------
// form.select_row
// ---------------------------------------------------------------------------

func (h *Handler) selectRow(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormSelectRowArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid select_row args: %w", err)
	}
	h.logger.Printf("selecting row %s in grid %s", a.RowNumber, a.GridName)

	result, err := h.session.evalString(fmt.Sprintf(`() => {
		var grid = document.querySelector('[data-dyn-controlname="%s"]');
		if (!grid) return JSON.stringify({error: "grid not found: %s"});
		var rows = grid.querySelectorAll('[role="row"]:not(:has([role="columnheader"])), tr.row');
		var idx = parseInt(%q, 10) - 1;
		if (idx < 0 || idx >= rows.length) return JSON.stringify({error: "row out of range"});
		rows[idx].click();
		if (%q === 'Marked') { var cb = rows[idx].querySelector('input[type="checkbox"]'); if (cb && !cb.checked) cb.click(); }
		if (%q === 'Unmarked') { var cb = rows[idx].querySelector('input[type="checkbox"]'); if (cb && cb.checked) cb.click(); }
		return JSON.stringify({status: "selected", row: %q});
	}`, a.GridName, a.GridName, a.RowNumber, a.Marking, a.Marking, a.RowNumber))
	if err != nil {
		return nil, fmt.Errorf("row selection failed: %w", err)
	}

	time.Sleep(1 * time.Second)

	var resp map[string]interface{}
	if json.Unmarshal([]byte(result), &resp) == nil {
		if errMsg, ok := resp["error"].(string); ok {
			return nil, fmt.Errorf("%s", errMsg)
		}
	}
	return h.getFormState(ctx)
}

// ---------------------------------------------------------------------------
// form.sort_grid
// ---------------------------------------------------------------------------

func (h *Handler) sortGrid(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormSortGridArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid sort_grid args: %w", err)
	}
	h.logger.Printf("sorting grid %s by %s (%s)", a.GridName, a.GridColumnName, a.SortDirection)

	result, err := h.session.evalString(fmt.Sprintf(`() => {
		var grid = document.querySelector('[data-dyn-controlname="%s"]');
		if (!grid) return JSON.stringify({error: "grid not found: %s"});
		var header = grid.querySelector('[data-dyn-columnname="%s"] .column-header') ||
		             grid.querySelector('th[data-dyn-columnname="%s"]');
		if (!header) return JSON.stringify({error: "column not found: %s"});
		header.click();
		if ('%s' === 'Descending') setTimeout(function() { header.click(); }, 500);
		return JSON.stringify({status: "sorted"});
	}`, a.GridName, a.GridName, a.GridColumnName, a.GridColumnName, a.GridColumnName, a.SortDirection))
	if err != nil {
		return nil, fmt.Errorf("sort failed: %w", err)
	}

	time.Sleep(2 * time.Second)

	var resp map[string]interface{}
	if json.Unmarshal([]byte(result), &resp) == nil {
		if errMsg, ok := resp["error"].(string); ok {
			return nil, fmt.Errorf("%s", errMsg)
		}
	}
	return h.getFormState(ctx)
}

// ---------------------------------------------------------------------------
// form.find_controls
// ---------------------------------------------------------------------------

func (h *Handler) findControls(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a daemon.FormFindControlsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid find_controls args: %w", err)
	}
	h.logger.Printf("finding controls matching %q", a.SearchTerm)

	result, err := h.session.evalString(fmt.Sprintf(`() => {
		var term = %q.toLowerCase();
		var matches = [];
		document.querySelectorAll('[data-dyn-controlname]').forEach(function(el) {
			var name = el.getAttribute('data-dyn-controlname') || '';
			var role = el.getAttribute('data-dyn-role') || '';
			var label = '';
			var labelEl = el.querySelector('label, .dynLabel');
			if (labelEl) label = labelEl.textContent.trim();
			if (name.toLowerCase().indexOf(term) !== -1 || label.toLowerCase().indexOf(term) !== -1) {
				matches.push({ name: name, label: label, type: role, visible: el.offsetParent !== null });
			}
		});
		return JSON.stringify({controls: matches});
	}`, a.SearchTerm))
	if err != nil {
		return nil, fmt.Errorf("find controls failed: %w", err)
	}
	return json.RawMessage(result), nil
}

// ---------------------------------------------------------------------------
// Save verification JS
// ---------------------------------------------------------------------------

const jsSaveVerification = `() => {
	var result = { status: 'saved', errors: [], warnings: [], infos: [] };
	var msgBar = document.querySelector('[data-dyn-role="MessageBar"]');
	if (msgBar) {
		msgBar.querySelectorAll('.messageBar-message, [data-dyn-role="MessageBarMessage"]').forEach(function(msg) {
			var text = msg.textContent.trim();
			if (!text) return;
			var isError = msg.closest('.error, [class*="error"]') !== null;
			if (isError) { result.errors.push(text); result.status = 'error'; }
			else { result.infos.push(text); }
		});
	}
	var dialog = document.querySelector('[data-dyn-role="Dialog"]');
	if (dialog && dialog.offsetParent !== null) {
		var errText = dialog.querySelector('.error, [class*="error"]');
		if (errText) {
			result.errors.push(errText.textContent.trim().substring(0, 500));
			result.status = 'error';
		}
	}
	return JSON.stringify(result);
}`
