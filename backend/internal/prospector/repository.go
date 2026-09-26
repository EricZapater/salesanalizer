package prospector

import (
	"context"
	"database/sql"
	"fmt"
	"salesanalizer/backend/internal/db"
	"time"
)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) ListOffers(ctx context.Context, status string, minScore int, limit, offset int) ([]JobOffer, int, error) {
	if limit <= 0 {
		limit = 50
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if status == "active" {
		whereClause += " AND jo.status != 'discarded'"
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
		return nil, 0, fmt.Errorf("error counting offers: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			jo.id, jo.source, jo.external_id, jo.title, jo.company, jo.location, jo.url, jo.status, jo.created_at, jo.updated_at,
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
		return nil, 0, fmt.Errorf("error querying offers: %w", err)
	}
	defer rows.Close()

	var offers []JobOffer
	for rows.Next() {
		var o JobOffer
		var extID, company, location sql.NullString
		var oaID, oaInef, oaProp, oaDecisor, oaGanxo sql.NullString
		var oaScore sql.NullInt32
		var oaAnalyzedAt sql.NullTime

		err := rows.Scan(
			&o.ID, &o.Source, &extID, &o.Title, &company, &location, &o.URL, &o.Status, &o.CreatedAt, &o.UpdatedAt,
			&oaID, &oaInef, &oaProp, &oaScore, &oaDecisor, &oaGanxo, &oaAnalyzedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning offer row: %w", err)
		}

		if extID.Valid {
			o.ExternalID = &extID.String
		}
		if company.Valid {
			o.Company = &company.String
		}
		if location.Valid {
			o.Location = &location.String
		}

		if oaID.Valid {
			o.Analysis = &OpportunityAnalysis{
				ID:                 oaID.String,
				JobOfferID:         o.ID,
				IneficienciaManual: oaInef.String,
				PropostaMicroSaas:  oaProp.String,
				ViabilitatPLGScore: int(oaScore.Int32),
				DecisorCompra:      oaDecisor.String,
				GanxoVenda:         oaGanxo.String,
				AnalyzedAt:         oaAnalyzedAt.Time,
			}
		}

		offers = append(offers, o)
	}

	if offers == nil {
		offers = []JobOffer{}
	}

	return offers, total, nil
}

func (r *Repository) GetOfferByID(ctx context.Context, id string) (*JobOffer, error) {
	query := `
		SELECT 
			jo.id, jo.source, jo.external_id, jo.title, jo.company, jo.location, jo.url, jo.raw_text, jo.status, jo.created_at, jo.updated_at,
			oa.id, oa.ineficiencia_manual, oa.proposta_micro_saas, oa.viabilitat_plg_score, oa.decisor_compra, oa.ganxo_venda, oa.analyzed_at
		FROM job_offers jo
		LEFT JOIN opportunity_analyses oa ON jo.id = oa.job_offer_id
		WHERE jo.id = $1
	`
	var o JobOffer
	var extID, company, location sql.NullString
	var oaID, oaInef, oaProp, oaDecisor, oaGanxo sql.NullString
	var oaScore sql.NullInt32
	var oaAnalyzedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&o.ID, &o.Source, &extID, &o.Title, &company, &location, &o.URL, &o.RawText, &o.Status, &o.CreatedAt, &o.UpdatedAt,
		&oaID, &oaInef, &oaProp, &oaScore, &oaDecisor, &oaGanxo, &oaAnalyzedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error querying offer by id: %w", err)
	}

	if extID.Valid {
		o.ExternalID = &extID.String
	}
	if company.Valid {
		o.Company = &company.String
	}
	if location.Valid {
		o.Location = &location.String
	}

	if oaID.Valid {
		o.Analysis = &OpportunityAnalysis{
			ID:                 oaID.String,
			JobOfferID:         o.ID,
			IneficienciaManual: oaInef.String,
			PropostaMicroSaas:  oaProp.String,
			ViabilitatPLGScore: int(oaScore.Int32),
			DecisorCompra:      oaDecisor.String,
			GanxoVenda:         oaGanxo.String,
			AnalyzedAt:         oaAnalyzedAt.Time,
		}
	}

	return &o, nil
}

func (r *Repository) DiscardOffer(ctx context.Context, id string) error {
	query := `UPDATE job_offers SET status = 'discarded', updated_at = NOW() WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error discarding offer: %w", err)
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

func (r *Repository) CreateJobOffer(ctx context.Context, o *JobOffer) error {
	query := `
		INSERT INTO job_offers (source, external_id, title, company, location, url, raw_text, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (url) DO UPDATE SET raw_text = EXCLUDED.raw_text, updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query, o.Source, o.ExternalID, o.Title, o.Company, o.Location, o.URL, o.RawText, o.Status).
		Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
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
	return r.db.QueryRowContext(ctx, query, a.JobOfferID, a.IneficienciaManual, a.PropostaMicroSaas, a.ViabilitatPLGScore, a.DecisorCompra, a.GanxoVenda, a.RawLLMResponse).
		Scan(&a.ID, &a.AnalyzedAt)
}

func (r *Repository) CountSignalsToday(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM job_offers WHERE created_at >= CURRENT_DATE`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting signals today: %w", err)
	}
	return count, nil
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
		// Provide default placeholders if no runs yet
		now := time.Now()
		runs = []ScraperRun{
			{ID: 1, Source: "Feina Activa (SOC)", Status: "ok", ItemsFound: 0, RunAt: now},
			{ID: 2, Source: "Infofeina", Status: "ok", ItemsFound: 0, RunAt: now},
		}
	}

	return runs, nil
}
