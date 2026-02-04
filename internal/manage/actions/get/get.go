package get

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/logger"

	"github.com/OpenSlides/openslides-go/datastore"
	"github.com/OpenSlides/openslides-go/datastore/dsfetch"
	"github.com/OpenSlides/openslides-go/environment"
)

const (
	GetHelp      = "Get models from the datastore"
	GetHelpExtra = `Provide a collection to list contained models.
Use options to narrow down output.

Examples:
  # Filter by field
  osmanage get user --filter is_active=true \
    --postgres-host localhost --postgres-port 5432 \
    --postgres-user openslides --postgres-database openslides \
    --postgres-password-file ./secrets/postgres_password

  # Select specific fields
  osmanage get user --fields first_name,last_name,email \
    --postgres-host localhost --postgres-port 5432 \
    --postgres-user openslides --postgres-database openslides \
    --postgres-password-file ./secrets/postgres_password

  # Complex filter with operators
  osmanage get meeting --filter-raw '{"field":"start_time","operator":">=","value":1609459200}' \
    --postgres-host localhost --postgres-port 5432 \
    --postgres-user openslides --postgres-database openslides \
    --postgres-password-file ./secrets/postgres_password

  # Combined AND filter
  osmanage get user --filter-raw '{"and_filter":[{"field":"first_name","operator":"~=","value":"^Ad"},{"field":"is_active","operator":"=","value":true}]}' \
    --postgres-host localhost --postgres-port 5432 \
    --postgres-user openslides --postgres-database openslides \
    --postgres-password-file ./secrets/postgres_password

Supported operators in filter-raw:
  =   : Equal
  !=  : Not equal
  >   : Greater than
  <   : Less than
  >=  : Greater than or equal
  <=  : Less than or equal
  ~=  : Regex match (pattern matching)

Supported collections:
  - user
  - meeting
  - organization

Note: Filtering is done in-memory after fetching. Field selection reduces memory usage by only loading requested fields.`
)

// RawFilter represents the complex filter structure
type RawFilter struct {
	Field     string      `json:"field,omitempty"`
	Operator  string      `json:"operator,omitempty"`
	Value     any         `json:"value,omitempty"`
	AndFilter []RawFilter `json:"and_filter,omitempty"`
	OrFilter  []RawFilter `json:"or_filter,omitempty"`
	NotFilter *RawFilter  `json:"not_filter,omitempty"`
}

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get collection",
		Short: GetHelp,
		Long:  GetHelp + "\n\n" + GetHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	// PostgreSQL connection flags
	postgresHost := cmd.Flags().String("postgres-host", "", "PostgreSQL host (required)")
	postgresPort := cmd.Flags().String("postgres-port", "", "PostgreSQL port (required)")
	postgresUser := cmd.Flags().String("postgres-user", "", "PostgreSQL user (required)")
	postgresDatabase := cmd.Flags().String("postgres-database", "", "PostgreSQL database (required)")
	postgresPasswordFile := cmd.Flags().String("postgres-password-file", "", "PostgreSQL password file (required)")

	// Mark PostgreSQL flags as required
	_ = cmd.MarkFlagRequired("postgres-host")
	_ = cmd.MarkFlagRequired("postgres-port")
	_ = cmd.MarkFlagRequired("postgres-user")
	_ = cmd.MarkFlagRequired("postgres-database")
	_ = cmd.MarkFlagRequired("postgres-password-file")

	// Query flags
	fields := cmd.Flags().StringSlice("fields", nil, "only include the provided fields in output")
	filter := cmd.Flags().StringToString("filter", nil, "simple filter using '=' operator, multiple filters are AND'ed")
	rawFilter := cmd.Flags().String("filter-raw", "", "complex filter in JSON format with operators (=, !=, >, <, >=, <=, ~=)")
	exists := cmd.Flags().Bool("exists", false, "check only for existence (requires --filter or --filter-raw)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== GET COLLECTION ===")

		collection := args[0]
		logger.Debug("Collection: %s", collection)

		// Validate flags
		if *exists && len(*filter) == 0 && *rawFilter == "" {
			return fmt.Errorf("--exists requires --filter or --filter-raw")
		}

		if len(*filter) > 0 && *rawFilter != "" {
			return fmt.Errorf("cannot use both --filter and --filter-raw")
		}

		// Parse raw filter if provided
		var parsedRawFilter *RawFilter
		if *rawFilter != "" {
			parsedRawFilter = &RawFilter{}
			if err := json.Unmarshal([]byte(*rawFilter), parsedRawFilter); err != nil {
				return fmt.Errorf("parsing filter-raw: %w", err)
			}
		}

		// Create environment map for datastore connection
		envMap := map[string]string{
			constants.EnvDatabaseHost:          *postgresHost,
			constants.EnvDatabasePort:          *postgresPort,
			constants.EnvDatabaseUser:          *postgresUser,
			constants.EnvDatabaseName:          *postgresDatabase,
			constants.EnvDatabasePasswordFile:  *postgresPasswordFile,
			constants.EnvOpenSlidesDevelopment: constants.DevelopmentModeDisabled,
		}

		// Initialize datastore flow
		env := environment.ForTests(envMap)
		dsFlow, err := datastore.NewFlowPostgres(env, nil)
		if err != nil {
			return fmt.Errorf("creating datastore flow: %w", err)
		}

		logger.Info("Connected to database successfully")

		// Create fetcher
		fetch := dsfetch.New(dsFlow)
		ctx := context.Background()

		// Execute query
		result, err := executeQuery(ctx, fetch, collection, *filter, parsedRawFilter, *fields, *exists)
		if err != nil {
			return fmt.Errorf("executing query: %w", err)
		}

		// Output as JSON
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling result to JSON: %w", err)
		}

		fmt.Println(string(jsonBytes))
		logger.Info("Query completed successfully")
		return nil
	}

	return cmd
}

func executeQuery(ctx context.Context, fetch *dsfetch.Fetch, collection string, filter map[string]string, rawFilter *RawFilter, fields []string, existsOnly bool) (any, error) {
	logger.Debug("Executing query for collection: %s", collection)

	switch collection {
	case "user":
		return queryUsers(ctx, fetch, filter, rawFilter, fields, existsOnly)
	case "meeting":
		return queryMeetings(ctx, fetch, filter, rawFilter, fields, existsOnly)
	case "organization":
		return queryOrganization(ctx, fetch, fields, existsOnly)
	default:
		return nil, fmt.Errorf("collection '%s' not yet supported", collection)
	}
}

func queryUsers(ctx context.Context, fetch *dsfetch.Fetch, filter map[string]string, rawFilter *RawFilter, fields []string, existsOnly bool) (any, error) {
	logger.Debug("Querying users with fields: %v, filter: %v, rawFilter: %v", fields, filter, rawFilter)

	// Get user IDs from organization
	var userIDs []int
	fetch.Organization_UserIDs(constants.DefaultOrganizationID).Lazy(&userIDs)
	if err := fetch.Execute(ctx); err != nil {
		return nil, fmt.Errorf("fetching user IDs: %w", err)
	}

	logger.Debug("Found %d total users", len(userIDs))

	fieldsToFetch := determineFieldsToFetch(fields, filter, rawFilter)
	logger.Debug("Fields to fetch: %v", fieldsToFetch)

	// Fetch fields for each user
	users := make([]map[string]any, 0, len(userIDs))
	for _, userID := range userIDs {
		user := map[string]any{"id": userID}
		for _, field := range fieldsToFetch {
			if field == "id" {
				continue
			}
			value, err := fetchField(fetch, "user", userID, field)
			if err != nil {
				return nil, fmt.Errorf("fetching user %d field %s: %w", userID, field, err)
			}
			user[field] = value
		}
		users = append(users, user)
	}

	// Execute all lazy fetches
	if err := fetch.Execute(ctx); err != nil {
		return nil, fmt.Errorf("executing batch fetch: %w", err)
	}

	users = applyFilters(users, filter, rawFilter)

	if existsOnly {
		return len(users) > 0, nil
	}

	if len(fields) > 0 {
		users = selectFields(users, fields)
	}

	return convertToMapFormat(users), nil
}

func queryMeetings(ctx context.Context, fetch *dsfetch.Fetch, filter map[string]string, rawFilter *RawFilter, fields []string, existsOnly bool) (any, error) {
	logger.Debug("Querying meetings with fields: %v, filter: %v, rawFilter: %v", fields, filter, rawFilter)

	// Get active and archived meeting IDs
	var activeMeetingIDs, archivedMeetingIDs []int
	fetch.Organization_ActiveMeetingIDs(constants.DefaultOrganizationID).Lazy(&activeMeetingIDs)
	fetch.Organization_ArchivedMeetingIDs(constants.DefaultOrganizationID).Lazy(&archivedMeetingIDs)

	if err := fetch.Execute(ctx); err != nil {
		return nil, fmt.Errorf("fetching meeting IDs: %w", err)
	}

	meetingIDs := append(activeMeetingIDs, archivedMeetingIDs...)
	logger.Debug("Found %d total meetings", len(meetingIDs))

	fieldsToFetch := determineFieldsToFetch(fields, filter, rawFilter)
	logger.Debug("Fields to fetch: %v", fieldsToFetch)

	// Fetch fields for each meeting
	meetings := make([]map[string]any, 0, len(meetingIDs))
	for _, meetingID := range meetingIDs {
		meeting := map[string]any{"id": meetingID}
		for _, field := range fieldsToFetch {
			if field == "id" {
				continue
			}
			value, err := fetchField(fetch, "meeting", meetingID, field)
			if err != nil {
				return nil, fmt.Errorf("fetching meeting %d field %s: %w", meetingID, field, err)
			}
			meeting[field] = value
		}
		meetings = append(meetings, meeting)
	}

	// Execute all lazy fetches
	if err := fetch.Execute(ctx); err != nil {
		return nil, fmt.Errorf("executing batch fetch: %w", err)
	}

	meetings = applyFilters(meetings, filter, rawFilter)

	if existsOnly {
		return len(meetings) > 0, nil
	}

	if len(fields) > 0 {
		meetings = selectFields(meetings, fields)
	}

	return convertToMapFormat(meetings), nil
}

func queryOrganization(ctx context.Context, fetch *dsfetch.Fetch, fields []string, existsOnly bool) (any, error) {
	if existsOnly {
		var orgID int
		fetch.Organization_ID(constants.DefaultOrganizationID).Lazy(&orgID)
		if err := fetch.Execute(ctx); err != nil {
			return false, nil
		}
		return orgID == constants.DefaultOrganizationID, nil
	}

	fieldsToFetch := fields
	if len(fieldsToFetch) == 0 {
		// Use default organization fields
		fieldsToFetch = strings.Split(constants.DefaultOrganizationFields, ",")
	}

	org := make(map[string]any)
	for _, field := range fieldsToFetch {
		value, err := fetchField(fetch, "organization", constants.DefaultOrganizationID, field)
		if err != nil {
			return nil, fmt.Errorf("fetching organization field %s: %w", field, err)
		}
		org[field] = value
	}

	if err := fetch.Execute(ctx); err != nil {
		return nil, fmt.Errorf("executing fetch: %w", err)
	}

	return org, nil
}

// determineFieldsToFetch calculates which fields need to be loaded
func determineFieldsToFetch(requestedFields []string, filter map[string]string, rawFilter *RawFilter) []string {
	fieldsSet := map[string]bool{"id": true}

	for _, field := range requestedFields {
		fieldsSet[field] = true
	}

	for field := range filter {
		fieldsSet[field] = true
	}

	if rawFilter != nil {
		extractFieldsFromRawFilter(rawFilter, fieldsSet)
	}

	fields := make([]string, 0, len(fieldsSet))
	for field := range fieldsSet {
		fields = append(fields, field)
	}

	return fields
}

// extractFieldsFromRawFilter recursively extracts all fields used in a raw filter
func extractFieldsFromRawFilter(rf *RawFilter, fieldsSet map[string]bool) {
	if rf.Field != "" {
		fieldsSet[rf.Field] = true
	}

	for i := range rf.AndFilter {
		extractFieldsFromRawFilter(&rf.AndFilter[i], fieldsSet)
	}

	for i := range rf.OrFilter {
		extractFieldsFromRawFilter(&rf.OrFilter[i], fieldsSet)
	}

	if rf.NotFilter != nil {
		extractFieldsFromRawFilter(rf.NotFilter, fieldsSet)
	}
}

// fetchField dynamically fetches a single field using reflection
func fetchField(fetch *dsfetch.Fetch, collection string, id int, field string) (any, error) {
	methodName := snakeToPascal(collection) + "_" + snakeToPascal(field)

	fetchValue := reflect.ValueOf(fetch)
	method := fetchValue.MethodByName(methodName)

	if !method.IsValid() {
		return nil, fmt.Errorf("unsupported %s field: %s (method %s not found)", collection, field, methodName)
	}

	results := method.Call([]reflect.Value{reflect.ValueOf(id)})
	if len(results) == 0 {
		return nil, fmt.Errorf("method %s returned no results", methodName)
	}

	valueObj := results[0]
	valueType := valueObj.Type().String()

	logger.Debug("Field %s has type: %s", field, valueType)

	switch {
	case strings.Contains(valueType, "ValueIntSlice"):
		var v []int
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	case strings.Contains(valueType, "ValueStringSlice"):
		var v []string
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	case strings.Contains(valueType, "ValueString"):
		var v string
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	case strings.Contains(valueType, "ValueMaybeString"):
		var v dsfetch.Maybe[string]
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	case strings.Contains(valueType, "ValueInt"):
		var v int
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	case strings.Contains(valueType, "ValueMaybeInt"):
		var v dsfetch.Maybe[int]
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	case strings.Contains(valueType, "ValueBool"):
		var v bool
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	case strings.Contains(valueType, "ValueFloat"):
		var v float64
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	case strings.Contains(valueType, "ValueJSON"):
		var v json.RawMessage
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	case strings.Contains(valueType, "ValueDecimal"):
		var v decimal.Decimal
		valueObj.MethodByName("Lazy").Call([]reflect.Value{reflect.ValueOf(&v)})
		return &v, nil
	default:
		return nil, fmt.Errorf("unsupported value type: %s for field %s", valueType, field)
	}
}

// applyFilters applies simple or raw filters to records
func applyFilters(records []map[string]any, filter map[string]string, rawFilter *RawFilter) []map[string]any {
	if len(filter) > 0 {
		return filterSimple(records, filter)
	}
	if rawFilter != nil {
		return filterRaw(records, rawFilter)
	}
	return records
}

// filterSimple applies simple equality filters
func filterSimple(records []map[string]any, filter map[string]string) []map[string]any {
	filtered := make([]map[string]any, 0)
	for _, record := range records {
		if matchesSimpleFilter(record, filter) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

// filterRaw applies complex raw filters
func filterRaw(records []map[string]any, rawFilter *RawFilter) []map[string]any {
	filtered := make([]map[string]any, 0)
	for _, record := range records {
		if matchesRawFilter(record, rawFilter) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

// matchesSimpleFilter checks if a record matches all simple filter conditions
func matchesSimpleFilter(record map[string]any, filter map[string]string) bool {
	for key, value := range filter {
		recordValue, ok := record[key]
		if !ok {
			return false
		}

		// Dereference and compare as strings for simple filters
		recordStr := fmt.Sprintf("%v", dereferenceValue(recordValue))
		if recordStr != value {
			return false
		}
	}
	return true
}

// matchesRawFilter checks if a record matches a raw filter condition
func matchesRawFilter(record map[string]any, rf *RawFilter) bool {
	if len(rf.AndFilter) > 0 {
		for i := range rf.AndFilter {
			if !matchesRawFilter(record, &rf.AndFilter[i]) {
				return false
			}
		}
		return true
	}

	if len(rf.OrFilter) > 0 {
		for i := range rf.OrFilter {
			if matchesRawFilter(record, &rf.OrFilter[i]) {
				return true
			}
		}
		return false
	}

	if rf.NotFilter != nil {
		return !matchesRawFilter(record, rf.NotFilter)
	}

	// Handle single condition
	if rf.Field != "" {
		return matchesCondition(record, rf.Field, rf.Operator, rf.Value)
	}

	return true
}

// matchesCondition checks if a record field matches a condition with the given operator
func matchesCondition(record map[string]any, field, operator string, value any) bool {
	recordValue, ok := record[field]
	if !ok {
		return false
	}

	recordValue = dereferenceValue(recordValue)

	// Special handling for JSON fields
	if _, isJSON := recordValue.(json.RawMessage); isJSON {
		return matchesJSONCondition(recordValue, operator, value)
	}

	switch operator {
	case "=":
		return reflect.DeepEqual(recordValue, value)
	case "!=":
		return !reflect.DeepEqual(recordValue, value)
	case ">":
		return compareNumeric(recordValue, value, func(a, b float64) bool { return cmp.Compare(a, b) > 0 })
	case "<":
		return compareNumeric(recordValue, value, func(a, b float64) bool { return cmp.Compare(a, b) < 0 })
	case ">=":
		return compareNumeric(recordValue, value, func(a, b float64) bool { return cmp.Compare(a, b) >= 0 })
	case "<=":
		return compareNumeric(recordValue, value, func(a, b float64) bool { return cmp.Compare(a, b) <= 0 })
	case "~=":
		return matchesRegex(recordValue, value)
	default:
		logger.Debug("Unsupported operator: %s", operator)
		return false
	}
}

// matchesJSONCondition handles comparison of JSON fields
func matchesJSONCondition(recordValue any, operator string, value any) bool {
	switch operator {
	case "=":
		recordStr := string(recordValue.(json.RawMessage))
		valueStr := fmt.Sprintf("%v", value)
		return recordStr == valueStr
	case "~=":
		// Regex match on JSON string
		return matchesRegex(string(recordValue.(json.RawMessage)), value)
	default:
		logger.Debug("Operator %s not supported for JSON fields", operator)
		return false
	}
}

// compareNumeric converts values to numbers and applies comparison function
func compareNumeric(recordValue, filterValue any, compareFn func(float64, float64) bool) bool {
	rNum, rOk := toNumber(recordValue)
	fNum, fOk := toNumber(filterValue)
	return rOk && fOk && compareFn(rNum, fNum)
}

// matchesRegex checks if recordValue matches the regex pattern in filterValue
func matchesRegex(recordValue, filterValue any) bool {
	recordStr := fmt.Sprintf("%v", recordValue)
	patternStr := fmt.Sprintf("%v", filterValue)

	matched, err := regexp.MatchString(patternStr, recordStr)
	if err != nil {
		logger.Debug("Regex error: %v", err)
		return false
	}
	return matched
}

// toNumber converts a value to float64 for numeric comparison
func toNumber(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case decimal.Decimal:
		f, _ := v.Float64()
		return f, true
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

// dereferenceValue dereferences pointer values
func dereferenceValue(value any) any {
	switch v := value.(type) {
	case *string:
		if v != nil {
			return *v
		}
		return ""
	case *int:
		if v != nil {
			return *v
		}
		return 0
	case *bool:
		if v != nil {
			return *v
		}
		return false
	case *float64:
		if v != nil {
			return *v
		}
		return 0.0
	case *[]int:
		if v != nil {
			return *v
		}
		return []int{}
	case *[]string:
		if v != nil {
			return *v
		}
		return []string{}
	case *json.RawMessage:
		if v != nil {
			return *v
		}
		return json.RawMessage(nil)
	case *decimal.Decimal:
		if v != nil {
			return *v
		}
		return decimal.Zero
	case *dsfetch.Maybe[int]:
		if v != nil && !v.Null() {
			val, _ := v.Value()
			return val
		}
		return 0
	case *dsfetch.Maybe[string]:
		if v != nil && !v.Null() {
			val, _ := v.Value()
			return val
		}
		return ""
	default:
		return value
	}
}

// selectFields returns the requested fields (and id) from each record
func selectFields(records []map[string]any, fields []string) []map[string]any {
	filtered := make([]map[string]any, len(records))
	for i, record := range records {
		filtered[i] = make(map[string]any)
		if id, ok := record["id"]; ok {
			filtered[i]["id"] = id
		}
		for _, field := range fields {
			if value, ok := record[field]; ok {
				filtered[i][field] = dereferenceValue(value)
			}
		}
	}
	return filtered
}

// convertToMapFormat converts an array of records to a map keyed by ID
// This matches the old datastorereader output format for backward compatibility
func convertToMapFormat(records []map[string]any) map[string]any {
	result := make(map[string]any, len(records))
	for _, record := range records {
		id := dereferenceValue(record["id"])
		idStr := fmt.Sprintf("%v", id)
		result[idStr] = record
	}
	return result
}

// snakeToPascal converts snake_case to PascalCase
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	caser := cases.Title(language.English)

	for i := range parts {
		parts[i] = caser.String(parts[i])
	}

	result := strings.Join(parts, "")
	result = strings.ReplaceAll(result, "Id", "ID")
	result = strings.ReplaceAll(result, "Ids", "IDs")

	return result
}
