# WebSocket Gateway + Subscription Aggregator

*Marzo 2026 — Design session*

Questo documento raccoglie le decisioni architetturali sul layer real-time: WebSocket Gateway, Subscription Aggregator, modello di presenza e protocollo client-server.

---

## Il contesto

Il layer real-time è composto da tre servizi distinti con responsabilità singole (documentati in `architecture-backend-revised.md`):

1. **RT Event Service** — consuma `messages.events`, produce su `channel:{id}`
2. **Subscription Aggregator** — consuma `channel:{id}` in base alle subscription, produce su `client:{clientID}`
3. **WebSocket Gateway** — gestisce connessioni WS, consuma `client:{clientID}`, forwarda comandi su `subscriptions.commands`

Questo documento approfondisce le decisioni di design su Gateway e Aggregator.

---

## WebSocket Gateway — struttura Hub + Client

### Hub

Il Hub è una struttura semplice con `sync.RWMutex` su `map[clientID]*Client`. Non usa l'actor pattern — le sue operazioni sono troppo semplici per giustificarlo: aggiunge e rimuove client, e fa shutdown. Un mutex è sufficiente e più leggibile.

```
Hub
├── clients  map[uuid.UUID]*Client   // clientID → Client
├── mu       sync.RWMutex
└── producer                          // shared producer per subscriptions.commands
```

Il producer Kafka per `subscriptions.commands` vive sul Hub come dipendenza condivisa — il client riceve un riferimento in costruzione. Un solo producer condiviso tra tutte le connessioni (il producer Kafka è thread-safe).

### Client e goroutine

Ogni connessione WS ha tre goroutine che comunicano tramite un canale `send chan []byte`:

```
Kafka client:{clientID} → kafkaConsumer goroutine ─┐
                                                    ├─→ send chan []byte → writePump goroutine → WS
                    WS → readPump goroutine ─────────────────────────────→ subscriptions.commands
```

- **readPump** — legge messaggi dal WS (subscribe, unsubscribe, focus), li serializza come `WebSocketCommand` e li produce su `subscriptions.commands`
- **writePump** — legge dal canale `send`, scrive sul WS
- **kafkaConsumer** — legge dal topic `client:{clientID}`, manda sul canale `send`

Shutdown tramite `context.CancelFunc` per connessione. `Hub.Shutdown()` cancella tutti i context.

### Lifecycle di una connessione

1. HTTP upgrade → WS
2. Genera `clientID` (uuid)
3. Crea topic Kafka `client:{clientID}` (effimero)
4. Crea consumer Kafka su quel topic
5. `hub.Register(client)`
6. Spawna le tre goroutine
7. Alla chiusura WS: `cancel()` → produce `WebSocketCommandDisconnect` su `subscriptions.commands` → `hub.Unregister(client)` → elimina topic `client:{clientID}`

---

## Topic effimeri per client

Il topic di output dell'Aggregator è `client:{clientID}`, creato alla connessione e eliminato alla disconnessione — non `user:{userID}`.

**Perché clientID invece di userID:**

La scelta di passare a topic per clientID è guidata dal supporto alle connessioni multiple dello stesso utente (multi-tab). Con `user:{id}`, gestire due tab aperte richiederebbe reference counting nel Hub (invia `user_disconnected` solo quando l'ultimo client si disconnette) e introdurrebbe conflitti di focus (due tab, due canali diversi, stesso userID). Con `client:{id}`, ogni connessione è completamente autonoma: l'Aggregator traccia stato per clientID, il Gateway invia `WebSocketCommandDisconnect` ad ogni chiusura WS senza logica aggiuntiva.

I topic `user:{id}` persistenti rimangono disponibili per un futuro Notification Service (vedi sezione notifiche push).

**Sul costo dei topic effimeri:** Kafka moderno gestisce migliaia di topic senza problemi. La metadata operation di creazione/eliminazione è trascurabile per connessioni che durano minuti od ore.

---

## Subscription Aggregator — actor pattern

L'Aggregator usa l'actor pattern: una singola goroutine "manager" possiede tutte le map e non richiede mutex. Le goroutine Kafka consumer inviano comandi tramite canale Go al manager, che è l'unico a toccare lo stato.

```
subscriptions.commands consumer goroutine ─┐
channel:{id} consumer goroutines           ─┼─→ commands chan → manager goroutine (owns maps)
                                            ┘
```

**Map del manager:**

```
subscriptionsByClientID   map[clientID]set[channelID]
clientsByChannelID        map[channelID]set[clientID]
focusedChannelByClientID  map[clientID]channelID
```

La presenza a livello workspace è **derivata** internamente: un utente è presente in un workspace se almeno uno dei suoi client è iscritto a un canale di quel workspace. Il client non manda mai un `subscribe` a livello workspace — l'Aggregator deduce la presenza workspace dalle subscription ai canali.

---

## Protocollo subscribe / focus

### Semantica distinta

`subscribe` e `focus` sono segnali ortogonali con semantiche diverse:

- **subscribe** — "voglio ricevere eventi da questo canale" (unread counts, notifiche in background). Non implica presenza attiva.
- **focus** — "sto attivamente guardando questo canale adesso". Implica sempre subscribe.

Un utente può essere iscritto a 20 canali ma focused su uno solo. Le subscription seguono l'utente anche quando non sta guardando il canale (notifiche). Il focus cambia ad ogni navigazione.

### Implicazioni gestite dall'Aggregator

- `focus {channel_id: X}` → subscribe a X se non già presente + aggiorna focus map
- `subscribe {channel_id: X}` → solo subscription list, focus invariato
- `unsubscribe {channel_id: X}` → rimuove da subscription list + clear focus se era X
- `WebSocketCommandDisconnect` → rimuove tutto lo stato per quel clientID

Il client non manda mai `unfocus` esplicitamente — un nuovo focus rimpiazza il precedente. Questo riduce il numero di messaggi sul protocollo e la logica lato client.

---

## Modello di presenza

Tre granularità:

| Livello | Significato | Come si determina |
|---|---|---|
| Online | Connessione WS attiva | Hub ha il clientID |
| Presente nel workspace | Online + iscritto ad ≥1 canale del workspace | Aggregator deduce dalle subscription |
| Focused nel canale | Sta attivamente guardando quel canale | Evento `focus` esplicito dal client |

### Snapshot di presenza

Il frontend riceve tutto via WebSocket, inclusa la snapshot iniziale. L'Aggregator emette un evento `presence_snapshot` su `client:{clientID}` come effetto collaterale del processing di ogni `focus`. Questo copre sia il primo accesso al canale che ogni successiva navigazione — la snapshot arriva sempre quando serve, senza distinguere i due casi.

### Propagazione agli altri utenti

Quando l'Aggregator processa un `focus` per client A sul canale X, produce:
- `presence_snapshot` → `client:{clientID di A}` (stato corrente del canale per A)
- `presence_update` → `client:{clientID}` di ogni altro client focused/iscritto al canale X

---

## WebSocketCommandType — separazione dal sistema di action key

I messaggi su `subscriptions.commands` implementano un'interfaccia leggera (`Bytes()` + `GetPartitionKey()`), non `KafkaDomainMessage`. Vivono nel package `websocketcommands`, separato da `shared`.

**Perché non usare le action key di dominio:**

Le action key (`resource:parent:verb`) sono il contratto del sistema di dominio — entrano in ABAC, audit log, event builder, topic routing. I comandi WebSocket non passano per nessuno di questi layer. Forzarli nel sistema di action key inquinerebbe il registry con concetti infrastrutturali (`connection`, `subscription`) privi di senso nel modello di dominio, e richiederebbe aggiungere voci in `MapParent()` per risorse che non hanno un parent di dominio.

`websocketcommands` è importato sia dal Gateway (producer) che dall'Aggregator (consumer) — è il contratto condiviso tra i due servizi.

```
websocketcommands/
├── WebSocketCommandType   // string type con costanti
├── SubscribePayload
├── UnsubscribePayload
├── FocusPayload
└── DisconnectPayload
```

---

## Delivery guarantee — at-most-once (MVP)

Il Subscription Aggregator usa l'actor pattern: le goroutine consumer Kafka inoltrano i messaggi su `rtChan` o `cmdChan` e ritornano `nil` immediatamente. Il consumer committa l'offset Kafka **prima** che l'Actor abbia processato il messaggio.

Se l'Actor fallisce nella fase di processing (errore su `handleMessage` o `HandleWsCommand`), il messaggio è già committato e non verrà reprocessato — **at-most-once**.

**Perché è accettabile per MVP:** gli RT events (presenza, invalidazioni, rich events) sono notifiche best-effort. Se un evento viene perso, il frontend si riallinea alla prossima interazione, al prossimo fetch HTTP, o al prossimo evento WS. Non c'è perdita di dati persistenti — l'event store è intatto.

**Miglioramento futuro — at-least-once:** introdurre un ack channel. Il consumer invia il messaggio sul canale insieme a un `chan error`, blocca in attesa dell'ack dall'Actor, e committa solo se l'ack è nil. Questo aggiunge latenza ma garantisce che nessun messaggio venga perso silenziosamente.

---

## Notifiche push per utenti offline (nota futura)

L'Aggregator produce eventi solo per client connessi — se l'utente è offline, nessuna subscription attiva, nessun evento. Per notifiche push (mobile, email) a utenti offline, il percorso è separato:

```
messages.events → RT Event Service → channel:{id} → Aggregator → client:{id} → Gateway → WS
                                          ↓
                                 Notification Service → push/email
```

Il Notification Service consuma direttamente da `channel:{id}`, ha le preferenze di notifica su DB, e decide autonomamente. L'Aggregator non cambia. I topic `user:{id}` persistenti (creati alla registrazione) restano disponibili come alternativa di output per questo servizio.
