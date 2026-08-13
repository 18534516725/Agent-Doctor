package costs

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

const publicPriceUnit = "USD-per-million-tokens"

// CatalogSnapshot is an offline-verifiable copy of public price information.
// It contains no routing, supplier, account, or credential data.
type CatalogSnapshot struct {
	Version     string         `json:"version"`
	GeneratedAt time.Time      `json:"generatedAt"`
	Entries     []CatalogEntry `json:"entries"`
	Signature   string         `json:"signature"`
}

type CatalogEntry struct {
	PublicModelName string    `json:"publicModelName"`
	EffectiveAt     time.Time `json:"effectiveAt"`
	SourceURL       string    `json:"sourceUrl"`
	Unit            string    `json:"unit"`
	Price           Price     `json:"price"`
}

type ExchangeRateSnapshot struct {
	BaseCurrency  string
	QuoteCurrency string
	RateMicros    int64
	EffectiveAt   time.Time
}

func (rate ExchangeRateSnapshot) IsStale(now time.Time, maxAge time.Duration) bool {
	return rate.EffectiveAt.IsZero() || now.Before(rate.EffectiveAt) || maxAge < 0 || now.Sub(rate.EffectiveAt) > maxAge
}

func SignCatalog(snapshot *CatalogSnapshot, privateKey ed25519.PrivateKey) error {
	if snapshot == nil || len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("catalog signing input is invalid")
	}
	payload, err := catalogPayload(*snapshot)
	if err != nil {
		return err
	}
	snapshot.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	return nil
}

func VerifyCatalog(snapshot CatalogSnapshot, publicKey ed25519.PublicKey, now time.Time) error {
	if len(publicKey) != ed25519.PublicKeySize || now.IsZero() || snapshot.Version == "" || snapshot.GeneratedAt.IsZero() || snapshot.GeneratedAt.After(now) || len(snapshot.Entries) == 0 {
		return fmt.Errorf("catalog metadata is invalid")
	}
	for _, entry := range snapshot.Entries {
		if err := validateCatalogEntry(entry); err != nil {
			return err
		}
	}
	signature, err := base64.RawStdEncoding.DecodeString(snapshot.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("catalog signature is invalid")
	}
	payload, err := catalogPayload(snapshot)
	if err != nil {
		return err
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return fmt.Errorf("catalog signature does not match")
	}
	return nil
}

func catalogPayload(snapshot CatalogSnapshot) ([]byte, error) {
	return json.Marshal(struct {
		Version     string         `json:"version"`
		GeneratedAt time.Time      `json:"generatedAt"`
		Entries     []CatalogEntry `json:"entries"`
	}{Version: snapshot.Version, GeneratedAt: snapshot.GeneratedAt.UTC(), Entries: snapshot.Entries})
}

func validateCatalogEntry(entry CatalogEntry) error {
	parsed, err := url.Parse(entry.SourceURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || entry.PublicModelName == "" || entry.EffectiveAt.IsZero() || entry.Unit != publicPriceUnit || !validPrice(entry.Price) {
		return fmt.Errorf("catalog entry is invalid")
	}
	return nil
}
