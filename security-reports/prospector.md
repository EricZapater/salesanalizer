# Security Report — Mòdul Prospector

- **Data**: 2026-09-26
- **Mòdul**: `prospector`
- **Veredicte**: **SEGUR**

---

## 1. Anàlisi de Superfície d'Atac i Seguretat

| Vector d'Atac | Avaluació | Mitigació Implementada |
|---|---|---|
| **Exposició de Secrets / Tokens** | PASS | La clau `GROQ_API_KEY` i `ADMIN_SECRET` viuen estrictament com a variables d'entorn al backend. Mai exposades al frontend. |
| **Autenticació & IDOR** | PASS | `AuthMiddleware` intercepta totes les rutes de negoci (`/api/offers`, `/api/scrapers/run`, `/api/system/status`). Retorn ràpid 401 si el token no és vàlid. |
| **Injecció SQL** | PASS | Ús exclusiu de queries parametritzades (`$1, $2...`) a `internal/prospector/repository.go`. Cap query dinàmica concatenada. |
| **SSRF / Ingestió d'URL** | PASS | Validació d'esquema (`http`/`https`), `url.ParseRequestURI`, timeout de 15s al client HTTP i límit de mida de buffer a l'extracció HTML. |
| **Límit de Quota / DoS** | PASS | Comprovació estricta del llindar `DAILY_SIGNAL_LIMIT=50` abans d'executar qualsevol inferència o rastreig nou. Retorn `429 Too Many Requests`. |
| **Protecció de Dades (GDPR)** | PASS | El sistema només rastreja i persisteix ofertes laborals públiques corporatives, descartant dades personals de tercers. |

---

## 2. Conclusió
No s'han identificat vulnerabilitats crítiques ni exposicions de seguretat. El mòdul és apte i segur per a producció.
