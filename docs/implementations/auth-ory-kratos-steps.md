# Auth — Ory Kratos + Oathkeeper — Implementazione step by step

Ogni step e' testabile in isolamento prima di passare al successivo.
Il goal finale e' un sistema di autenticazione completo integrato con l'architettura esistente di Relay.

**Prerequisiti:** Docker Compose funzionante con Kafka e PostgreSQL.

---

## Panoramica dei componenti

```
Frontend (React)
    |
    |-- Login/Register --> Kratos API (self-service flows)
    |                        |
    |                        v
    |                      Kratos DB (PostgreSQL separato)
    |
    |-- HTTP requests --> Oathkeeper (reverse proxy :4455)
    |                        |
    |                        | 1. valida session cookie con Kratos
    |                        | 2. inietta X-User-ID nel header
    |                        |
    |                        v
    |                      Backend Go (:8080)
    |                        |
    |                        | AuthMiddleware legge X-User-ID (gia' esistente)
    |                        | ABAC check con user_id
    |                        v
    |                      Kafka commands / Gin handlers
    |
    |-- WebSocket --> WsHandler
                        |
                        | valida session cookie con Kratos SDK
                        | estrae userID
                        v
                      Hub + Aggregator (flusso esistente invariato)
```

### Ruolo di ogni componente

| Componente | Responsabilita' | Non fa |
|---|---|---|
| **Kratos** | Gestisce identita': registrazione, login, sessioni, recovery, MFA | Non autorizza, non conosce workspace/canali |
| **Oathkeeper** | Reverse proxy: valida sessioni, inietta identity headers, inoltra | Non gestisce credenziali, non fa ABAC |
| **AuthMiddleware (Go)** | Legge `X-User-ID` dal header trusted di Oathkeeper | Non valida token, non parla con Kratos |
| **ABAC (Go)** | Decide se l'utente puo' eseguire l'azione | Non autentica, non gestisce sessioni |

---

## Step 0 — Prerequisiti e struttura file

### Struttura directory da creare

```
backend/
  ory/
    kratos/
      kratos.yml              # configurazione principale Kratos
      identity.schema.json    # schema dell'identita' (campi utente)
    oathkeeper/
      oathkeeper.yml          # configurazione principale Oathkeeper
      access-rules.yml        # regole di routing e autenticazione
```

### Concetti chiave prima di iniziare

**Identity Schema:** Kratos non ha un modello utente fisso. Definisci tu i campi tramite un JSON Schema. Per Relay servono: email (credenziale di login) + display_name (profilo).

**Self-Service Flows:** Kratos espone operazioni utente come "flow" in due fasi:
1. `GET /self-service/{action}/browser` --> Kratos ritorna un `flow_id` + i campi da mostrare
2. `POST /self-service/{action}` con `flow_id` + dati compilati --> Kratos esegue l'azione

**Session Cookie:** Dopo il login, Kratos setta un cookie `ory_kratos_session`. Il browser lo manda automaticamente su ogni request. Oathkeeper lo valida.

---

## Step 1 — Kratos in Docker Compose

**Obiettivo:** Kratos avviato con un database PostgreSQL dedicato, raggiungibile su `localhost:4433` (API pubblica) e `localhost:4434` (API admin).

### Docker Compose — nuovi servizi

Aggiungere al `docker-compose.yml` esistente:

```yaml
  kratos-db:
    image: postgres:16
    container_name: kratos-db
    environment:
      POSTGRES_USER: kratos
      POSTGRES_PASSWORD: kratos
      POSTGRES_DB: kratos
    ports:
      - "5433:5432"
    volumes:
      - kratos-db-data:/var/lib/postgresql/data

  kratos-migrate:
    image: oryd/kratos:v1.3.1
    container_name: kratos-migrate
    depends_on:
      - kratos-db
    environment:
      DSN: postgres://kratos:kratos@kratos-db:5432/kratos?sslmode=disable
    volumes:
      - ./ory/kratos:/etc/config/kratos
    command: migrate sql -e --yes --config /etc/config/kratos/kratos.yml

  kratos:
    image: oryd/kratos:v1.3.1
    container_name: kratos
    depends_on:
      kratos-migrate:
        condition: service_completed_successfully
    environment:
      DSN: postgres://kratos:kratos@kratos-db:5432/kratos?sslmode=disable
    volumes:
      - ./ory/kratos:/etc/config/kratos
    command: serve --config /etc/config/kratos/kratos.yml --dev --watch-courier
    ports:
      - "4433:4433"   # public API (frontend chiama qui)
      - "4434:4434"   # admin API (solo interno)
```

Aggiungere il volume:

```yaml
volumes:
  kratos-db-data:
```

### Identity Schema — `ory/kratos/identity.schema.json`

```json
{
  "$id": "https://relay.local/identity.schema.json",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Relay User",
  "type": "object",
  "properties": {
    "traits": {
      "type": "object",
      "properties": {
        "email": {
          "type": "string",
          "format": "email",
          "title": "Email",
          "ory.sh/kratos": {
            "credentials": {
              "password": {
                "identifier": true
              }
            },
            "verification": {
              "via": "email"
            },
            "recovery": {
              "via": "email"
            }
          }
        },
        "display_name": {
          "type": "string",
          "title": "Display Name",
          "minLength": 1,
          "maxLength": 100
        }
      },
      "required": ["email", "display_name"],
      "additionalProperties": false
    }
  }
}
```

**Nota:** `traits` e' il campo dove Kratos mette i dati di profilo dell'utente. L'`email` con `credentials.password.identifier: true` dice a Kratos di usarla come login identifier. Il `display_name` e' un campo custom che Relay usa per il profilo.

### Configurazione Kratos — `ory/kratos/kratos.yml`

```yaml
version: v1.3.1

dsn: memory # sovrascritto da env var DSN nel Docker Compose

serve:
  public:
    base_url: http://localhost:4433/
    cors:
      enabled: true
      allowed_origins:
        - http://localhost:5173
      allowed_methods:
        - GET
        - POST
        - PUT
        - DELETE
        - PATCH
      allowed_headers:
        - Content-Type
        - Authorization
        - Cookie
        - X-Session-Token
      exposed_headers:
        - Set-Cookie
      allow_credentials: true
  admin:
    base_url: http://localhost:4434/

selfservice:
  default_browser_return_url: http://localhost:5173/

  methods:
    password:
      enabled: true

  flows:
    registration:
      ui_url: http://localhost:5173/auth/register
      after:
        password:
          hooks:
            - hook: session  # auto-login dopo registrazione
            - hook: web_hook
              config:
                url: http://host.docker.internal:8080/api/hooks/registration
                method: POST
                body: file:///etc/config/kratos/webhook-body.jsonnet
                auth:
                  type: api_key
                  config:
                    name: X-Webhook-Secret
                    value: relay-webhook-secret-change-me
                    in: header

    login:
      ui_url: http://localhost:5173/auth/login

    logout:
      after:
        default_browser_return_url: http://localhost:5173/auth/login

    verification:
      enabled: false  # MVP: disabilitato

    recovery:
      enabled: false  # MVP: disabilitato

identity:
  default_schema_id: relay-user
  schemas:
    - id: relay-user
      url: file:///etc/config/kratos/identity.schema.json

log:
  level: info
  format: text

session:
  cookie:
    name: ory_kratos_session
    same_site: Lax
    domain: localhost
  lifespan: 24h
```

**Punti chiave della configurazione:**
- `selfservice.flows.registration.after.password.hooks`: due hook dopo la registrazione:
  1. `session` — auto-login (crea sessione immediatamente, l'utente non deve rifare login)
  2. `web_hook` — chiama il backend Relay per creare l'utente nell'event store
- `ui_url` punta al frontend React — Kratos non ha UI, redirect al tuo form
- `session.cookie` — cookie-based, il browser lo manda automaticamente

### Webhook body template — `ory/kratos/webhook-body.jsonnet`

```jsonnet
function(ctx) {
  identity_id: ctx.identity.id,
  email: ctx.identity.traits.email,
  display_name: ctx.identity.traits.display_name,
}
```

Questo template Jsonnet dice a Kratos quali dati mandare al webhook di Relay quando un utente si registra. Il backend ricevera' `identity_id`, `email`, e `display_name`.

### Test

```bash
docker compose up -d kratos-db kratos-migrate kratos
```

Verifica che Kratos risponda:

```bash
curl http://localhost:4433/health/alive
# {"status":"ok"}
```

Crea un flow di registrazione:

```bash
curl -s http://localhost:4433/self-service/registration/api | jq .id
# ritorna un flow_id UUID
```

**Concetti:** Kratos come servizio Docker, identity schema, self-service flows.

---

## Step 2 — Webhook di registrazione nel backend Go

**Obiettivo:** Quando un utente si registra su Kratos, il backend Relay riceve il webhook e produce l'evento `user:platform:register` nell'event store, creando l'utente nel dominio Relay.

### Endpoint webhook — `POST /api/hooks/registration`

```go
// Payload ricevuto dal webhook Kratos
type KratosRegistrationWebhook struct {
    IdentityID  string `json:"identity_id"`
    Email       string `json:"email"`
    DisplayName string `json:"display_name"`
}
```

### Flusso

```
Kratos (after registration hook)
    |
    | POST /api/hooks/registration
    | Header: X-Webhook-Secret: relay-webhook-secret-change-me
    | Body: { identity_id, email, display_name }
    |
    v
Backend webhook handler
    |
    | 1. Valida X-Webhook-Secret
    | 2. Usa identity_id come AggregateID (l'ID Kratos diventa l'ID utente Relay)
    | 3. Produce Command su messages.commands con action_key user:platform:register
    |
    v
Consumer service
    |
    | 4. Persiste EventRecord + UserSnapshot nell'event store
    | 5. Produce Event su messages.events
    |
    v
View Materializer + RT Event Service (flusso standard)
```

### Decisione chiave: identity_id di Kratos = user_id di Relay

L'ID generato da Kratos per l'identita' diventa il `user_id` usato ovunque in Relay. Non serve una tabella di mapping. Questo significa:
- Lo snapshot `user_snapshots` usa l'identity_id di Kratos come PK
- `metadata.user_id` in ogni EventRecord e' l'identity_id di Kratos
- Il topic `realtime.users` usa lo stesso ID
- Il frontend conosce un solo ID utente

**Perche':** due ID separati (uno Kratos, uno Relay) richiederebbero un JOIN o lookup su ogni richiesta. Un ID unico elimina il mapping.

### Sicurezza del webhook

Il webhook e' protetto da un shared secret nell'header `X-Webhook-Secret`. Il handler lo valida prima di processare il payload. Questo endpoint NON passa per Oathkeeper (e' chiamato da Kratos internamente, non dal browser).

### Route

L'endpoint webhook vive fuori dal gruppo API autenticato:

```go
// Gruppo pubblico (no auth middleware)
hooks := r.Group("/api/hooks")
hooks.POST("/registration", handler.HandleKratosRegistration)

// Gruppo autenticato (esistente)
api := r.Group("/api/", middleware.AuthMiddleware(), middleware.TraceIDMiddleware())
```

### Test

1. Avvia backend + Kratos
2. Registra un utente via Kratos API
3. Verifica nel log che il webhook arrivi e l'evento `user:platform:register` venga prodotto
4. Verifica che `user_snapshots` contenga il nuovo utente

**Concetti:** Webhook Kratos, sincronizzazione identita', ID unificato.

---

## Step 3 — Oathkeeper come reverse proxy

> **Stato: COMPLETATO e testato**

**Obiettivo:** Oathkeeper davanti al backend Go. Valida il session cookie (o bearer token) di Kratos, inietta `X-User-ID`, inoltra al backend. Le request non autenticate ricevono 401.

### Docker Compose — aggiungere Oathkeeper

```yaml
  oathkeeper:
    image: oryd/oathkeeper:v0.40.8
    container_name: oathkeeper
    depends_on:
      - kratos
    volumes:
      - ./ory/oathkeeper:/etc/config/oathkeeper
    command: serve --config /etc/config/oathkeeper/oathkeeper.yml
    ports:
      - "4455:4455"   # proxy (il frontend chiama qui)
      - "4456:4456"   # API admin
```

### Configurazione — `ory/oathkeeper/oathkeeper.yml`

```yaml
serve:
  proxy:
    port: 4455
    cors:
      enabled: true
      allowed_origins:
        - http://localhost:5173
      allowed_methods:
        - GET
        - POST
        - PUT
        - DELETE
        - PATCH
      allowed_headers:
        - Content-Type
        - Authorization
        - Cookie
      exposed_headers:
        - Set-Cookie
      allow_credentials: true
  api:
    port: 4456

access_rules:
  matching_strategy: glob      # IMPORTANTE: glob, non regexp — <**> e' la sintassi glob
  repositories:
    - file:///etc/config/oathkeeper/access-rules.yml

authenticators:
  cookie_session:
    enabled: true
    config:
      check_session_url: http://kratos:4433/sessions/whoami
      preserve_path: true
      force_method: GET
      extra_from: "@this"
      subject_from: "identity.id"
      only:
        - ory_kratos_session

  bearer_token:                 # Supporto per Authorization: Bearer <session_token>
    enabled: true               # Utile per API testing (curl) e client non-browser
    config:
      check_session_url: http://kratos:4433/sessions/whoami
      preserve_path: true
      force_method: GET
      subject_from: "identity.id"
      extra_from: "@this"
      token_from:
        header: Authorization

  noop:
    enabled: true

authorizers:
  allow:
    enabled: true

mutators:
  header:
    enabled: true
    config:
      headers:
        X-User-ID: "{{ print .Subject }}"

  noop:
    enabled: true
```

**Punti chiave:**

- **`matching_strategy: glob`** — Oathkeeper supporta `glob` o `regexp`. Con glob si usa `<**>` per wildcard multi-segment. NON usare `regexp` con pattern `<**>` — causa "invalid nested repetition operator".
- **`authenticators.cookie_session`** — Prende il cookie `ory_kratos_session`, chiama Kratos `GET /sessions/whoami`, estrae `identity.id` come Subject.
- **`authenticators.bearer_token`** — Prende il token dall'header `Authorization: Bearer ...`, stessa validazione con Kratos. Utile per curl/testing. Oathkeeper prova gli authenticators in ordine: se cookie fallisce, prova bearer. Se entrambi falliscono → 401.
- **`mutators.header`** — Aggiunge `X-User-ID: {identity.id}` alla request inoltrata. Questo e' l'header che `AuthMiddleware` legge.
- **CORS: niente OPTIONS** — Oathkeeper gestisce le preflight CORS internamente. Aggiungere OPTIONS a `allowed_methods` causa errore.

### Access Rules — `ory/oathkeeper/access-rules.yml`

**ORDINE CRITICO:** Le regole vengono matchate dall'alto verso il basso. Le regole piu' specifiche (hooks, ws) DEVONO essere prima della catch-all (`/api/<**>`).

```yaml
# 1. Webhook interno — nessuna autenticazione (Kratos -> backend)
#    DEVE essere prima di relay-api-protected per matchare prima
- id: "relay-hooks"
  upstream:
    url: "http://host.docker.internal:8080"
  match:
    url: "http://localhost:4455/api/hooks/<**>"
    methods:
      - POST
  authenticators:
    - handler: noop
  authorizer:
    handler: allow
  mutators:
    - handler: noop

# 2. WebSocket — richiede sessione valida
- id: "relay-ws"
  upstream:
    url: "http://host.docker.internal:8080"
  match:
    url: "http://localhost:4455/api/ws"
    methods:
      - GET
  authenticators:
    - handler: cookie_session
    - handler: bearer_token
  authorizer:
    handler: allow
  mutators:
    - handler: header

# 3. Rotte protette — catch-all, richiedono sessione valida
- id: "relay-api-protected"
  upstream:
    url: "http://host.docker.internal:8080"
  match:
    url: "http://localhost:4455/api/<**>"
    methods:
      - GET
      - POST
      - PUT
      - DELETE
      - PATCH
  authenticators:
    - handler: cookie_session
    - handler: bearer_token
  authorizer:
    handler: allow
  mutators:
    - handler: header
```

### AuthMiddleware — stato attuale

`AuthMiddleware` in `middleware/auth.go` legge `X-User-ID`, valida che sia un UUID valido, e lo salva nel context Gin **come stringa**:

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        userIDStr := c.GetHeader("X-User-ID")
        if userIDStr == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
            return
        }
        if _, err := uuid.Parse(userIDStr); err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
            return
        }
        c.Set("UserID", userIDStr)  // stringa, non uuid.UUID
        c.Next()
    }
}
```

**Nota:** Il valore e' salvato come `string` (non `uuid.UUID`) perche' tutti gli handler esistenti fanno `c.MustGet("UserID").(string)`. Cambiare il tipo romperebbe l'intera codebase.

### Modifica al router Go (gia' fatta)

```go
// router.go — hooks FUORI dal gruppo autenticato
hooks := r.Group("/api/hooks")
hooks.POST("/registration", authHandler.HandleKratosRegistrationHook)

// Gruppo autenticato — AuthMiddleware al posto di TestAuthMiddleware
api := r.Group("/api/", middleware.AuthMiddleware(), middleware.TraceIDMiddleware())
```

### Modifica al frontend — apiClient.ts (gia' fatta)

```typescript
export const api = axios.create({
    baseURL: 'http://localhost:4455/api',   // Oathkeeper, non backend diretto
    withCredentials: true,                   // CRITICO: invia il cookie di sessione
    headers: { 'Content-Type': 'application/json' },
})
```

### Test (gia' eseguiti e verificati)

```bash
# Con session token (bearer) — 200
curl.exe -H "Authorization: Bearer {session_token}" http://localhost:4455/api/workspaces

# Senza auth — 401
curl.exe http://localhost:4455/api/workspaces
```

---

## Step 4 — Frontend: route di autenticazione

**Obiettivo:** Il frontend ha le pagine `/auth/login` e `/auth/register` che interagiscono con Kratos tramite i self-service flow, un session check al mount dell'app, e un AuthGuard che protegge le route.

### 4.0 — Contesto: cosa e' gia' implementato

File gia' pronti:

| File | Stato |
|---|---|
| `src/lib/kratosClient.ts` | Axios instance → `localhost:4433`, `withCredentials: true` |
| `src/lib/apiClient.ts` | Axios instance → `localhost:4455/api`, `withCredentials: true` |
| `src/stores/authStore.ts` | Zustand store con `user`, `isAuthenticated`, `setUser`, `clearUser` |
| `src/hooks/useKratosFlow.ts` | Hook completo con tipi, multi-step, error extraction |
| `src/types/entities.ts` | `User` type con `Id`, `Email`, `Username`, `DisplayName`, `AvatarUrl` |

### 4.1 — Kratos self-service flow: come funziona davvero

Kratos non ha form HTML. Espone un'API che descrive quali campi servono. Il frontend li renderizza.

#### Login flow (singolo step)

```
1. Frontend naviga a /auth/login

2. useKratosFlow('login') chiama:
   GET http://localhost:4433/self-service/login/browser

3. Kratos risponde con un flow object:
   {
     id: "flow-uuid",
     state: "choose_method",
     ui: {
       action: "http://localhost:4433/self-service/login?flow=flow-uuid",
       method: "POST",
       nodes: [
         { attributes: { name: "identifier", type: "email" } },
         { attributes: { name: "password", type: "password" } },
         { attributes: { name: "method", value: "password", type: "hidden" } },
         { attributes: { name: "csrf_token", value: "...", type: "hidden" } }
       ]
     }
   }

4. Il frontend mostra un form con email + password

5. L'utente compila e submitte

6. submitFlow() fa POST a flow.ui.action:
   POST http://localhost:4433/self-service/login?flow=flow-uuid
   Body: { identifier: "user@example.com", password: "...", method: "password", csrf_token: "..." }

7. Kratos valida, crea sessione:
   - Set-Cookie: ory_kratos_session=...
   - Body: { session: { identity: { id: "user-uuid", traits: { email, display_name, username } } } }

8. submitFlow() ritorna il KratosSession → il frontend popola authStore e redirect a /
```

#### Registration flow (DUE step — Kratos v1.3+)

**Questa e' la differenza chiave da capire.** In Kratos v1.3+, la registrazione con password e' un flow multi-step:

```
STEP 1 — Profilo (traits)
==========================

1. useKratosFlow('registration') chiama:
   GET http://localhost:4433/self-service/registration/browser

2. Kratos risponde con:
   {
     id: "flow-uuid",
     state: "choose_method",     <-- indica che siamo allo step 1
     ui: {
       action: "http://localhost:4433/self-service/registration?flow=flow-uuid",
       nodes: [
         { attributes: { name: "traits.email", type: "email" } },
         { attributes: { name: "traits.display_name", type: "text" } },
         { attributes: { name: "traits.username", type: "text" } },
         { attributes: { name: "method", value: "profile", type: "hidden" } },
         { attributes: { name: "csrf_token", value: "...", type: "hidden" } }
       ]
     }
   }

   NOTA: Il method e' "profile", NON "password"! Questo e' lo step 1.

3. Il frontend mostra il form con email + display_name + username

4. submitFlow() manda:
   POST .../self-service/registration?flow=flow-uuid
   Body: {
     csrf_token: "...",
     method: "profile",              <-- CRITICO: deve essere "profile" al primo step
     traits: {
       email: "user@example.com",
       display_name: "Marco Rossi",
       username: "marco"
     }
   }

5. Kratos NON crea ancora l'utente. Risponde con un NUOVO flow object:
   {
     id: "flow-uuid" (stesso),
     state: "sent_email",            <-- indica che siamo allo step 2
     ui: {
       action: "http://localhost:4433/self-service/registration?flow=flow-uuid",
       nodes: [
         { attributes: { name: "password", type: "password" } },
         { attributes: { name: "method", value: "password", type: "hidden" } },
         { attributes: { name: "csrf_token", value: "...", type: "hidden" } }
       ]
     }
   }

6. submitFlow() ritorna null (non c'e' session) e aggiorna il flow interno.
   Il frontend rileva che ora ci sono campi diversi e mostra il form password.


STEP 2 — Password
==================

7. submitFlow() manda:
   POST .../self-service/registration?flow=flow-uuid
   Body: {
     csrf_token: "...",
     method: "password",             <-- ora e' "password"
     password: "SecurePass!2026"
   }

8. Kratos crea l'identita', esegue gli after-hooks:
   a) hook: session  → auto-login, setta il cookie
   b) hook: web_hook → chiama POST /api/hooks/registration sul backend Relay

9. Kratos risponde con:
   - Set-Cookie: ory_kratos_session=...
   - Body: { session: { identity: { id: "user-uuid", traits: { ... } } } }

10. submitFlow() ritorna KratosSession → frontend popola authStore e redirect a /
```

**Come il frontend gestisce i due step:** L'hook `useKratosFlow` gestisce tutto internamente. Quando `submitFlow()` riceve un flow aggiornato (senza session), salva il nuovo flow e ritorna `null`. Quando riceve una session, ritorna il `KratosSession`. La pagina non deve conoscere i dettagli del multi-step — guarda solo se `submitFlow()` ha ritornato una session o no.

### 4.2 — Struttura file e route

Il frontend usa **TanStack Router con file-based routing**. Le route vivono in `src/routes/`, non in `src/pages/`.

```
src/routes/
  __root.tsx                           # Root layout (Outlet + Toaster + devtools)
  _app.tsx                             # App shell autenticato (WebSocket, WorkspaceSwitcher)
  _app.index.tsx                       # Redirect a ultimo workspace
  _app.workspaces.$workspaceId.tsx     # Vista workspace

  auth/                                # <-- NUOVO: route di autenticazione
    login.tsx                          # /auth/login
    register.tsx                       # /auth/register
```

**Perche' in `routes/auth/` e non in `pages/`:** TanStack Router genera automaticamente il route tree da `src/routes/`. I file in `pages/` non vengono registrati come route. I file `pages/LoginPage.tsx` e `pages/RegisterPage.tsx` esistenti sono vuoti e possono essere rimossi.

**Gerarchia route risultante:**

```
/                          → __root.tsx
├── /auth/login            → auth/login.tsx      (pubblica, no auth check)
├── /auth/register         → auth/register.tsx   (pubblica, no auth check)
└── /_app                  → _app.tsx            (protetta, richiede sessione)
    ├── /                  → _app.index.tsx
    └── /workspaces/:id    → _app.workspaces.$workspaceId.tsx
```

### 4.3 — useKratosFlow: il hook (gia' implementato)

L'hook in `src/hooks/useKratosFlow.ts` e' gia' completo. Riassunto dell'interfaccia:

```typescript
export function useKratosFlow(flowType: 'login' | 'registration') {
    // State
    const [flow, setFlow] = useState<KratosFlow | null>(null)
    const [errors, setErrors] = useState<string[]>([])
    const [isLoading, setIsLoading] = useState(false)

    // Al mount: GET /self-service/{flowType}/browser → setFlow()

    // Submit: POST a flow.ui.action con i valori
    // - Se la risposta contiene session → ritorna KratosSession
    // - Se la risposta e' un flow aggiornato → setFlow() e ritorna null
    // - Se errore con flow nella risposta → estrae errori da ui.messages e nodes
    const submitFlow = async (values: Record<string, unknown>): Promise<KratosSession | null>

    return { flow, submitFlow, errors, isLoading }
}
```

**Tipi chiave definiti nell'hook:**

```typescript
type KratosSession = {
    identity: {
        id: string
        traits: {
            email: string
            display_name: string
            username: string
        }
    }
    session_token?: string
}
```

### 4.4 — LoginPage: `/auth/login`

**File:** `src/routes/auth/login.tsx`

```
Flow:
1. useKratosFlow('login') → inizializza il flow
2. Mostra form: email + password
3. onSubmit:
   a. submitFlow({ identifier: email, password, method: "password", csrf_token })
   b. Se ritorna KratosSession:
      - authStore.setUser(mapTraitsToUser(session))
      - navigate({ to: '/' })
   c. Se ritorna null: non dovrebbe succedere per login (single step)
   d. Se errors[]: mostra sotto il form
```

**csrf_token:** Ogni flow contiene un nodo con `name: "csrf_token"` e un `value` precompilato. Va incluso in ogni submit. Per estrarlo dal flow:

```typescript
const csrfToken = flow.ui.nodes
    .find(n => n.attributes.name === 'csrf_token')
    ?.attributes.value ?? ''
```

**Mapping KratosSession → User (authStore):**

```typescript
function sessionToUser(session: KratosSession): User {
    return {
        Id: session.identity.id,
        Email: session.identity.traits.email,
        Username: session.identity.traits.username,
        DisplayName: session.identity.traits.display_name,
        AvatarUrl: null,
        CreatedAt: '',    // non disponibile da Kratos, verra' dal backend
        UpdatedAt: '',
        DeletedAt: null,
    }
}
```

**Nota:** `CreatedAt` e `UpdatedAt` non sono disponibili dalla session Kratos. Servono solo come placeholder per soddisfare il tipo `User`. Il frontend non li usa nelle pagine auth.

**UI:** Usa i componenti shadcn gia' disponibili: `Card`, `Input`, `Button`, `Label`, `Field`. Tema dark monospace coerente con il resto dell'app.

**Link a register:** In fondo al form, link a `/auth/register`.

### 4.5 — RegisterPage: `/auth/register`

**File:** `src/routes/auth/register.tsx`

Qui la complessita' e' il multi-step. Il componente deve gestire due stati diversi del form.

```
Flow:
1. useKratosFlow('registration') → inizializza il flow (state: "choose_method")
2. Rileva lo step corrente guardando i nodi del flow:
   - Se esiste un nodo con name="traits.email" → step profilo
   - Se esiste un nodo con name="password" (e NON traits.email) → step password

STEP PROFILO (primo submit):
3. Mostra form: email + display_name + username
4. onSubmit:
   a. submitFlow({
        csrf_token,
        method: "profile",
        "traits.email": email,
        "traits.display_name": displayName,
        "traits.username": username,
      })
   b. Ritorna null → il flow si aggiorna automaticamente (useKratosFlow fa setFlow)
   c. Il componente ri-renderizza e mostra lo step password

STEP PASSWORD (secondo submit):
5. Mostra form: password + conferma password (la conferma e' solo frontend)
6. onSubmit:
   a. submitFlow({
        csrf_token,
        method: "password",
        password: password,
      })
   b. Ritorna KratosSession → authStore.setUser() + navigate('/')
```

**Come rilevare lo step corrente:**

```typescript
function getCurrentStep(flow: KratosFlow): 'profile' | 'password' {
    const hasTraits = flow.ui.nodes.some(
        n => n.attributes.name?.startsWith('traits.')
    )
    return hasTraits ? 'profile' : 'password'
}
```

**Validazione username (frontend):** Il pattern nell'identity schema e' `^[a-z0-9._-]+$`, minLength 3, maxLength 30. Il frontend puo' validare prima del submit per feedback istantaneo, ma Kratos validera' comunque server-side e ritornera' errori nei nodi del flow.

**Errori:** Gli errori estratti dall'hook (`errors[]`) vengono mostrati sopra il form. Kratos ritorna errori come:
- "The password has been found in data breaches..." (password troppo comune)
- "Property email is missing" (campo obbligatorio)
- "Does not match pattern..." (username non valido)

### 4.6 — traits come campi flat vs nested

**Dettaglio importante:** Quando Kratos renderizza i nodi del flow di registrazione, i campi dei traits hanno nomi "flat" con dot notation:

```json
{ "attributes": { "name": "traits.email" } }
{ "attributes": { "name": "traits.display_name" } }
{ "attributes": { "name": "traits.username" } }
```

Nel body del POST, questi campi possono essere inviati in due modi equivalenti:

```typescript
// Flat (dot notation) — consigliato, piu' semplice
{
    "csrf_token": "...",
    "method": "profile",
    "traits.email": "user@example.com",
    "traits.display_name": "Marco Rossi",
    "traits.username": "marco"
}

// Nested (oggetto traits) — funziona ugualmente
{
    "csrf_token": "...",
    "method": "profile",
    "traits": {
        "email": "user@example.com",
        "display_name": "Marco Rossi",
        "username": "marco"
    }
}
```

Kratos accetta entrambi. La dot notation e' piu' semplice perche' non richiede di ricostruire l'oggetto.

### 4.7 — Session check al mount dell'app

Quando l'utente apre l'app (o fa refresh), il frontend deve verificare se ha gia' una sessione Kratos attiva. Questo si fa con `GET /sessions/whoami`.

**Dove:** Nel `beforeLoad` della route layout `_app.tsx`, usando TanStack Router.

```typescript
// _app.tsx
export const Route = createFileRoute('/_app')({
    beforeLoad: async () => {
        try {
            const res = await kratosClient.get('/sessions/whoami')
            const session = res.data as KratosSession
            useAuthStore.getState().setUser(sessionToUser(session))
        } catch {
            // Nessuna sessione valida → redirect a login
            throw redirect({ to: '/auth/login' })
        }
    },
    component: AppShell,
})
```

**Perche' `beforeLoad` e non `useEffect`:**
- `beforeLoad` esegue PRIMA del render del componente e dei suoi figli
- Se non c'e' sessione, il redirect avviene prima che qualsiasi componente tenti di fare API call
- Evita flash di contenuto non autenticato
- I loader dei figli (`_app.index.tsx`, workspace route) non vengono eseguiti se `beforeLoad` fallisce

**Perche' non nel root `__root.tsx`:**
- Le route `/auth/login` e `/auth/register` devono essere accessibili senza sessione
- Solo le route sotto `_app` richiedono autenticazione
- `__root.tsx` renderizza solo `<Outlet />` + `<Toaster />`, non ha logica auth

### 4.8 — Route auth: non protette

Le route `/auth/login` e `/auth/register` vivono sotto `src/routes/auth/`, che e' un percorso separato da `_app`. Non passano per il `beforeLoad` di `_app.tsx`, quindi sono accessibili senza sessione.

Se l'utente e' GIA' autenticato e naviga a `/auth/login`, e' buona pratica fare redirect a `/`:

```typescript
// auth/login.tsx
export const Route = createFileRoute('/auth/login')({
    beforeLoad: async () => {
        try {
            await kratosClient.get('/sessions/whoami')
            // Ha gia' una sessione → redirect alla home
            throw redirect({ to: '/' })
        } catch (err) {
            // Se e' un redirect, propagalo
            if (err instanceof Error && 'to' in err) throw err
            // Altrimenti: nessuna sessione, procedi normalmente
        }
    },
    component: LoginPage,
})
```

### 4.9 — Flusso completo login (riepilogo visivo)

```
Browser                          Frontend                     Kratos :4433
  |                                  |                            |
  | naviga /auth/login               |                            |
  |--------------------------------->|                            |
  |                                  | GET /self-service/login/browser
  |                                  |--------------------------->|
  |                                  |    { id, ui: { nodes } }  |
  |                                  |<---------------------------|
  |    form (email + password)       |                            |
  |<---------------------------------|                            |
  |                                  |                            |
  | compila + submit                 |                            |
  |--------------------------------->|                            |
  |                                  | POST /self-service/login?flow=id
  |                                  | { identifier, password, method, csrf_token }
  |                                  |--------------------------->|
  |                                  |    Set-Cookie + session    |
  |                                  |<---------------------------|
  |                                  |                            |
  |                                  | authStore.setUser(traits)  |
  |                                  | navigate('/')              |
  |                                  |                            |
  |    redirect /                    |                            |
  |<---------------------------------|                            |
  |                                  |                            |
  | _app.tsx beforeLoad              |                            |
  |--------------------------------->|                            |
  |                                  | GET /sessions/whoami       |
  |                                  |--------------------------->|
  |                                  |    200 OK (cookie valido)  |
  |                                  |<---------------------------|
  |                                  |                            |
  |    AppShell renderizza           |                            |
  |<---------------------------------|                            |
```

### 4.10 — Flusso completo registrazione (riepilogo visivo)

```
Browser                Frontend                    Kratos :4433              Backend :8080
  |                        |                            |                        |
  | /auth/register         |                            |                        |
  |----------------------->|                            |                        |
  |                        | GET /self-service/         |                        |
  |                        |   registration/browser     |                        |
  |                        |--------------------------->|                        |
  |                        |   flow (state:choose)      |                        |
  |                        |<---------------------------|                        |
  |                        |                            |                        |
  | form step 1            |                            |                        |
  | (email,name,username)  |                            |                        |
  |<-----------------------|                            |                        |
  |                        |                            |                        |
  | submit step 1          |                            |                        |
  |----------------------->|                            |                        |
  |                        | POST method:"profile"      |                        |
  |                        | + traits                   |                        |
  |                        |--------------------------->|                        |
  |                        |   flow (state:sent_email)  |                        |
  |                        |<---------------------------|                        |
  |                        |                            |                        |
  | form step 2            |                            |                        |
  | (password)             |                            |                        |
  |<-----------------------|                            |                        |
  |                        |                            |                        |
  | submit step 2          |                            |                        |
  |----------------------->|                            |                        |
  |                        | POST method:"password"     |                        |
  |                        | + password                 |                        |
  |                        |--------------------------->|                        |
  |                        |                            |                        |
  |                        |                            | [crea identity]        |
  |                        |                            | [hook: session]        |
  |                        |                            | [hook: web_hook] ----->|
  |                        |                            |                        | POST /api/hooks/registration
  |                        |                            |                        | produce Command su Kafka
  |                        |                            |                        | Consumer crea UserSnapshot
  |                        |                            |<----- 200 OK ---------|
  |                        |                            |                        |
  |                        |   Set-Cookie + session     |                        |
  |                        |<---------------------------|                        |
  |                        |                            |                        |
  |                        | setUser() + navigate('/')  |                        |
  |                        |                            |                        |
  | redirect /             |                            |                        |
  |<-----------------------|                            |                        |
```

### 4.11 — Componenti UI consigliati

Il frontend ha gia' questi componenti shadcn in `src/components/ui/`:

| Componente | Uso nelle pagine auth |
|---|---|
| `Card` + `CardHeader` + `CardContent` | Wrapper del form |
| `Input` | Campi email, password, username, display_name |
| `Button` | Submit |
| `Label` | Label dei campi |
| `Field` | Wrapper campo + errore (se disponibile) |

**Stile:** Il tema dell'app e' dark-only, monospace (JetBrains Mono), denso. Le pagine auth devono essere coerenti: sfondo `bg-relay-base`, testo `text-relay-primary`, accent `relay-accent-blue`.

**Layout suggerito:**

```
+-----------------------------------------------+
|                                                |
|         [logo/titolo Relay]                    |
|                                                |
|         +---------------------------+          |
|         | Card                      |          |
|         |                           |          |
|         |  Email    [__________]    |          |
|         |  Password [__________]    |          |
|         |                           |          |
|         |  [  Login  ]              |          |
|         |                           |          |
|         |  Non hai un account?      |          |
|         |  Registrati               |          |
|         +---------------------------+          |
|                                                |
+-----------------------------------------------+
```

Centrato verticalmente e orizzontalmente, max-width ~400px.

### Test Step 4

1. Naviga a `localhost:5173` → redirect automatico a `/auth/login` (nessuna sessione)
2. Clicca "Registrati" → naviga a `/auth/register`
3. Compila email + display_name + username → submit step 1
4. Form cambia mostrando solo password → submit step 2
5. Redirect a `/` → sessione attiva, workspace caricati (o pagina "nessun workspace")
6. Refresh pagina → `GET /sessions/whoami` conferma sessione, nessun re-login
7. Naviga manualmente a `/auth/login` → redirect a `/` (gia' autenticato)

---

## Step 5 — WebSocket: autenticazione al connection upgrade

**Obiettivo:** Il WebSocket upgrade passa per Oathkeeper, che valida il cookie e inietta `X-User-ID`. Il `WsHandler` legge l'header dal context Gin.

### Come funziona

La connessione WebSocket inizia come una request HTTP GET che viene "upgraded" a WebSocket. Questa request iniziale porta i cookie del browser. Oathkeeper la intercetta come una normale HTTP request.

```
Frontend
  |
  | new WebSocket('ws://localhost:4455/api/ws?connectionId=...')
  | (il browser invia automaticamente i cookie per localhost)
  |
  v
Oathkeeper :4455
  |
  | match: relay-ws rule (url: /api/ws, method: GET)
  | auth: cookie_session → Kratos /sessions/whoami → 200 OK
  |        (oppure bearer_token se presente)
  | mutator: header → X-User-ID: user-uuid
  | forward upgrade request
  |
  v
Backend :8080 /api/ws
  |
  | AuthMiddleware legge X-User-ID → c.Set("UserID", "user-uuid")
  |
  v
WsHandler.ServeWebsocket
  |
  | userIDStr := c.MustGet("UserID").(string)
  | userID, _ := uuid.Parse(userIDStr)
  | Upgrade HTTP → WS
  | Hub registra il Client con userID
```

### Modifica frontend — WebSocket URL

**File:** `src/hooks/useDomainWebSocket.ts`

Attualmente il WebSocket punta direttamente al backend:

```typescript
// PRIMA — bypass Oathkeeper, nessuna autenticazione
const WS_URL = 'ws://localhost:8080/api/ws'
```

Deve puntare a Oathkeeper:

```typescript
// DOPO — passa per Oathkeeper, cookie validato
const WS_URL = 'ws://localhost:4455/api/ws'
```

**Perche' funziona senza modifiche aggiuntive:**
- Il browser include automaticamente i cookie di `localhost` nella WebSocket handshake request
- Oathkeeper valida il cookie `ory_kratos_session` con Kratos
- Il backend riceve `X-User-ID` nel header, come per qualsiasi altra request

### Modifica backend — WsHandler

**Nessuna modifica necessaria.** Il `WsHandler` gia' legge `UserID` come stringa dal context Gin e fa `uuid.Parse()`:

```go
userIDStr := c.MustGet("UserID").(string)
userID, err := uuid.Parse(userIDStr)
```

Questo funziona identicamente sia con `TestAuthMiddleware` (che settava una stringa) sia con `AuthMiddleware` (che setta una stringa validata).

### Timing: quando il WebSocket si connette

Il WebSocket viene inizializzato in `_app.tsx` (AppShell), che e' dentro la route protetta. Questo significa:

1. `_app.tsx` `beforeLoad` → `GET /sessions/whoami` → sessione valida ✓
2. `AppShell` component monta → `useDomainWebSocket()` → apre WS
3. WS handshake → Oathkeeper valida cookie → Backend riceve `X-User-ID`

Se il `beforeLoad` fallisce (nessuna sessione), il redirect a `/auth/login` avviene PRIMA che il WebSocket tenti di connettersi. Non ci sono mai connessioni WS non autenticate.

### Sessione scaduta durante connessione WS attiva

Una volta che il WebSocket e' stabilito, il protocollo WS non ri-invia cookie su ogni messaggio. La sessione viene verificata solo al momento dell'upgrade.

**Strategia MVP:** Se la sessione Kratos scade (default: 24h), il WebSocket resta connesso. Ma il prossimo HTTP fetch (API call) fallisce con 401. L'interceptor axios (o il query error handler) rileva il 401 e:
1. `authStore.clearUser()`
2. Chiude il WebSocket
3. Redirect a `/auth/login`

**Interceptor axios consigliato:**

```typescript
// apiClient.ts
api.interceptors.response.use(
    (res) => res,
    (err) => {
        if (err.response?.status === 401) {
            useAuthStore.getState().clearUser()
            window.location.href = '/auth/login'
        }
        return Promise.reject(err)
    }
)
```

### Test Step 5

1. Login via `/auth/login` → sessione attiva
2. La pagina carica → WebSocket si connette via `ws://localhost:4455/api/ws`
3. Nel log backend: `userID` corretto (UUID Kratos dell'utente loggato)
4. Subscribe e focus funzionano normalmente
5. Test negativo: rimuovi il cookie manualmente (DevTools) → prossima API call → 401 → redirect login

---

## Step 6 — Logout

**Obiettivo:** L'utente puo' fare logout. La sessione Kratos viene invalidata, il cookie rimosso, lo stato locale pulito.

### Flusso logout di Kratos

Il logout in Kratos e' un flow a due step:

```
1. GET /self-service/logout/browser
   (con cookie di sessione — Kratos deve sapere QUALE sessione invalidare)

   Risposta: { logout_url: "http://localhost:4433/self-service/logout?token=abc123" }
   Il token e' monouso e legato alla sessione corrente.

2. GET {logout_url}
   Kratos invalida la sessione nel suo database e rimuove il cookie.

   Risposta: 204 No Content (oppure redirect se configurato browser flow)
```

**Perche' due step:** Prevenzione CSRF. Se il logout fosse un singolo GET senza token, un link malevolo potrebbe fare logout dell'utente. Il token nel `logout_url` garantisce che il logout sia intenzionale.

### Implementazione frontend

Creare un hook o utility function riusabile:

```typescript
// hooks/useLogout.ts oppure inline nel componente
async function logout() {
    try {
        // 1. Ottieni il logout URL da Kratos
        const { data } = await kratosClient.get('/self-service/logout/browser')

        // 2. Chiama il logout URL (invalida sessione + rimuove cookie)
        await kratosClient.get(data.logout_url)
    } catch {
        // Se fallisce (es. sessione gia' scaduta), procedi comunque col cleanup
    }

    // 3. Pulisci stato locale
    useAuthStore.getState().clearUser()

    // 4. Redirect a login
    // Nota: window.location.href forza un full reload, pulendo tutto lo stato in-memory
    window.location.href = '/auth/login'
}
```

**Perche' `window.location.href` e non `navigate()`:** Dopo il logout vogliamo un clean slate. `navigate()` mantiene lo stato React in memoria (store Zustand, cache React Query, WebSocket). `window.location.href` forza un full page reload che resetta tutto. Alternativa: usare `navigate()` ma fare anche `queryClient.clear()` e `wsRef.close()`.

### Dove mettere il bottone logout

Nel footer della `ChannelSidebar` o nel `WorkspaceSwitcher`, dove attualmente c'e' il placeholder user. Il design e' a discrezione, ma il posto naturale e' in basso nella sidebar.

### Test Step 6

1. Login → sessione attiva, workspace visibili
2. Click logout → redirect a `/auth/login`
3. Prova a navigare a `/` → redirect a `/auth/login` (nessuna sessione)
4. Verifica che il cookie `ory_kratos_session` sia stato rimosso (DevTools → Application → Cookies)
5. Refresh → resta su `/auth/login`

---

## Step 7 — Cleanup e test end-to-end

**Obiettivo:** Il sistema funziona end-to-end. Nessun mock, nessun TestAuthMiddleware, tutti i flussi verificati.

### Checklist

**Backend (gia' completato):**
- [x] `AuthMiddleware` in uso nel router (sostituisce `TestAuthMiddleware`)
- [x] `AuthMiddleware` salva `UserID` come `string` nel context Gin
- [x] Webhook `/api/hooks/registration` fuori dal gruppo autenticato
- [x] `WsHandler` legge `UserID` come `string` e parsa con `uuid.Parse()`
- [x] Docker Compose: kratos-db, kratos-migrate, kratos, oathkeeper
- [x] Oathkeeper: `matching_strategy: glob`, `cookie_session` + `bearer_token`
- [x] Access rules: hooks (noop) prima di api-protected (catch-all)
- [x] Identity schema: email + display_name + username
- [x] Webhook body: include username

**Frontend (da completare):**
- [ ] `src/routes/auth/login.tsx` — pagina login funzionante
- [ ] `src/routes/auth/register.tsx` — pagina register multi-step
- [ ] `_app.tsx` — `beforeLoad` con session check (`GET /sessions/whoami`)
- [ ] `_app.tsx` — `beforeLoad` popola `authStore` con dati utente
- [ ] Auth route — redirect a `/` se gia' autenticato
- [ ] `useDomainWebSocket.ts` — `WS_URL` cambiato a `ws://localhost:4455/api/ws`
- [ ] `apiClient.ts` — interceptor 401 → clearUser + redirect login
- [ ] Logout — funzione + bottone nell'UI
- [ ] Rimuovere `src/pages/LoginPage.tsx` e `src/pages/RegisterPage.tsx` (vuoti)

### Test end-to-end completo

```
1. docker compose up -d
   (kratos-db, kratos-migrate, kratos, oathkeeper, zookeeper, kafka, postgres, backend)

2. npm run dev (frontend)

3. Apri http://localhost:5173
   → _app.tsx beforeLoad: GET /sessions/whoami → 401
   → redirect a /auth/login

4. Clicca "Registrati" → naviga a /auth/register

5. Step 1: compila email + display_name + username → submit
   → Kratos valida traits
   → Flow aggiornato: mostra step password

6. Step 2: inserisci password → submit
   → Kratos crea identity
   → Kratos setta cookie ory_kratos_session
   → Kratos chiama webhook → backend produce Command su Kafka
   → Consumer crea UserSnapshot + EventRecord
   → Frontend riceve KratosSession
   → authStore.setUser(traits)
   → navigate('/')

7. _app.tsx beforeLoad: GET /sessions/whoami → 200 OK
   → Conferma authStore
   → AppShell monta

8. WebSocket si connette: ws://localhost:4455/api/ws
   → Oathkeeper valida cookie → X-User-ID iniettato
   → Backend logga userID corretto
   → "Connect" ack ricevuto

9. _app.index.tsx loader: GET /api/workspaces → lista (probabilmente vuota)
   → Mostra "Nessun workspace disponibile" oppure redirect

10. Crea un workspace via "+" button
    → POST /api/workspaces → Oathkeeper → Backend
    → Workspace creato, RT event ricevuto via WebSocket

11. Naviga nel workspace → canali visibili, Subscribe inviato via WS

12. Apri altra tab → http://localhost:5173
    → GET /sessions/whoami → 200 OK (stesso cookie)
    → Stessi workspace visibili

13. Logout dalla prima tab
    → Cookie rimosso, stato locale pulito
    → Redirect a /auth/login

14. Nella seconda tab: prossima azione (es. crea canale)
    → API call → Oathkeeper → 401 (cookie invalidato)
    → Interceptor axios → clearUser → redirect /auth/login
```

---

## Riepilogo architettura finale

```
                                   +-------------------+
                                   |   Kratos :4433    |
                                   |   Identity Server |
                                   |                   |
                  +-- login/reg -->|  • session mgmt  |
                  |   (browser     |  • password hash  |
                  |    flows)      |  • webhook on reg |
                  |                +--------+----------+
                  |                         |
                  |                         | whoami (session check)
                  |                         | chiamato da Oathkeeper
                  |                         | E dal frontend (beforeLoad)
                  |                         |
Frontend :5173   |               +---------v----------+
  |               |               |  Oathkeeper :4455  |
  |               |               |  Reverse Proxy     |
  +-- API req ---------------------->                  |
  |                               |  auth:             |
  +-- WS upgrade ------------------->  cookie_session  |
                                  |  + bearer_token    |
                                  |                    |
                                  |  mutator:          |
                                  |  X-User-ID header  |
                                  +---------+----------+
                                            |
                                            v
                                  +---------+----------+
                                  |  Backend Go :8080  |
                                  |                    |
                                  |  AuthMiddleware    |
                                  |  legge X-User-ID   |
                                  |  (stringa UUID)    |
                                  |                    |
                                  |  ABAC check        |
                                  |  Event Store       |
                                  |  Kafka             |
                                  +--------------------+
```

### Porte

| Servizio | Porta | Chi lo chiama | Protocollo |
|---|---|---|---|
| Kratos public | 4433 | Frontend (login/register/logout/whoami), Oathkeeper (whoami) | HTTP |
| Kratos admin | 4434 | Solo interno | HTTP |
| Oathkeeper proxy | 4455 | Frontend (API + WebSocket) | HTTP + WS |
| Backend Go | 8080 | Oathkeeper, Kratos webhook | HTTP + WS |
| Frontend | 5173 | Browser | HTTP |

### Flusso dati identita'

```
Kratos identity.id -----> Oathkeeper X-User-ID -----> AuthMiddleware c.Set("UserID")
        |                 (cookie_session o            (string, validato UUID)
        |                  bearer_token)                       |
        |                                                      v
        |                                               ABAC subject_attrs
        |                                               EventRecord metadata.user_id
        |                                               Kafka Command metadata.user_id
        |
        +---> Frontend authStore.user.Id
        +---> Relay user_snapshots.id
        +---> Relay workspace_views.user_id
        +---> Kafka topic realtime.users partition key
```

Un singolo UUID attraversa tutto il sistema: dall'identita' Kratos fino al topic Kafka real-time. Nessuna tabella di mapping.
