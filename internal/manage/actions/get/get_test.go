package get

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/OpenSlides/openslides-go/datastore/dsfetch"
	"github.com/shopspring/decimal"
)

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"first_name", "FirstName"},
		{"last_name", "LastName"},
		{"is_active", "IsActive"},
		{"user_id", "UserID"},
		{"user_ids", "UserIDs"},
		{"meeting_id", "MeetingID"},
		{"start_time", "StartTime"},
		{"id", "ID"},
		{"single", "Single"},
		{"multiple_word_field", "MultipleWordField"},
		{"field_with_id", "FieldWithID"},
		{"field_with_ids", "FieldWithIDs"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := snakeToPascal(tt.input)
			if result != tt.expected {
				t.Errorf("snakeToPascal(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetermineFieldsToFetch(t *testing.T) {
	tests := []struct {
		name            string
		requestedFields []string
		filter          map[string]string
		rawFilter       *RawFilter
		expected        []string
	}{
		{
			name:            "no fields requested",
			requestedFields: nil,
			filter:          nil,
			rawFilter:       nil,
			expected:        []string{"id"},
		},
		{
			name:            "only requested fields",
			requestedFields: []string{"first_name", "last_name"},
			filter:          nil,
			rawFilter:       nil,
			expected:        []string{"id", "first_name", "last_name"},
		},
		{
			name:            "with simple filter",
			requestedFields: []string{"first_name"},
			filter:          map[string]string{"is_active": "true"},
			rawFilter:       nil,
			expected:        []string{"id", "first_name", "is_active"},
		},
		{
			name:            "with raw filter",
			requestedFields: []string{"first_name"},
			filter:          nil,
			rawFilter: &RawFilter{
				Field:    "last_name",
				Operator: "~=",
				Value:    "Smith",
			},
			expected: []string{"id", "first_name", "last_name"},
		},
		{
			name:            "with complex raw filter",
			requestedFields: []string{"first_name"},
			filter:          nil,
			rawFilter: &RawFilter{
				AndFilter: []RawFilter{
					{Field: "is_active", Operator: "=", Value: true},
					{Field: "age", Operator: ">", Value: 18},
				},
			},
			expected: []string{"id", "first_name", "is_active", "age"},
		},
		{
			name:            "deduplication",
			requestedFields: []string{"first_name", "id"},
			filter:          map[string]string{"first_name": "John"},
			rawFilter:       nil,
			expected:        []string{"id", "first_name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineFieldsToFetch(tt.requestedFields, tt.filter, tt.rawFilter)

			// Sort both for comparison
			slices.Sort(result)
			slices.Sort(tt.expected)

			if !slices.Equal(result, tt.expected) {
				t.Errorf("determineFieldsToFetch() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractFieldsFromRawFilter(t *testing.T) {
	tests := []struct {
		name      string
		rawFilter *RawFilter
		expected  []string
	}{
		{
			name: "simple filter",
			rawFilter: &RawFilter{
				Field:    "name",
				Operator: "=",
				Value:    "test",
			},
			expected: []string{"name"},
		},
		{
			name: "and filter",
			rawFilter: &RawFilter{
				AndFilter: []RawFilter{
					{Field: "first_name", Operator: "=", Value: "John"},
					{Field: "last_name", Operator: "=", Value: "Doe"},
				},
			},
			expected: []string{"first_name", "last_name"},
		},
		{
			name: "or filter",
			rawFilter: &RawFilter{
				OrFilter: []RawFilter{
					{Field: "status", Operator: "=", Value: "active"},
					{Field: "status", Operator: "=", Value: "pending"},
				},
			},
			expected: []string{"status"},
		},
		{
			name: "not filter",
			rawFilter: &RawFilter{
				NotFilter: &RawFilter{
					Field:    "is_deleted",
					Operator: "=",
					Value:    true,
				},
			},
			expected: []string{"is_deleted"},
		},
		{
			name: "nested filters",
			rawFilter: &RawFilter{
				AndFilter: []RawFilter{
					{Field: "age", Operator: ">", Value: 18},
					{
						OrFilter: []RawFilter{
							{Field: "city", Operator: "=", Value: "NYC"},
							{Field: "city", Operator: "=", Value: "LA"},
						},
					},
				},
			},
			expected: []string{"age", "city"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldsSet := make(map[string]bool)
			extractFieldsFromRawFilter(tt.rawFilter, fieldsSet)

			result := make([]string, 0, len(fieldsSet))
			for field := range fieldsSet {
				result = append(result, field)
			}

			slices.Sort(result)
			slices.Sort(tt.expected)

			if !slices.Equal(result, tt.expected) {
				t.Errorf("extractFieldsFromRawFilter() found %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDereferenceValue(t *testing.T) {
	stringVal := "test"
	intVal := 42
	boolVal := true
	floatVal := 3.14
	intSlice := []int{1, 2, 3}
	stringSlice := []string{"a", "b"}
	jsonVal := json.RawMessage(`{"key":"value"}`)
	decimalVal := decimal.NewFromFloat(3.14159)

	// Create Maybe values
	maybeIntWithValue := dsfetch.MaybeValue(99)
	maybeIntNull := dsfetch.Maybe[int]{}
	maybeStringWithValue := dsfetch.MaybeValue("maybe-string")
	maybeStringNull := dsfetch.Maybe[string]{}

	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{"string pointer", &stringVal, "test"},
		{"int pointer", &intVal, 42},
		{"bool pointer", &boolVal, true},
		{"float pointer", &floatVal, 3.14},
		{"int slice pointer", &intSlice, []int{1, 2, 3}},
		{"string slice pointer", &stringSlice, []string{"a", "b"}},
		{"json pointer", &jsonVal, json.RawMessage(`{"key":"value"}`)},
		{"decimal pointer", &decimalVal, decimal.NewFromFloat(3.14159)},
		{"maybe int with value", &maybeIntWithValue, 99},
		{"maybe int null", &maybeIntNull, 0},
		{"maybe string with value", &maybeStringWithValue, "maybe-string"},
		{"maybe string null", &maybeStringNull, ""},
		{"nil string pointer", (*string)(nil), ""},
		{"nil int pointer", (*int)(nil), 0},
		{"nil bool pointer", (*bool)(nil), false},
		{"nil float pointer", (*float64)(nil), 0.0},
		{"nil int slice pointer", (*[]int)(nil), []int{}},
		{"nil string slice pointer", (*[]string)(nil), []string{}},
		{"nil json pointer", (*json.RawMessage)(nil), json.RawMessage(nil)},
		{"nil decimal pointer", (*decimal.Decimal)(nil), decimal.Zero},
		{"direct string", "direct", "direct"},
		{"direct int", 100, 100},
		{"direct bool", false, false},
		{"direct float", 2.71, 2.71},
		{"direct json", json.RawMessage(`{"test":1}`), json.RawMessage(`{"test":1}`)},
		{"direct decimal", decimal.NewFromInt(42), decimal.NewFromInt(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dereferenceValue(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("dereferenceValue(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToNumber(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expectedNum float64
		expectedOk  bool
	}{
		{"int", 42, 42.0, true},
		{"int64", int64(100), 100.0, true},
		{"float64", 3.14, 3.14, true},
		{"float32", float32(2.5), 2.5, true},
		{"decimal", decimal.NewFromFloat(3.14), 3.14, true},
		{"decimal zero", decimal.Zero, 0.0, true},
		{"decimal negative", decimal.NewFromFloat(-5.5), -5.5, true},
		{"string number", "123.45", 123.45, true},
		{"string not number", "abc", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, ok := toNumber(tt.input)
			if ok != tt.expectedOk {
				t.Errorf("toNumber(%v) ok = %v, want %v", tt.input, ok, tt.expectedOk)
			}
			if ok && num != tt.expectedNum {
				t.Errorf("toNumber(%v) = %v, want %v", tt.input, num, tt.expectedNum)
			}
		})
	}
}

func TestCompareNumeric(t *testing.T) {
	tests := []struct {
		name      string
		val1      any
		val2      any
		compareFn func(float64, float64) bool
		expected  bool
	}{
		{"10 > 5", 10, 5, func(a, b float64) bool { return a > b }, true},
		{"5 > 10", 5, 10, func(a, b float64) bool { return a > b }, false},
		{"5 < 10", 5, 10, func(a, b float64) bool { return a < b }, true},
		{"10 < 5", 10, 5, func(a, b float64) bool { return a < b }, false},
		{"5 >= 5", 5, 5, func(a, b float64) bool { return a >= b }, true},
		{"5 <= 5", 5, 5, func(a, b float64) bool { return a <= b }, true},
		{"float > int", 10.5, 10, func(a, b float64) bool { return a > b }, true},
		{"string numbers", "20", "10", func(a, b float64) bool { return a > b }, true},
		{"decimal > int", decimal.NewFromFloat(10.5), 10, func(a, b float64) bool { return a > b }, true},
		{"int > decimal", 20, decimal.NewFromFloat(10.5), func(a, b float64) bool { return a > b }, true},
		{"invalid comparison", "abc", 10, func(a, b float64) bool { return a > b }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareNumeric(tt.val1, tt.val2, tt.compareFn)
			if result != tt.expected {
				t.Errorf("compareNumeric(%v, %v) = %v, want %v", tt.val1, tt.val2, result, tt.expected)
			}
		})
	}
}

func TestMatchesRegex(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		pattern  any
		expected bool
	}{
		{"simple match", "test", "test", true},
		{"no match", "test", "other", false},
		{"prefix match", "testing", "^test", true},
		{"suffix match", "testing", "ing$", true},
		{"case sensitive", "Test", "test", false},
		{"case insensitive", "Test", "(?i)test", true},
		{"numeric match", 42, "42", true},
		{"wildcard", "anything", ".*", true},
		{"invalid regex", "test", "[", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesRegex(tt.value, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesRegex(%v, %v) = %v, want %v", tt.value, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestMatchesJSONCondition(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		operator string
		filter   any
		expected bool
	}{
		{
			name:     "json equality match",
			value:    json.RawMessage(`{"key":"value"}`),
			operator: "=",
			filter:   `{"key":"value"}`,
			expected: true,
		},
		{
			name:     "json equality no match",
			value:    json.RawMessage(`{"key":"value"}`),
			operator: "=",
			filter:   `{"key":"other"}`,
			expected: false,
		},
		{
			name:     "json regex match",
			value:    json.RawMessage(`{"status":"active"}`),
			operator: "~=",
			filter:   "active",
			expected: true,
		},
		{
			name:     "json regex no match",
			value:    json.RawMessage(`{"status":"active"}`),
			operator: "~=",
			filter:   "inactive",
			expected: false,
		},
		{
			name:     "unsupported operator",
			value:    json.RawMessage(`{"num":42}`),
			operator: ">",
			filter:   40,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesJSONCondition(tt.value, tt.operator, tt.filter)
			if result != tt.expected {
				t.Errorf("matchesJSONCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMatchesCondition(t *testing.T) {
	tests := []struct {
		name     string
		record   map[string]any
		field    string
		operator string
		value    any
		expected bool
	}{
		{"equal match", map[string]any{"name": "test"}, "name", "=", "test", true},
		{"equal no match", map[string]any{"name": "test"}, "name", "=", "other", false},
		{"not equal", map[string]any{"name": "test"}, "name", "!=", "other", true},
		{"greater than", map[string]any{"age": 30}, "age", ">", 20, true},
		{"less than", map[string]any{"age": 15}, "age", "<", 20, true},
		{"greater or equal", map[string]any{"age": 20}, "age", ">=", 20, true},
		{"less or equal", map[string]any{"age": 20}, "age", "<=", 20, true},
		{"regex match", map[string]any{"name": "admin"}, "name", "~=", "^ad", true},
		{"field not exists", map[string]any{"name": "test"}, "missing", "=", "test", false},
		{"unsupported operator", map[string]any{"name": "test"}, "name", "??", "test", false},
		{"json field equality", map[string]any{"data": json.RawMessage(`{"x":1}`)}, "data", "=", `{"x":1}`, true},
		{"json field regex", map[string]any{"data": json.RawMessage(`{"status":"ok"}`)}, "data", "~=", "ok", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesCondition(tt.record, tt.field, tt.operator, tt.value)
			if result != tt.expected {
				t.Errorf("matchesCondition(%v, %q, %q, %v) = %v, want %v",
					tt.record, tt.field, tt.operator, tt.value, result, tt.expected)
			}
		})
	}
}

func TestMatchesSimpleFilter(t *testing.T) {
	tests := []struct {
		name     string
		record   map[string]any
		filter   map[string]string
		expected bool
	}{
		{
			name:     "single match",
			record:   map[string]any{"name": "John", "age": 30},
			filter:   map[string]string{"name": "John"},
			expected: true,
		},
		{
			name:     "single no match",
			record:   map[string]any{"name": "John", "age": 30},
			filter:   map[string]string{"name": "Jane"},
			expected: false,
		},
		{
			name:     "multiple match",
			record:   map[string]any{"name": "John", "age": 30, "city": "NYC"},
			filter:   map[string]string{"name": "John", "city": "NYC"},
			expected: true,
		},
		{
			name:     "multiple partial match",
			record:   map[string]any{"name": "John", "age": 30, "city": "NYC"},
			filter:   map[string]string{"name": "John", "city": "LA"},
			expected: false,
		},
		{
			name:     "field missing",
			record:   map[string]any{"name": "John"},
			filter:   map[string]string{"age": "30"},
			expected: false,
		},
		{
			name:     "empty filter",
			record:   map[string]any{"name": "John"},
			filter:   map[string]string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesSimpleFilter(tt.record, tt.filter)
			if result != tt.expected {
				t.Errorf("matchesSimpleFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMatchesRawFilter(t *testing.T) {
	tests := []struct {
		name      string
		record    map[string]any
		rawFilter *RawFilter
		expected  bool
	}{
		{
			name:   "simple condition match",
			record: map[string]any{"name": "John"},
			rawFilter: &RawFilter{
				Field:    "name",
				Operator: "=",
				Value:    "John",
			},
			expected: true,
		},
		{
			name:   "and filter all match",
			record: map[string]any{"age": 30, "active": true},
			rawFilter: &RawFilter{
				AndFilter: []RawFilter{
					{Field: "age", Operator: ">", Value: 20},
					{Field: "active", Operator: "=", Value: true},
				},
			},
			expected: true,
		},
		{
			name:   "and filter partial match",
			record: map[string]any{"age": 15, "active": true},
			rawFilter: &RawFilter{
				AndFilter: []RawFilter{
					{Field: "age", Operator: ">", Value: 20},
					{Field: "active", Operator: "=", Value: true},
				},
			},
			expected: false,
		},
		{
			name:   "or filter one match",
			record: map[string]any{"status": "active"},
			rawFilter: &RawFilter{
				OrFilter: []RawFilter{
					{Field: "status", Operator: "=", Value: "active"},
					{Field: "status", Operator: "=", Value: "pending"},
				},
			},
			expected: true,
		},
		{
			name:   "or filter no match",
			record: map[string]any{"status": "inactive"},
			rawFilter: &RawFilter{
				OrFilter: []RawFilter{
					{Field: "status", Operator: "=", Value: "active"},
					{Field: "status", Operator: "=", Value: "pending"},
				},
			},
			expected: false,
		},
		{
			name:   "not filter",
			record: map[string]any{"deleted": false},
			rawFilter: &RawFilter{
				NotFilter: &RawFilter{
					Field:    "deleted",
					Operator: "=",
					Value:    true,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesRawFilter(tt.record, tt.rawFilter)
			if result != tt.expected {
				t.Errorf("matchesRawFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMatchesRawFilter_Complex(t *testing.T) {
	// Test: (age > 18 AND city = "NYC") OR (age > 65 AND city = "FL")
	filter := &RawFilter{
		OrFilter: []RawFilter{
			{
				AndFilter: []RawFilter{
					{Field: "age", Operator: ">", Value: 18},
					{Field: "city", Operator: "=", Value: "NYC"},
				},
			},
			{
				AndFilter: []RawFilter{
					{Field: "age", Operator: ">", Value: 65},
					{Field: "city", Operator: "=", Value: "FL"},
				},
			},
		},
	}

	tests := []struct {
		name     string
		record   map[string]any
		expected bool
	}{
		{"young in NYC", map[string]any{"age": 25, "city": "NYC"}, true},
		{"senior in FL", map[string]any{"age": 70, "city": "FL"}, true},
		{"young in FL", map[string]any{"age": 25, "city": "FL"}, false},
		{"senior in NYC", map[string]any{"age": 70, "city": "NYC"}, true},
		{"minor in NYC", map[string]any{"age": 16, "city": "NYC"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesRawFilter(tt.record, filter)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMatchesRawFilter_TripleNested(t *testing.T) {
	// NOT (age > 18 AND is_active = true)
	// Equivalent to: age <= 18 OR is_active = false
	filter := &RawFilter{
		NotFilter: &RawFilter{
			AndFilter: []RawFilter{
				{Field: "age", Operator: ">", Value: 18},
				{Field: "is_active", Operator: "=", Value: true},
			},
		},
	}

	tests := []struct {
		name     string
		record   map[string]any
		expected bool
	}{
		{"young and active", map[string]any{"age": 15, "is_active": true}, true},
		{"adult and inactive", map[string]any{"age": 25, "is_active": false}, true},
		{"adult and active", map[string]any{"age": 25, "is_active": true}, false},
		{"young and inactive", map[string]any{"age": 15, "is_active": false}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesRawFilter(tt.record, filter)
			if result != tt.expected {
				t.Errorf("got %v, want %v for record %v", result, tt.expected, tt.record)
			}
		})
	}
}

func TestSelectFields(t *testing.T) {
	stringVal := "John"
	intVal := 30

	tests := []struct {
		name     string
		records  []map[string]any
		fields   []string
		expected []map[string]any
	}{
		{
			name: "select subset",
			records: []map[string]any{
				{"id": 1, "name": &stringVal, "age": &intVal, "city": "NYC"},
			},
			fields: []string{"id", "name"},
			expected: []map[string]any{
				{"id": 1, "name": "John"},
			},
		},
		{
			name: "select all present",
			records: []map[string]any{
				{"id": 1, "name": &stringVal},
			},
			fields: []string{"id", "name"},
			expected: []map[string]any{
				{"id": 1, "name": "John"},
			},
		},
		{
			name: "select with missing field",
			records: []map[string]any{
				{"id": 1, "name": &stringVal},
			},
			fields: []string{"id", "name", "missing"},
			expected: []map[string]any{
				{"id": 1, "name": "John"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectFields(tt.records, tt.fields)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("selectFields() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertToMapFormat(t *testing.T) {
	tests := []struct {
		name     string
		records  []map[string]any
		expected map[string]any
	}{
		{
			name: "single record",
			records: []map[string]any{
				{"id": 1, "name": "test"},
			},
			expected: map[string]any{
				"1": map[string]any{"id": 1, "name": "test"},
			},
		},
		{
			name: "multiple records",
			records: []map[string]any{
				{"id": 1, "name": "first"},
				{"id": 2, "name": "second"},
			},
			expected: map[string]any{
				"1": map[string]any{"id": 1, "name": "first"},
				"2": map[string]any{"id": 2, "name": "second"},
			},
		},
		{
			name:     "empty records",
			records:  []map[string]any{},
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToMapFormat(tt.records)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertToMapFormat() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestApplyFilters(t *testing.T) {
	records := []map[string]any{
		{"id": 1, "name": "John", "age": 30},
		{"id": 2, "name": "Jane", "age": 25},
		{"id": 3, "name": "Bob", "age": 35},
	}

	t.Run("simple filter", func(t *testing.T) {
		filter := map[string]string{"name": "John"}
		result := applyFilters(records, filter, nil)

		if len(result) != 1 {
			t.Errorf("applyFilters() returned %d records, want 1", len(result))
		}
		if result[0]["name"] != "John" {
			t.Errorf("applyFilters() returned wrong record")
		}
	})

	t.Run("raw filter", func(t *testing.T) {
		rawFilter := &RawFilter{
			Field:    "age",
			Operator: ">",
			Value:    28,
		}
		result := applyFilters(records, nil, rawFilter)

		if len(result) != 2 {
			t.Errorf("applyFilters() returned %d records, want 2", len(result))
		}
	})

	t.Run("no filter", func(t *testing.T) {
		result := applyFilters(records, nil, nil)

		if len(result) != 3 {
			t.Errorf("applyFilters() returned %d records, want 3", len(result))
		}
	})
}

func TestMaybeTypeFiltering(t *testing.T) {
	// Simulate records with Maybe fields (as they would appear after dereferencing)
	maybeInt1 := dsfetch.MaybeValue(10)
	maybeInt2 := dsfetch.MaybeValue(20)
	maybeIntNull := dsfetch.Maybe[int]{}

	maybeStr1 := dsfetch.MaybeValue("active")
	maybeStr2 := dsfetch.MaybeValue("inactive")
	maybeStrNull := dsfetch.Maybe[string]{}

	records := []map[string]any{
		{"id": 1, "maybe_field": &maybeInt1, "status": &maybeStr1},
		{"id": 2, "maybe_field": &maybeInt2, "status": &maybeStr2},
		{"id": 3, "maybe_field": &maybeIntNull, "status": &maybeStrNull},
	}

	t.Run("filter maybe int field", func(t *testing.T) {
		rawFilter := &RawFilter{
			Field:    "maybe_field",
			Operator: ">",
			Value:    15,
		}
		result := applyFilters(records, nil, rawFilter)

		if len(result) != 1 {
			t.Errorf("Expected 1 record with maybe_field > 15, got %d", len(result))
		}
		if len(result) > 0 && result[0]["id"] != 2 {
			t.Errorf("Expected record with id=2, got id=%v", result[0]["id"])
		}
	})

	t.Run("filter maybe string field", func(t *testing.T) {
		filter := map[string]string{"status": "active"}
		result := applyFilters(records, filter, nil)

		if len(result) != 1 {
			t.Errorf("Expected 1 record with status=active, got %d", len(result))
		}
		if len(result) > 0 && result[0]["id"] != 1 {
			t.Errorf("Expected record with id=1, got id=%v", result[0]["id"])
		}
	})

	t.Run("filter null maybe fields", func(t *testing.T) {
		rawFilter := &RawFilter{
			Field:    "maybe_field",
			Operator: "=",
			Value:    0, // Null Maybe[int] dereferences to 0
		}
		result := applyFilters(records, nil, rawFilter)

		if len(result) != 1 {
			t.Errorf("Expected 1 record with null maybe_field, got %d", len(result))
		}
		if len(result) > 0 && result[0]["id"] != 3 {
			t.Errorf("Expected record with id=3, got id=%v", result[0]["id"])
		}
	})

	t.Run("regex on maybe string", func(t *testing.T) {
		rawFilter := &RawFilter{
			Field:    "status",
			Operator: "~=",
			Value:    "^act",
		}
		result := applyFilters(records, nil, rawFilter)

		if len(result) != 1 {
			t.Errorf("Expected 1 record matching regex, got %d", len(result))
		}
	})
}
