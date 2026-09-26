package signal

import (
	"context"
)

// RawSignal representa un senyal brut extret d'un canal (oferta de feina, fil de fòrum, etc.)
type RawSignal struct {
	SourceURL   string  `json:"source_url"`
	CompanyName *string `json:"company_name,omitempty"`
	Title       string  `json:"title"`
	RawText     string  `json:"raw_text"`
	SignalType  string  `json:"signal_type"` // "oferta_feina" o "queixa_forum"
	Source      string  `json:"source"`      // Identificador del canal (ex: "feina_activa", "reddit_smallbusiness", etc.)
}

// Scraper és la interfície Strategy per a qualsevol extractor multicanal
type Scraper interface {
	Name() string
	Fetch(ctx context.Context) ([]RawSignal, error)
}
