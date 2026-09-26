package signal

import (
	"context"
	"database/sql"
	"fmt"
	"salesanalizer/backend/internal/db"
	"strings"
	"time"
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
