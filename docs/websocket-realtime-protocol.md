# WebSocket Real-Time Protocol — Implementazione

*Marzo 2026*

Questo documento descrive il protocollo completo implementato tra frontend e backend per il layer real-time: connessione, comandi, ack, routing degli eventi. È complementare a `websocket-gateway-aggregator.md` (decisioni architetturali di design) e copre la fase di implementazione concreta.

---

## Panoramica del flusso

```
Frontend                     Gateway (Hub)              Aggregator
   │                              │                          │
   │  WS upgrade ?connectionId=X  │                          │
   ├─────────────────────────────►│                          │
   │                              │  ConnectClientChan       │
   │                              ├─────────────────────────►│ auto-subscribe UserTopic
   │                              │                          │
   │                              │◄──────────────ackChan────┤ WsCommandAck{Connect, X}
   │◄── WsCommandAck{Connect, X} ─┤                          │
   │ isConnectReady = true        │                          │
   │                              │                          │
   │  WsCommandMessage{Subscribe} │                          │
   ├─────────────────────────────►│  subscriptions.commands  │
   │                              ├─────────────────────────►│ aggiunge subscription
   │                              │                          │
   │                              │◄──────────────ackChan────┤ WsCommandAck{Subscribe, Y}
   │◄── WsCommandAck{Subscribe,Y}─┤                          │
   │ UI sbloccata                 │                          │
   │                              │                          │
   │◄── WsAggregatedEvent ────────┤◄── wsclient.{clientID} ──┤ RT event arrivato
```

---

## Connessione — connectionId come correlation ID

Il frontend non può aggiungere header HTTP custom a una WebSocket (limitazione del browser). La correlazione del Connect avviene tramite query parameter:

```
ws://host/api/ws?connectionId=<uuid>
```

Il `connectionId` è generato dal frontend con `crypto.randomUUID()` (Web Crypto API, no import necessario). È distinto dal `clientID` generato server-side — i due coesistono:

| ID | Generato da | Scopo |
|---|---|---|
| `connectionId` | Frontend | Correlazione del Connect ack |
| `clientID` | Backend (WsHandler) | Identity del client in Hub e Aggregator |

Il `connectionId` viene parsato dal `WsHandler` come `uuid.UUID` e passato al `Client` struct. Viene poi usato come `WsCommandID` nel messaggio Connect inviato su `ConnectClientChan`.

---

## Canali in-process Hub ↔ Aggregator

Il Connect e il WsCommandAck **non passano per Kafka**. Hub e Aggregator condividono due canali Go creati in `main.go` e passati a entrambi i costruttori:

```go
ackChan          := make(chan websocketcommands.AggregatorAck, 100)
connectClientChan := make(chan websocketcommands.WsCommandMessage, 100) // dentro Aggregator

// costruttori
wsAggregator = NewWsAggregator(..., ackChan)
wsHub        = NewHub(..., wsAggregator.ConnectClientChan, ackChan)
```

**Perché in-process:** Connect e Disconnect sono eventi interni al processo — non c'è motivo di aggiungere latenza Kafka per comunicazione tra due attori che vivono nello stesso binario. I comandi del frontend (Subscribe, Unsubscribe, Focus) passano ancora per Kafka perché arrivano dall'esterno via WebSocket.

### ConnectClientChan

Direzione: Hub → Aggregator.
Attivato da: `Hub.ConnectClient()` al momento della registrazione del client.
Payload: `WsCommandMessage{WsCommandKey: "Connect", WsCommandID: connectionId, Payload: ConnectCommandPayload{ClientID, UserID}}`.

L'Actor dell'Aggregator ha un case dedicato su `ConnectClientChan`, separato da `cmdChan` (Kafka).

### ackChan

Direzione: Aggregator → Hub.
Tipo: `AggregatorAck{ClientID uuid.UUID, Ack WsCommandAck}`.
Il Hub legge da `ackChan` nel suo actor loop, trova il client per `ClientID`, e scrive `json.Marshal(Ack)` su `client.send`.

---

## WsCommandID — correlazione request/ack

Ogni comando ha un `WsCommandID` (UUID) che permette al frontend di correlare la risposta alla richiesta.

### Backend

```go
// FrontendWsMessage — letto da ReadPump
type FrontendWsMessage struct {
    WsCommandKey WsCommandKey    `json:"WsCommandKey"`
    WsCommandID  uuid.UUID       `json:"WsCommandID"`
    Payload      json.RawMessage `json:"Payload"`
}

// WsCommandMessage — prodotto su Kafka
type WsCommandMessage struct {
    WsCommandKey WsCommandKey    `json:"WsCommandKey"`
    WsCommandID  uuid.UUID       `json:"WsCommandID"`
    PartitionKey PartitionKey    `json:"-"`
    Payload      json.RawMessage `json:"Payload"`
}

// WsCommandAck — inviato al frontend
type WsCommandAck struct {
    WsCommandKey WsCommandKey `json:"WsCommandKey"`
    WsCommandID  uuid.UUID    `json:"WsCommandID"`
    Success      bool         `json:"Success"`
    Error        string       `json:"Error,omitempty"`
}
```

`ReadPump` legge `WsCommandID` dal `FrontendWsMessage` e lo propaga nel `WsCommandMessage` Kafka. L'Aggregator lo legge da `command.WsCommandID` e lo passa a `sendAck`. Il casing `WsCommandID` (D maiuscolo) è uniforme in tutto il codebase.

### Frontend

```typescript
type WsCommandBase = { WsCommandID: string }
type WsCommandMessage =
  | WsCommandBase & { WsCommandKey: 'Subscribe';   Payload: SubscribePayload }
  | WsCommandBase & { WsCommandKey: 'Unsubscribe'; Payload: UnsubscribePayload }
  | WsCommandBase & { WsCommandKey: 'Focus';       Payload: FocusPayload }

type WsCommandAck = {
    WsCommandKey: WsCommandKey
    WsCommandID:  string
    Success:      boolean
    Error?:       string
}
```

Il `WsCommandID` è generato dal frontend con `crypto.randomUUID()` al momento dell'invio. Il Connect usa il `connectionId` del query param come `WsCommandID`.

---

## Protocollo messaggi in entrata

Il frontend riceve due tipi di messaggi distinti sullo stesso WebSocket:

```typescript
// RT event — prodotto dall'Aggregator su wsclient.{clientID}
type WsAggregatedEvent = { RTEvent: RTEvent }

// Command ack — inviato dall'Aggregator via ackChan → Hub → client.send
type WsCommandAck = { WsCommandKey, WsCommandID, Success, Error? }
```

### Parsing — separazione responsabilità

Il parsing è separato dalla gestione della connessione in due hook distinti:

```
useDomainWebSocket   — connessione WS, reconnect, lastMessage: MessageEvent | null
       ↓ compone
useWsMessages        — parsing, espone lastRtEvent: RTEvent | null, lastAck: WsCommandAck | null
       ↓ usato da
_app.tsx             — applica eventi, risolve ack
```

Il discriminator in `useWsMessages`:
```typescript
function parseWsMessage(data: string): ParsedWsMessage | null {
    const raw = JSON.parse(data)
    if (raw.RTEvent)                                          return { kind: 'rt_event',    event: raw.RTEvent }
    if (raw.WsCommandKey !== undefined && raw.Success !== undefined) return { kind: 'command_ack', ack: raw }
    return null
}
```

---

## Stato pending ack — wsStore

Il frontend gestisce tutti gli ack pendenti in `wsStore` (Zustand):

```typescript
type WsStore = {
    pendingAcks:    Record<string, WsCommandAck>  // WsCommandID → ack atteso
    isConnectReady: boolean
    addPendingAck:    (ack: WsCommandAck) => void
    resolveAck:       (received: WsCommandAck) => boolean  // rimuove + setta isConnectReady se Connect
    cancelPendingAck: (id: string) => void                 // rimuove senza resolvere (on WS close)
    hasPendingAck:    (key: WsCommandKey) => boolean
}
```

### isConnectReady

Flag reattivo settato a `true` quando il Connect ack viene risolto, e a `false` quando la connessione si chiude (`cancelPendingAck` in `ws.onclose`). Usato come gate nel `useEffect` che invia Subscribe:

```typescript
useEffect(() => {
    if (activeWorkspaceId && isConnectReady) sendSubscribeToWorkspace(activeWorkspaceId)
}, [activeWorkspaceId, isConnectReady])
```

Questo garantisce che Subscribe parta automaticamente quando entrambe le condizioni sono soddisfatte — anche se `activeWorkspaceId` era già settato prima che il Connect completasse.

### cancelPendingAck on close

Quando la connessione WS si chiude (prima che il backend abbia risposto), il pending Connect ack verrebbe lasciato nello store indefinitamente. `ws.onclose` chiama `cancelPendingAck(connectionId)` per rimuovere l'ack stale e resettare `isConnectReady: false`, garantendo che il reconnect parta da uno stato pulito.

### Loading derivato

Il loading globale dell'app è derivato direttamente dai pending ack — nessun sync via `useEffect`, nessuna doppia fonte di verità:

```typescript
// nessun isWaitingForAck separato — tutto deriva da pendingAcks
const isWsLoading    = useWsStore(s => Object.keys(s.pendingAcks).length > 0)
const isOtherLoading = useUIStore(s => Object.values(s.loadingStack).some(Boolean))
const isAppLoading   = isWsLoading || isOtherLoading
```

`uiStore.loadingStack` rimane disponibile per altre sorgenti di loading non-WS.

---

## Auto-subscribe UserTopic al Connect

Quando l'Aggregator riceve un Connect command da `ConnectClientChan`, esegue automaticamente:

1. Inizializza `SubscriptionsByClient[clientID] = []`
2. Aggiunge subscription `{RealTimeTopic: realtime.users, ID: userID}`
3. Crea il producer per `wsclient.{clientID}`
4. Invia `WsCommandAck{Connect, connectionId, Success: true}` su `ackChan`

Il client non deve mai inviare esplicitamente una subscription per il proprio canale utente — è garantito dall'infrastruttura alla connessione. Questo assicura che eventi user-scoped (inviti, workspace create, kick) vengano ricevuti immediatamente senza attendere una Subscribe esplicita dal frontend.

---

## Subscribe bloccato se Connect pendente

`useWsCommands.sendSubscribeToWorkspace` controlla `hasPendingAck('Connect')` prima di inviare. Se il Connect non è ancora stato confermato, il Subscribe viene scartato con un warning:

```typescript
if (hasPendingAck('Connect')) {
    console.warn('Subscribe blocked: Connect ack not yet received')
    return
}
```

In pratica il gate su `isConnectReady` nel `useEffect` di `_app.tsx` è il punto di controllo primario. Il check in `sendSubscribeToWorkspace` è una difesa secondaria per chiamate dirette fuori dal flusso standard.

---

## Sequence completa — primo caricamento

```
1. _app.tsx monta
2. useDomainWebSocket: crypto.randomUUID() → connectionId
                       addPendingAck({Connect, connectionId})
                       new WebSocket(url?connectionId=...)
3. isWsLoading = true → isAppLoading = true → UI mostra loading

4. WsHandler: parsea connectionId dal query param
              genera clientID (server-side uuid)
              crea topic wsclient.{clientID}
              NewClient(clientID, connectionId, userID, ...)
              hub.Register ← client

5. Hub actor: ConnectClient → ConnectClientChan ← {Connect, connectionId, {ClientID, UserID}}
              RunClientConsumer (goroutine per leggere da wsclient.{clientID})

6. Aggregator actor: HandleConnectCommand
                     SubscriptionsByClient[clientID] = [{realtime.users, userID}]
                     ProducerByClient[clientID] = producer(wsclient.{clientID})
                     sendAck → ackChan ← {clientID, {Connect, connectionId, Success}}

7. Hub actor: legge ackChan → client.send ← json(WsCommandAck)

8. Client WritePump: scrive WsCommandAck sul WebSocket

9. Frontend useWsMessages: parseWsMessage → {kind: 'command_ack', ack}
   _app.tsx useEffect[lastAck]: resolveAck(ack)
                                isConnectReady = true
                                pendingAcks = {}

10. isWsLoading = false → isAppLoading = false → UI sbloccata

11. useEffect[activeWorkspaceId, isConnectReady]: se workspace attivo
    sendSubscribeToWorkspace → addPendingAck({Subscribe, Y}) → WS.send(...)
    → subscriptions.commands → Aggregator → sendAck → UI sbloccata per Subscribe
```
