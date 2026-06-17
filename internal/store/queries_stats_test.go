package store

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"
)

// TestGetStats_PopulatesLoadDurationAndSource reproduces ndc-loader#9:
// GET /api/ndc/stats always returned load_duration_seconds=null and was missing
// the "source" field entirely. The stats response must carry both per the
// specs/query-api.md AC-009 contract.
//
// Named for the observed symptom (null duration, absent source), not a cause.
func TestGetStats_PopulatesLoadDurationAndSource(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping store integration test")
	}

	ctx := context.Background()
	db, err := NewDB(ctx, dbURL)
	if err != nil {
		t.Skipf("cannot connect to database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Seed a single completed load run lasting exactly 6.5 seconds.
	if _, err := db.Exec(ctx, "TRUNCATE load_checkpoints"); err != nil {
		t.Fatalf("failed to truncate load_checkpoints: %v", err)
	}
	_, err = db.Exec(ctx, `
		INSERT INTO load_checkpoints (load_id, dataset, table_name, status, started_at, completed_at)
		VALUES ('run-1', 'ndc_directory', 'products', 'loaded',
		        TIMESTAMPTZ '2026-03-26 22:21:46.5Z',
		        TIMESTAMPTZ '2026-03-26 22:21:53Z')
	`)
	if err != nil {
		t.Fatalf("failed to seed checkpoint: %v", err)
	}

	const wantSource = "https://www.accessdata.fda.gov/cder/ndctext.zip"
	q := NewQueryStore(db, wantSource)

	stats, err := q.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats returned error: %v", err)
	}

	// load_duration_seconds must be populated from the latest load run.
	if stats.LoadDuration == nil {
		t.Fatal("load_duration_seconds is nil; expected ~6.5")
	}
	if math.Abs(*stats.LoadDuration-6.5) > 0.01 {
		t.Errorf("load_duration_seconds = %v; want ~6.5", *stats.LoadDuration)
	}

	// "source" must appear in the JSON contract with the configured URL.
	// Inspect the serialized form so the test is robust to the Go field name.
	raw, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("failed to marshal stats: %v", err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("failed to unmarshal stats: %v", err)
	}
	if _, ok := keys["source"]; !ok {
		t.Errorf("stats JSON is missing the \"source\" field: %s", raw)
	}
	var decoded struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to decode source: %v", err)
	}
	if decoded.Source != wantSource {
		t.Errorf("source = %q; want %q", decoded.Source, wantSource)
	}
}
