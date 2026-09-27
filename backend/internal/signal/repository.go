package signal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"salesanalizer/backend/internal/db"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) ExistsByURL(ctx context.Context, targetURL string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM job_offers WHERE url = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, targetURL).Scan(&exists)
	return exists, err
}

func (r *Repository) SaveRawSignal(ctx context.Context, sig *RawSignal) (string, error) {
	query := `
		INSERT INTO job_offers (source, signal_type, title, company, url, raw_text, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'analyzed', NOW(), NOW())
		ON CONFLICT (url) DO UPDATE SET raw_text = EXCLUDED.raw_text, updated_at = NOW()
		RETURNING id
	`
	var id string
	err := r.db.QueryRowContext(ctx, query, sig.Source, sig.SignalType, sig.Title, sig.CompanyName, sig.SourceURL, sig.RawText).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("error desant senyal: %w", err)
	}
	return id, nil
}

func (r *Repository) SaveOpportunityAnalysis(ctx context.Context, a *OpportunityAnalysis) error {
	query := `
		INSERT INTO opportunity_analyses (job_offer_id, ineficiencia_manual, proposta_micro_saas, viabilitat_plg_score, decisor_compra, ganxo_venda, raw_llm_response, analyzed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (job_offer_id) DO UPDATE SET 
			ineficiencia_manual = EXCLUDED.ineficiencia_manual,
			proposta_micro_saas = EXCLUDED.proposta_micro_saas,
			viabilitat_plg_score = EXCLUDED.viabilitat_plg_score,
			decisor_compra = EXCLUDED.decisor_compra,
			ganxo_venda = EXCLUDED.ganxo_venda,
			raw_llm_response = EXCLUDED.raw_llm_response,
			analyzed_at = NOW()
		RETURNING id, analyzed_at
	`
	return r.db.QueryRowContext(ctx, query, a.SignalID, a.IneficienciaManual, a.PropostaMicroSaas, a.ViabilitatPLGScore, a.DecisorCompra, a.GanxoVenda, a.RawLLMResponse).
		Scan(&a.ID, &a.AnalyzedAt)
}

func (r *Repository) ListSignals(ctx context.Context, status string, minScore int, limit, offset int) ([]Signal, int, error) {
	if limit <= 0 {
		limit = 50
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if status == "active" {
		whereClause += " AND jo.status NOT IN ('discarded', 'descartada')"
	} else if status == "pendent" || status == "pending" {
		whereClause += " AND jo.status IN ('pendent', 'analyzed', 'pending', 'active')"
	} else if status != "" && status != "all" {
		whereClause += fmt.Sprintf(" AND jo.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if minScore > 0 {
		whereClause += fmt.Sprintf(" AND oa.viabilitat_plg_score >= $%d", argIdx)
		args = append(args, minScore)
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT jo.id)
		FROM job_offers jo
		LEFT JOIN opportunity_analyses oa ON jo.id = oa.job_offer_id
		%s
	`, whereClause)

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting signals: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			jo.id, jo.source, COALESCE(jo.signal_type, 'oferta_feina'), jo.external_id, jo.title, jo.company, jo.location, jo.url, jo.status, jo.created_at, jo.updated_at,
			oa.id, oa.ineficiencia_manual, oa.proposta_micro_saas, oa.viabilitat_plg_score, oa.decisor_compra, oa.ganxo_venda, oa.analyzed_at
		FROM job_offers jo
		LEFT JOIN opportunity_analyses oa ON jo.id = oa.job_offer_id
		%s
		ORDER BY COALESCE(oa.viabilitat_plg_score, 0) DESC, jo.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying signals: %w", err)
	}
	defer rows.Close()

	var signals []Signal
	for rows.Next() {
		var s Signal
		var extID, company, location sql.NullString
		var oaID, oaInef, oaProp, oaDecisor, oaGanxo sql.NullString
		var oaScore sql.NullInt32
		var oaAnalyzedAt sql.NullTime

		err := rows.Scan(
			&s.ID, &s.Source, &s.SignalType, &extID, &s.Title, &company, &location, &s.URL, &s.Status, &s.CreatedAt, &s.UpdatedAt,
			&oaID, &oaInef, &oaProp, &oaScore, &oaDecisor, &oaGanxo, &oaAnalyzedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning signal row: %w", err)
		}

		if extID.Valid {
			s.ExternalID = &extID.String
		}
		if company.Valid {
			s.Company = &company.String
		}
		if location.Valid {
			s.Location = &location.String
		}

		if oaID.Valid {
			s.Analysis = &OpportunityAnalysis{
				ID:                 oaID.String,
				SignalID:           s.ID,
				IneficienciaManual: oaInef.String,
				PropostaMicroSaas:  oaProp.String,
				ViabilitatPLGScore: int(oaScore.Int32),
				DecisorCompra:      oaDecisor.String,
				GanxoVenda:         oaGanxo.String,
				AnalyzedAt:         oaAnalyzedAt.Time,
			}
		}

		signals = append(signals, s)
	}

	if signals == nil {
		signals = []Signal{}
	}

	return signals, total, nil
}

func (r *Repository) GetSignalByID(ctx context.Context, id string) (*Signal, error) {
	query := `
		SELECT 
			jo.id, jo.source, COALESCE(jo.signal_type, 'oferta_feina'), jo.external_id, jo.title, jo.company, jo.location, jo.url, jo.raw_text, jo.status, jo.created_at, jo.updated_at,
			oa.id, oa.ineficiencia_manual, oa.proposta_micro_saas, oa.viabilitat_plg_score, oa.decisor_compra, oa.ganxo_venda, oa.analyzed_at
		FROM job_offers jo
		LEFT JOIN opportunity_analyses oa ON jo.id = oa.job_offer_id
		WHERE jo.id = $1
	`
	var s Signal
	var extID, company, location sql.NullString
	var oaID, oaInef, oaProp, oaDecisor, oaGanxo sql.NullString
	var oaScore sql.NullInt32
	var oaAnalyzedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Source, &s.SignalType, &extID, &s.Title, &company, &location, &s.URL, &s.RawText, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		&oaID, &oaInef, &oaProp, &oaScore, &oaDecisor, &oaGanxo, &oaAnalyzedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error querying signal by id: %w", err)
	}

	if extID.Valid {
		s.ExternalID = &extID.String
	}
	if company.Valid {
		s.Company = &company.String
	}
	if location.Valid {
		s.Location = &location.String
	}

	if oaID.Valid {
		s.Analysis = &OpportunityAnalysis{
			ID:                 oaID.String,
			SignalID:           s.ID,
			IneficienciaManual: oaInef.String,
			PropostaMicroSaas:  oaProp.String,
			ViabilitatPLGScore: int(oaScore.Int32),
			DecisorCompra:      oaDecisor.String,
			GanxoVenda:         oaGanxo.String,
			AnalyzedAt:         oaAnalyzedAt.Time,
		}
	}

	return &s, nil
}

func (r *Repository) UpdateSignalStatus(ctx context.Context, id, status string) error {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "discarded" {
		normalized = "descartada"
	} else if normalized == "pending" || normalized == "analyzed" {
		normalized = "pendent"
	}

	query := `UPDATE job_offers SET status = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, normalized, id)
	if err != nil {
		return fmt.Errorf("error actualitzant estat del senyal: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) DiscardSignal(ctx context.Context, id string) error {
	return r.UpdateSignalStatus(ctx, id, "descartada")
}

func (r *Repository) CountSignalsToday(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM job_offers WHERE created_at >= CURRENT_DATE`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *Repository) RecordScraperRun(ctx context.Context, source, status string, itemsFound int, errMsg *string) error {
	query := `
		INSERT INTO scraper_runs (source, status, items_found, error_message, run_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := r.db.ExecContext(ctx, query, source, status, itemsFound, errMsg)
	return err
}

func (r *Repository) GetLatestScraperRuns(ctx context.Context) ([]ScraperRun, error) {
	query := `
		SELECT DISTINCT ON (source) id, source, status, items_found, error_message, run_at
		FROM scraper_runs
		ORDER BY source, run_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying latest scraper runs: %w", err)
	}
	defer rows.Close()

	var runs []ScraperRun
	for rows.Next() {
		var run ScraperRun
		var errMsg sql.NullString
		if err := rows.Scan(&run.ID, &run.Source, &run.Status, &run.ItemsFound, &errMsg, &run.RunAt); err != nil {
			return nil, err
		}
		if errMsg.Valid {
			run.ErrorMessage = &errMsg.String
		}
		runs = append(runs, run)
	}

	if len(runs) == 0 {
		now := time.Now()
		runs = []ScraperRun{
			{ID: 1, Source: "Feina Activa (SOC)", Status: "ok", ItemsFound: 0, RunAt: now},
			{ID: 2, Source: "Reddit SmallBusiness (RSS)", Status: "ok", ItemsFound: 0, RunAt: now},
		}
	}

	return runs, nil
}

func (r *Repository) GetScraperSettings(ctx context.Context) (bool, error) {
	query := `SELECT deep_fetch_enabled FROM signal_settings WHERE id = 1`
	var deepFetch bool
	err := r.db.QueryRowContext(ctx, query).Scan(&deepFetch)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = r.UpdateScraperSettings(ctx, false)
			return false, nil
		}
		return false, fmt.Errorf("error obtenint signal_settings: %w", err)
	}
	return deepFetch, nil
}

func (r *Repository) UpdateScraperSettings(ctx context.Context, deepFetch bool) error {
	query := `
		INSERT INTO signal_settings (id, deep_fetch_enabled, updated_at)
		VALUES (1, $1, NOW())
		ON CONFLICT (id) DO UPDATE SET deep_fetch_enabled = EXCLUDED.deep_fetch_enabled, updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, deepFetch)
	if err != nil {
		return fmt.Errorf("error actualitzant signal_settings: %w", err)
	}
	return nil
}

// FindEvidenceByHashOrURL cerca si ja existeix una evidència amb la mateixa URL normalitzada o el mateix hash de contingut
func (r *Repository) FindEvidenceByHashOrURL(ctx context.Context, normalizedURL, contentHash string) (*Evidence, error) {
	query := `
		SELECT id, raw_content, normalized_url, content_hash, source, source_id, author_or_company,
		       extracted_process, task_description, frequency, manuality_score, tools_mentioned,
		       sector, evidence_type, source_evidence_quote, evidence_confidence, is_duplicate_of, published_at, created_at
		FROM evidences
		WHERE normalized_url = $1 OR content_hash = $2
		LIMIT 1
	`
	var ev Evidence
	var srcID, authComp, extProc, taskDesc, sector, evType, srcQuote, isDup *string
	var pubAt *time.Time
	var tools pq.StringArray

	err := r.db.QueryRowContext(ctx, query, normalizedURL, contentHash).Scan(
		&ev.ID, &ev.RawContent, &ev.NormalizedURL, &ev.ContentHash, &ev.Source,
		&srcID, &authComp, &extProc, &taskDesc, &ev.Frequency, &ev.ManualityScore,
		&tools, &sector, &evType, &srcQuote, &ev.EvidenceConfidence,
		&isDup, &pubAt, &ev.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error cercant duplicat d'evidència: %w", err)
	}

	ev.SourceID = srcID
	ev.AuthorOrCompany = authComp
	ev.ExtractedProcess = extProc
	ev.TaskDescription = taskDesc
	ev.ToolsMentioned = []string(tools)
	ev.Sector = sector
	ev.EvidenceType = evType
	ev.SourceEvidenceQuote = srcQuote
	ev.IsDuplicateOf = isDup
	ev.PublishedAt = pubAt

	return &ev, nil
}

// SaveEvidence desa una evidència amb fets i quotes literals a la base de dades
func (r *Repository) SaveEvidence(ctx context.Context, ev *Evidence) error {
	query := `
		INSERT INTO evidences (
			raw_content, normalized_url, content_hash, source, source_id,
			author_or_company, extracted_process, task_description, frequency,
			manuality_score, tools_mentioned, sector, evidence_type,
			source_evidence_quote, evidence_confidence, is_duplicate_of, published_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW()
		)
		RETURNING id, created_at
	`
	var tools pq.StringArray = ev.ToolsMentioned
	return r.db.QueryRowContext(
		ctx, query,
		ev.RawContent, ev.NormalizedURL, ev.ContentHash, ev.Source, ev.SourceID,
		ev.AuthorOrCompany, ev.ExtractedProcess, ev.TaskDescription, ev.Frequency,
		ev.ManualityScore, tools, ev.Sector, ev.EvidenceType,
		ev.SourceEvidenceQuote, ev.EvidenceConfidence, ev.IsDuplicateOf, ev.PublishedAt,
	).Scan(&ev.ID, &ev.CreatedAt)
}

// ListEvidences llista les evidències ordenades cronològicament
func (r *Repository) ListEvidences(ctx context.Context, limit, offset int) ([]Evidence, int, error) {
	if limit <= 0 {
		limit = 50
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM evidences`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error comptant evidències: %w", err)
	}

	query := `
		SELECT id, raw_content, normalized_url, content_hash, source, source_id, author_or_company,
		       extracted_process, task_description, frequency, manuality_score, tools_mentioned,
		       sector, evidence_type, source_evidence_quote, evidence_confidence, is_duplicate_of, published_at, created_at
		FROM evidences
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error llistant evidències: %w", err)
	}
	defer rows.Close()

	var evidences []Evidence
	for rows.Next() {
		var ev Evidence
		var srcID, authComp, extProc, taskDesc, sector, evType, srcQuote, isDup *string
		var pubAt *time.Time
		var tools pq.StringArray

		if err := rows.Scan(
			&ev.ID, &ev.RawContent, &ev.NormalizedURL, &ev.ContentHash, &ev.Source,
			&srcID, &authComp, &extProc, &taskDesc, &ev.Frequency, &ev.ManualityScore,
			&tools, &sector, &evType, &srcQuote, &ev.EvidenceConfidence,
			&isDup, &pubAt, &ev.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("error llegint fila d'evidència: %w", err)
		}

		ev.SourceID = srcID
		ev.AuthorOrCompany = authComp
		ev.ExtractedProcess = extProc
		ev.TaskDescription = taskDesc
		ev.ToolsMentioned = []string(tools)
		ev.Sector = sector
		ev.EvidenceType = evType
		ev.SourceEvidenceQuote = srcQuote
		ev.IsDuplicateOf = isDup
		ev.PublishedAt = pubAt

		evidences = append(evidences, ev)
	}

	return evidences, total, nil
}

// CountEvidencesToday compta quantes evidències s'han creat avui
func (r *Repository) CountEvidencesToday(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM evidences WHERE created_at >= CURRENT_DATE`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// ListCanonicalProcesses retorna tots els processos canònics
func (r *Repository) ListCanonicalProcesses(ctx context.Context) ([]ProcessNormalized, error) {
	query := `
		SELECT id, canonical_name, category, typical_tools, description, created_at
		FROM processes_normalized
		ORDER BY canonical_name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error llistant processos canònics: %w", err)
	}
	defer rows.Close()

	var list []ProcessNormalized
	for rows.Next() {
		var p ProcessNormalized
		var tools pq.StringArray
		var desc *string
		if err := rows.Scan(&p.ID, &p.CanonicalName, &p.Category, &tools, &desc, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("error llegint fila de procés canònic: %w", err)
		}
		p.TypicalTools = []string(tools)
		p.Description = desc
		list = append(list, p)
	}
	return list, nil
}

// FindOrCreateCanonicalProcess cerca o crea un procés canònic
func (r *Repository) FindOrCreateCanonicalProcess(ctx context.Context, name, category, desc string, typicalTools []string) (*ProcessNormalized, error) {
	query := `
		INSERT INTO processes_normalized (canonical_name, category, typical_tools, description, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (canonical_name) DO UPDATE SET 
			category = EXCLUDED.category,
			typical_tools = EXCLUDED.typical_tools
		RETURNING id, canonical_name, category, typical_tools, description, created_at
	`
	var p ProcessNormalized
	var tools pq.StringArray
	var descPtr *string
	var toolsArg pq.StringArray = typicalTools

	err := r.db.QueryRowContext(ctx, query, name, category, toolsArg, desc).Scan(
		&p.ID, &p.CanonicalName, &p.Category, &tools, &descPtr, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error desant procés canònic: %w", err)
	}
	p.TypicalTools = []string(tools)
	p.Description = descPtr
	return &p, nil
}

// FindClustersByProcess cerca tots els clústers actius associats a un procés
func (r *Repository) FindClustersByProcess(ctx context.Context, processID string) ([]PainCluster, error) {
	query := `
		SELECT pc.id, pc.process_id, pn.canonical_name, pn.category, pc.title, pc.summary, pc.status,
		       pc.evidence_count, pc.company_count, pc.source_count, pc.sector_breadth,
		       pc.last_evidence_at, pc.created_at, pc.updated_at
		FROM pain_clusters pc
		JOIN processes_normalized pn ON pc.process_id = pn.id
		WHERE pc.process_id = $1 AND pc.status != 'discarded'
		ORDER BY pc.evidence_count DESC
	`
	rows, err := r.db.QueryContext(ctx, query, processID)
	if err != nil {
		return nil, fmt.Errorf("error cercant clústers per procés: %w", err)
	}
	defer rows.Close()

	var list []PainCluster
	for rows.Next() {
		var c PainCluster
		var lastEv *time.Time
		if err := rows.Scan(
			&c.ID, &c.ProcessID, &c.ProcessName, &c.Category, &c.Title, &c.Summary, &c.Status,
			&c.EvidenceCount, &c.CompanyCount, &c.SourceCount, &c.SectorBreadth,
			&lastEv, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error llegint fila de clúster: %w", err)
		}
		c.LastEvidenceAt = lastEv
		list = append(list, c)
	}
	return list, nil
}

// SavePainCluster desa o actualitza un clúster de dolor
func (r *Repository) SavePainCluster(ctx context.Context, c *PainCluster) error {
	query := `
		INSERT INTO pain_clusters (
			process_id, title, summary, status, evidence_count,
			company_count, source_count, sector_breadth, last_evidence_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(
		ctx, query,
		c.ProcessID, c.Title, c.Summary, c.Status, c.EvidenceCount,
		c.CompanyCount, c.SourceCount, c.SectorBreadth, c.LastEvidenceAt,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// LinkEvidenceToCluster enllaça una evidència a un clúster amb un score de rellevància
func (r *Repository) LinkEvidenceToCluster(ctx context.Context, clusterID, evidenceID string, relevance float64) error {
	query := `
		INSERT INTO pain_cluster_evidences (cluster_id, evidence_id, relevance_score, added_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (cluster_id, evidence_id) DO UPDATE SET relevance_score = EXCLUDED.relevance_score
	`
	_, err := r.db.ExecContext(ctx, query, clusterID, evidenceID, relevance)
	if err != nil {
		return fmt.Errorf("error enllaçant evidència a clúster: %w", err)
	}
	return nil
}

// RecalculateClusterMetrics recalcula les mètriques d'evidències, empreses i fonts d'un clúster
func (r *Repository) RecalculateClusterMetrics(ctx context.Context, clusterID string) error {
	calcQuery := `
		SELECT 
			COALESCE(COUNT(DISTINCT e.id) FILTER (WHERE e.is_duplicate_of IS NULL), 0) as ev_count,
			COALESCE(COUNT(DISTINCT e.author_or_company) FILTER (WHERE e.author_or_company IS NOT NULL AND e.author_or_company != ''), 0) as comp_count,
			COALESCE(COUNT(DISTINCT e.source), 0) as src_count,
			COALESCE(COUNT(DISTINCT e.sector) FILTER (WHERE e.sector IS NOT NULL AND e.sector != ''), 0) as sec_count,
			MAX(e.created_at) as last_ev_at
		FROM pain_cluster_evidences pce
		JOIN evidences e ON pce.evidence_id = e.id
		WHERE pce.cluster_id = $1
	`
	var evCount, compCount, srcCount, secCount int
	var lastEvAt *time.Time

	err := r.db.QueryRowContext(ctx, calcQuery, clusterID).Scan(&evCount, &compCount, &srcCount, &secCount, &lastEvAt)
	if err != nil {
		return fmt.Errorf("error calculant mètriques de clúster: %w", err)
	}

	updateQuery := `
		UPDATE pain_clusters
		SET evidence_count = $1,
		    company_count = $2,
		    source_count = $3,
		    sector_breadth = $4,
		    last_evidence_at = $5,
		    status = CASE 
		        WHEN status IN ('validated', 'discarded') THEN status
		        WHEN $1 >= 3 AND $3 >= 2 THEN 'consolidated'
		        ELSE 'emerging'
		    END,
		    updated_at = NOW()
		WHERE id = $6
	`
	_, err = r.db.ExecContext(ctx, updateQuery, evCount, compCount, srcCount, secCount, lastEvAt, clusterID)
	if err != nil {
		return fmt.Errorf("error actualitzant mètriques de clúster: %w", err)
	}
	return nil
}

// ListPainClusters llista els clústers ordenats per evidències i data
func (r *Repository) ListPainClusters(ctx context.Context, status string, minEvidence int, limit, offset int) ([]PainCluster, int, error) {
	if limit <= 0 {
		limit = 50
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if status != "" && status != "all" {
		whereClause += fmt.Sprintf(" AND pc.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if minEvidence > 0 {
		whereClause += fmt.Sprintf(" AND pc.evidence_count >= $%d", argIdx)
		args = append(args, minEvidence)
		argIdx++
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM pain_clusters pc %s`, whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error comptant clústers: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT pc.id, pc.process_id, pn.canonical_name, pn.category, pc.title, pc.summary, pc.status,
		       pc.evidence_count, pc.company_count, pc.source_count, pc.sector_breadth,
		       pc.last_evidence_at, pc.created_at, pc.updated_at
		FROM pain_clusters pc
		JOIN processes_normalized pn ON pc.process_id = pn.id
		%s
		ORDER BY pc.evidence_count DESC, pc.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error llistant clústers: %w", err)
	}
	defer rows.Close()

	var clusters []PainCluster
	for rows.Next() {
		var c PainCluster
		var lastEv *time.Time
		if err := rows.Scan(
			&c.ID, &c.ProcessID, &c.ProcessName, &c.Category, &c.Title, &c.Summary, &c.Status,
			&c.EvidenceCount, &c.CompanyCount, &c.SourceCount, &c.SectorBreadth,
			&lastEv, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("error llegint fila de clúster: %w", err)
		}
		c.LastEvidenceAt = lastEv
		clusters = append(clusters, c)
	}

	return clusters, total, nil
}

// GetClusterByID retorna un clúster amb totes les seves evidències associades
func (r *Repository) GetClusterByID(ctx context.Context, id string) (*PainClusterWithDetails, error) {
	query := `
		SELECT pc.id, pc.process_id, pn.canonical_name, pn.category, pc.title, pc.summary, pc.status,
		       pc.evidence_count, pc.company_count, pc.source_count, pc.sector_breadth,
		       pc.last_evidence_at, pc.created_at, pc.updated_at
		FROM pain_clusters pc
		JOIN processes_normalized pn ON pc.process_id = pn.id
		WHERE pc.id = $1
	`
	var c PainClusterWithDetails
	var lastEv *time.Time
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.ProcessID, &c.ProcessName, &c.Category, &c.Title, &c.Summary, &c.Status,
		&c.EvidenceCount, &c.CompanyCount, &c.SourceCount, &c.SectorBreadth,
		&lastEv, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error cercant clúster per ID: %w", err)
	}
	c.LastEvidenceAt = lastEv

	evQuery := `
		SELECT e.id, e.raw_content, e.normalized_url, e.content_hash, e.source, e.source_id, e.author_or_company,
		       e.extracted_process, e.task_description, e.frequency, e.manuality_score, e.tools_mentioned,
		       e.sector, e.evidence_type, e.source_evidence_quote, e.evidence_confidence, e.is_duplicate_of, e.published_at, e.created_at
		FROM pain_cluster_evidences pce
		JOIN evidences e ON pce.evidence_id = e.id
		WHERE pce.cluster_id = $1
		ORDER BY pce.relevance_score DESC, e.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, evQuery, id)
	if err != nil {
		return nil, fmt.Errorf("error obtenint evidències del clúster: %w", err)
	}
	defer rows.Close()

	c.Evidences = make([]Evidence, 0)
	for rows.Next() {
		var ev Evidence
		var srcID, authComp, extProc, taskDesc, sector, evType, srcQuote, isDup *string
		var pubAt *time.Time
		var tools pq.StringArray

		if err := rows.Scan(
			&ev.ID, &ev.RawContent, &ev.NormalizedURL, &ev.ContentHash, &ev.Source,
			&srcID, &authComp, &extProc, &taskDesc, &ev.Frequency, &ev.ManualityScore,
			&tools, &sector, &evType, &srcQuote, &ev.EvidenceConfidence,
			&isDup, &pubAt, &ev.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("error llegint fila d'evidència: %w", err)
		}
		ev.SourceID = srcID
		ev.AuthorOrCompany = authComp
		ev.ExtractedProcess = extProc
		ev.TaskDescription = taskDesc
		ev.ToolsMentioned = []string(tools)
		ev.Sector = sector
		ev.EvidenceType = evType
		ev.SourceEvidenceQuote = srcQuote
		ev.IsDuplicateOf = isDup
		ev.PublishedAt = pubAt

		c.Evidences = append(c.Evidences, ev)
	}

	return &c, nil
}

// SaveOpportunity desa o actualitza una oportunitat Micro-SaaS
func (r *Repository) SaveOpportunity(ctx context.Context, opp *Opportunity) error {
	scoresJSON, err := json.Marshal(opp.Scores)
	if err != nil {
		return fmt.Errorf("error serialitzant scores a JSON: %w", err)
	}

	query := `
		INSERT INTO opportunities (
			cluster_id, title, target_user, buyer_persona, core_workflow,
			value_prop, pricing_model, outreach_hook, scores, global_score,
			viability_tier, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
		)
		ON CONFLICT (cluster_id) DO UPDATE SET
			title = EXCLUDED.title,
			target_user = EXCLUDED.target_user,
			buyer_persona = EXCLUDED.buyer_persona,
			core_workflow = EXCLUDED.core_workflow,
			value_prop = EXCLUDED.value_prop,
			pricing_model = EXCLUDED.pricing_model,
			outreach_hook = EXCLUDED.outreach_hook,
			scores = EXCLUDED.scores,
			global_score = EXCLUDED.global_score,
			viability_tier = EXCLUDED.viability_tier,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(
		ctx, query,
		opp.ClusterID, opp.Title, opp.TargetUser, opp.BuyerPersona, opp.CoreWorkflow,
		opp.ValueProp, opp.PricingModel, opp.OutreachHook, scoresJSON, opp.GlobalScore,
		opp.ViabilityTier,
	).Scan(&opp.ID, &opp.CreatedAt, &opp.UpdatedAt)
}

// GetOpportunityByClusterID retorna l'oportunitat associada a un clúster
func (r *Repository) GetOpportunityByClusterID(ctx context.Context, clusterID string) (*Opportunity, error) {
	query := `
		SELECT o.id, o.cluster_id, pc.title, pn.canonical_name, pn.category,
		       o.title, o.target_user, o.buyer_persona, o.core_workflow,
		       o.value_prop, o.pricing_model, o.outreach_hook, o.scores,
		       o.global_score, o.viability_tier, o.created_at, o.updated_at
		FROM opportunities o
		JOIN pain_clusters pc ON o.cluster_id = pc.id
		JOIN processes_normalized pn ON pc.process_id = pn.id
		WHERE o.cluster_id = $1
	`
	var opp Opportunity
	var scoresRaw []byte
	err := r.db.QueryRowContext(ctx, query, clusterID).Scan(
		&opp.ID, &opp.ClusterID, &opp.ClusterTitle, &opp.ProcessName, &opp.Category,
		&opp.Title, &opp.TargetUser, &opp.BuyerPersona, &opp.CoreWorkflow,
		&opp.ValueProp, &opp.PricingModel, &opp.OutreachHook, &scoresRaw,
		&opp.GlobalScore, &opp.ViabilityTier, &opp.CreatedAt, &opp.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error cercant oportunitat per clúster: %w", err)
	}

	if err := json.Unmarshal(scoresRaw, &opp.Scores); err != nil {
		return nil, fmt.Errorf("error deserialitzant scores d'oportunitat: %w", err)
	}

	return &opp, nil
}

// GetOpportunityByID retorna una oportunitat per la seva ID única
func (r *Repository) GetOpportunityByID(ctx context.Context, id string) (*Opportunity, error) {
	query := `
		SELECT o.id, o.cluster_id, pc.title, pn.canonical_name, pn.category,
		       o.title, o.target_user, o.buyer_persona, o.core_workflow,
		       o.value_prop, o.pricing_model, o.outreach_hook, o.scores,
		       o.global_score, o.viability_tier, o.created_at, o.updated_at
		FROM opportunities o
		JOIN pain_clusters pc ON o.cluster_id = pc.id
		JOIN processes_normalized pn ON pc.process_id = pn.id
		WHERE o.id = $1
	`
	var opp Opportunity
	var scoresRaw []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&opp.ID, &opp.ClusterID, &opp.ClusterTitle, &opp.ProcessName, &opp.Category,
		&opp.Title, &opp.TargetUser, &opp.BuyerPersona, &opp.CoreWorkflow,
		&opp.ValueProp, &opp.PricingModel, &opp.OutreachHook, &scoresRaw,
		&opp.GlobalScore, &opp.ViabilityTier, &opp.CreatedAt, &opp.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error cercant oportunitat per ID: %w", err)
	}

	if err := json.Unmarshal(scoresRaw, &opp.Scores); err != nil {
		return nil, fmt.Errorf("error deserialitzant scores d'oportunitat: %w", err)
	}

	return &opp, nil
}

// ListOpportunities llista oportunitats ordenades per global_score descendent
func (r *Repository) ListOpportunities(ctx context.Context, minScore float64, tier string, limit, offset int) ([]Opportunity, int, error) {
	if limit <= 0 {
		limit = 50
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if minScore > 0 {
		whereClause += fmt.Sprintf(" AND o.global_score >= $%d", argIdx)
		args = append(args, minScore)
		argIdx++
	}

	if tier != "" && tier != "all" {
		whereClause += fmt.Sprintf(" AND o.viability_tier = $%d", argIdx)
		args = append(args, tier)
		argIdx++
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM opportunities o %s`, whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error comptant oportunitats: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT o.id, o.cluster_id, pc.title, pn.canonical_name, pn.category,
		       o.title, o.target_user, o.buyer_persona, o.core_workflow,
		       o.value_prop, o.pricing_model, o.outreach_hook, o.scores,
		       o.global_score, o.viability_tier, o.created_at, o.updated_at
		FROM opportunities o
		JOIN pain_clusters pc ON o.cluster_id = pc.id
		JOIN processes_normalized pn ON pc.process_id = pn.id
		%s
		ORDER BY o.global_score DESC, o.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error llistant oportunitats: %w", err)
	}
	defer rows.Close()

	var list []Opportunity
	for rows.Next() {
		var opp Opportunity
		var scoresRaw []byte
		if err := rows.Scan(
			&opp.ID, &opp.ClusterID, &opp.ClusterTitle, &opp.ProcessName, &opp.Category,
			&opp.Title, &opp.TargetUser, &opp.BuyerPersona, &opp.CoreWorkflow,
			&opp.ValueProp, &opp.PricingModel, &opp.OutreachHook, &scoresRaw,
			&opp.GlobalScore, &opp.ViabilityTier, &opp.CreatedAt, &opp.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("error llegint fila d'oportunitat: %w", err)
		}

		_ = json.Unmarshal(scoresRaw, &opp.Scores)
		list = append(list, opp)
	}

	return list, total, nil
}




