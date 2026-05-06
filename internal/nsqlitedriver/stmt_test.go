package nsqlitedriver

import (
	"database/sql/driver"
	"reflect"
	"testing"

	"github.com/varavelio/nsqlitego/nsqlitehttp"
)

func TestConvertNamedValueToQueryParam(t *testing.T) {
	tests := []struct {
		name  string
		input []driver.NamedValue
		want  []nsqlitehttp.QueryParam
	}{
		{
			name:  "Empty input",
			input: []driver.NamedValue{},
			want:  []nsqlitehttp.QueryParam{},
		},
		{
			name:  "Single value",
			input: []driver.NamedValue{{Ordinal: 1, Value: "test"}},
			want:  []nsqlitehttp.QueryParam{{Value: "test"}},
		},
		{
			name: "Multiple values",
			input: []driver.NamedValue{
				{Ordinal: 1, Value: 42},
				{Name: "exampleName", Value: "exampleValue"},
				{Ordinal: 3, Value: 3.14},
			},
			want: []nsqlitehttp.QueryParam{
				{Value: 42},
				{Name: "exampleName", Value: "exampleValue"},
				{Value: 3.14},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertNamedValueToQueryParam(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertNamedValueToQueryParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertValueToNamedValue(t *testing.T) {
	tests := []struct {
		name  string
		input []driver.Value
		want  []driver.NamedValue
	}{
		{
			name:  "Empty input",
			input: []driver.Value{},
			want:  nil,
		},
		{
			name:  "Single value",
			input: []driver.Value{"test"},
			want:  []driver.NamedValue{{Ordinal: 1, Value: "test"}},
		},
		{
			name:  "Multiple values",
			input: []driver.Value{42, "example", 3.14},
			want: []driver.NamedValue{
				{Ordinal: 1, Value: 42},
				{Ordinal: 2, Value: "example"},
				{Ordinal: 3, Value: 3.14},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertValueToNamedValue(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertValueToNamedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
