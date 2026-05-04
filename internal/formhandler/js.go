package formhandler

// jsGetFormState is the JavaScript executed in the D365 page context to
// extract the current form's controls, values, and grid data.
// Scoped to the Form element to exclude D365 shell/chrome noise.
const jsGetFormState = `() => {
	var result = {
		formName: '',
		controls: [],
		tabs: [],
		grids: [],
		actionPanes: []
	};

	// --- Shell noise filters ---
	var shellPrefixes = [
		'messageCenter', 'navBar', 'NavBar', 'EmbeddedBar',
		'UserButton', 'MessageBar', 'SysGen_', 'FormRightActions',
		'FactBoxToggle', 'NavigationBar', 'NavigationSearch',
		'companybutton', 'UserCompany', 'NavBarOffice',
		'NavBarDashboard', 'NavBarActions'
	];
	var shellRoles = [
		'NavigationBar', 'NavigationSearchButton', 'SearchBox',
		'CompanyButton', 'UserButton', 'MessageBar',
		'MessageBarSection', 'MessageCenterButton'
	];
	function isShellControl(name, role) {
		for (var i = 0; i < shellPrefixes.length; i++) {
			if (name.indexOf(shellPrefixes[i]) === 0) return true;
		}
		for (var i = 0; i < shellRoles.length; i++) {
			if (role === shellRoles[i]) return true;
		}
		// Numeric-only names are usually auto-generated shell elements.
		if (/^\d+$/.test(name)) return true;
		return false;
	}

	// --- Locate the form element ---
	var formEl = document.querySelector('[data-dyn-role="Form"]');
	var scope = formEl || document;

	if (formEl) {
		result.formName = formEl.getAttribute('data-dyn-formname') ||
		                  formEl.getAttribute('data-dyn-controlname') || '';
	}
	if (!result.formName) {
		var title = document.title || '';
		result.formName = title.replace(/ -+ .*$/, '').trim();
	}

	// --- Layout containers to skip ---
	var skipRoles = {
		'Group': 1, 'ContentPanel': 1, 'PanoramaSection': 1,
		'FormContent': 1, 'StatusBar': 1
	};

	// --- Collect controls ---
	var seenControls = new Set();
	scope.querySelectorAll('[data-dyn-controlname]').forEach(function(el) {
		var name = el.getAttribute('data-dyn-controlname');
		if (!name || name.startsWith('__') || seenControls.has(name)) return;
		seenControls.add(name);

		var role = el.getAttribute('data-dyn-role') || '';
		var controlType = el.getAttribute('data-dyn-controltype') || role;

		// Skip shell noise.
		if (isShellControl(name, role || controlType)) return;

		// Skip layout containers.
		if (skipRoles[role]) return;

		// --- Tabs / FastTabs ---
		if (role === 'TabPage' || role === 'FastTab' || role === 'FastTabSection' ||
		    controlType === 'TabPage' || controlType === 'FastTab') {
			var labelEl = el.querySelector(':scope > .fasttab-header .caption, :scope > [class*="header"] .caption, label');
			var tabLabel = labelEl ? labelEl.textContent.trim() : '';
			var isExpanded = el.classList.contains('expanded') ||
			                el.getAttribute('aria-expanded') === 'true' ||
			                !el.classList.contains('collapsed');
			result.tabs.push({
				name: name,
				label: tabLabel || name,
				expanded: isExpanded
			});
			return;
		}

		// --- Grids (ReactList / legacy Grid) ---
		if (role === 'ReactList' || role === 'Grid' || controlType === 'ReactList' || controlType === 'Grid') {
			var gridInfo = extractGrid(el, name);
			result.grids.push(gridInfo);
			return;
		}

		// --- Grid column definitions (children of ReactList) ---
		// These have no role but are direct children of a ReactList parent.
		if (!role && !controlType) {
			var parentGrid = el.closest('[data-dyn-role="ReactList"], [data-dyn-role="Grid"]');
			if (parentGrid) return; // Will be handled by grid extraction.
		}

		// --- Action pane tabs ---
		if (role === 'AppBarTab' || controlType === 'AppBarTab') {
			result.actionPanes.push({
				name: name,
				label: el.textContent.trim() || name
			});
			return;
		}

		// Skip action pane infrastructure.
		if (role === 'AppBarSection' || role === 'AppBarSearchField' ||
		    role === 'OverflowButton' || controlType === 'AppBarSection' ||
		    controlType === 'AppBarSearchField' || controlType === 'OverflowButton') return;

		// Skip action groups (just containers for buttons).
		if (role === 'ActionGroup') return;

		// --- Get label ---
		var label = '';
		var lblEl = el.querySelector(':scope > label, :scope > .dynLabel, :scope > .field-label');
		if (!lblEl) lblEl = el.querySelector('label, .dynLabel, .field-label');
		if (lblEl) label = lblEl.textContent.trim();

		// --- Get value ---
		var value = '';
		var input = el.querySelector('input:not([type="hidden"]), textarea, select');
		if (input) {
			if (input.tagName === 'SELECT') {
				value = input.options[input.selectedIndex] ? input.options[input.selectedIndex].text : input.value;
			} else {
				value = input.value || '';
			}
		} else {
			var valEl = el.querySelector('.dynField, .field-value, .staticText');
			if (valEl) value = valEl.textContent.trim();
		}

		// --- Editability ---
		var editable = false;
		if (input) {
			editable = !input.disabled && !input.readOnly;
		}

		// --- Visibility ---
		var visible = el.offsetParent !== null;

		// Only include visible controls, plus buttons which may be in action panes.
		var isButton = role === 'CommandButton' || role === 'MenuItemButton' ||
		               role === 'Button' || role === 'MenuButton' ||
		               role === 'DropDialogButton' || role === 'ToggleButton';

		if (visible || isButton) {
			result.controls.push({
				name: name,
				label: label,
				value: value,
				type: controlType || 'Unknown',
				editable: editable,
				visible: visible
			});
		}
	});

	return JSON.stringify(result);

	// --- Grid extraction helper ---
	function extractGrid(gridEl, gridName) {
		var columns = [];
		var rows = [];

		// D365 ReactList: column definitions are child elements with
		// data-dyn-controlname like "Table_FieldName".
		gridEl.querySelectorAll(':scope > [data-dyn-controlname]').forEach(function(col) {
			var colName = col.getAttribute('data-dyn-controlname');
			if (colName) {
				// Extract header text from aria-label or column header.
				var headerText = col.getAttribute('aria-label') || '';
				if (!headerText) {
					var hdr = col.querySelector('.column-header, [role="columnheader"]');
					if (hdr) headerText = hdr.textContent.trim();
				}
				columns.push({
					name: colName,
					label: headerText || colName
				});
			}
		});

		// If no child column defs, try the ARIA grid role approach.
		if (columns.length === 0) {
			gridEl.querySelectorAll('[role="columnheader"]').forEach(function(hdr) {
				var text = hdr.textContent.trim();
				var colName = hdr.getAttribute('data-dyn-columnname') ||
				              hdr.closest('[data-dyn-controlname]')?.getAttribute('data-dyn-controlname') || text;
				if (text) columns.push({ name: colName, label: text });
			});
		}

		// Extract row data from the rendered grid.
		// ReactList renders rows as div containers with role="row" or class "row".
		var rowEls = gridEl.querySelectorAll('[role="row"]:not([role="row"] [role="row"]), .row-container');
		if (rowEls.length === 0) {
			// Fallback: try table rows.
			rowEls = gridEl.querySelectorAll('tbody tr');
		}

		rowEls.forEach(function(rowEl, rowIdx) {
			// Skip header rows.
			if (rowEl.querySelector('[role="columnheader"]')) return;

			var cells = rowEl.querySelectorAll('[role="gridcell"], td');
			if (cells.length === 0) return;

			var row = {};
			cells.forEach(function(cell, cellIdx) {
				var colKey = cellIdx < columns.length ? columns[cellIdx].name : 'col_' + cellIdx;
				var cellInput = cell.querySelector('input, select, textarea');
				var cellValue = cellInput ? cellInput.value : cell.textContent.trim();
				row[colKey] = cellValue;
			});
			rows.push(row);
		});

		return {
			name: gridName,
			columns: columns.map(function(c) { return c.label; }),
			columnNames: columns.map(function(c) { return c.name; }),
			rows: rows,
			rowCount: rows.length
		};
	}
}`
