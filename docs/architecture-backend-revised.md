**Slack Clone**

Architectural Design Summary

*Go · React · Kafka*

Marzo 2026 — Revisione MVP

*Documento aggiornato con le revisioni architetturali per l’MVP: PostgreSQL per materialized view, WebSocket Gateway in Go, Subscription Aggregator come microservizio separato.*

# **Parte I — Overview degli obiettivi architetturali**

## **1. Stack tecnologico**

| **Backend** | Go — microservizi indipendenti |
| --- | --- |
| **Frontend** | React — client con semantica consumer Kafka |
| **Message broker** | Kafka — spina dorsale tra microservizi e gateway real-time |
| **Persistenza write** | PostgreSQL — event store append-only per ogni servizio |
| **Persistenza read** | PostgreSQL — materialized view denormalizzate (MVP) |
| **Search (futuro)** | Elasticsearch — full-text search (post-MVP; tsvector per MVP) |

*Nota MVP: Redis rimosso dallo stack iniziale. Le materialized view risiedono su PostgreSQL con tabelle denormalizzate e indici mirati. Redpanda rimosso: il delivery real-time è gestito da un WebSocket Gateway in Go.*

## **2. Layer core**

**Action key system trasversale —**stringa resource:scope:verb come contratto condiviso tra handler, ABAC, event builder e broker.

**ABAC authz con policy engine —**subject attrs + action key + resource attrs → allow/deny, policy dichiarative per action key.

**Event builder tipizzato —**ogni action key mappa a payload Go tipizzato; topic e partition key derivati automaticamente.

**Kafka broker unico —**Kafka come bus interno tra microservizi e come infrastruttura per i topic real-time verso il WebSocket Gateway.

## **3. Pattern di gestione dati distribuita**

**Event Sourcing —**su message-service e channel-service. Append-only event store come unica source of truth; stato ricostruito per replay degli eventi.

**CQRS —**separazione netta tra command side (event store) e query side (materialized view su PostgreSQL); read handler bypassa interamente il percorso write.

**Choreography Saga —**compensazioni via eventi Kafka; nessun orchestratore centrale; ogni servizio conosce solo i topic che produce e consuma.

**Snapshot DB —**stato materializzato ogni N eventi per evitare replay completo dell’event store ad ogni ricostruzione.

## **4. Comunicazione e real-time**

### **Topic e flusso write**

**messages.commands —**topic Kafka; payload: action_key + event_id; partizione per workspace_id; prodotto dal command handler dopo ABAC.

**messages.events —**topic Kafka interno; payload: event_id only; partizione per channel_id; prodotto da message-service solo dopo persist su event store.

**View Materializer —**consumer group su messages.events; legge evento per event_id dall’event store; aggiorna materialized view su PostgreSQL.

### **Delivery real-time (revisione MVP)**

**RT Event Service —**consumer group su messages.events; produce su topic Kafka channel:{id}; nessuna logica di dispatch, nessuna conoscenza degli utenti.

**Subscription Aggregator —**microservizio dedicato. Consuma dai topic channel:{id} dichiarati nella subscription list di ogni utente. Produce su topic user:{id} un singolo stream aggregato con tutti gli eventi rilevanti per quell’utente. Riceve istruzioni di subscribe/unsubscribe dal topic Kafka subscriptions.commands.

**WebSocket Gateway —**microservizio Go stateless rispetto al dominio. Gestisce connessioni WebSocket (una per client). Per ogni client consuma dal topic Kafka user:{id}. Quando il client manda subscribe/unsubscribe, li inoltra sul topic subscriptions.commands. Non contiene logica di business.

**subscriptions.commands —**topic Kafka che trasporta i comandi subscribe/unsubscribe dal Gateway all’Aggregator. Comunicazione asincrona e disaccoppiata, coerente con il pattern del resto del sistema.

**Topic user:{id} —**topic Kafka persistenti, uno per utente registrato. Creati al momento della registrazione, non alla connessione WebSocket. Il costo di topic vuoti è trascurabile.

### **Read path**

**Read path sincrono —**Read Handler chiama Message/Channel Reader Service via HTTP diretto; nessun Kafka sul percorso query.

### **Client come consumer**

Il client React mantiene una singola connessione WebSocket verso il Gateway. Il client dichiara i topic a cui vuole iscriversi (canali, DM, thread) tramite messaggi subscribe/unsubscribe sulla stessa connessione. Il Gateway inoltra le sottoscrizioni all’Aggregator via subscriptions.commands. L’Aggregator filtra gli eventi e li produce sul topic user:{id} del client. Il Gateway consuma da user:{id} e li consegna al client.

La semantica consumer è preservata: il backend non decide mai chi notificare. Il client dichiara cosa vuole ricevere, e il sistema si occupa di aggregare e consegnare. L’unica differenza rispetto al modello originale con Redpanda è che il trasporto verso il browser è un WebSocket gestito da un servizio Go, non un protocollo Kafka nativo via gateway HTTP.

## **5. Osservabilità e resilienza**

**Audit log —**derivato automaticamente dalle action key; tabella (user_id, action_key, resource_id, timestamp) senza codice extra negli handler.

**Distributed tracing (OpenTelemetry) —**trace_id che attraversa HTTP, Kafka e WebSocket; un unico filo per debuggare flussi multi-servizio.

**Dead Letter Queue + retry —**eventi non processabili finiscono in DLQ con backoff esponenziale; nessun evento perso silenziosamente.

**Rate limiting per action key —**limiti configurati per azione specifica, integrati nativamente con il sistema di action key.

**Health check + graceful shutdown —**/health su ogni servizio; SIGTERM drena i consumer Kafka prima dello spegnimento.

## **6. Infrastruttura e developer experience**

**Docker Compose locale —**tutti i servizi Go, Kafka (3 broker), Zookeeper, PostgreSQL, MinIO. Redis e Elasticsearch rimossi dall’MVP.

**Monitoring —**Kafka UI + ispezione topic e consumer group; Prometheus + Grafana per metriche lag e throughput.

# **Parte II — Architettura: le scelte e il filo che le connette**

## **Il punto di partenza**

Il progetto nasce dall’esigenza di costruire un Slack clone che fosse, fin dall’inizio, un’architettura seria — non un prototipo da riscalare in seguito, ma un sistema progettato per reggere la complessità fin dalle prime righe di codice. La scelta dello stack — Go per il backend, React per il frontend, Kafka come infrastruttura di messaggistica — è stata guidata da affinità naturale: Go per la concorrenza e la semplicità dei binari, Kafka per la natura event-driven di un sistema di messaggistica in tempo reale.

Il primo insight architetturale è arrivato ragionando su Kafka: non è solo un modo per connettere frontend e backend, ma una spina dorsale tra microservizi. Questa distinzione — Kafka come bus interno tra servizi, non solo come canale verso il client — ha cambiato radicalmente la struttura del sistema.

## **La action key: un contratto che attraversa tutto**

Il nucleo dell’architettura è la action key — una stringa gerarchica nella forma resource:scope:verb (ad esempio messages:channel:send, channels:workspace:create, users:workspace:kick). Non è solo una convenzione di naming: è il contratto condiviso tra tutti i layer del sistema.

Ogni richiesta che entra nel sistema porta con sé una action key. Questa chiave determina come il layer ABAC valuta l’autorizzazione, quale payload tipizzato costruisce l’event builder, su quale topic Kafka viene pubblicato l’evento, con quale partition key viene distribuito, e come viene scritto l’audit log. Non ci sono stringhe magiche sparse nel codice: tutto deriva dalla action key.

Il formato resource:scope:verb offre tre dimensioni naturali per l’autorizzazione ABAC — chi vuole fare cosa su quale risorsa — e garantisce che aggiungere una nuova azione significhi semplicemente definire la sua chiave: il resto del sistema sa già come trattarla.

## **HTTP e Kafka: la regola di separazione**

Una regola pratica guida tutte le scelte di comunicazione: si usa HTTP dove serve la risposta adesso per continuare, si usa Kafka dove si sta comunicando che qualcosa è successo. Questa distinzione elimina l’ambiguità in ogni punto del sistema.

Le query sono il caso più ovvio della prima categoria. Quando l’utente apre un canale e aspetta i messaggi, il Read Handler chiama il Message Reader Service via HTTP diretto — nessun Kafka in mezzo, nessun correlation ID, nessuna complessità di request-reply. La risposta torna sulla stessa connessione in modo naturale. I Reader Service rimangono servizi indipendenti e scalabili, ma il bus di messaggistica non è lo strumento giusto per il percorso sincrono.

Gli eventi scritti sull’event store, al contrario, appartengono alla seconda categoria. Quando message-service persiste un nuovo evento, non sa chi ha bisogno di saperlo — e non deve saperlo. Produce su messages.events e il sistema si occupa del resto.

La comunicazione tra WebSocket Gateway e Subscription Aggregator segue la stessa regola: il Gateway non ha bisogno di una risposta sincrona quando un client si iscrive a un canale. Sta comunicando che qualcosa è successo (un utente vuole ricevere eventi da un topic). Quindi usa Kafka — il topic subscriptions.commands — coerente con il pattern del resto del sistema.

## **Event Sourcing e CQRS: la source of truth unica**

Per message-service e channel-service, il database non contiene lo stato corrente ma la sequenza di eventi che lo ha prodotto. Ogni messaggio inviato, modificato, cancellato — ogni cambio di membership in un canale — è un evento immutabile scritto in append-only sull’event store. Lo stato corrente è una proiezione di questa sequenza.

Questa scelta porta con sé CQRS come conseguenza naturale: il percorso write passa per l’event store, il percorso read interroga le materialized view — strutture ottimizzate per le query specifiche del frontend, non per la struttura normalizzata del write. Il View Materializer ascolta messages.events, legge il nuovo evento dall’event store tramite event_id, e aggiorna la materialized view.

La decisione di tenere l’event store come unica source of truth ha implicazioni precise sul payload degli eventi Kafka interni: il topic messages.events trasporta solo event_id, non i dati dell’evento. Il View Materializer e tutti gli altri consumer vanno a rileggersi l’evento dall’event store. Questo elimina qualsiasi ambiguità su cosa credere quando i dati divergono, e garantisce che la notifica Kafka venga prodotta solo dopo che la scrittura sull’event store è andata a buon fine — quindi il consumer può sempre trovare l’evento che cerca.

Per evitare di ricostruire lo stato riproducendo migliaia di eventi ad ogni richiesta, uno Snapshot DB materializza lo stato corrente ogni N eventi. Il replay parte dallo snapshot più recente, non dall’inizio del log.

## **Materialized view su PostgreSQL (revisione MVP)**

*Questa sezione documenta una revisione rispetto al design originale che prevedeva Redis + Elasticsearch.*

Per l’MVP, le materialized view risiedono su PostgreSQL in tabelle denormalizzate con indici mirati, anziché su Redis. Questa scelta riduce la complessità operativa da tre sistemi di persistenza (PostgreSQL, Redis, Elasticsearch) a uno solo, con un unico backup da gestire e un unico failure mode da comprendere.

PostgreSQL con connection pooling (PgBouncer) e indici appropriati regge carichi di lettura più che sufficienti per l’MVP. La ricerca full-text è gestita con tsvector e tsquery nativi di PostgreSQL, rimandando Elasticsearch a quando il volume o la complessità delle query lo renderanno necessario.

La migrazione futura a Redis o a un altro datastore per le read model resta possibile senza redesign: il View Materializer è già disaccoppiato, e cambiare il datastore sottostante è un refactor localizzato.

## **Due topic Kafka, due ruoli distinti**

Il flusso write attraversa due topic Kafka con responsabilità distinte. Il primo, messages.commands, è il canale tra il command handler e il message-service: trasporta action_key e event_id, è partizionato per workspace_id, e viene prodotto dopo che ABAC ha autorizzato la richiesta. Il message-service lo consuma, esegue la business logic, scrive sull’event store.

Il secondo, messages.events, è il canale interno tra message-service e tutti i consumer downstream: trasporta solo event_id, è partizionato per channel_id per garantire l’ordinamento per canale, e viene prodotto solo dopo il persist sull’event store. Su questo topic si iscrivono il View Materializer per aggiornare le read model, il RT Event Service per il delivery real-time, e qualsiasi altro consumer futuro — notification service, search indexer, presence service — senza modificare nulla di ciò che esiste.

## **Il delivery real-time: tre servizi, tre responsabilità**

*Questa sezione documenta una revisione rispetto al design originale che prevedeva Redpanda come gateway WebSocket/SSE.*

Il percorso real-time verso il frontend è stato ripensato per eliminare la dipendenza da Redpanda e mantenere pieno controllo sull’infrastruttura, preservando la semantica del client come consumer. Il delivery è ora gestito da tre microservizi distinti, ciascuno con una responsabilità singola.

### **RT Event Service**

Consuma da messages.events e produce su topic Kafka nella forma channel:{id}. Non sa nulla degli utenti, non sa nulla del WebSocket. La sua unica responsabilità è trasformare eventi interni in eventi per canale. Scala con il throughput dei topic interni.

### **Subscription Aggregator**

Microservizio dedicato alla logica di aggregazione. Per ogni utente connesso, consuma dai topic channel:{id} dichiarati nella sua subscription list e produce su un singolo topic user:{id} l’aggregato di tutti gli eventi rilevanti.

Riceve i comandi di subscribe e unsubscribe dal topic Kafka subscriptions.commands. Non comunica direttamente con il WebSocket Gateway — conosce solo i topic che consuma e quelli che produce, coerente con il pattern dell’intero sistema.

La subscription list è dinamica: quando un utente entra in un nuovo canale o apre un thread, il client manda un subscribe sulla connessione WebSocket, il Gateway lo inoltra su subscriptions.commands, e l’Aggregator inizia a includere quel topic nello stream dell’utente. Scala con il volume di eventi da filtrare.

### **WebSocket Gateway**

Microservizio Go completamente stateless rispetto al dominio. Gestisce le connessioni WebSocket con i client (una goroutine per connessione). Per ogni client connesso consuma dal topic Kafka user:{id} e consegna gli eventi sulla connessione WebSocket.

Quando il client manda subscribe o unsubscribe, il Gateway li inoltra sul topic subscriptions.commands senza interpretarli. Non contiene logica di business, non sa quali canali esistono, non decide cosa mandare a chi. Scala con il numero di connessioni WebSocket.

Il vantaggio rispetto a Redpanda è il pieno controllo: backpressure, buffering, autenticazione sul WebSocket, metriche custom — tutto senza dipendere da un vendor su un percorso critico.

### **Il canale virtuale aggregato**

L’idea alla base di questa architettura è il “canale virtuale”: tra utente e backend esiste un singolo stream che raccoglie tutti gli eventi di cui l’utente ha bisogno, calcolato on-the-fly dall’Aggregator. Il client non gestisce decine di sottoscrizioni separate — mantiene una connessione WebSocket e riceve un flusso unificato.

I topic user:{id} sono persistenti su Kafka, creati alla registrazione dell’utente. Il costo di topic vuoti è trascurabile. La scelta di topic persistenti anziché effimeri evita race condition tra connessione e creazione del topic.

### **Comunicazione Gateway → Aggregator**

La comunicazione tra Gateway e Aggregator avviene via il topic Kafka subscriptions.commands, non via HTTP diretto. Questa scelta è coerente con il principio architetturale “Kafka dove si sta comunicando che qualcosa è successo”: il Gateway non ha bisogno di una risposta sincrona quando inoltra un subscribe. La latenza aggiuntiva è nell’ordine dei millisecondi ed è impercettibile per l’utente.

Il topic subscriptions.commands offre vantaggi operativi aggiuntivi: è osservabile, tracciabile, e permette il replay in caso di crash dell’Aggregator per ricostruire lo stato delle sottoscrizioni attive.

## **Osservabilità come conseguenza del design**

Le scelte architetturali rendono l’osservabilità quasi gratuita. L’audit log non richiede codice extra negli handler: ogni evento che attraversa il sistema porta action_key, user_id, resource_id e timestamp — basta scrivere questa tupla in una tabella dedicata come effetto collaterale della produzione su Kafka. Il distributed tracing con OpenTelemetry porta un trace_id attraverso HTTP, Kafka e WebSocket — un unico filo per ricostruire il percorso di qualsiasi richiesta attraverso tutti i microservizi.

La Dead Letter Queue gestisce i fallimenti senza perdere eventi: se un consumer è temporaneamente down, gli eventi si accumulano in DLQ e vengono riprocessati con backoff esponenziale quando il servizio torna disponibile. Il rate limiting è configurato per action key — limits semantici, non per endpoint HTTP — e il graceful shutdown garantisce che ogni servizio dreni i consumer Kafka prima di spegnersi, senza lasciare messaggi a metà elaborazione.

## **Principi trasversali**

Tre principi guidano tutte le decisioni architetturali del progetto. Il primo: la action key è l’unica source of truth per routing, autorizzazione, eventi, audit, rate limiting e tracing — aggiungere una nuova operazione significa definire la sua chiave, non toccare infrastruttura. Il secondo: ogni microservizio ignora l’esistenza degli altri — conosce solo i topic che consuma e quelli che produce. Il terzo: HTTP dove serve la risposta adesso, Kafka dove si sta comunicando che qualcosa è successo. Queste tre regole, applicate con consistenza, producono un sistema dove le scelte locali restano coerenti con l’architettura globale.

# **Appendice — Riepilogo revisioni MVP**

Questo documento incorpora le seguenti revisioni rispetto al design originale:

### **1. PostgreSQL per materialized view**

Le materialized view risiedono su PostgreSQL anziché Redis. Riduce la complessità operativa a un solo sistema di persistenza. Full-text search via tsvector/tsquery nativo. Migrazione a Redis o Elasticsearch resta possibile come refactor localizzato.

### **2. WebSocket Gateway in Go al posto di Redpanda**

Redpanda rimosso dallo stack. Il delivery real-time è gestito da un WebSocket Gateway in Go con goroutine per connessione. Pieno controllo su backpressure, autenticazione e metriche. La semantica client-as-consumer è preservata.

### **3. Subscription Aggregator come microservizio separato**

L’aggregazione degli eventi per utente è responsabilità di un microservizio dedicato, non del Gateway. I due servizi scalano indipendentemente per ragioni diverse (connessioni vs volume eventi). Coerente con il principio di responsabilità singola applicato ovunque nel sistema.

### **4. Topic user:{id} persistenti**

Un topic Kafka per utente registrato, creato alla registrazione. Persistente, non effimero. Evita race condition tra connessione e creazione del topic.

### **5. subscriptions.commands via Kafka**

La comunicazione Gateway → Aggregator avviene via topic Kafka, non HTTP. Coerente con il pattern “Kafka dove si comunica che qualcosa è successo”. Osservabile e tracciabile. Latenza trascurabile per il caso d’uso.
