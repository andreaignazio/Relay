# Slack Clone — Decisioni architetturali

**Go · React · Kafka · PostgreSQL**
**Marzo 2026 — Sessione di design**

Questo documento raccoglie tutte le decisioni prese durante la sessione di design, organizzate per area. Integra i documenti già esistenti nel progetto (architecture, architecture-revised).

---

## 1. Entity Model — MVP

### Scope

11 entità per l'MVP. Unica entità esclusa: **Custom Emoji** (post-MVP — le emoji Unicode native coprono i casi d'uso iniziali).

### Decisioni chiave

- **Canali e DM unificati** nella stessa entità `Channel` con campo `type` (public | private | dm). Un DM è un canale con tipo "dm" e partecipanti fissi. Semplifica la logica dei messaggi: tutto passa per la stessa tabella e lo stesso flusso event-sourced.

- **Multi-workspace** fin dall'inizio, sia nel modello dati che nella UI. Il `workspace_id` è presente su Channel, Notification, Invite e indirettamente su tutte le entità figlie.

- **User Status per workspace** — modellato direttamente sulla `WorkspaceMembership` (campi `status_emoji`, `status_text`, `status_expires_at`) anziché in tabella separata.

- **Thread via parent_message_id** — nullable su Message. Campi denormalizzati `reply_count` e `last_reply_at` sul messaggio padre per evitare COUNT query.

- **Notification polimorfiche** — `reference_id` + `reference_type` per puntare a qualsiasi entità sorgente senza colonne FK multiple.

### Lista entità MVP

1. User
2. Workspace
3. WorkspaceMembership
4. Channel
5. ChannelMembership
6. Message
7. Reaction
8. Attachment
9. Notification
10. Invite
11. (User Status — vive su WorkspaceMembership)

### Post-MVP

- Custom Emoji (id, workspace_id, name, image_url, created_by, created_at)
- Pinned Message
- Bookmark
- User Preferences
- Webhook / Bot
- Audit Log Entry (già previsto nell'architettura, derivato dalle action key)

---

## 2. Action Key

### Formato

```
resource:parent:verb
```

Tutto singolare. Underscore per entità composte (`channel_member`).

### Significato dei segmenti

- **resource** — l'entità su cui si agisce
- **parent** — il contesto strutturale nel modello dati (dove vive l'entità nella gerarchia del dominio). NON è una decisione di autorizzazione — la policy engine ABAC decide autonomamente in base al contesto completo (subject attrs + resource attrs)
- **verb** — l'azione eseguita. Set tipizzato ma non rigidamente CRUD. Verbi domain-specific quando il CRUD generico non cattura la semantica

### Gerarchia dei parent

| Entità | Parent | Motivazione |
|--------|--------|-------------|
| workspace | platform | Root della gerarchia |
| user | platform | Entità globale |
| member | workspace | La membership vive nel workspace |
| channel | workspace | Il canale appartiene al workspace |
| channel_member | channel | L'iscrizione vive nel canale |
| message | channel | Il messaggio appartiene al canale |
| reaction | message | La reaction è su un messaggio |
| attachment | message | L'allegato è su un messaggio |
| notification | user | La notifica è per un utente |
| invite | workspace | L'invito è per un workspace |
| status | workspace | Lo stato è per-workspace |

### Catalogo

49 action key totali: 33 write, 16 read. Le operazioni read (path sincrono via HTTP) non producono eventi Kafka.

Il catalogo completo con partition key e note è nel documento `slack-clone-conventions.docx`.

---

## 3. Partition Key

### Convenzione

**Commands:** `cmd:<service>:<id>` — l'ID è lo scope di ordinamento, tipicamente `workspace_id`.

**Events:** `evt:<parent-entity>:<parent-id>` — il parent è l'aggregato che definisce l'ordinamento.

### Albero

| Operazione su | Partition key | Motivazione |
|---------------|--------------|-------------|
| workspace | `evt:workspace:<workspace_id>` | Root — è il proprio aggregato |
| channel | `evt:workspace:<workspace_id>` | Creare un canale modifica la struttura del workspace |
| member | `evt:workspace:<workspace_id>` | La membership è parte della struttura del workspace |
| invite | `evt:workspace:<workspace_id>` | L'invito è un'operazione sul workspace |
| status | `evt:workspace:<workspace_id>` | Lo stato vive nel contesto del workspace |
| channel_member | `evt:channel:<channel_id>` | L'iscrizione modifica la struttura del canale |
| message | `evt:channel:<channel_id>` | Il messaggio modifica il contenuto del canale |
| reaction | `evt:channel:<channel_id>` | La reaction modifica come il messaggio appare nel canale |
| attachment | `evt:channel:<channel_id>` | L'allegato è parte del messaggio nel canale |
| notification | `evt:user:<user_id>` | Il contesto di ordinamento è l'utente destinatario |
| user | `evt:user:<user_id>` | Operazioni sul profilo globale |

### Pre-generazione ID

Alla creazione di una nuova entità, l'UUID viene generato dal chiamante (command handler / API gateway) prima di inviare il comando su Kafka. L'ID esiste già prima che l'entità sia persistita.

### Nota su reaction e attachment

Il parent strutturale è il messaggio, ma la partition key è `evt:channel:<channel_id>`. Il consumer che aggiorna la materialized view del canale deve vedere reaction, attachment e messaggi in ordine.

---

## 4. Event Store

### Struttura

```sql
CREATE TABLE event_store (
    event_id        UUID PRIMARY KEY,
    aggregate_type  VARCHAR(50)  NOT NULL,
    aggregate_id    UUID         NOT NULL,
    version         INTEGER      NOT NULL,
    action_key      VARCHAR(100) NOT NULL,
    trace_id        VARCHAR(64)  NOT NULL,
    payload         JSONB        NOT NULL,
    metadata        JSONB        NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    UNIQUE (aggregate_id, version)
);

CREATE INDEX idx_events_aggregate ON event_store (aggregate_id, version ASC);
CREATE INDEX idx_events_trace     ON event_store (trace_id);
```

### Payload

Contiene i dati dell'evento (dominio), NON l'intero stato dell'aggregato. Un evento di rename porta `old_name` e `name`, non tutto il workspace. Il payload è dominio puro.

### Metadata

Struttura con campo obbligatorio + extra libero:

```go
type MessageMetadata struct {
    UserID string         `json:"user_id"`
    Extra  map[string]any `json:"extra,omitempty"`
}
```

- **Obbligatorio:** `user_id` (chi ha eseguito l'azione — audit)
- **Extra (libero):** `ip`, `user_agent`, e qualsiasi campo futuro

Il `trace_id` è colonna diretta sulla riga, fuori dal JSONB, per query di debug indicizzate.

### Version e optimistic concurrency

Il vincolo UNIQUE su `(aggregate_id, version)` gestisce la concorrenza. Se due comandi concorrenti sullo stesso aggregato provano a scrivere la stessa version, uno fallisce con constraint violation e riprova.

---

## 5. Snapshot

### Pattern: dual write transazionale

Ogni operazione write esegue, nella stessa transazione PostgreSQL:
1. INSERT evento nell'event store
2. UPSERT dello snapshot

Solo dopo il commit il service produce l'event_id su Kafka.

### Struttura

Una tabella per tipo di aggregato, con colonne tipizzate (non JSONB). Ogni snapshot table ha i campi dell'entità corrispondente + `version` (INTEGER) + `updated_at` (TIMESTAMPTZ).

### Snapshot table necessarie (7 su 11)

- `workspace_snapshots`
- `channel_snapshots`
- `message_snapshots`
- `workspace_membership_snapshots`
- `channel_membership_snapshots`
- `user_snapshots`
- `invite_snapshots`

### Senza snapshot dedicato (3)

- **Reaction** — validazione con UNIQUE constraint sulla tripletta (message_id, user_id, emoji)
- **Attachment** — validazione minima (esiste?)
- **Notification** — `is_read` è l'unico stato mutabile, `read` è idempotente

### Snapshot vs Materialized View

| | Snapshot | Materialized View |
|---|---------|-------------------|
| Serve a | Command side (write) | Query side (read) |
| Vive in | DB del service, stessa transazione | DB del read path, aggiornato via Kafka |
| Modellato su | Dominio — forma dell'aggregato | UI — esigenze del frontend |
| Scope | Mono-servizio | Cross-servizio (join tra aggregati) |
| Aggiornato da | Il service stesso (sincrono) | View Materializer (asincrono) |

### Replay

Non necessario in condizioni normali. Utile solo per: ricostruire snapshot corrotti, riproiettare lo stato dopo bugfix, creare nuovi tipi di snapshot da eventi storici.

### Apply pattern

Metodo Apply sullo snapshot (switch/case per action key). Idiomatico Go, tutta la logica di mutazione in un posto. Per aggregati con molti case, estrarre in metodi privati (`s.applyCreate(payload)`, `s.applyArchive()`).

---

## 6. Struct Go — Layer Shared

### KafkaDomainMessage interface

```go
type KafkaDomainMessage interface {
    Bytes() ([]byte, error)
    GetPartitionKey() PartitionKey
    GetActionKey() ActionKey
    GetMessageID() uuid.UUID
    GetAggregateID() uuid.UUID
}
```

Cambiamenti rispetto alla versione precedente:
- `Bytes()` ritorna `([]byte, error)` — niente più panic
- `GetEntityID()` diventa `GetMessageID()` + `GetAggregateID()` — concetti separati
- `GetTraceID()` rimosso dall'interfaccia — solo Command ce l'ha
- `FromBytes` resta fuori dall'interfaccia — funzioni libere per tipo (`CommandFromBytes`, `EventFromBytes`, `RTEventFromBytes`)

### Naming unificato

- **EntityID → AggregateID** ovunque. Consistente con il vocabolario event sourcing.
- **CommandID + EventID → MessageID**. Un unico ID che attraversa tutto il flusso: generato dall'handler HTTP, viaggia nel Command, diventa la PK dell'EventRecord, viaggia nel Kafka Event. Un solo filo da seguire.
- **TraceID** resta separato dal MessageID. Lifecycle diverso (il trace inizia prima del comando e finisce dopo il delivery), formato diverso (OpenTelemetry 128-bit hex vs UUID), e in scenari di retry/saga possono divergere.

### Command

```go
type Command struct {
    MessageID    uuid.UUID           `json:"message_id"`
    AggregateID  uuid.UUID           `json:"aggregate_id"`
    ActionKey    ActionKey           `json:"action_key"`
    PartitionKey CommandPartitionKey `json:"-"`
    TraceID      string              `json:"trace_id"`
    Metadata     MessageMetadata     `json:"metadata"`
    Payload      json.RawMessage     `json:"payload"`
}
```

Cambiamenti:
- `TraceID` e `Metadata` aggiunti — senza questi il consumer non può scrivere l'event store
- `Payload` da `interface{}` a `json.RawMessage` — il payload viaggia come bytes grezzi, il consumer lo deserializza nella struct corretta guidato dall'action key
- `PartitionKey` con `json:"-"` — non va nel body JSON del messaggio Kafka. Serve solo al producer per il routing alle partizioni. Il consumer non ne ha bisogno
- `CommandJSON` eliminata — non serve più la struct intermedia

### Event (Kafka message leggero)

```go
type Event struct {
    MessageID    uuid.UUID         `json:"message_id"`
    AggregateID  uuid.UUID         `json:"aggregate_id"`
    ActionKey    ActionKey         `json:"action_key"`
    PartitionKey EventPartitionKey `json:"-"`
}
```

Messaggio leggero su `messages.events`. I consumer lo ricevono e fetchano i dati completi dall'event store tramite MessageID.

### EventRecord (event store row)

```go
type EventRecord struct {
    EventID       uuid.UUID       `db:"event_id"`
    AggregateType string          `db:"aggregate_type"`
    AggregateID   uuid.UUID       `db:"aggregate_id"`
    Version       int             `db:"version"`
    ActionKey     ActionKey       `db:"action_key"`
    TraceID       string          `db:"trace_id"`
    Payload       json.RawMessage `db:"payload"`
    Metadata      MessageMetadata `db:"metadata"`
    CreatedAt     time.Time       `db:"created_at"`
}
```

`EventID` = `MessageID` dal Command. Stesso valore, naming diverso per il contesto DB.

### RTEvent (real-time delivery)

```go
type RTEventType string

const (
    RTEventTypeRich         RTEventType = "rich"
    RTEventTypeInvalidation RTEventType = "invalidation"
    RTEventTypeSidebar      RTEventType = "sidebar"
)

type RTEvent struct {
    MessageID    uuid.UUID         `json:"message_id"`
    AggregateID  uuid.UUID         `json:"aggregate_id"`
    ActionKey    ActionKey         `json:"action_key"`
    Type         RTEventType       `json:"type"`
    PartitionKey EventPartitionKey `json:"-"`
    Payload      json.RawMessage   `json:"payload,omitempty"`
}
```

Struct unica per i tre tipi di delivery. I servizi intermedi (Aggregator, Gateway) trattano tutto allo stesso modo. Solo il client finale guarda `type` per decidere come comportarsi.

### Payload RT per tipo

```go
// Rich — update immediato della UI
type RTRichPayload struct {
    // Varia per action key. Esempio per message:channel:send:
    MessageID       uuid.UUID  `json:"message_id"`
    ChannelID       uuid.UUID  `json:"channel_id"`
    ParentMessageID *uuid.UUID `json:"parent_message_id,omitempty"`
    Content         string     `json:"content"`
    Author          RTAuthor   `json:"author"`
    CreatedAt       time.Time  `json:"created_at"`
}

type RTAuthor struct {
    UserID      uuid.UUID `json:"user_id"`
    DisplayName string    `json:"display_name"`
    AvatarURL   string    `json:"avatar_url"`
}

// Invalidation — segnale di re-fetch
type RTInvalidationPayload struct {
    Resource string    `json:"resource"`
    ID       uuid.UUID `json:"id"`
}

// Sidebar hint — aggiorna contatori senza re-fetch
type RTSidebarPayload struct {
    ChannelID uuid.UUID `json:"channel_id"`
    Hint      string    `json:"hint"`
}
```

---

## 7. Flusso HTTP → Kafka → Consumer

### Handler HTTP

L'handler fa solo due cose:
1. **ABAC check** — fail fast → 403
2. **Payload validation** — JSON valido, campi obbligatori → 400

Se passa, produce il Command su Kafka e ritorna **202** senza body significativo (solo `message_id`). Non legge snapshot, non fa validazione di business logic.

### Consumer Service

Il consumer fa tutto il resto:
1. Fetch messaggio da Kafka
2. Router interno usa action key per scegliere il service handler (`map[ActionKey]ServiceHandler`)
3. Router fa unmarshal del Command
4. Service handler:
   - Unmarshal payload tipizzato (da `json.RawMessage` a struct concreta)
   - Legge snapshot per validazione business logic
   - Dual write transazionale (event store + snapshot)
   - Produce Event su Kafka
5. Se nessun errore → commit offset Kafka
6. Se errore → non committa (retry)

### Conseguenza per il client

Il client scopre i fallimenti di business logic solo via WebSocket (come error event o assenza di conferma). Per l'MVP è ragionevole — i comandi che passano ABAC quasi sempre vanno a buon fine nel consumer. I fallimenti sono edge case (race condition).

---

## 8. Real-Time Delivery

### Strategia: ibrido rich + invalidation + sidebar

Non tutti gli eventi RT sono uguali. Il criterio è: "l'utente percepisce il ritardo di un re-fetch?"

**Rich event (5 action key)** — payload arricchito con dati utente, update immediato della UI:
- `message:channel:send`
- `message:channel:edit`
- `message:channel:delete`
- `reaction:message:add`
- `reaction:message:remove`

**Sidebar hint** — segnale leggero per aggiornare contatori unread nella sidebar senza re-fetch:
- `message:channel:send` → "new_message"
- `message:channel:delete` → "message_deleted"
- `reaction:message:add` → "activity"
- `channel:workspace:update` → "updated"

**Invalidation** — tutto il resto. Il client invalida la cache e fa re-fetch HTTP dalla materialized view.

**No RT** — azioni locali che non necessitano notifica:
- `user:platform:register`
- `channel_member:channel:mark_read`
- `notification:user:read` / `read_all`
- `invite:workspace:revoke`

### Tre livelli di topic RT

Il client deve ricevere eventi a tre livelli diversi:

**`user:<user_id>`** — eventi personali, indipendenti dal contesto. Il client è sempre iscritto.
- `workspace:platform:create` (conferma al creatore)
- `member:workspace:kick` (notifica al kickato)
- `invite:workspace:create` (notifica invito)

**`workspace:<ws_id>`** — eventi strutturali del workspace. Il client si iscrive quando entra in un workspace.
- `channel:workspace:create/archive/delete`
- `member:workspace:join/leave/kick`
- Sidebar hint per messaggi nei canali non aperti

**`channel:<ch_id>`** — eventi del canale attivo. Il client si iscrive quando apre un canale, si disiscrive quando lo chiude.
- Rich event per messaggi, reaction, edit, delete
- Invalidation per pin, member changes, attachment

### Sidebar: il problema dell'utente fuori dal canale

Un utente non iscritto al topic `channel:<ch_id>` deve comunque vedere i badge unread nella sidebar. La soluzione: per i messaggi, il RT Event Service produce su **due topic** — rich su `rt.channel` per chi ha il canale aperto, sidebar hint su `rt.workspace` per tutti i membri del workspace. Quando l'utente apre il canale, fa fetch HTTP per allineare lo stato iniziale e si iscrive al topic channel per i rich event.

### Arricchimento dei rich event

Il **RT Event Service** arricchisce i payload — una volta per canale, non per utente. Ha una cache locale di user profile (`user_id` → `display_name`, `avatar_url`) mantenuta consumando eventi `user:platform:update`.

### 4 Topic Kafka RT

| Topic | Partition key | Producer | Consumer |
|-------|--------------|----------|----------|
| `rt.channel` | `channel:<ch_id>` | RT Event Service | Subscription Aggregator |
| `rt.workspace` | `workspace:<ws_id>` | RT Event Service | Subscription Aggregator |
| `rt.user` | `user:<user_id>` | RT Event Service | Subscription Aggregator |
| `rt.delivery` | `user:<user_id>` | Subscription Aggregator | WebSocket Gateway |

Più `subscriptions.commands` per il flusso inverso subscribe/unsubscribe.

### Pipeline dei tre servizi RT

**RT Event Service**
- Consuma da `messages.events`
- Fetcha EventRecord dall'event store
- Usa routing table (action key → lista di route) per decidere:
  - Su quale topic produrre (rt.channel, rt.workspace, rt.user)
  - Quale tipo di RTEvent costruire (rich, invalidation, sidebar)
  - Quale payload builder usare
- Per i rich event: arricchisce con dati utente dalla cache locale
- Per le action key multi-destinazione: produce più RTEvent

**Subscription Aggregator**
- Consuma da `rt.channel`, `rt.workspace`, `rt.user`
- Per ogni messaggio, consulta la subscription list: "quali utenti sono iscritti a questo canale/workspace?"
- Produce su `rt.delivery` con partition key `user:<user_id>` — un messaggio per utente destinatario
- Riceve subscribe/unsubscribe da `subscriptions.commands`
- Per `rt.user`: copia su `rt.delivery` senza filtraggio (l'utente destinatario è già noto)

**WebSocket Gateway**
- Consumer pool che legge `rt.delivery` (poche goroutine)
- Dispatcher interno `map[user_id]connection` che smista ai WebSocket
- Quando il client manda subscribe/unsubscribe, li inoltra su `subscriptions.commands`
- Completamente stateless rispetto al dominio

### Flusso sottoscrizioni del client

```
Al connect:
  → subscribe user:<user_id>        (sempre, automatico)

Quando entra in un workspace:
  → subscribe workspace:<ws_id>     (sidebar)

Quando apre un canale:
  → subscribe channel:<ch_id>       (messaggi real-time)
  → HTTP GET per stato iniziale     (allineamento)

Quando chiude un canale:
  → unsubscribe channel:<ch_id>
```

### RT Event Routing Table completa

#### Rich + Sidebar
| Action key | rt.channel | rt.workspace |
|------------|-----------|-------------|
| `message:channel:send` | rich | sidebar (new_message) |
| `message:channel:edit` | rich | — |
| `message:channel:delete` | rich | sidebar (message_deleted) |
| `reaction:message:add` | rich | sidebar (activity) |
| `reaction:message:remove` | rich | — |

#### Invalidation su workspace
`workspace:platform:update`, `workspace:platform:delete`, `channel:workspace:create`, `channel:workspace:archive`, `channel:workspace:unarchive`, `channel:workspace:delete`, `member:workspace:join`, `member:workspace:leave`, `member:workspace:update_role`, `status:workspace:set`, `status:workspace:clear`

#### Invalidation su channel
`channel_member:channel:join`, `channel_member:channel:leave`, `channel_member:channel:update`, `message:channel:pin`, `message:channel:unpin`, `attachment:message:upload`, `attachment:message:delete`

#### Invalidation + Sidebar su channel update
`channel:workspace:update` → sidebar su rt.workspace + invalidation su rt.channel

#### Multi-destinazione
| Action key | Destinazioni |
|------------|-------------|
| `member:workspace:kick` | inv su rt.workspace + inv su rt.user (kickato) |
| `channel_member:channel:kick` | inv su rt.channel + inv su rt.user (kickato) |
| `invite:workspace:create` | inv su rt.user (invitato) |
| `invite:workspace:accept` | inv su rt.workspace + inv su rt.user (invitato) |

#### Diretto su user
`workspace:platform:create` → inv su rt.user (creatore)
`user:platform:update` → inv su rt.user
`user:platform:deactivate` → inv su rt.user

#### No RT
`user:platform:register`, `channel_member:channel:mark_read`, `notification:user:read`, `notification:user:read_all`, `invite:workspace:revoke`

---

## 9. Riepilogo topic Kafka

| Topic | Tipo | Partition key | Scopo |
|-------|------|--------------|-------|
| `messages.commands` | Commands | `cmd:<service>:<ws_id>` | Handler HTTP → Consumer Service |
| `messages.events` | Events | `evt:<parent>:<id>` | Service → consumer downstream |
| `rt.channel` | RT | `channel:<ch_id>` | RT Event Service → Aggregator |
| `rt.workspace` | RT | `workspace:<ws_id>` | RT Event Service → Aggregator |
| `rt.user` | RT | `user:<user_id>` | RT Event Service → Aggregator |
| `rt.delivery` | RT | `user:<user_id>` | Aggregator → WS Gateway |
| `subscriptions.commands` | Control | — | WS Gateway → Aggregator |

7 topic totali.
