# Especificació de l'Èpica: Prospector & Radar d'Oportunitats Micro-SaaS

- **Mòdul**: `prospector`
- **Versió d'Spec**: 1.0.0
- **Estat**: Pendent de validació (Checkpoint 1)

---

## 1. Visió General
SalesAnalizer és una eina d'ús exclusivament intern (un sol usuari) orientada a identificar ineficiències operatives manuals en PIMEs i indústries catalanes a partir d'ofertes de feina públiques (Feina Activa / SOC i Infofeina), generant hipòtesis de productes Micro-SaaS (<100€/mes) amb alt potencial Product-Led Growth (PLG).

---

## 2. Rols d'Usuari
- **Administrador Únic**: Únic usuari autoritzat mitjançant una clau mestra (`ADMIN_SECRET`).

---

## 3. Històries d'Usuari i Criteris d'Acceptació

### HU-01: Autenticació per Clau Mestra
- **Com a** Administrador,
- **Vull** introduir la clau d'accés en un formulari senzill a la web,
- **Per** protegir el dashboard i les dades sense requerir registre d'usuaris ni gestió complexa de comptes.
- **Criteris d'acceptació**:
  1. El backend valida el token/clau enviat a la capçalera (`Authorization: Bearer <ADMIN_SECRET>`) contra la variable d'entorn `ADMIN_SECRET`.
  2. Si la clau és invàlida, retorna `401 Unauthorized`.
  3. El frontend emmagatzema el token a nivell local (ex. `localStorage` o store Zustand) i redirigeix a la vista del Radar si la sessió és vàlida.

---

### HU-02: Rastreig de Feina Activa i Infofeina (Cron Nocturn + Sota Demanda)
- **Com a** Administrador,
- **Vull** que el sistema executi una tasca programada a les 03:00h i també poder prémer un botó "Rastrejar portals ara" des de la interfície,
- **Per** disposar d'oportunitats automàticament o forçar una nova cerca quan vulgui sense haver d'esperar a la nit.
- **Criteris d'acceptació**:
  1. El backend executa un cron diari a les 03:00h (hora local).
  2. La interfície disposa d'un botó "Rastrejar portals ara" que activa el procés sota demanda via API.
  3. Cerca ofertes als portals Feina Activa (SOC) i Infofeina amb les paraules clau:
     - `auxiliar administratiu`, `control de planta`, `gestió d'estocs`, `quadrants`, `introducció de dades`, `gestió de rutes`.
  4. Respecta el límit màxim de 50 senyals/ofertes recollides per dia (si s'arriba al límit, s'atura i avisa).
  5. Emmagatzema les noves ofertes a PostgreSQL evitant duplicats per URL o identificador d'oferta.
  6. Espaia les peticions HTTP amb retards aleatoris per evitar bloquejos d'IP sense fer servir proxies de pagament.

---

### HU-03: Ingestió i Anàlisi Manual per URL
- **Com a** Administrador,
- **Vull** enganxar la URL d'una oferta directament al dashboard i demanar la seva anàlisi immediata,
- **Per** validar ràpidament idees d'ofertes que descobreixo navegant manualment.
- **Criteris d'acceptació**:
  1. Input de text visible al Radar amb botó "Analitzar oferta".
  2. El backend descarrega i extreu el text de l'oferta en temps real.
  3. Envia el text a l'API de Groq i desa tant l'oferta com l'anàlisi a la base de dades.
  4. Retorna el resultat immediatament i l'afegeix a la taula del Radar.

---

### HU-04: Anàlisi d'IA de Micro-SaaS (Groq / Llama 3)
- **Com a** Administrador,
- **Vull** que cada oferta sigui analitzada per un model LLM mitjançant Groq,
- **Per** extreure automàticament les tasques manuals ineficients i avaluar la seva viabilitat com a eina PLG autònoma.
- **Criteris d'acceptació**:
  1. El backend fa la crida a l'API de Groq utilitzant la clau d'entorn `GROQ_API_KEY`.
  2. El prompt força una resposta estricta en format JSON amb els camps:
     - `ineficiencia_manual`: Descripció de la tasca manual o full de càlcul detectat.
     - `proposta_micro_saas`: Solució autònoma d'una sola funció per resoldre-ho.
     - `viabilitat_plg_score`: Enter de 1 a 5 (1 = complex/integració a mida; 5 = self-onboarding 100% autònom).
     - `decisor_compra`: Càrrec objectiu (ex: Cap de producció, Propietari, Gerent).
     - `ganxo_venda`: Frase curta directa per a futur correu en fred.
  3. L'anàlisi es guarda associada a l'oferta corresponent.

---

### HU-05: Taula del Radar de Senyals (Feed Principal)
- **Com a** Administrador,
- **Vull** visualitzar totes les ofertes analitzades en una taula prioritzada pel `viabilitat_plg_score`,
- **Per** focalitzar-me directament en les oportunitats amb puntuació 4 i 5.
- **Criteris d'acceptació**:
  1. La taula mostra per defecte les ofertes ordenades descendentment per `viabilitat_plg_score` (els 5 a dalt).
  2. Columnes principals: Data, Títol de l'oferta / Empresa, Font (SOC/Infofeina/Manual), Ineficiència resumida, Proposta Micro-SaaS, Puntuació PLG (1-5), Accions.
  3. Acció ràpida de descartar (paperera) que fa soft-delete / arxivat de l'oportunitat.
  4. Clic en qualsevol fila obre el panell lateral (Drawer) de detall.

---

### HU-06: Panell Lateral de Detall i Estat del Sistema
- **Com a** Administrador,
- **Vull** obrir un calaix lateral (Drawer) amb dues pestanyes (Anàlisi i Estat del Sistema),
- **Per** consultar el detall profund d'una oportunitat i monitoritzar el consum d'ofertes diàries sense canviar de pantalla.
- **Criteris d'acceptació**:
  1. **Pestanya 1 (Anàlisi)**:
     - Mostra: Ineficiència manual completa, Proposta Micro-SaaS, Decisor de compra, Ganxo de venda (amb botó de copiar al portapapers), Score PLG, i el text original complet de l'oferta amb enllaç a la font.
  2. **Pestanya 2 (Estat del Sistema)**:
     - Comptador de senyals utilitzats avui (ex: `12 / 50 senyals`).
     - Estat de l'última execució del scraper nocturn (Data/hora, estat ÈXIT/ERROR, número d'ofertes trobades).

---

## 4. Model de Dades Proposat (PostgreSQL)

### Taula `job_offers`
- `id` (UUID / SERIAL, PK)
- `source` (VARCHAR: `feina_activa`, `infofeina`, `manual`)
- `external_id` (VARCHAR, NULLABLE, indexat)
- `title` (VARCHAR)
- `company` (VARCHAR, NULLABLE)
- `location` (VARCHAR, NULLABLE)
- `url` (TEXT, UNIQUE)
- `raw_text` (TEXT)
- `scraped_at` (TIMESTAMPTZ)
- `status` (VARCHAR: `pending_analysis`, `analyzed`, `discarded`, `error`)
- `created_at` (TIMESTAMPTZ)
- `updated_at` (TIMESTAMPTZ)

### Taula `opportunity_analyses`
- `id` (UUID / SERIAL, PK)
- `job_offer_id` (FK a `job_offers.id`, UNIQUE)
- `ineficiencia_manual` (TEXT)
- `proposta_micro_saas` (TEXT)
- `viabilitat_plg_score` (INT: 1..5)
- `decisor_compra` (VARCHAR)
- `ganxo_venda` (TEXT)
- `raw_llm_response` (JSONB)
- `analyzed_at` (TIMESTAMPTZ)

### Taula `scraper_runs`
- `id` (SERIAL, PK)
- `source` (VARCHAR)
- `status` (VARCHAR: `success`, `partial`, `failed`)
- `items_found` (INT)
- `error_message` (TEXT, NULLABLE)
- `run_at` (TIMESTAMPTZ)

---

## 5. Mètriques i Controls de Límit
- Variable de configuració: `DAILY_SIGNAL_LIMIT=50`.
- El backend comptabilitza les ofertes creades avui (`WHERE created_at >= CURRENT_DATE`).
- Si s'arriba a 50, l'orquestrador del scraper atura l'execució automàtica fins l'endemà. L'anàlisi manual d'URL pot permetre's o alertar si supera el límit segons convingui.
