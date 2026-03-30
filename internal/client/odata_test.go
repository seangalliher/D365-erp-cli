package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildQueryString_Empty(t *testing.T) {

	qs := BuildQueryString(QueryOptions{})
	// Should auto-inject cross-company=true when CrossCompany is nil.
	if !strings.Contains(qs, "cross-company=true") {
		t.Errorf("expected cross-company=true in query string, got: %s", qs)
	}
}

func TestBuildQueryString_AllFields(t *testing.T) {

	crossCompanyTrue := true
	qs := BuildQueryString(QueryOptions{
		Filter:       "Name eq 'Contoso'",
		Select:       "CustomerAccount,Name",
		Expand:       "SalesOrders",
		Top:          10,
		Skip:         5,
		OrderBy:      "Name asc",
		Count:        true,
		CrossCompany: &crossCompanyTrue,
	})

	checks := []string{
		"cross-company=true",
		"%24filter=Name+eq+%27Contoso%27",
		"%24select=CustomerAccount%2CName",
		"%24expand=SalesOrders",
		"%24top=10",
		"%24skip=5",
		"%24orderby=Name+asc",
		"%24count=true",
	}
	for _, check := range checks {
		if !strings.Contains(qs, check) {
			t.Errorf("expected %q in query string, got: %s", check, qs)
		}
	}
}

func TestBuildQueryString_CrossCompanyAutoInject(t *testing.T) {

	// nil CrossCompany should auto-inject cross-company=true.
	qs := BuildQueryString(QueryOptions{Filter: "Status eq 'Active'"})
	if !strings.Contains(qs, "cross-company=true") {
		t.Errorf("expected cross-company=true to be auto-injected, got: %s", qs)
	}
}

func TestBuildQueryString_CrossCompanyExplicitFalse(t *testing.T) {

	crossCompanyFalse := false
	qs := BuildQueryString(QueryOptions{
		CrossCompany: &crossCompanyFalse,
		Filter:       "Name eq 'X'",
	})
	if strings.Contains(qs, "cross-company") {
		t.Errorf("cross-company should be omitted when explicitly false, got: %s", qs)
	}
}

func TestBuildQueryString_TopOnly(t *testing.T) {

	qs := BuildQueryString(QueryOptions{Top: 5})
	if !strings.Contains(qs, "%24top=5") {
		t.Errorf("expected $top=5, got: %s", qs)
	}
}

func TestODataResponseParsing(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"@odata.context":  "https://env.operations.dynamics.com/data/$metadata#Customers",
			"@odata.count":    2,
			"@odata.nextLink": "https://env.operations.dynamics.com/data/Customers?$skip=2",
			"value": []map[string]interface{}{
				{"CustomerAccount": "US-001", "Name": "Contoso"},
				{"CustomerAccount": "US-002", "Name": "Fabrikam"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	result, err := oc.QueryEntities(context.Background(), "Customers", QueryOptions{Top: 2, Count: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Value) != 2 {
		t.Errorf("expected 2 records, got %d", len(result.Value))
	}
	if result.NextLink == "" {
		t.Error("expected NextLink to be set")
	}
	if result.Value[0]["CustomerAccount"] != "US-001" {
		t.Errorf("expected US-001, got %v", result.Value[0]["CustomerAccount"])
	}
}

func TestODataResponseParsing_EmptyValue(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"@odata.context": "https://env.operations.dynamics.com/data/$metadata#Customers",
			"value":          []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	result, err := oc.QueryEntities(context.Background(), "Customers", QueryOptions{
		Filter: "Name eq 'NonExistent'",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Value) != 0 {
		t.Errorf("expected 0 records, got %d", len(result.Value))
	}
}

func TestODataErrorParsing(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errResp := ODataErrorResponse{
			Error: ODataErrorDetail{
				Code:    "BadRequest",
				Message: "Invalid filter expression",
				Details: []ODataErrorInner{
					{Message: "Syntax error near 'eq'"},
				},
				Inner: &ODataErrorInnerMsg{
					Message: "Could not parse filter",
				},
			},
		}
		json.NewEncoder(w).Encode(errResp)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	_, err := oc.QueryEntities(context.Background(), "Customers", QueryOptions{
		Filter: "bad filter",
	})
	if err == nil {
		t.Fatal("expected OData error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "ODATA_ERROR") {
		t.Errorf("expected ODATA_ERROR code, got: %s", errStr)
	}
	if !strings.Contains(errStr, "Invalid filter expression") {
		t.Errorf("expected error message to contain 'Invalid filter expression', got: %s", errStr)
	}
}

func TestODataClient_CreateEntity(t *testing.T) {

	var capturedMethod string
	var capturedBody map[string]interface{}
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"CustomerAccount":"US-099","Name":"New Customer"}`)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	result, err := oc.CreateEntity(context.Background(), "Customers", map[string]interface{}{
		"CustomerAccount": "US-099",
		"Name":            "New Customer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if capturedPath != "/data/Customers" {
		t.Errorf("expected path /data/Customers, got %s", capturedPath)
	}
	if result["CustomerAccount"] != "US-099" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestODataClient_UpdateEntity(t *testing.T) {

	var capturedMethod string
	var capturedPath string
	var capturedBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	err := oc.UpdateEntity(context.Background(),
		"Customers(dataAreaId='USMF',CustomerAccount='US-001')",
		map[string]interface{}{"Name": "Updated Name"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "Customers") {
		t.Errorf("expected Customers in path, got %s", capturedPath)
	}
	if capturedBody["Name"] != "Updated Name" {
		t.Errorf("expected 'Updated Name', got %v", capturedBody["Name"])
	}
}

func TestODataClient_DeleteEntity(t *testing.T) {

	var capturedMethod string
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	err := oc.DeleteEntity(context.Background(),
		"Customers(dataAreaId='USMF',CustomerAccount='US-001')",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "Customers") {
		t.Errorf("expected Customers in path, got %s", capturedPath)
	}
}

func TestODataClient_GetMetadata(t *testing.T) {

	metadataXML := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0">
  <edmx:DataServices>
    <Schema Namespace="Microsoft.Dynamics.DataEntities">
      <EntityType Name="Customer">
        <Property Name="CustomerAccount" Type="Edm.String"/>
      </EntityType>
      <EntityContainer Name="Resources">
        <EntitySet Name="Customers" EntityType="Microsoft.Dynamics.DataEntities.Customer"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "$metadata") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, metadataXML)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	result, err := oc.GetMetadata(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "EntityType") {
		t.Errorf("expected $metadata XML, got: %s", result[:100])
	}
}

func TestODataClient_FindEntityTypes(t *testing.T) {

	metadataXML := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0">
  <edmx:DataServices>
    <Schema Namespace="Microsoft.Dynamics.DataEntities">
      <EntityType Name="Customer">
        <Property Name="CustomerAccount" Type="Edm.String"/>
      </EntityType>
      <EntityType Name="CustomerGroup">
        <Property Name="CustomerGroupId" Type="Edm.String"/>
      </EntityType>
      <EntityType Name="Vendor">
        <Property Name="VendorAccount" Type="Edm.String"/>
      </EntityType>
      <EntityContainer Name="Resources">
        <EntitySet Name="Customers" EntityType="Microsoft.Dynamics.DataEntities.Customer"/>
        <EntitySet Name="CustomerGroups" EntityType="Microsoft.Dynamics.DataEntities.CustomerGroup"/>
        <EntitySet Name="Vendors" EntityType="Microsoft.Dynamics.DataEntities.Vendor"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, metadataXML)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	matches, err := oc.FindEntityTypes(context.Background(), "customer", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches for 'customer', got %d", len(matches))
	}

	// Verify names.
	names := make(map[string]bool)
	for _, m := range matches {
		names[m.Name] = true
	}
	if !names["Customer"] {
		t.Error("expected 'Customer' in matches")
	}
	if !names["CustomerGroup"] {
		t.Error("expected 'CustomerGroup' in matches")
	}

	// Verify entity set name resolution.
	for _, m := range matches {
		if m.Name == "Customer" && m.EntitySetName != "Customers" {
			t.Errorf("expected EntitySetName 'Customers', got %q", m.EntitySetName)
		}
	}
}

func TestODataClient_FindEntityTypes_TopLimit(t *testing.T) {

	metadataXML := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0">
  <edmx:DataServices>
    <Schema Namespace="Microsoft.Dynamics.DataEntities">
      <EntityType Name="SalesOrderHeader">
        <Property Name="SalesOrderNumber" Type="Edm.String"/>
      </EntityType>
      <EntityType Name="SalesOrderLine">
        <Property Name="ItemNumber" Type="Edm.String"/>
      </EntityType>
      <EntityType Name="SalesTable">
        <Property Name="SalesId" Type="Edm.String"/>
      </EntityType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, metadataXML)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	matches, err := oc.FindEntityTypes(context.Background(), "sales", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 match with top=1, got %d", len(matches))
	}
}

func TestSplitPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"LedgerChartOfAccounts", []string{"ledger", "chart", "of", "accounts"}},
		{"HRWorker", []string{"hr", "worker"}},
		{"Customer", []string{"customer"}},
		{"SalesOrderHeader", []string{"sales", "order", "header"}},
		{"ABCDef", []string{"abc", "def"}},
		{"", nil},
		{"lowercase", []string{"lowercase"}},
		{"MainAccounts", []string{"main", "accounts"}},
	}
	for _, tt := range tests {
		got := splitPascalCase(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitPascalCase(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitPascalCase(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMatchesSearch(t *testing.T) {
	tests := []struct {
		name       string
		searchTerm string
		want       bool
	}{
		// Single-word (backward compatible)
		{"Customer", "customer", true},
		{"CustomerGroup", "customer", true},
		{"Vendor", "customer", false},
		// Multi-word
		{"LedgerChartOfAccounts", "chart accounts", true},
		{"LedgerChartOfAccounts", "chart of accounts", true},
		{"SalesOrderHeader", "sales order", true},
		{"SalesOrderHeader", "purchase order", false},
		{"MainAccounts", "main account", true},
		{"ReleasedDistinctProducts", "released products", true},
		// Edge cases
		{"Customer", "", false},
		{"Customer", "   ", false},
	}
	for _, tt := range tests {
		got := matchesSearch(tt.name, tt.searchTerm)
		if got != tt.want {
			t.Errorf("matchesSearch(%q, %q) = %v, want %v", tt.name, tt.searchTerm, got, tt.want)
		}
	}
}

func TestSearchMetadataForTypes_MultiWord(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0">
  <edmx:DataServices>
    <Schema Namespace="Microsoft.Dynamics.DataEntities">
      <EntityType Name="Customer">
      </EntityType>
      <EntityType Name="LedgerChartOfAccounts">
      </EntityType>
      <EntityType Name="SalesOrderHeader">
      </EntityType>
      <EntityType Name="MainAccount">
      </EntityType>
      <EntityContainer Name="Resources">
        <EntitySet Name="Customers" EntityType="Microsoft.Dynamics.DataEntities.Customer"/>
        <EntitySet Name="LedgerChartOfAccountsEntities" EntityType="Microsoft.Dynamics.DataEntities.LedgerChartOfAccounts"/>
        <EntitySet Name="SalesOrderHeaders" EntityType="Microsoft.Dynamics.DataEntities.SalesOrderHeader"/>
        <EntitySet Name="MainAccounts" EntityType="Microsoft.Dynamics.DataEntities.MainAccount"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	matches := searchMetadataForTypes(xml, "chart accounts", 10)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'chart accounts', got %d", len(matches))
	}
	if matches[0].Name != "LedgerChartOfAccounts" {
		t.Errorf("expected LedgerChartOfAccounts, got %s", matches[0].Name)
	}

	matches = searchMetadataForTypes(xml, "sales order", 10)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'sales order', got %d", len(matches))
	}

	// Single word still works
	matches = searchMetadataForTypes(xml, "customer", 10)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'customer', got %d", len(matches))
	}
}

func TestODataClient_QueryEntities_PathForwarding(t *testing.T) {

	var capturedPath string
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"value":[]}`)
	}))
	defer srv.Close()

	crossTrue := true
	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	_, err := oc.QueryEntities(context.Background(), "Customers", QueryOptions{
		Select:       "CustomerAccount",
		Top:          5,
		CrossCompany: &crossTrue,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedPath != "/data/Customers" {
		t.Errorf("expected /data/Customers, got %s", capturedPath)
	}
	if !strings.Contains(capturedQuery, "cross-company=true") {
		t.Errorf("expected cross-company=true in query, got %s", capturedQuery)
	}
}

func TestODataErrorResponse_StructuredParsing(t *testing.T) {

	body := `{
		"error": {
			"code": "ValidationError",
			"message": "Field 'Name' is required",
			"details": [
				{"code": "MissingField", "message": "Name cannot be empty"}
			],
			"innererror": {
				"message": "Validation failed in CustTable",
				"type": "System.ArgumentException"
			}
		}
	}`

	var parsed ODataErrorResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if parsed.Error.Code != "ValidationError" {
		t.Errorf("expected code 'ValidationError', got %q", parsed.Error.Code)
	}
	if parsed.Error.Message != "Field 'Name' is required" {
		t.Errorf("unexpected message: %s", parsed.Error.Message)
	}
	if len(parsed.Error.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(parsed.Error.Details))
	}
	if parsed.Error.Details[0].Message != "Name cannot be empty" {
		t.Errorf("unexpected detail message: %s", parsed.Error.Details[0].Message)
	}
	if parsed.Error.Inner == nil {
		t.Fatal("expected inner error")
	}
	if parsed.Error.Inner.Message != "Validation failed in CustTable" {
		t.Errorf("unexpected inner message: %s", parsed.Error.Inner.Message)
	}
}

func TestParseODataError_FromResponse(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{
			"error": {
				"code": "0x80040216",
				"message": "An error occurred while validating input parameters",
				"innererror": {
					"message": "dataAreaId is required"
				}
			}
		}`)
	}))
	defer srv.Close()

	oc := NewODataClient(srv.URL, stubTokenProvider, 5*time.Second)
	_, err := oc.CreateEntity(context.Background(), "Customers", map[string]interface{}{
		"Name": "Missing company",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "ODATA_ERROR") {
		t.Errorf("expected ODATA_ERROR, got: %s", errStr)
	}
	if !strings.Contains(errStr, "validating input parameters") {
		t.Errorf("expected OData message in error, got: %s", errStr)
	}
	if !strings.Contains(errStr, "dataAreaId is required") {
		t.Errorf("expected inner error detail, got: %s", errStr)
	}
}

func TestExtractXMLAttr(t *testing.T) {

	tests := []struct {
		name     string
		input    string
		attr     string
		expected string
	}{
		{
			name:     "entity type name",
			input:    `<EntityType Name="Customer">`,
			attr:     "Name",
			expected: "Customer",
		},
		{
			name:     "entity set with type",
			input:    `<EntitySet Name="Customers" EntityType="Microsoft.Dynamics.DataEntities.Customer"/>`,
			attr:     "EntityType",
			expected: "Microsoft.Dynamics.DataEntities.Customer",
		},
		{
			name:     "missing attr",
			input:    `<EntityType Name="Customer">`,
			attr:     "Label",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractXMLAttr(tt.input, tt.attr)
			if got != tt.expected {
				t.Errorf("extractXMLAttr(%q, %q) = %q, want %q", tt.input, tt.attr, got, tt.expected)
			}
		})
	}
}
