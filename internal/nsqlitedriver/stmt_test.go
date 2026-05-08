package nsqlitedriver

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/varavelio/nsqlitego/internal/vdl"
	"github.com/varavelio/nsqlitego/nsqlitehttp"
)

func TestConvertNamedValueToQueryParam(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		got, err := convertNamedValueToQueryParam(nil)
		if err != nil {
			t.Fatalf("convertNamedValueToQueryParam returned error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty params, got %+v", got)
		}
	})

	t.Run("converts supported driver values to sqlite values", func(t *testing.T) {
		at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		got, err := convertNamedValueToQueryParam([]driver.NamedValue{
			{Ordinal: 1, Value: int64(42)},
			{Name: "name", Value: "alice"},
			{Ordinal: 3, Value: 3.14},
			{Name: "payload", Value: []byte{0x01, 0x02}},
			{Name: "missing", Value: nil},
			{Name: "created_at", Value: at},
		})
		if err != nil {
			t.Fatalf("convertNamedValueToQueryParam returned error: %v", err)
		}
		want := []nsqlitehttp.QueryParam{
			{Value: vdl.SqliteValue{Integer: vdl.Ptr(int64(42))}},
			{Name: vdl.Ptr("name"), Value: vdl.SqliteValue{Text: vdl.Ptr("alice")}},
			{Value: vdl.SqliteValue{Real: vdl.Ptr(3.14)}},
			{Name: vdl.Ptr("payload"), Value: vdl.SqliteValue{Blob: vdl.Ptr("AQI=")}},
			{Name: vdl.Ptr("missing"), Value: vdl.SqliteValue{Null: vdl.Ptr(true)}},
			{
				Name:  vdl.Ptr("created_at"),
				Value: vdl.SqliteValue{Text: vdl.Ptr("2026-01-02T03:04:05Z")},
			},
		}
		assertQueryParamsEqual(t, got, want)
	})

	t.Run("returns an error for unsupported values", func(t *testing.T) {
		_, err := convertNamedValueToQueryParam(
			[]driver.NamedValue{{Ordinal: 1, Value: struct{}{}}},
		)
		if err == nil {
			t.Fatal("expected error for unsupported value")
		}
	})
}

func TestConvertValueToNamedValue(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		got := convertValueToNamedValue([]driver.Value{})
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("multiple values", func(t *testing.T) {
		got := convertValueToNamedValue([]driver.Value{42, "example", 3.14})
		if len(got) != 3 {
			t.Fatalf("expected 3 values, got %+v", got)
		}
		if got[0].Ordinal != 1 || got[0].Value != 42 {
			t.Fatalf("unexpected first named value %+v", got[0])
		}
		if got[1].Ordinal != 2 || got[1].Value != "example" {
			t.Fatalf("unexpected second named value %+v", got[1])
		}
		if got[2].Ordinal != 3 || got[2].Value != 3.14 {
			t.Fatalf("unexpected third named value %+v", got[2])
		}
	})
}

func assertQueryParamsEqual(t *testing.T, got, want []nsqlitehttp.QueryParam) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d params, got %d", len(want), len(got))
	}

	for i := range want {
		if vdl.Val(got[i].Name) != vdl.Val(want[i].Name) {
			t.Fatalf(
				"param %d name mismatch: got %q want %q",
				i,
				vdl.Val(got[i].Name),
				vdl.Val(want[i].Name),
			)
		}
		assertSqliteValueEqual(t, i, got[i].Value, want[i].Value)
	}
}

func assertSqliteValueEqual(t *testing.T, index int, got, want vdl.SqliteValue) {
	t.Helper()

	switch {
	case got.Null != nil || want.Null != nil:
		if vdl.Val(got.Null) != vdl.Val(want.Null) {
			t.Fatalf(
				"param %d null mismatch: got %v want %v",
				index,
				vdl.Val(got.Null),
				vdl.Val(want.Null),
			)
		}
	case got.Integer != nil || want.Integer != nil:
		if vdl.Val(got.Integer) != vdl.Val(want.Integer) {
			t.Fatalf(
				"param %d integer mismatch: got %d want %d",
				index,
				vdl.Val(got.Integer),
				vdl.Val(want.Integer),
			)
		}
	case got.Real != nil || want.Real != nil:
		if vdl.Val(got.Real) != vdl.Val(want.Real) {
			t.Fatalf(
				"param %d real mismatch: got %v want %v",
				index,
				vdl.Val(got.Real),
				vdl.Val(want.Real),
			)
		}
	case got.Text != nil || want.Text != nil:
		if vdl.Val(got.Text) != vdl.Val(want.Text) {
			t.Fatalf(
				"param %d text mismatch: got %q want %q",
				index,
				vdl.Val(got.Text),
				vdl.Val(want.Text),
			)
		}
	case got.Blob != nil || want.Blob != nil:
		if vdl.Val(got.Blob) != vdl.Val(want.Blob) {
			t.Fatalf(
				"param %d blob mismatch: got %q want %q",
				index,
				vdl.Val(got.Blob),
				vdl.Val(want.Blob),
			)
		}
	default:
		t.Fatalf("param %d had no sqlite value set", index)
	}
}
