# WebSocket Gateway — Implementazione step by step

Ogni step è testabile in isolamento prima di passare al successivo.
Il goal finale è un Gateway completo che gestisce connessioni WS, forwarda comandi su Kafka e consegna eventi dal topic `client:{clientID}`.

---

## Step 1 — Connessione raw

**Backend:** endpoint `GET /ws` che fa l'upgrade HTTP→WS, accetta la connessione, la tiene aperta.

**Frontend:** `useWebSocket.ts` chiama `new WebSocket(url)`, logga `onopen` / `onclose` / `onerror`.

**Test:** apri il browser → backend logga "client connesso". Chiudi la tab → backend logga "client disconnesso".

**Concetti:** HTTP upgrade, goroutine che tiene aperta la connessione.

---

## Step 2 — Ricezione messaggio e log

**Backend:** la connessione legge un messaggio WS grezzo e lo stampa nel log.

**Frontend:** al `onopen`, manda `JSON.stringify({ type: "ping" })`.

**Test:** il backend stampa il payload ricevuto nella console.

**Concetti:** loop di lettura sul WS (`ReadMessage`), parsing JSON grezzo.

---

## Step 3 — Hub + registrazione Client con goroutine per connessione

**Backend:** introduci `Hub` (map + `sync.RWMutex`) e `Client` (con `send chan []byte`). Alla connessione: genera `clientID`, registra il client nell'Hub, spawna `readPump` e `writePump` come goroutine separate. La `readPump` logga i messaggi ricevuti.

**Frontend:** nessuna modifica.

**Test:** apri due tab → il log mostra due `clientID` distinti. Chiudi una → solo quella viene rimossa dall'Hub.

**Concetti:** goroutine per connessione, `send chan` come bridge tra `readPump` e `writePump`, `sync.RWMutex` sulla map del Hub.

> **Nota:** questo è lo step architetturalmente più importante. Il modello "una goroutine per connessione" è la differenza tra un Hub che scala e uno che blocca.

---

## Step 4 — `readPump` forwarda su `subscriptions.commands`

**Backend:** la `readPump` deserializza i messaggi WS come `WsCommandMessage` e li produce su Kafka `subscriptions.commands` usando il producer del Hub. Per ora nessun consumer.

**Frontend:** al `onopen`, manda un `WsCommandKeySubscribe` con payload hardcoded (es. un `channelID` fisso).

**Test:** Kafka UI mostra il messaggio su `subscriptions.commands` con partition key `ws:client:{clientID}`.

**Concetti:** `WsCommandMessage`, `WsCommandPartitionKey`, produzione su Kafka da goroutine.

---

## Step 5 — Topic effimero + `kafkaConsumer` + `writePump`

**Backend:** alla connessione, crea il topic `client:{clientID}` via Kafka admin. Avvia una goroutine `kafkaConsumer` che legge dal topic e manda sul `send chan`. La `writePump` legge dal `send chan` e scrive sul WS.

**Frontend:** `onmessage` logga il payload ricevuto.

**Test:** produci manualmente un messaggio su `client:{clientID}` da Kafka UI → il messaggio appare nella console del browser.

**Concetti:** Kafka admin client (topic create), consumer goroutine, pipeline `kafkaConsumer → send chan → writePump → WS`.

---

## Step 6 — Disconnect e cleanup

**Backend:** quando la `readPump` esce (WS chiusa o errore), il cleanup:
1. `cancel()` — ferma tutte le goroutine del client
2. Produce `WsCommandKeyDisconnect` su `subscriptions.commands`
3. `hub.Unregister(client)`
4. Elimina il topic `client:{clientID}`

**Test:** chiudi la tab → Kafka UI mostra il messaggio di disconnect + il topic `client:{clientID}` sparisce.

**Concetti:** `context.CancelFunc` come segnale di shutdown, `defer` nella `readPump`, ordine delle operazioni al cleanup.

---

## Step 7 — `wsStore` e status UI

**Frontend:** `wsStore` Zustand con `status: 'connecting' | 'connected' | 'disconnected'`. `useWebSocket.ts` aggiorna lo store su `onopen`/`onclose`/`onerror`. Un componente UI mostra il badge di stato.

**Backend:** nessuna modifica.

**Test:** simula disconnessione (ferma il backend) → il badge UI passa a "disconnected". Riavvia → torna "connected" (con reconnect logic o manuale).

**Concetti:** Zustand store, `useWebSocket` come hook singleton, stato WS nel client.

---

## Step 8 — Subscribe iniziale batch al connect

**Frontend:** al `onopen`, manda `WsCommandKeySubscribe` con la lista completa dei canali dell'utente (fetch da TanStack Query prima della connessione, o al mount).

**Backend:** la `readPump` processa il batch, produce N messaggi su `subscriptions.commands`.

**Test:** Kafka UI mostra N messaggi subscribe con i channel ID corretti all'apertura della connessione.

**Concetti:** coordinamento tra TanStack Query (dati iniziali) e `useWebSocket` (subscribe al connect).

---

## Prossimi step (post-Gateway)

| Step | Feature |
|---|---|
| 9 | Subscription Aggregator — actor pattern, maps, consume `subscriptions.commands` |
| 10 | Aggregator produce su `client:{clientID}` → frontend riceve primo RT event reale |
| 11 | Presenza — focus, snapshot, propagazione agli altri client |
| 12 | RT events routing nel frontend (`useRTEvents.ts`) |
