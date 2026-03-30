package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	clierrors "github.com/seangalliher/d365-erp-cli/internal/errors"
)

// ODataClient wraps Client with OData-specific functionality for D365 F&O.
type ODataClient struct {
	client *Client
}

// NewODataClient creates an ODataClient backed by a new HTTP Client.
func NewODataClient(baseURL string, tokenProvider TokenProvider, timeout time.Duration) *ODataClient {
	return &ODataClient{
		client: NewClient(baseURL, tokenProvider, timeout),
	}
}

// SetVerbose enables or disables verbose logging on the underlying HTTP client.
func (o *ODataClient) SetVerbose(v bool) {
	o.client.SetVerbose(v)
}

// QueryOptions defines OData query parameters.
type QueryOptions struct {
	Filter       string // $filter value
	Select       string // $select value (comma-separated fields)
	Expand       string // $expand value (comma-separated navigation properties)
	Top          int    // $top value
	Skip         int    // $skip value
	OrderBy      string // $orderby value
	Count        bool   // $count=true
	CrossCompany *bool  // cross-company flag; nil = auto-inject true (guardrail)
}

// ODataResponse represents the parsed OData JSON response envelope.
type ODataResponse struct {
	Context  string                   `json:"@odata.context,omitempty"`
	Count    *int                     `json:"@odata.count,omitempty"`
	Value    []map[string]interface{} `json:"value,omitempty"`
	NextLink string                   `json:"@odata.nextLink,omitempty"`
}

// ODataErrorResponse is the standard OData error body returned by D365.
type ODataErrorResponse struct {
	Error ODataErrorDetail `json:"error"`
}

// ODataErrorDetail holds the structured fields of an OData error.
type ODataErrorDetail struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Details []ODataErrorInner   `json:"details,omitempty"`
	Inner   *ODataErrorInnerMsg `json:"innererror,omitempty"`
}

// ODataErrorInner holds a single detail entry in an OData error.
type ODataErrorInner struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// ODataErrorInnerMsg holds the inner-error message chain from D365.
type ODataErrorInnerMsg struct {
	Message    string              `json:"message,omitempty"`
	Type       string              `json:"type,omitempty"`
	Stacktrace string              `json:"stacktrace,omitempty"`
	Inner      *ODataErrorInnerMsg `json:"innererror,omitempty"`
}

// BuildQueryString converts QueryOptions into a URL query string.
// If CrossCompany is nil (unset), it auto-injects cross-company=true as an
// AI-agent guardrail to prevent accidental single-company filtering.
func BuildQueryString(opts QueryOptions) string {
	params := url.Values{}

	// Guardrail: auto-inject cross-company=true unless explicitly set.
	if opts.CrossCompany == nil {
		params.Set("cross-company", "true")
	} else if *opts.CrossCompany {
		params.Set("cross-company", "true")
	}
	// If CrossCompany is explicitly false, omit it.

	if opts.Filter != "" {
		params.Set("$filter", opts.Filter)
	}
	if opts.Select != "" {
		params.Set("$select", opts.Select)
	}
	if opts.Expand != "" {
		params.Set("$expand", opts.Expand)
	}
	if opts.Top > 0 {
		params.Set("$top", strconv.Itoa(opts.Top))
	}
	if opts.Skip > 0 {
		params.Set("$skip", strconv.Itoa(opts.Skip))
	}
	if opts.OrderBy != "" {
		params.Set("$orderby", opts.OrderBy)
	}
	if opts.Count {
		params.Set("$count", "true")
	}

	return params.Encode()
}

// QueryEntities performs a GET on the given entity path with OData query options.
// It returns the parsed ODataResponse containing the value array and pagination link.
func (o *ODataClient) QueryEntities(ctx context.Context, entityPath string, opts QueryOptions) (*ODataResponse, error) {
	qs := BuildQueryString(opts)
	path := "data/" + strings.TrimLeft(entityPath, "/")
	if qs != "" {
		path += "?" + qs
	}

	resp, err := o.client.Get(ctx, path, nil)
	if err != nil {
		// If we got a response body with an OData error, try to parse it.
		if resp != nil {
			defer resp.Body.Close()
			if odataErr := parseODataError(resp); odataErr != nil {
				return nil, odataErr
			}
		}
		return nil, err
	}
	defer resp.Body.Close()

	var odataResp ODataResponse
	if err := json.NewDecoder(resp.Body).Decode(&odataResp); err != nil {
		return nil, clierrors.ODataError("failed to parse OData response", err)
	}
	return &odataResp, nil
}

// CreateEntity performs a POST to create a new entity instance.
// The data map is serialized to JSON and sent as the request body.
func (o *ODataClient) CreateEntity(ctx context.Context, entityPath string, data map[string]interface{}) (map[string]interface{}, error) {
	path := "data/" + strings.TrimLeft(entityPath, "/")

	body, err := json.Marshal(data)
	if err != nil {
		return nil, clierrors.ODataError("failed to marshal entity data", err)
	}

	resp, err := o.client.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			if odataErr := parseODataError(resp); odataErr != nil {
				return nil, odataErr
			}
		}
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, clierrors.ODataError("failed to parse create response", err)
	}
	return result, nil
}

// UpdateEntity performs a PATCH to update an existing entity.
// The odataPath should include the entity key, e.g. "Customers(dataAreaId='USMF',CustomerAccount='US-001')".
// The fields map contains only the fields to update.
func (o *ODataClient) UpdateEntity(ctx context.Context, odataPath string, fields map[string]interface{}) error {
	path := "data/" + strings.TrimLeft(odataPath, "/")

	body, err := json.Marshal(fields)
	if err != nil {
		return clierrors.ODataError("failed to marshal update fields", err)
	}

	resp, err := o.client.Patch(ctx, path, bytes.NewReader(body))
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			if odataErr := parseODataError(resp); odataErr != nil {
				return odataErr
			}
		}
		return err
	}
	defer resp.Body.Close()
	return nil
}

// DeleteEntity performs a DELETE on the given OData path.
// The odataPath should include the entity key, e.g. "Customers(dataAreaId='USMF',CustomerAccount='US-001')".
func (o *ODataClient) DeleteEntity(ctx context.Context, odataPath string) error {
	path := "data/" + strings.TrimLeft(odataPath, "/")

	resp, err := o.client.Delete(ctx, path)
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			if odataErr := parseODataError(resp); odataErr != nil {
				return odataErr
			}
		}
		return err
	}
	defer resp.Body.Close()
	return nil
}

// GetMetadata fetches the $metadata document and returns the raw XML string.
func (o *ODataClient) GetMetadata(ctx context.Context) (string, error) {
	headers := map[string]string{"Accept": "application/xml"}
	resp, err := o.client.Get(ctx, "data/$metadata", nil, headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", clierrors.ODataError("failed to read $metadata response", err)
	}
	return string(raw), nil
}

// FindEntityTypes searches the $metadata XML for EntityType definitions whose
// name contains the searchTerm (case-insensitive). Returns up to top results.
func (o *ODataClient) FindEntityTypes(ctx context.Context, searchTerm string, top int) ([]EntityTypeMatch, error) {
	metadata, err := o.GetMetadata(ctx)
	if err != nil {
		return nil, err
	}
	return searchMetadataForTypes(metadata, searchTerm, top), nil
}

// EntityTypeMatch is a search result from FindEntityTypes.
type EntityTypeMatch struct {
	Name          string `json:"name"`
	EntitySetName string `json:"entity_set_name,omitempty"`
}

// searchMetadataForTypes is a simple string-based search through $metadata XML.
// It looks for <EntityType Name="..."> elements matching the search term.
// A full XML parser would be more robust, but this avoids pulling in encoding/xml
// for a simple search-and-match use case on potentially large metadata documents.
func searchMetadataForTypes(metadata string, searchTerm string, top int) []EntityTypeMatch {
	var matches []EntityTypeMatch

	// Search for EntityType definitions (not EntitySet which also has EntityType attr).
	lines := strings.Split(metadata, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "<EntityType ") || !strings.Contains(trimmed, "Name=") {
			continue
		}

		name := extractXMLAttr(trimmed, "Name")
		if name == "" {
			continue
		}

		if matchesSearch(name, searchTerm) {
			matches = append(matches, EntityTypeMatch{Name: name})
			if top > 0 && len(matches) >= top {
				break
			}
		}
	}

	// Attempt to find corresponding EntitySet names.
	for i, m := range matches {
		setName := findEntitySetName(metadata, m.Name)
		if setName != "" {
			matches[i].EntitySetName = setName
		}
	}

	return matches
}

// extractXMLAttr extracts an attribute value from a string like Name="Foo".
func extractXMLAttr(s string, attr string) string {
	prefix := attr + `="`
	idx := strings.Index(s, prefix)
	if idx < 0 {
		return ""
	}
	start := idx + len(prefix)
	end := strings.Index(s[start:], `"`)
	if end < 0 {
		return ""
	}
	return s[start : start+end]
}

// findEntitySetName looks for an EntitySet element whose EntityType matches the given type name.
func findEntitySetName(metadata string, typeName string) string {
	// Look for <EntitySet Name="..." EntityType="...typeName"...>
	lines := strings.Split(metadata, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "EntitySet") || !strings.Contains(trimmed, "Name=") {
			continue
		}
		entityType := extractXMLAttr(trimmed, "EntityType")
		// EntityType attribute often has a namespace prefix, e.g. "Microsoft.Dynamics.DataEntities.Customer"
		if strings.HasSuffix(entityType, "."+typeName) || entityType == typeName {
			return extractXMLAttr(trimmed, "Name")
		}
	}
	return ""
}

// splitPascalCase splits a PascalCase identifier into lowercase words.
// e.g. "LedgerChartOfAccounts" -> ["ledger", "chart", "of", "accounts"]
func splitPascalCase(name string) []string {
	var words []string
	var current []rune
	runes := []rune(name)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			// Start new word if previous char was lowercase,
			// or if next char is lowercase (handles acronyms like "HR" in "HRWorker").
			prevLower := unicode.IsLower(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prevLower || nextLower {
				if len(current) > 0 {
					words = append(words, strings.ToLower(string(current)))
					current = nil
				}
			}
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		words = append(words, strings.ToLower(string(current)))
	}
	return words
}

// matchesSearch checks if an entity name matches a search term.
// Single-word: substring match (backward-compatible).
// Multi-word: ALL words must appear in the PascalCase-split name.
func matchesSearch(name, searchTerm string) bool {
	searchWords := strings.Fields(strings.ToLower(searchTerm))
	if len(searchWords) == 0 {
		return false
	}
	if len(searchWords) == 1 {
		return strings.Contains(strings.ToLower(name), searchWords[0])
	}
	// Multi-word: check each search word against the split name
	nameWords := splitPascalCase(name)
	nameLower := strings.ToLower(name)
	for _, sw := range searchWords {
		// Check if the word appears as substring in the full name OR matches a split word
		found := strings.Contains(nameLower, sw)
		if !found {
			for _, nw := range nameWords {
				if strings.Contains(nw, sw) {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// parseODataError tries to parse an OData error response body.
// Returns a structured CLIError if the body is a valid OData error, nil otherwise.
func parseODataError(resp *http.Response) *clierrors.CLIError {
	if resp.Body == nil {
		return nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(bodyBytes) == 0 {
		return nil
	}
	// Replace the body so callers can still read it.
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var odataErr ODataErrorResponse
	if err := json.Unmarshal(bodyBytes, &odataErr); err != nil {
		return nil
	}

	if odataErr.Error.Message == "" {
		return nil
	}

	msg := fmt.Sprintf("OData error [%s]: %s", odataErr.Error.Code, odataErr.Error.Message)

	// Collect additional details from the inner error chain.
	var details []string
	for _, d := range odataErr.Error.Details {
		if d.Message != "" {
			details = append(details, d.Message)
		}
	}
	if inner := odataErr.Error.Inner; inner != nil && inner.Message != "" {
		details = append(details, inner.Message)
	}
	if len(details) > 0 {
		msg += " | details: " + strings.Join(details, "; ")
	}

	return clierrors.ODataError(msg, nil)
}
