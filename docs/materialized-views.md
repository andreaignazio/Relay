# Relay — Materialized Views

**Go · PostgreSQL · CQRS**
**Marzo 2026**

---

## Cos'è una materialized view in questo sistema

Nel contesto di questo progetto, "materialized view" non indica le `MATERIALIZED VIEW` native di PostgreSQL (che si rinfrescano in bulk con `REFRESH MATERIALIZED VIEW`). Sono **tabelle denormalizzate** aggiornate in modo incrementale dal View Materializer ad ogni evento Kafka.

Questo è necessario per il real-time: non si può aspettare un refresh bulk ogni volta che arriva un messaggio. Il View Materializer consuma da `messages.events`, fetcha l'EventRecord dall'event store tramite `event_id`, e aggiorna la tabella corrispondente.

### Snapshot vs Materialized View

| | Snapshot | Materialized View |
|---|---|---|
| Serve a | Command side (write) | Query side (read) |
| Vive in | DB del service, stessa transazione | DB del read path, aggiornato via Kafka |
| Modellato su | Dominio — forma dell'aggregato | UI — esigenze del frontend |
| Scope | Mono-servizio | Cross-servizio (join tra aggregati) |
| Aggiornato da | Il service stesso (sincrono) | View Materializer (asincrono) |

---

## Principio guida

> Modellare le MV sull'UI, non sul dominio.

Le query key del frontend (`['workspaces']`, `['channels', wsId]`, `['messages', chId]`) definiscono esattamente quali tabelle servono e con quali campi. Il criterio per decidere la struttura di una MV è: "cosa deve renderizzare il frontend con una singola query senza JOIN?"

---

## Query `['workspaces']` — Lista workspace dell'utente

### Cosa chiede il frontend

Tutti i workspace di cui l'utente è membro, con abbastanza dati per renderizzare la lista: nome, slug, avatar, ruolo.

### Perché denormalizzare

La risposta nasce da un JOIN tra `workspace_snapshots` e `workspace_membership_snapshots`. Ma gli snapshot vivono sul command side — il read path non dovrebbe toccarli. La MV pre-computa il JOIN in una tabella ottimizzata per la chiave di accesso `user_id`.

### Trade-off: fan-out su update

Denormalizzare i dati del workspace (nome, slug, icon_url) nella tabella di membership significa che `workspace:platform:update` richiede un UPDATE su N righe (una per ogni membro). Il ragionamento:

- Le letture sono dominate dalle query degli utenti — molti utenti, molte query
- Le modifiche al workspace sono rare e privilegio di pochi admin
- Il Materializer lavora in modo asincrono — il fan-out non blocca mai il client

Conclusione: il fan-out è accettabile. Denormalizzare è l'opzione migliore.

### Struttura tabella

```sql
CREATE TABLE workspace_views (
    workspace_id UUID        NOT NULL,
    user_id      UUID        NOT NULL,
    name         TEXT        NOT NULL,
    slug         TEXT        NOT NULL,
    icon_url     TEXT,
    role         TEXT        NOT NULL,
    joined_at    TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, workspace_id)
);

CREATE INDEX idx_workspace_views_user_id ON workspace_views (user_id);
```

### Eventi che aggiornano questa tabella

| Evento | Operazione |
|---|---|
| `workspace:platform:create` | INSERT riga (il creatore diventa membro) |
| `member:workspace:join` | INSERT riga |
| `member:workspace:leave` / `kick` | DELETE riga |
| `workspace:platform:update` | UPDATE tutte le righe con quel `workspace_id` |
| `workspace:platform:delete` | DELETE tutte le righe con quel `workspace_id` |
| `member:workspace:update_role` | UPDATE `role` sulla riga specifica |

---

## Contatori unread — separati dalla lista workspace

### Perché non nella stessa tabella

I contatori unread cambiano ad ogni messaggio — frequenza altissima. I dati del workspace cambiano rarissimamente. Tenerli insieme significa aggiornare una tabella quasi-immutabile ad ogni `message:channel:send`.

La separazione segue il principio: **separare per frequenza di cambiamento**.

### Granularità: canale, non workspace

I contatori sono tenuti a livello canale (con `workspace_id` denormalizzato per filtrare senza JOIN). L'aggregazione per workspace è responsabilità del frontend (somma delle righe per `workspace_id`).

Ragionamento: se fossero pre-aggregati per workspace, ogni `message:channel:send` richiederebbe di conoscere il workspace del canale per aggiornare la riga giusta — un join implicito nel Materializer.

### Struttura tabella (da implementare)

```sql
CREATE TABLE channel_unread_view (
    user_id      UUID    NOT NULL,
    channel_id   UUID    NOT NULL,
    workspace_id UUID    NOT NULL,
    unread_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, channel_id)
);

CREATE INDEX idx_channel_unread_view_user_workspace ON channel_unread_view (user_id, workspace_id);
```

### Lifecycle degli unread

Al caricamento della pagina il frontend fetcha HTTP per lo stato iniziale. Poi i sidebar hint Kafka aggiornano solo Zustand — nessun HTTP finché la sessione è aperta.

### Workspace in background

Un utente con il workspace B non aperto deve comunque ricevere badge unread. Soluzione: l'opzione "notifica tutti i workspace" è un **comando al Subscription Aggregator** via `subscriptions.commands` — non una preferenza frontend. L'Aggregator iscrive l'utente a tutti i topic `channel:<id>` dei suoi workspace in base a questa preferenza.

---

## Query `['channels', workspaceId]` — Sidebar canali

### Query richiesta

Tutti i canali di un workspace di cui l'utente è membro (esclusi DM), con dati sufficienti per renderizzare la sidebar: nome, tipo, ruolo, stato archivio.

### Perché solo i canali joined

La sidebar mostra esclusivamente i canali a cui l'utente ha aderito — i canali pubblici non-joined sono visibili solo nella pagina "Browse channels". Questo elimina il problema del fan-out sulla creazione di un canale pubblico: se la tabella fosse user-scoped su tutti i canali visibili, ogni nuovo canale pubblico richiederebbe un INSERT per ogni membro del workspace.

### Due tabelle, due use case

**`channel_membership_view`** — sidebar (user-scoped, joined only)

```sql
CREATE TABLE channel_membership_views (
    user_id      UUID        NOT NULL,
    channel_id   UUID        NOT NULL,
    workspace_id UUID        NOT NULL,
    name         TEXT        NOT NULL,
    type         TEXT        NOT NULL,
    description  TEXT,
    topic        TEXT,
    is_archived  BOOLEAN     NOT NULL DEFAULT FALSE,
    role         TEXT        NOT NULL,
    joined_at    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, channel_id)
);

CREATE INDEX idx_channel_membership_views_user_workspace ON channel_membership_views (user_id, workspace_id);
```

**`channel_views`** — browse pubblico (workspace-scoped, nessuna dimensione utente)

```sql
CREATE TABLE channel_views (
    channel_id   UUID        NOT NULL,
    workspace_id UUID        NOT NULL,
    created_by   UUID        NOT NULL,
    name         TEXT        NOT NULL,
    type         TEXT        NOT NULL,
    description  TEXT,
    topic        TEXT,
    is_archived  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (channel_id)
);

CREATE INDEX idx_channel_views_workspace_id ON channel_views (workspace_id);
```

La query browse legge da `channel_views` (una riga per canale, nessun fan-out) e la deduplicazione rispetto ai canali già joined avviene lato frontend o API — è una query rara, il costo è accettabile.

### Fan-out su channel update

`channel:workspace:update` richiede UPDATE su N righe in `channel_membership_views` (una per membro). I canali tendono ad avere meno membri dei workspace, quindi il ragionamento già applicato alla `workspace_views` vale a maggior ragione qui.

### Eventi che aggiornano `channel_membership_views`

| Evento | Operazione |
|---|---|
| `channel:workspace:create` | INSERT riga per il creatore |
| `member:channel:join` | INSERT riga |
| `member:channel:leave` / `kick` | DELETE riga |
| `channel:workspace:update` | UPDATE tutte le righe con quel `channel_id` |
| `channel:workspace:archive` | UPDATE `is_archived` su tutte le righe |
| `channel:workspace:delete` | DELETE tutte le righe con quel `channel_id` |
| `member:channel:update_role` | UPDATE `role` sulla riga specifica |

### Eventi che aggiornano `channel_views`

| Evento | Operazione |
|---|---|
| `channel:workspace:create` | INSERT riga |
| `channel:workspace:update` | UPDATE riga |
| `channel:workspace:archive` | UPDATE `is_archived` |
| `channel:workspace:delete` | DELETE riga |

---

## Query `['dms', workspaceId]` — Sidebar DM

### Perché una tabella separata

I DM non hanno una query di browse pubblico (sono conversazioni private), quindi non serve la seconda tabella. Non esiste alcuna relazione da mantenere con `channel_views`. Una tabella separata è più semplice e più economica.

### Nessuna denormalizzazione dei partecipanti

Nei DM a più utenti non è possibile denormalizzare i partecipanti in modo sostenibile: ogni cambiamento di display name o avatar di un partecipante richiederebbe un UPDATE su tutte le sue conversazioni DM. La complessità non vale il guadagno.

### Soluzione: query che ritorna tutti i partecipanti

Invece di filtrare `WHERE user_id = ?`, la query ritorna **tutte le righe** dei canali DM di cui l'utente è membro. Il frontend riceve una struttura raggruppabile per `channel_id` e ottiene così la lista completa dei partecipanti per ogni DM.

I profili utente vengono risolti dalla cache Zustand. Se un `user_id` manca in cache, il frontend invalida e fetcha il profilo — pattern già in uso per altri tipi di dati.

### Struttura `dm_membership_views`

```sql
CREATE TABLE dm_membership_views (
    user_id      UUID        NOT NULL,
    channel_id   UUID        NOT NULL,
    workspace_id UUID        NOT NULL,
    joined_at    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, channel_id)
);

CREATE INDEX idx_dm_membership_views_user_workspace ON dm_membership_views (user_id, workspace_id);
```

### Query pattern

```sql
-- Step 1: trova i channel_id DM dell'utente
-- Step 2: ritorna tutti i partecipanti di quei canali
SELECT * FROM dm_membership_views
WHERE channel_id IN (
    SELECT channel_id FROM dm_membership_views
    WHERE user_id = ? AND workspace_id = ?
);
```

### Eventi che aggiornano `dm_membership_views`

| Evento | Operazione |
| --- | --- |
| `channel:workspace:create` (type=dm) | INSERT riga per ogni partecipante |
| `member:channel:leave` | DELETE riga |

---

## Nota implementativa: GORM e primary key composita

Le MV con primary key composita (senza campo `ID`) richiedono tag GORM espliciti. Senza questi tag, GORM non sa dove fare il conflict nell'upsert e il `clause.OnConflict{UpdateAll: true}` fallisce.

```go
// Corretto
type WorkspaceView struct {
    WorkspaceID uuid.UUID `gorm:"primaryKey" json:"WorkspaceId"`
    UserID      uuid.UUID `gorm:"primaryKey" json:"UserId"`
    // ...
}
```

Gli snapshot non hanno questo problema perché embeddano entity structs con `ID uuid.UUID` — GORM riconosce `ID` come primary key per convenzione.
