package costs

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestVerifyCatalogAcceptsSignedPublicEntry(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := CatalogSnapshot{
		Version:     "catalog-2026-08-13",
		GeneratedAt: mustTime(t, "2026-08-13T00:00:00Z"),
		Entries: []CatalogEntry{{
			PublicModelName: "Example Code Model", EffectiveAt: mustTime(t, "2026-08-12T00:00:00Z"),
			SourceURL: "https://www.nexotoken.net/official/pricing", Unit: "USD-per-million-tokens",
			Price: Price{Currency: "USD", InputMicrosPerMTok: 500_000, OutputMicrosPerMTok: 3_000_000, Version: "catalog-2026-08-13"},
		}},
	}
	if err := SignCatalog(&snapshot, privateKey); err != nil {
		t.Fatal(err)
	}
	if err := VerifyCatalog(snapshot, publicKey, mustTime(t, "2026-08-13T12:00:00Z")); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyCatalogRejectsFutureOrTamperedSnapshots(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := CatalogSnapshot{Version: "catalog-1", GeneratedAt: mustTime(t, "2026-08-14T00:00:00Z"), Entries: []CatalogEntry{{PublicModelName: "Example", EffectiveAt: mustTime(t, "2026-08-13T00:00:00Z"), SourceURL: "https://www.nexotoken.net/official/pricing", Unit: "USD-per-million-tokens", Price: Price{Currency: "USD", Version: "catalog-1"}}}}
	if err := SignCatalog(&snapshot, privateKey); err != nil {
		t.Fatal(err)
	}
	if err := VerifyCatalog(snapshot, publicKey, mustTime(t, "2026-08-13T12:00:00Z")); err == nil {
		t.Fatal("expected future snapshot to be rejected")
	}
	snapshot.GeneratedAt = mustTime(t, "2026-08-13T00:00:00Z")
	if err := VerifyCatalog(snapshot, publicKey, mustTime(t, "2026-08-13T12:00:00Z")); err == nil {
		t.Fatal("expected changed signed payload to be rejected")
	}
}

func TestExchangeRateMarksStaleDataInsteadOfSilentlyUsingIt(t *testing.T) {
	rate := ExchangeRateSnapshot{BaseCurrency: "USD", QuoteCurrency: "CNY", RateMicros: 7_200_000, EffectiveAt: mustTime(t, "2026-08-10T00:00:00Z")}
	if !rate.IsStale(mustTime(t, "2026-08-13T00:00:01Z"), 48*time.Hour) {
		t.Fatal("expected stale rate")
	}
	if rate.IsStale(mustTime(t, "2026-08-11T00:00:00Z"), 48*time.Hour) {
		t.Fatal("rate should still be current")
	}
}
