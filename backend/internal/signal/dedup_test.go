package signal

import (
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Strip UTM and tracking parameters",
			input:    "https://feinaactiva.gencat.cat/oferta?id=123&utm_source=twitter&utm_medium=social&ref=partner",
			expected: "https://feinaactiva.gencat.cat/oferta?id=123",
			wantErr:  false,
		},
		{
			name:     "Strip fbclid, gclid, mc_eid",
			input:    "http://example.com/job/456/?fbclid=IwAR123&gclid=xyz&mc_eid=abc&sector=logistics",
			expected: "http://example.com/job/456?sector=logistics",
			wantErr:  false,
		},
		{
			name:     "Normalize default ports and uppercase scheme/host",
			input:    "HTTP://WWW.EXAMPLE.COM:80/path/to/page/",
			expected: "http://www.example.com/path/to/page",
			wantErr:  false,
		},
		{
			name:     "Normalize HTTPS default port 443",
			input:    "HTTPS://EXAMPLE.COM:443/process",
			expected: "https://example.com/process",
			wantErr:  false,
		},
		{
			name:     "Sort remaining query parameters deterministically",
			input:    "https://example.com/search?z=3&a=1&m=2",
			expected: "https://example.com/search?a=1&m=2&z=3",
			wantErr:  false,
		},
		{
			name:     "Root path trailing slash handling",
			input:    "https://example.com/",
			expected: "https://example.com/",
			wantErr:  false,
		},
		{
			name:     "Invalid scheme rejected",
			input:    "ftp://example.com/file",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Empty URL rejected",
			input:    "   ",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeURL(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("NormalizeURL() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.expected {
				t.Errorf("NormalizeURL() = %q, expected %q", got, tc.expected)
			}
		})
	}
}

func TestCalculateContentHash(t *testing.T) {
	textA := "  Empresa de logística cerca administratiu per gestionar albarans en paper i fulls Excel.  \n\n"
	textB := "empresa de logística cerca administratiu per gestionar albarans en paper i fulls excel."

	hashA := CalculateContentHash(textA)
	hashB := CalculateContentHash(textB)

	if hashA == "" {
		t.Fatal("CalculateContentHash returned empty string")
	}
	if hashA != hashB {
		t.Errorf("Expected identical hash for text with different casing/whitespace, got %s vs %s", hashA, hashB)
	}

	textDifferent := "Empresa de neteja cerca operari per manteniment"
	hashDiff := CalculateContentHash(textDifferent)
	if hashA == hashDiff {
		t.Errorf("Expected different hashes for different texts")
	}
}

func TestSimilarityAndSyndication(t *testing.T) {
	orig := "Empresa de transports i logística a Girona cerca auxiliar administratiu per al control de rutes, albarans i gestió d'estocs amb Excel."
	syndicated := "Empresa de transports i logística a Girona cerca auxiliar administratiu per al control de rutes, albarans i gestió d'estocs amb Excel. Contactar a rrhh@empresa.cat"
	different := "Restaurant familiar al centre de Tarragona busca cambrer de sala per caps de setmana i torns rotatius."

	simHigh := CalculateTextSimilarity(orig, syndicated)
	if simHigh < 0.75 {
		t.Errorf("Expected high similarity for syndicated text, got %f", simHigh)
	}

	if !IsSyndicated(orig, syndicated, 0.70) {
		t.Errorf("Expected IsSyndicated to be true for syndicated text at 0.70 threshold")
	}

	simLow := CalculateTextSimilarity(orig, different)
	if simLow > 0.3 {
		t.Errorf("Expected low similarity for completely different texts, got %f", simLow)
	}

	if IsSyndicated(orig, different, 0.70) {
		t.Errorf("Expected IsSyndicated to be false for distinct texts")
	}
}

func TestVerifyEvidenceQuote(t *testing.T) {
	rawText := `Oferta de feina: Auxiliar de magatzem.
La persona seleccionada s'encarregarà de la planificació dels torns dels 15 operaris mitjançant fulls de càlcul Excel i comunicació diària per WhatsApp.
Requisits: Domini d'Excel i carnet de conduir.`

	tests := []struct {
		name     string
		quote    string
		expected bool
	}{
		{
			name:     "Exact match in text",
			quote:    "planificació dels torns dels 15 operaris mitjançant fulls de càlcul Excel i comunicació diària per WhatsApp",
			expected: true,
		},
		{
			name:     "Case insensitive & whitespace match",
			quote:    "PLANIFICACIÓ DELS TORNS DELS 15 OPERARIS  MITJANÇANT   FULLS DE CÀLCUL EXCEL",
			expected: true,
		},
		{
			name:     "Hallucinated quote not in text",
			quote:    "Utilitzem el software SAP i Factusol per enviar factures electròniques",
			expected: false,
		},
		{
			name:     "Quote exceeding 200 chars limit",
			quote:    strings.Repeat("a", 201),
			expected: false,
		},
		{
			name:     "Empty quote",
			quote:    "",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := VerifyEvidenceQuote(rawText, tc.quote)
			if got != tc.expected {
				t.Errorf("VerifyEvidenceQuote() = %v, expected %v", got, tc.expected)
			}
		})
	}
}
