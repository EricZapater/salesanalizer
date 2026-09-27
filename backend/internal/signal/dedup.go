package signal

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var trackingParams = map[string]bool{
	"utm_source":    true,
	"utm_medium":    true,
	"utm_campaign":  true,
	"utm_term":      true,
	"utm_content":   true,
	"utm_id":        true,
	"utm_name":      true,
	"ref":           true,
	"ref_src":       true,
	"ref_url":       true,
	"referrer":      true,
	"fbclid":        true,
	"gclid":         true,
	"gbraid":        true,
	"wbraid":        true,
	"msclkid":       true,
	"twclid":        true,
	"igshid":        true,
	"mc_eid":        true,
	"_ga":           true,
	"_gl":           true,
	"_hsenc":        true,
	"_hsmi":         true,
	"__hssc":        true,
	"__hstc":        true,
	"hsctatracking": true,
	"yclid":         true,
	"spjobid":       true,
	"trk":           true,
	"trkinfo":       true,
}

var multiSpaceRegex = regexp.MustCompile(`\s+`)
var nonAlphaNumRegex = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

// NormalizeURL neteja paràmetres de seguiment (UTM, ref, fbclid), normalitza protocol, host i elimina barres finals
func NormalizeURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("URL buida")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("URL invàlida: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("URL sense scheme o host")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("protocol no suportat: %s", scheme)
	}

	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}

	hostWithPort := host
	if port != "" {
		hostWithPort = host + ":" + port
	}

	path := parsed.Path
	if path == "" {
		path = "/"
	} else if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimRight(path, "/")
	}

	// Filtrar query params
	queryParams := parsed.Query()
	filteredKeys := make([]string, 0, len(queryParams))

	for key := range queryParams {
		lowerKey := strings.ToLower(key)
		if !trackingParams[lowerKey] {
			filteredKeys = append(filteredKeys, key)
		}
	}
	sort.Strings(filteredKeys)

	var queryStr string
	if len(filteredKeys) > 0 {
		var pairs []string
		for _, k := range filteredKeys {
			vals := queryParams[k]
			sort.Strings(vals)
			for _, v := range vals {
				pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(v))
			}
		}
		queryStr = "?" + strings.Join(pairs, "&")
	}

	return fmt.Sprintf("%s://%s%s%s", scheme, hostWithPort, path, queryStr), nil
}

// CanonicalizeText normalitza el text traient espais redundants, salts de línia i minúscules
func CanonicalizeText(text string) string {
	lower := strings.ToLower(text)
	normalized := multiSpaceRegex.ReplaceAllString(lower, " ")
	return strings.TrimSpace(normalized)
}

// CalculateContentHash genera un hash SHA-256 a partir del text canònic
func CalculateContentHash(text string) string {
	canonical := CanonicalizeText(text)
	hash := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", hash)
}

// TokenizeText separa el text en paraules netes en minúscules
func TokenizeText(text string) []string {
	cleaned := nonAlphaNumRegex.ReplaceAllString(strings.ToLower(text), " ")
	fields := strings.Fields(cleaned)
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		if len([]rune(f)) >= 2 {
			tokens = append(tokens, f)
		}
	}
	return tokens
}

// GenerateShingles genera n-grames de paraules (shingles)
func GenerateShingles(tokens []string, n int) map[string]struct{} {
	shingles := make(map[string]struct{})
	if len(tokens) == 0 {
		return shingles
	}
	if len(tokens) < n {
		shingles[strings.Join(tokens, " ")] = struct{}{}
		return shingles
	}
	for i := 0; i <= len(tokens)-n; i++ {
		shingle := strings.Join(tokens[i:i+n], " ")
		shingles[shingle] = struct{}{}
	}
	return shingles
}

// CalculateTextSimilarity calcula la similitud de Jaccard basada en 2-grames de paraules (0.0 a 1.0)
func CalculateTextSimilarity(textA, textB string) float64 {
	tokensA := TokenizeText(textA)
	tokensB := TokenizeText(textB)

	if len(tokensA) == 0 && len(tokensB) == 0 {
		return 1.0
	}
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0.0
	}

	shingleSize := 2
	if len(tokensA) < 2 || len(tokensB) < 2 {
		shingleSize = 1
	}

	shinglesA := GenerateShingles(tokensA, shingleSize)
	shinglesB := GenerateShingles(tokensB, shingleSize)

	intersection := 0
	for s := range shinglesA {
		if _, exists := shinglesB[s]; exists {
			intersection++
		}
	}

	union := len(shinglesA) + len(shinglesB) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// CalculateWordOverlapSimilarity calcula la similitud de Jaccard basada en paraules úniques (>2 caràcters)
func CalculateWordOverlapSimilarity(textA, textB string) float64 {
	tokensA := TokenizeText(textA)
	tokensB := TokenizeText(textB)

	if len(tokensA) == 0 && len(tokensB) == 0 {
		return 1.0
	}
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0.0
	}

	setA := make(map[string]struct{}, len(tokensA))
	for _, t := range tokensA {
		setA[t] = struct{}{}
	}

	setB := make(map[string]struct{}, len(tokensB))
	for _, t := range tokensB {
		setB[t] = struct{}{}
	}

	intersection := 0
	for t := range setA {
		if _, exists := setB[t]; exists {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// IsSyndicated determina si dos textos són sindicació/duplicat (>90% similitud per defecte)
func IsSyndicated(textA, textB string, threshold float64) bool {
	if threshold <= 0 {
		threshold = 0.90
	}
	return CalculateTextSimilarity(textA, textB) >= threshold
}

// VerifyEvidenceQuote comprova que la cita textual extreta pel LLM sigui literalment al text font (<200 caràcters)
func VerifyEvidenceQuote(rawContent, quote string) bool {
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return false
	}
	// Màxim 200 caràcters
	if len([]rune(quote)) > 200 {
		return false
	}

	canonicalRaw := CanonicalizeText(rawContent)
	canonicalQuote := CanonicalizeText(quote)

	// Comprovar substring directe
	if strings.Contains(canonicalRaw, canonicalQuote) {
		return true
	}

	// Comprovar eliminant puntuació si hi ha petites diferències de comes/punts
	cleanRaw := cleanPunctuation(canonicalRaw)
	cleanQuote := cleanPunctuation(canonicalQuote)
	return strings.Contains(cleanRaw, cleanQuote)
}

func cleanPunctuation(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return ' '
		}
		return r
	}, s)
}
