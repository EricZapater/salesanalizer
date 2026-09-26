# Product Functional Specification — SalesAnalizer (Agent Prospector)

## 1. Visió i Objectiu de Negoci
SalesAnalizer és una eina d'ús exclusivament intern dissenyada per reduir a zero el Cost d'Adquisició de Clients (CAC) en la fase de recerca de mercat. El seu objectiu és rastrejar ofertes de feina i portals públics de Catalunya (focalitzant-se en la petita indústria, manufactura i serveis) per identificar ineficiències operatives manuals que es puguin resoldre mitjançant micro-SaaS de baix cost (< 100€/mes).

## 2. Usuaris i Públic Objectiu
- **Únic usuari:** Intern (Administrador / Propietari).
- L'eina no té registre públic ni usuaris externs. No requereix rols complexos ni gestió de permisos avançada, només una capa d'autenticació bàsica per protegir el dashboard.

## 3. Restriccions i Regles de Negoci
- **Cost d'operació estrictament 0 €:** 
  - Prohibida l'extracció de LinkedIn o qualsevol portal que requereixi proxies de pagament.
  - La inferència d'IA s'ha de delegar exclusivament a l'API gratuïta de Groq (models Llama 3 o similars).
- **Abast geogràfic:** Catalunya (processament bilingüe català/castellà).
- **Volum de processament:** Limitat a un màxim de 50 senyals/dia per evitar bloquejos d'IP del VPS i no saturar el *free tier* de l'API.

## 4. Dades Sensibles i Seguretat
- No es processen dades personals de tercers (GDPR), només textos d'ofertes de feina públiques de corporacions.
- Els tokens/API Keys de Groq han de viure exclusivament com a variables d'entorn al backend.

## 5. Exclusions Explícites (Out of Scope)
- **Mòbil:** No s'ha de desenvolupar cap aplicació React Native per a aquest projecte. Accés exclusivament via web d'escriptori.
- **Accions automàtiques de sortida:** El sistema només llegeix i analitza. L'enviament de correus en fred (Setter) queda fora de l'abast d'aquesta primera iteració.
