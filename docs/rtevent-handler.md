# RT Event Handler

*Marzo 2026 — Design session*

Questo documento descrive l'architettura del `rteventshandler`: il servizio che consuma gli eventi di dominio da Kafka e produce gli RT event sui topic real-time (`realtime.workspaces`, `realtime.channels`, `realtime.users`).

---

## Il problema che risolve

Quando un'azione utente produce un evento di dominio (es. `message:channel:send`), il frontend deve aggiornarsi. Esistono tre strategie di aggiornamento per il frontend, ognuna con un costo diverso:

| Tipo | Comportamento frontend | Costo |
|---|---|---|
| **Rich** | Aggiorna la UI direttamente con i dati dell'evento | Nessun refetch |
| **InvalidateCache** | Invalida una query, triggera un refetch HTTP | 1 request HTTP |
| **Hint** | Aggiorna un contatore/badge senza refetch | Nessun refetch |

Il `rteventshandler` decide quale strategia usare per ogni `ActionKey` e su quale topic RT pubblicare l'evento.

---

## Separazione delle responsabilità

Il handler è strutturato in quattro file con responsabilità nette e non sovrapposte:

```
rteventshandler/
├── rteventshandler.go   // orchestratore puro
├── router.go            // dove va + che tipo di evento
├── builder.go           // cosa contiene il payload
└── rtpayloads.go        // definizioni dei tipi di payload
```

### Principio guida

> Chi è coinvolto in un evento è un **fatto di dominio** — lo sa il service.
> Come il frontend deve reagire è una **decisione UI** — la sa solo l'rteventshandler.

Questa distinzione governa dove vive ogni pezzo di informazione.

---

## EntityContext — il contratto tra service e handler

Ogni evento di dominio su Kafka porta un campo `EntityContext`:

```go
EntityContext map[EntityKeys]uuid.UUID
```

Il service popola questo campo con tutti gli aggregati che conosce già in memoria al momento della creazione dell'evento — zero query aggiuntive.

**Esempi:**

```go
// message:channel:send
EntityContext: map[EntityKeys]uuid.UUID{
    EntityKeysWorkspace: workspaceID,
    EntityKeysChannel:   channelID,
    EntityKeysUser:      userID,
}

// workspace:platform:create
EntityContext: map[EntityKeys]uuid.UUID{
    EntityKeysWorkspace: workspaceID,
    EntityKeysUser:      userID,
}
```

Il service non sa niente di topic RT, tipi di evento, o comportamento frontend. Sa solo quali entity sono coinvolte — fatto di dominio puro.

L'`rteventshandler` usa `EntityContext` per ricavare il `partitionID` per ogni route, senza mai fare unmarshal del payload dell'evento per scopi di routing.

---

## router.go — tabella di routing statica

`router.go` contiene la **tabella statica** che mappa ogni `ActionKey` alle sue route RT:

```go
type RouteSpec struct {
    Topic     RealTimeDestination // workspaces | channels | users
    EventType RTEventType         // rich | invalidate_cache | hint
}

var routeTable = map[ActionKey][]RouteSpec{
    ActionKeyMessageSend: {
        {RealTimeDestinationChannels,   RTEventTypeRich},
        {RealTimeDestinationWorkspaces, RTEventTypeHint},
    },
    ActionKeyChannelCreate: {
        {RealTimeDestinationWorkspaces, RTEventTypeInvalidateCache},
    },
    // ...
}
```

Ogni `ActionKey` può produrre **zero, una o più route** — una per ogni topic destinatario. Ad esempio, `message:channel:send` produce sia un evento `rich` sul topic canale (per aggiornare la message list di chi è nel canale) sia un evento `hint` sul topic workspace (per aggiornare i badge unread nella sidebar).

Il mapping `Topic → EntityKeys` permette di ricavare il partition ID dall'`EntityContext` senza unmarshal:

```go
var topicToEntity = map[RealTimeDestination]EntityKeys{
    RealTimeDestinationWorkspaces: EntityKeysWorkspace,
    RealTimeDestinationChannels:   EntityKeysChannel,
    RealTimeDestinationUsers:      EntityKeysUser,
}
```

**Regola:** `router.go` non fa mai unmarshal del payload. Lavora solo su `ActionKey`, `EntityContext`, e costanti statiche.

---

## builder.go — costruzione del payload

`builder.go` è l'**unico punto del sistema** in cui avviene unmarshal del payload di dominio o query al database.

Ogni `RTEventType` ha il suo builder:

### InvalidateCache
Passthrough diretto del payload del domain event. Il frontend sa già quale risorsa deve refetchare perché il payload contiene gli ID necessari.

### Rich
Fa le query necessarie per costruire un payload completo che il frontend può applicare direttamente alla UI senza refetch.

Per `message:channel:send` / `message:channel:edit` / `message:channel:delete`:
1. Query `MessageSnapshot` tramite `AggregateID` → `Content`, `ChannelID`, `ParentMessageID`, `CreatedAt`
2. Query `UserSnapshot` tramite `msg.UserID` → `DisplayName`, `AvatarURL`
3. Produce `RTRichPayload` con tutti i dati necessari al rendering

```go
type RTRichPayload struct {
    MessageID       uuid.UUID
    ChannelID       uuid.UUID
    ParentMessageID *uuid.UUID
    Content         string
    Author          RTAuthor
    CreatedAt       time.Time
}
```

Lo snapshot è garantito essere già committed quando arriva il domain event: il service fa commit su DB e poi pubblica su Kafka; il consumer Kafka arriva dopo.

### Hint
Produce un segnale leggero per aggiornare badge/contatori nella sidebar senza refetch. Contiene `ChannelID` e un `Hint` string (es. `"increment"`). Il design del hint è ancora in evoluzione.

---

## rteventshandler.go — orchestratore

L'orchestratore non contiene logica — coordina gli altri tre componenti:

```
for ogni ActionKey ricevuto:
    routes = routeTable[actionKey]

    for ogni route in routes:
        entityKey  = topicToEntity[route.Topic]
        partitionID = event.EntityContext[entityKey]
        payload    = builder.buildPayload(ctx, event, route.EventType)
        rtEvent    = assemble(event, route, partitionID, payload)
        producer(route.Topic).WriteMessage(ctx, rtEvent)
```

Se un `ActionKey` non è nella `routeTable`, l'handler lo ignora silenziosamente. Questo permette di estendere il sistema aggiungendo righe alla tabella senza toccare l'orchestratore.

---

## Aggiungere un nuovo evento RT

Per aggiungere il supporto a un nuovo `ActionKey`:

1. **`router.go`** — aggiungere la entry nella `routeTable` con i `RouteSpec` corretti
2. **`builder.go`** — se il tipo di evento è `rich`, aggiungere un case nel `buildRichPayload` switch con le query necessarie
3. **Service** — assicurarsi che l'`EntityContext` contenga tutti gli ID necessari per le route definite

Nessuna modifica a `rteventshandler.go` o `rtpayloads.go` nella maggior parte dei casi.

---

## Perché non mettere il routing nel service

Il service potrebbe, in teoria, decidere lui stesso su quale topic RT pubblicare. Questo è stato deliberatamente evitato perché:

- Il service è **command-side** — la sua responsabilità termina quando l'evento è committed su DB e pubblicato su Kafka
- Decidere "il frontend deve fare un refetch o un update diretto" è una **concern del layer di presentazione**, non del dominio
- Se il comportamento frontend cambia (es. si passa da invalidate a rich per un certo evento), la modifica deve essere in `router.go` — non distribuita nei service

Il service conosce le entity. L'handler conosce la UI. Il confine è netto.
