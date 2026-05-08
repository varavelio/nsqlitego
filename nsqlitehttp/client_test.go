package nsqlitehttp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	json "github.com/goccy/go-json"
	"github.com/varavelio/nsqlitego/internal/vdl"
)

func TestSendPingUsesVDLSystemHealthRPC(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method %q, got %q", http.MethodPost, r.Method)
		}
		if r.URL.Path != "/rpc/System/health" {
			t.Fatalf("expected path %q, got %q", "/rpc/System/health", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("expected auth header %q, got %q", "Bearer token", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "{}" {
			t.Fatalf("expected body %q, got %q", "{}", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(`{"ok":true,"output":{"healthy":true,"database":true,"message":"ok"}}`),
		)
	}))
	defer server.Close()

	client, err := NewClient(fmt.Sprintf("%s?authToken=Bearer%%20token", server.URL))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.SendPing(context.Background()); err != nil {
		t.Fatalf("send ping: %v", err)
	}
}

func TestSendQueriesUsesVDLClientTypes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method %q, got %q", http.MethodPost, r.Method)
		}
		if r.URL.Path != "/rpc/Database/query" {
			t.Fatalf("expected path %q, got %q", "/rpc/Database/query", r.URL.Path)
		}

		var body vdl.DatabaseQueryInput
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body.Queries) != 1 {
			t.Fatalf("expected 1 query, got %d", len(body.Queries))
		}

		query := body.Queries[0]
		if query.Query != "SELECT :id, :name, :missing" {
			t.Fatalf("unexpected query %q", query.Query)
		}
		if query.GetTxId() != "tx-123" {
			t.Fatalf("unexpected txId %q", query.GetTxId())
		}

		params := query.GetParams()
		if len(params) != 3 {
			t.Fatalf("expected 3 params, got %d", len(params))
		}
		if params[0].GetName() != "id" || params[0].Value.GetInteger() != 7 {
			t.Fatalf("unexpected first param %+v", params[0])
		}
		if params[1].GetName() != "name" || params[1].Value.GetText() != "alice" {
			t.Fatalf("unexpected second param %+v", params[1])
		}
		if params[2].GetName() != "missing" || !params[2].Value.GetNull() {
			t.Fatalf("unexpected third param %+v", params[2])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(
				`{"ok":true,"output":{"time":0.02,"results":[{"type":"read","time":0.01,"columns":["id","name","missing"],"types":["INTEGER","TEXT","NULL"],"rows":[[{"integer":7},{"text":"alice"},{"null":true}]]}]}}`,
			),
		)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	results, err := client.SendQueries(context.Background(), []Query{{
		Query: "SELECT :id, :name, :missing",
		TxId:  vdl.Ptr("tx-123"),
		Params: &[]QueryParam{
			{Name: vdl.Ptr("id"), Value: vdl.SqliteValue{Integer: vdl.Ptr(int64(7))}},
			{Name: vdl.Ptr("name"), Value: vdl.SqliteValue{Text: vdl.Ptr("alice")}},
			{Name: vdl.Ptr("missing"), Value: vdl.SqliteValue{Null: vdl.Ptr(true)}},
		},
	}})
	if err != nil {
		t.Fatalf("send queries: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != QueryResponseTypeRead {
		t.Fatalf("unexpected result type %q", results[0].Type)
	}
	rows := results[0].GetRows()
	if len(rows) != 1 || len(rows[0]) != 3 {
		t.Fatalf("unexpected rows %+v", rows)
	}
	if rows[0][0].GetInteger() != 7 || rows[0][1].GetText() != "alice" || !rows[0][2].GetNull() {
		t.Fatalf("unexpected row values %+v", rows[0])
	}
}

func TestGetStatsReturnsVDLStats(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method %q, got %q", http.MethodPost, r.Method)
		}
		if r.URL.Path != "/rpc/System/status" {
			t.Fatalf("expected path %q, got %q", "/rpc/System/status", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(
				`{"ok":true,"output":{"name":"nsqlite","version":"v2.0.0","stats":{"startedAt":"2026-01-01T00:00:00Z","uptimeSeconds":65,"totals":{"reads":1,"writes":2,"begins":3,"commits":4,"rollbacks":5,"errors":6,"httpRequests":7},"queued":{"begins":8,"writes":9,"httpRequests":10},"minutes":{"2026-01-01T00:01:00Z":{"reads":11,"writes":12,"begins":13,"commits":14,"rollbacks":15,"errors":16,"httpRequests":17}}}}}`,
			),
		)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	stats, err := client.GetStats(context.Background())
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}

	if !stats.StartedAt.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected StartedAt %v", stats.StartedAt)
	}
	if stats.UptimeSeconds != 65 {
		t.Fatalf("unexpected UptimeSeconds %v", stats.UptimeSeconds)
	}
	if stats.Queued.HttpRequests != 10 {
		t.Fatalf("unexpected queued stats %+v", stats.Queued)
	}
	minuteStats, ok := stats.Minutes["2026-01-01T00:01:00Z"]
	if !ok {
		t.Fatalf("expected minute stats map entry, got %+v", stats.Minutes)
	}
	if minuteStats.HttpRequests != 17 {
		t.Fatalf("unexpected minute HTTP requests %d", minuteStats.HttpRequests)
	}
}
