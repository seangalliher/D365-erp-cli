package daemon

import (
	"encoding/json"
	"testing"
)

func TestRequest_Marshal(t *testing.T) {
	t.Parallel()

	args, _ := json.Marshal(&FormOpenArgs{
		MenuItemName: "CustTable",
		MenuItemType: "Display",
		Company:      "USMF",
	})

	req := &Request{
		ID:      "req-1",
		Command: CmdFormOpen,
		Args:    args,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ID != "req-1" {
		t.Errorf("expected ID 'req-1', got %q", decoded.ID)
	}
	if decoded.Command != CmdFormOpen {
		t.Errorf("expected command %q, got %q", CmdFormOpen, decoded.Command)
	}

	var openArgs FormOpenArgs
	if err := json.Unmarshal(decoded.Args, &openArgs); err != nil {
		t.Fatalf("unmarshal args: %v", err)
	}
	if openArgs.MenuItemName != "CustTable" {
		t.Errorf("expected menu item 'CustTable', got %q", openArgs.MenuItemName)
	}
}

func TestResponse_Success(t *testing.T) {
	t.Parallel()

	data, _ := json.Marshal(map[string]string{"status": "ok"})
	resp := &Response{
		ID:      "req-1",
		Success: true,
		Data:    data,
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !decoded.Success {
		t.Error("expected success=true")
	}
	if decoded.Error != nil {
		t.Error("expected no error")
	}
}

func TestResponse_Error(t *testing.T) {
	t.Parallel()

	resp := &Response{
		ID:      "req-2",
		Success: false,
		Error: &ErrorDetail{
			Code:    "FORM_REQUIRED",
			Message: "no form is open",
		},
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Success {
		t.Error("expected success=false")
	}
	if decoded.Error == nil {
		t.Fatal("expected error")
	}
	if decoded.Error.Code != "FORM_REQUIRED" {
		t.Errorf("expected error code 'FORM_REQUIRED', got %q", decoded.Error.Code)
	}
}

func TestPingResponse_Marshal(t *testing.T) {
	t.Parallel()

	ping := PingResponse{
		Status:   "ok",
		Uptime:   120,
		FormOpen: true,
		FormName: "CustTable",
	}

	data, err := json.Marshal(ping)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded PingResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Uptime != 120 {
		t.Errorf("expected uptime 120, got %d", decoded.Uptime)
	}
	if !decoded.FormOpen {
		t.Error("expected form_open=true")
	}
	if decoded.FormName != "CustTable" {
		t.Errorf("expected form_name 'CustTable', got %q", decoded.FormName)
	}
}

func TestFormArgs_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args interface{}
	}{
		{"open", &FormOpenArgs{MenuItemName: "CustTable", MenuItemType: "Display"}},
		{"click", &FormClickArgs{ControlName: "PostButton", ActionID: "action1"}},
		{"set", &FormSetValuesArgs{Values: []ControlValue{{ControlName: "Name", Value: "Test"}}}},
		{"lookup", &FormLookupArgs{LookupName: "CustomerLookup"}},
		{"tab", &FormTabArgs{TabName: "General", TabAction: "Open"}},
		{"filter", &FormFilterArgs{ControlName: "Status", FilterValue: "Active"}},
		{"grid_filter", &FormGridFilterArgs{GridName: "Grid", GridColumnName: "Name", GridColumnValue: "Test"}},
		{"select_row", &FormSelectRowArgs{GridName: "Grid", RowNumber: "1", Marking: "Marked"}},
		{"sort", &FormSortGridArgs{GridName: "Grid", GridColumnName: "Name", SortDirection: "Ascending"}},
		{"find_controls", &FormFindControlsArgs{SearchTerm: "customer"}},
		{"find_menu", &FormFindMenuArgs{Filter: "Cust", ResponseSize: "10"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("marshal %s: %v", tt.name, err)
			}
			if len(data) == 0 {
				t.Errorf("expected non-empty JSON for %s", tt.name)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	t.Parallel()

	commands := []string{
		CmdPing, CmdShutdown, CmdFormOpen, CmdFormClose, CmdFormSave,
		CmdFormState, CmdFormClick, CmdFormSetValues, CmdFormOpenLookup,
		CmdFormOpenTab, CmdFormFilter, CmdFormFilterGrid, CmdFormSelectRow,
		CmdFormSortGrid, CmdFormFind, CmdFormFindMenu,
	}

	seen := make(map[string]bool)
	for _, cmd := range commands {
		if cmd == "" {
			t.Error("empty command constant")
		}
		if seen[cmd] {
			t.Errorf("duplicate command constant: %q", cmd)
		}
		seen[cmd] = true
	}
}
