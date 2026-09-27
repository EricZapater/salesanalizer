# Especificació de l'Èpica: Radar de Processos Empresarials & Oportunitats Micro-SaaS

- **Mòdul**: `prospector`
- **Versió d'Spec**: 2.0.0
- **Estat**: Implementat i Validat (v2.0.0)
- **Data d'Actualització**: 2026-09-26

---

## 1. Visió de Producte i Objectius

SalesAnalizer deixa de ser un simple "generador d'idees d'IA a partir d'ofertes" per convertir-se en un **RADAR DE PROCESSOS EMPRESARIALS**: un sistema d'intel·ligència de mercat que detecta processos que PIMEs, tallers, comerços i autònoms encara resolen mitjançant mètodes artesanals o ineficients:
- Fulls de càlcul (Excel, Google Sheets).
- Missatgeria dispersa (WhatsApp, Telegram, trucades).
- Documents estàtics i paper (Word, PDFs, albarans físics, notes manuals).
- Tasques manuals repetitives (còpia/enganxa, re-entrada de dades, recordatoris mentals, quadrants en pissarres).
- Coneixement tàcit no estructurat ("una persona que és l'única que sap com funciona").

### Criteris de Producte Micro-SaaS Objectiu (No negociables)
1. **Un sol problema concret, un sol perfil de client, un sol workflow clar, un sol decisor de compra.**
2. **Aplicació web lleugera**, amb procés de *self-onboarding* directe i idealment **0 integracions inicials** (o mínimes).
3. **Model de preus orientatiu**: 30–50 €/mes per negoci, amb potencial per a desenes de clients (recurrents).
4. **Fita de negoci**: ~1.000–1.500 €/mes amb 4–5 micro-productes independents (25–40 clients en total).
5. **Principi de qualitat sobre volum**: 2 o 3 candidats sòlids i hipervalidats al mes tenen infinitament més valor que un panell ple de desenes de falsos positius o idees teòriques sense demanda real.
6. **Exclusions explícites**:
   - ❌ NO ERPs integrals ni solucions "tot en un".
   - ❌ NO problemes que requereixin consultoria, formació presencial o canvis organitzatius profunds.
   - ❌ NO software empresarial complex amb vendes corporatives de cicle llarg.
   - ❌ NO "idees startup" genèriques ni projectes basats en OCR / maquinari / IA sofisticada d'alt cost.

---

## 2. Nou Pipeline Conceptual en 5 Etapes

El pipeline substitueix la crida monolítica anterior per un flux progressiu i auditable:

```
RAW EVIDENCE (Extracció multicanal)
       ↓
1. PROCESS EXTRACTION (Extracció de fets literals, quotes i eines)
       ↓
2. NORMALIZATION (Mapeig cap a processos canònics de negoci)
       ↓
3. PAIN CLUSTERING (Agrupació d'evidències independents multi-font)
       ↓
4. OPPORTUNITY SYNTHESIS (Generació d'hipòtesi Micro-SaaS només amb clúster consolidat)
       ↓
5. MULTIDIMENSIONAL SCORING (Avaluació de 12 factors justificats amb dades)
```

### Tipologia estricta de les dades del pipeline:
- **OBSERVED FACT**: Dada literal extreta del text font (eines esmentades, freqüència declarada, cita textual).
- **INFERENCE**: Deducció raonada pel LLM a partir dels fets (sector, nivell de manualitat).
- **HYPOTHESIS**: Proposta de solució, proposta de valor i model de preu (exclusivament al pas final).

---

## 3. Rols d'Usuari

- **Operador / Administrador Únic**: Usuari autenticat mitjançant clau mestra (`ADMIN_SECRET`) que explora el radar de clústers, revisa les evidències literals i valida les hipòtesis de negoci generades.

---

## 4. Històries d'Usuari i Criteris d'Acceptació

### HU-01: Ingestió Multicanal i Deduplicació Avançada
- **Com a** Administrador,
- **Vull** que el sistema reculli senyals bruts de diverses fonts (SearXNG, Reddit RSS, Feina Activa, Licitacions públiques) i els dedupliqui abans de qualsevol processament de LLM,
- **Per** no malgastar recursos d'IA ni inflar la importància d'un problema per contingut duplicat o sindicat.
- **Criteris d'acceptació**:
  1. Normalització d'URL (eliminació de paràmetres de seguiment `utm_*`, `ref`, protocols i barres finals).
  2. Generació de `content_hash` (SHA-256 del text canònic normalitzat) per detectar mateix contingut en diferents dominis.
  3. Detecció de sindicació (>90% similitud): marca l'evidència com a `is_duplicate_of` i no suma com a evidència independent al clúster.
  4. Deduplicació dins del mateix fil/conversa per no comptar com a N evidències un mateix usuari.

---

### HU-02: Etapa 1 — Extracció Estricta de Processos (Evidence Extraction)
- **Com a** Administrador,
- **Vull** que la primera etapa de la IA extregui exclusivament què fa l'empresa i quines eines manuals utilitza,
- **Per** evitar al·lucinacions d'idees de producte quan només estem recollint fets.
- **Criteris d'acceptació**:
  1. El prompt de l'Etapa 1 té prohibit proposar solucions, Micro-SaaS o models de negoci.
  2. Extreu: `extracted_process`, `task_description`, `frequency` (diària|setmanal|mensual|puntual|unknown), `manuality_score` (0..3), `tools_mentioned` (array), `sector`, `evidence_type` i `source_evidence_quote`.
  3. **Regla dura de cita literal**: Si `source_evidence_quote` no conté una cita literal verificable (<200 caràcters) del text original, el backend assigna automàticament `evidence_confidence = 'baixa'`.

---

### HU-03: Etapa 2 i 3 — Normalització de Processos i Confirmació de Clústers
- **Com a** Administrador,
- **Vull** que les evidències s'agrupin al voltant de processos de negoci canònics i es confirmin com a clústers de dolor reals,
- **Per** descobrir patrons repetits en múltiples empreses i sectors independents.
- **Criteris d'acceptació**:
  1. **Etapa 2 (Normalització)**: Mapeja `extracted_process` a `ProcessNormalized` (creant-ne de nous o assignant un existent).
  2. **Etapa 3 (Confirmació de Clúster)**: El LLM confirma la fusió o separació de clústers tenint en compte la tasca, l'objecte i el context de negoci (no solament solapament de paraules clau).
  3. Cada `PainCluster` manté un recompte calculat de `evidence_count`, `company_count`, `source_count` i `sector_breadth`.

---

### HU-04: Etapa 4 — Síntesi d'Oportunitat Micro-SaaS (Opportunity Synthesis)
- **Com a** Administrador,
- **Vull** que el sistema només generi una proposta de Micro-SaaS quan un clúster assoleixi un llindar mínim d'evidència sòlida,
- **Per** dedicar temps només a problemes amb demanda provada i contrastada.
- **Criteris d'acceptació**:
  1. La síntesi d'oportunitat només s'executa si el clúster té un mínim d'evidències independents (ex: ≥3 evidències independents de ≥2 fonts diferents).
  2. Genera l'estructura completa:
     - `problem_statement`
     - `target_customer`
     - `buyer` (càrrec decisor)
     - `workflow_steps` (Seqüència: Reben → registren → assignen → seguiment → tanquen)
     - `current_workaround` (com ho fan avui amb Excel/WhatsApp/Paper)
     - `proposed_micro_saas` (nom i funcionalitat central)
     - `mvp_scope` (abast mínim per sortir al mercat)
     - `integrations_required` (cap | baixa | mitjana | alta)
     - `pricing_hypothesis_eur` (rang 30-50€/mes)
     - `onboarding_complexity` (baixa | mitjana | alta)

---

### HU-05: Etapa 5 — Puntuació Multidimensional Justificada (12 Factors)
- **Com a** Administrador,
- **Vull** una avaluació basada en 12 factors ponderats i cadascun amb la seva justificació explícita,
- **Per** entendre exactament per què una oportunitat és viable o descartable sense caixes negres.
- **Criteris d'acceptació**:
  1. Substitució del camp únic `viabilitat_plg_score` pels 12 factors (0..5 o null si no hi ha prou dades):
     - `score_evidence_strength`
     - `score_company_breadth`
     - `score_source_breadth`
     - `score_frequency`
     - `score_manual_effort`
     - `score_repetition`
     - `score_business_impact`
     - `score_buyer_clarity`
     - `score_market_breadth`
     - `score_implementation_simplicity`
     - `score_integration_dependency` (penalització)
     - `score_existing_software_saturation` (penalització)
  2. Cadascun dels 12 scores té associat un text de justificació que cita l'evidència o mètrica concreta.
  3. `opportunity_score_total`: Fórmula matemàtica transparent documentada al codi.

---

### HU-06: Nova Vista del Radar de Clústers (Frontend)
- **Com a** Administrador,
- **Vull** visualitzar el Radar organitzat per Top Clústers de Dolor en comptes d'una llista plana de senyals desconnectats,
- **Per** identificar d'un cop d'ull els problemes amb més tracció i profunditat de mercat.
- **Criteris d'acceptació**:
  1. Taula/targetes principals mostrant: Nom del Clúster, Procés Canònic, Núm. d'Evidències, Empreses, Fonts, Amplitud Sectorial, Score Total.
  2. En seleccionar un clúster s'obre el panell amb la hipòtesi Micro-SaaS (`Opportunity`) i la secció **"Per què ho creiem?"**, que mostra les cites literals, fonts originals i enllaços de cada evidència associada.
  3. Possibilitat de filtrar per nivell de confiança, fonts i estat.

---

### HU-07: Compatibilitat Retroactiva i Configuració
- **Com a** Administrador,
- **Vull** que els endpoints existents segueixin funcionant durant tota la transició,
- **Per** no trencar l'operativitat mentre el nou pipeline s'activa.
- **Criteris d'acceptació**:
  1. Endpoints `/offers`, `/offers/:id/status`, `/system/status`, `/signals/settings`, `/scrapers/run` i `/scrapers/stream` es mantenen operatius.
  2. Les taules noves coexisteixen amb les taules de llegat (`job_offers`, `opportunity_analyses`).

---

## 5. Model de Dades Proposat (Migració SQL)

### Taules noves:

```sql
-- 1. EVIDÈNCIES EXTRETES (Unitat atòmica d'informació)
CREATE TABLE IF NOT EXISTS evidences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type VARCHAR(50) NOT NULL, -- searxng, rss_forum, feina_activa, public_tender, manual, marketplace
    source_name VARCHAR(255) NOT NULL,
    source_url TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    title VARCHAR(500) NOT NULL,
    raw_content TEXT NOT NULL,
    publication_date TIMESTAMPTZ,
    company VARCHAR(255),
    sector VARCHAR(100),
    location VARCHAR(255),
    extracted_process TEXT NOT NULL,
    normalized_process_id UUID,
    task_description TEXT NOT NULL,
    frequency VARCHAR(50), -- diaria, setmanal, mensual, puntual, unknown
    manuality_score INT CHECK (manuality_score BETWEEN 0 AND 3),
    tools_mentioned TEXT[] DEFAULT '{}',
    evidence_type VARCHAR(50) NOT NULL, -- job_posting, complaint, question, review, template_listing, tender_description
    evidence_confidence VARCHAR(50) NOT NULL DEFAULT 'mitjana', -- alta, mitjana, baixa
    source_evidence_quote TEXT NOT NULL,
    is_duplicate_of UUID REFERENCES evidences(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_evidences_hash ON evidences(content_hash);
CREATE INDEX IF NOT EXISTS idx_evidences_source_url ON evidences(source_url);
CREATE INDEX IF NOT EXISTS idx_evidences_confidence ON evidences(evidence_confidence);

-- 2. PROCESSOS NORMALITZATS (Catàleg canònic)
CREATE TABLE IF NOT EXISTS processes_normalized (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. CLÚSTERS DE DOLOR (Agregació multi-evidència)
CREATE TABLE IF NOT EXISTS pain_clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    normalized_process_id UUID REFERENCES processes_normalized(id) ON DELETE SET NULL,
    sector_breadth INT DEFAULT 1,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    frequency_score INT,
    manuality_score INT,
    confidence_score NUMERIC(5,2) DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. TAULA PONT N:M (PainCluster <-> Evidence)
CREATE TABLE IF NOT EXISTS pain_cluster_evidences (
    pain_cluster_id UUID REFERENCES pain_clusters(id) ON DELETE CASCADE,
    evidence_id UUID REFERENCES evidences(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (pain_cluster_id, evidence_id)
);

-- 5. OPORTUNITATS MICRO-SAAS (1:1 amb PainCluster consolidat)
CREATE TABLE IF NOT EXISTS opportunities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pain_cluster_id UUID UNIQUE NOT NULL REFERENCES pain_clusters(id) ON DELETE CASCADE,
    problem_statement TEXT NOT NULL,
    target_customer VARCHAR(255) NOT NULL,
    buyer VARCHAR(255) NOT NULL,
    workflow_steps TEXT NOT NULL,
    current_workaround TEXT NOT NULL,
    proposed_micro_saas TEXT NOT NULL,
    mvp_scope TEXT NOT NULL,
    integrations_required VARCHAR(50) NOT NULL DEFAULT 'cap',
    pricing_hypothesis_eur INT NOT NULL DEFAULT 40,
    onboarding_complexity VARCHAR(50) NOT NULL DEFAULT 'baixa',
    -- 12 Scores Multidimensionals (0..5)
    score_evidence_strength INT,
    score_company_breadth INT,
    score_source_breadth INT,
    score_frequency INT,
    score_manual_effort INT,
    score_repetition INT,
    score_business_impact INT,
    score_buyer_clarity INT,
    score_market_breadth INT,
    score_implementation_simplicity INT,
    score_integration_dependency INT,
    score_existing_software_saturation INT,
    opportunity_score_total NUMERIC(5,2) NOT NULL DEFAULT 0.0,
    score_justifications JSONB NOT NULL DEFAULT '{}',
    risks TEXT,
    unknowns TEXT,
    evidence_summary TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_opportunities_score_total ON opportunities(opportunity_score_total DESC);
```

---

## 6. Fases d'Implementació i Checkpoints

1. **Fase 0 — Especificació Funcional (AQUEST DOCUMENT)**: Checkpoint 1 humà per aprovar abans de codificar.
2. **Fase 1 — Fonaments d'Extracció i Deduplicació**: Migració SQL `000003`, mòdul pur `dedup.go` + tests i Prompt 1 (`Evidence Extraction`).
3. **Fase 2 — Normalització i Clustering**: Prompts 2 i 3, taules `ProcessNormalized`, `PainCluster` i `PainClusterEvidence`.
4. **Fase 3 — Oportunitats i Scoring Multidimensional**: Prompts 4 i 5, taula `Opportunity`, mòdul de càlcul `scoring.go` i tests sense xarxa.
5. **Fase 4 — Frontend Revamp**: Vista de Top Pain Clusters, Drawer "Per què ho creiem?" i navegació d'evidències.
6. **Fase 5 — Comparativa Old vs New i Tancament**: Execució del dataset comparatiu per mesurar reducció de falsos positius i qualitat de clústers.
