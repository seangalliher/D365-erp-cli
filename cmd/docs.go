package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDocsCmd())
}

// docTopic holds the title and markdown content for a documentation topic.
type docTopic struct {
	Title   string
	Content string
}

// docTopics contains all available inline documentation topics.
var docTopics = map[string]docTopic{
	"odata-filters": {
		Title: "OData Filter Syntax for D365 F&O",
		Content: `# OData Filter Syntax for D365 F&O

## Basic Comparison Operators
- ` + "`eq`" + ` - Equals: ` + "`$filter=CustomerAccount eq 'US-001'`" + `
- ` + "`ne`" + ` - Not equals: ` + "`$filter=Status ne 'Closed'`" + `
- ` + "`gt`" + `, ` + "`ge`" + `, ` + "`lt`" + `, ` + "`le`" + ` - Comparison: ` + "`$filter=Amount gt 1000`" + `

## Logical Operators
- ` + "`and`" + `: ` + "`$filter=City eq 'Seattle' and State eq 'WA'`" + `
- ` + "`or`" + `: ` + "`$filter=Status eq 'Open' or Status eq 'InProgress'`" + `
- ` + "`not`" + `: ` + "`$filter=not endswith(Name, 'Test')`" + `

## String Functions
- ` + "`contains(Field, 'value')`" + ` - Substring match
- ` + "`startswith(Field, 'value')`" + ` - Prefix match
- ` + "`endswith(Field, 'value')`" + ` - Suffix match

## D365-Specific Notes
- Use single quotes for string values: ` + "`'value'`" + `
- For wildcards in D365, use: ` + "`$filter=Field eq '*pattern*'`" + `
- Enum filters require namespace: ` + "`$filter=Status eq Microsoft.Dynamics.DataEntities.Status'Active'`" + `
- Date format: ` + "`$filter=CreatedDate gt 2024-01-01T00:00:00Z`" + `
- Always include ` + "`cross-company=true`" + ` for cross-company queries (auto-injected by this CLI).
`,
	},

	"form-workflow": {
		Title: "Form Automation Workflow",
		Content: `# Form Automation Workflow

## Typical Form Interaction Sequence
1. Find the menu item: ` + "`.\\/d365 form find-menu <keyword>`" + `
2. Open the form: ` + "`.\\/d365 form open <menuItemName> --type Display`" + `
3. Read current state: ` + "`.\\/d365 form state`" + `
4. Set field values: ` + "`.\\/d365 form set --values '{\"FieldName\": \"Value\"}'`" + `
5. Save: ` + "`.\\/d365 form save`" + `
6. Close: ` + "`.\\/d365 form close`" + `

## Working with Grids
- Filter a grid column: ` + "`.\\/d365 form filter-grid --grid <name> --column <name> --value <value>`" + `
- Select a row: ` + "`.\\/d365 form select-row --grid <name> --row <number>`" + `
- Sort a column: ` + "`.\\/d365 form sort-grid --grid <name> --column <name> --direction Ascending`" + `

## Working with Lookups
- Open a lookup: ` + "`.\\/d365 form open-lookup <controlName>`" + `
- Lookups require selecting a value from the lookup results.

## Best Practices
- Always read form state before making changes to understand available controls.
- Use control names (not labels) for all operations.
- Save the form before closing to persist changes.
- Check is_dirty in form state to know if unsaved changes exist.
`,
	},

	"authentication": {
		Title: "Authentication Methods",
		Content: `# Authentication Methods

## Supported Auth Methods

### Azure CLI (default)
Uses your existing Azure CLI login. No additional configuration needed.
` + "```" + `
az login
.\d365 connect https://your-env.operations.dynamics.com
` + "```" + `

### Client Credentials (Service Principal)
For CI/CD pipelines and automated scenarios.
` + "```" + `
export D365_CLIENT_ID=your-app-id
export D365_CLIENT_SECRET=your-secret
export D365_TENANT_ID=your-tenant-id
.\d365 connect https://your-env.operations.dynamics.com --auth client_credentials
` + "```" + `

## Environment Variables
- ` + "`D365_URL`" + ` - Default environment URL
- ` + "`D365_AUTH_METHOD`" + ` - Auth method (az_cli, client_credentials)
- ` + "`D365_TENANT_ID`" + ` - Azure AD tenant ID (find in Azure Portal > Microsoft Entra ID, not the D365 environment ID)
- ` + "`D365_CLIENT_ID`" + ` - Application (client) ID
- ` + "`D365_CLIENT_SECRET`" + ` - Client secret
- ` + "`D365_COMPANY`" + ` - Default company/legal entity

## Token Management
- Tokens are cached in the session file and refreshed automatically.
- Use ` + "`.\\/d365 status`" + ` to check token expiry.
- Run ` + "`.\\/d365 connect`" + ` again to refresh credentials.
`,
	},

	"batch-mode": {
		Title: "Batch and Pipeline Mode",
		Content: `# Batch and Pipeline Mode

## CI Mode
Use ` + "`--ci`" + ` for pipeline-friendly output:
` + "```" + `
.\d365 --ci data find Customers --query '$top=5&$select=CustomerAccount,Name'
` + "```" + `
This implies ` + "`--output json --quiet --no-color`" + `.

## Output Formats
- ` + "`--output json`" + ` (default for non-TTY) - Structured JSON with metadata
- ` + "`--output table`" + ` (default for TTY) - Human-readable table
- ` + "`--output csv`" + ` - CSV for spreadsheet import
- ` + "`--output raw`" + ` - Raw data without envelope

## Exit Codes
- 0: Success
- 1: Command error
- 2: Connection/auth error
- 3: Validation error

## Scripting Patterns
Parse JSON output with jq:
` + "```powershell" + `
.\d365 --ci data find Customers --query '$top=5' | ConvertFrom-Json | Select-Object -ExpandProperty data
` + "```" + `

Check command success:
` + "```powershell" + `
$result = .\d365 --ci data find Customers --query '$top=1' | ConvertFrom-Json
if ($result.success) { Write-Host 'Connected and working' }
` + "```" + `
`,
	},

	"enum-syntax": {
		Title: "Enum Syntax in D365 OData",
		Content: `# Enum Syntax in D365 OData

## Filtering by Enum Values
D365 OData enums require the full namespace prefix:
` + "```" + `
$filter=Status eq Microsoft.Dynamics.DataEntities.Status'Active'
` + "```" + `

## Common Pattern
` + "`Namespace.EnumType'SymbolName'`" + `

The namespace is typically ` + "`Microsoft.Dynamics.DataEntities`" + `.

## Finding Enum Values
Use the metadata command with --enums to discover available enum values:
` + "```" + `
.\d365 data metadata <EntitySetName> --enums
` + "```" + `

## Creating/Updating with Enums
When creating or updating entities, pass the enum symbol name as a string:
` + "```json" + `
{"status": "Active", "type": "Customer"}
` + "```" + `

## Important Notes
- Enum values are case-sensitive.
- Use the symbol name (e.g., "Active"), not the integer value (e.g., 0).
- The CLI metadata command shows both symbol names and integer values.
- When filtering, you must use the full namespace prefix.
- When creating/updating, use just the symbol name string.
`,
	},
	"entities": {
		Title: "Common D365 F&O Data Entities Reference",
		Content: `# Common D365 F&O Data Entities Reference

A quick-reference for the most frequently used OData entity sets in Dynamics 365 Finance & Operations. Use these names with ` + "`d365 data find`" + `, ` + "`d365 data create`" + `, etc.

Tip: Use ` + "`d365 data find-type <keyword>`" + ` to search for entities not listed here. Multi-word searches work (e.g., "chart accounts").

## Master Data

### Customers
- **Entity set:** ` + "`Customers`" + `
- **Key fields:** dataAreaId, CustomerAccount
- **Common fields:** Name, CustomerGroupId, AddressCity, AddressState, Currency
- **Example:** ` + "`d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'`" + `

### Vendors
- **Entity set:** ` + "`Vendors`" + `
- **Key fields:** dataAreaId, VendorAccountNumber
- **Common fields:** VendorName, VendorGroupId, Currency, AddressCity
- **Example:** ` + "`d365 data find Vendors --query '$top=5&$select=VendorAccountNumber,VendorName'`" + `

### Products
- **Entity set:** ` + "`ReleasedDistinctProducts`" + ` (product masters) or ` + "`ReleasedProducts`" + ` (per-company variants)
- **Key fields:** ProductNumber (masters) or dataAreaId, ItemNumber (released)
- **Common fields:** ProductName, ProductType, SearchName
- **Example:** ` + "`d365 data find ReleasedDistinctProducts --query '$top=5&$select=ProductNumber,ProductName'`" + `

### Warehouses
- **Entity set:** ` + "`InventWarehouses`" + `
- **Key fields:** dataAreaId, WarehouseId
- **Common fields:** WarehouseName, SiteId

### Sites
- **Entity set:** ` + "`InventSites`" + `
- **Key fields:** dataAreaId, SiteId
- **Common fields:** SiteName

## Sales

### Sales Orders
- **Header entity set:** ` + "`SalesOrderHeaders`" + `
- **Key fields:** dataAreaId, SalesOrderNumber
- **Common fields:** OrderingCustomerAccountNumber, InvoiceCustomerAccountNumber, SalesOrderStatus
- **Line entity set:** ` + "`SalesOrderLines`" + `
- **Line key fields:** dataAreaId, SalesOrderNumber, LineNumber
- **Example:** ` + "`d365 data find SalesOrderHeaders --query '$top=5&$select=SalesOrderNumber,OrderingCustomerAccountNumber'`" + `

## Purchasing

### Purchase Orders
- **Header entity set:** ` + "`PurchaseOrderHeaders`" + `
- **Key fields:** dataAreaId, PurchaseOrderNumber
- **Common fields:** OrderVendorAccountNumber, PurchaseOrderStatus
- **Line entity set:** ` + "`PurchaseOrderLines`" + `
- **Line key fields:** dataAreaId, PurchaseOrderNumber, LineNumber
- **Example:** ` + "`d365 data find PurchaseOrderHeaders --query '$top=5&$select=PurchaseOrderNumber,OrderVendorAccountNumber'`" + `

## Financials

### Legal Entities
- **Entity set:** ` + "`LegalEntities`" + `
- **Key fields:** LegalEntityId
- **Common fields:** Name, CompanyType, AddressCountryRegion
- **Example:** ` + "`d365 data find LegalEntities --query '$select=LegalEntityId,Name'`" + `

### Main Accounts
- **Entity set:** ` + "`MainAccounts`" + `
- **Key fields:** dataAreaId, MainAccountId, ChartOfAccounts
- **Common fields:** Name, MainAccountType
- **Example:** ` + "`d365 data find MainAccounts --query '$top=10&$select=MainAccountId,Name,MainAccountType'`" + `

### Chart of Accounts
- **Entity set:** ` + "`ChartOfAccounts`" + ` (for create/update) or ` + "`LedgerChartOfAccounts`" + ` (read-only view)
- **Key fields:** ChartOfAccounts
- **Common fields:** Description

### Ledger
- **Entity set:** ` + "`Ledgers`" + `
- **Key fields:** LegalEntityId
- **Common fields:** ChartOfAccounts, Name, AccountingCurrency, FiscalCalendar

### General Journal
- **Header entity set:** ` + "`GeneralJournalHeaders`" + `
- **Key fields:** dataAreaId, JournalBatchNumber
- **Line entity set:** ` + "`GeneralJournalLines`" + `
- **Line key fields:** dataAreaId, JournalBatchNumber, LineNumber

## Inventory

### On-Hand Inventory
- **Entity set:** ` + "`InventoryOnhandEntities`" + `
- **Key fields:** dataAreaId, ItemNumber
- **Common fields:** AvailableQuantity, OnOrderQuantity

### Item Groups
- **Entity set:** ` + "`ItemGroups`" + `
- **Key fields:** dataAreaId, ItemGroupId
- **Common fields:** ItemGroupName

## Human Resources

### Workers
- **Entity set:** ` + "`Workers`" + ` (all types) or ` + "`Employees`" + ` (employees only)
- **Key fields:** PersonnelNumber
- **Common fields:** FirstName, LastName, EmploymentStartDate

### Positions
- **Entity set:** ` + "`Positions`" + `
- **Key fields:** PositionId
- **Common fields:** Description, Department, Job

## Configuration

### Company Info
- **Entity set:** ` + "`CompanyInfoEntities`" + `
- **Key fields:** dataAreaId

### Number Sequences
- **Entity set:** ` + "`NumberSequences`" + `
- **Key fields:** dataAreaId, NumberSequenceCode
`,
	},
}

// newDocsCmd creates the "docs" command for inline documentation.
func newDocsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs [topic]",
		Short: "Inline documentation for D365 CLI concepts",
		Long: `Access built-in documentation for key D365 CLI concepts and workflows.

Run without arguments to list all available topics.
Provide a topic name to view its documentation.

Available topics: odata-filters, form-workflow, authentication,
batch-mode, enum-syntax, entities`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			// No topic specified: list available topics.
			if len(args) == 0 {
				return listTopics(cmd, start)
			}

			topicName := args[0]

			// Look up the topic.
			topic, ok := docTopics[topicName]
			if !ok {
				return handleUnknownTopic(cmd, topicName, start)
			}

			// Output the documentation content.
			RenderSuccess(cmd, map[string]interface{}{
				"topic":   topicName,
				"title":   topic.Title,
				"content": topic.Content,
			}, start)
			return nil
		},
	}

	return cmd
}

// listTopics outputs the list of available documentation topics.
func listTopics(cmd *cobra.Command, start time.Time) error {
	var topics []map[string]string

	// Sort topic names for consistent output.
	names := make([]string, 0, len(docTopics))
	for name := range docTopics {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		topics = append(topics, map[string]string{
			"topic": name,
			"title": docTopics[name].Title,
		})
	}

	RenderSuccess(cmd, map[string]interface{}{
		"available_topics": topics,
		"usage":            "Run '.\\/d365 docs <topic>' to view documentation for a specific topic.",
	}, start)
	return nil
}

// handleUnknownTopic suggests close matches when a topic is not found.
func handleUnknownTopic(cmd *cobra.Command, topicName string, start time.Time) error {
	suggestions := findClosestTopics(topicName)

	data := map[string]interface{}{
		"error":   fmt.Sprintf("unknown topic: %s", topicName),
		"message": "Topic not found.",
	}
	if len(suggestions) > 0 {
		data["suggestions"] = suggestions
	}

	// List all available topics.
	names := make([]string, 0, len(docTopics))
	for name := range docTopics {
		names = append(names, name)
	}
	sort.Strings(names)
	data["available_topics"] = names

	RenderSuccess(cmd, data, start)
	return nil
}

// findClosestTopics returns topic names that share a substring with the query.
func findClosestTopics(query string) []string {
	queryLower := strings.ToLower(query)
	var matches []string

	// Sort topic names for consistent output.
	names := make([]string, 0, len(docTopics))
	for name := range docTopics {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		nameLower := strings.ToLower(name)
		titleLower := strings.ToLower(docTopics[name].Title)

		// Check if the query is a substring of the topic name or title,
		// or if the topic name is a substring of the query.
		if strings.Contains(nameLower, queryLower) ||
			strings.Contains(queryLower, nameLower) ||
			strings.Contains(titleLower, queryLower) {
			matches = append(matches, name)
		}

		// Also check individual words from the query.
		for _, word := range strings.Split(queryLower, "-") {
			if len(word) >= 3 && strings.Contains(nameLower, word) {
				alreadyAdded := false
				for _, m := range matches {
					if m == name {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					matches = append(matches, name)
				}
			}
		}
	}

	return matches
}
