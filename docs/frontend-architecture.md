# Relay — Frontend Architecture

**Stack:** React 19 · TypeScript · Vite 8 · TanStack Router · TanStack Query · Zustand · shadcn/ui · Tailwind v4

---

## 1. Stack e responsabilità

| Layer | Libreria | Responsabilità |
|---|---|---|
| Bundler / Dev | Vite 8 + React Compiler | Build, HMR, memoizzazione automatica |
| Routing | TanStack Router | URL state, params tipizzati, navigation |
| Server state | TanStack Query | Fetch HTTP, cache, invalidazione |
| Client state | Zustand | Stato client-only (unread, WS status, UI) |
| UI components | shadcn/ui (custom preset) | Componenti headless Radix-based |
| Styling | Tailwind v4 + Relay theme | Design system dark-first monospace |
| Real-time | Custom hook WebSocket | Ricezione RT events, dispatch a Query/Zustand |

---

## 2. Struttura cartelle

```
src/
├── api/                        # fetch functions — una per dominio
│   ├── workspaces.ts
│   ├── channels.ts
│   ├── messages.ts
│   └── members.ts
│
├── components/
│   ├── ui/                     # componenti shadcn (generati, non editare)
│   ├── layout/                 # shell app (AppShell, Sidebar, Header)
│   ├── workspace/              # WorkspaceList, WorkspaceAvatar
│   ├── channel/                # ChannelList, ChannelItem, CreateChannelDialog
│   ├── message/                # MessageList, MessageItem, Composer
│   └── common/                 # Avatar, Badge, StatusDot, Kbd
│
├── hooks/
│   ├── useWebSocket.ts         # connessione WS, subscribe/unsubscribe
│   └── useRTEvents.ts          # routing RT events → Query cache / Zustand
│
├── lib/
│   ├── utils.ts                # cn() shadcn utility
│   ├── apiClient.ts            # axios instance (baseURL, interceptors, error handling)
│   ├── queryClient.ts          # istanza QueryClient condivisa (staleTime, retry)
│   └── theme/
│       └── tokens.ts           # design tokens TypeScript
│
├── pages/                      # route components (montati da TanStack Router)
│   ├── WorkspaceListPage.tsx   # / → lista workspace dell'utente
│   ├── WorkspacePage.tsx       # /:workspaceId → sidebar + placeholder
│   └── ChannelPage.tsx         # /:workspaceId/:channelId → messaggi
│
├── routes/                     # definizione route TanStack Router
│   ├── __root.tsx              # root route (provider, layout shell)
│   ├── index.tsx               # / redirect
│   ├── $workspaceId.tsx        # /:workspaceId
│   └── $workspaceId.$channelId.tsx
│
├── stores/
│   ├── workspaceStore.ts       # activeWorkspaceId, unreadCounts
│   └── wsStore.ts              # WS connection status, subscriptions attive
│
├── types/
│   ├── entities.ts             # User, Workspace, Channel, Message, ... (mirror Go)
│   ├── rtevents.ts             # RTEvent, RTRichPayload, RTInvalidationPayload, ...
│   └── actionkeys.ts           # ActionKey enum (49 action key)
│
├── index.css                   # Relay theme + Tailwind v4
└── main.tsx
```

---

## 3. Ownership dello stato

### Regola fondamentale
> TanStack Query possiede il server state. Zustand possiede il client state.
> L'UI legge da entrambi direttamente — non si copia mai da Query a Zustand.

### TanStack Query — server state

| Query key | Contenuto | Invalidata da |
|---|---|---|
| `['workspaces']` | Lista workspace dell'utente | RT: `workspace:platform:create` |
| `['workspace', wsId]` | Dettaglio workspace | RT: `workspace:platform:update` |
| `['channels', wsId]` | Lista canali del workspace | RT: `channel:workspace:create/archive/delete` |
| `['channel', chId]` | Dettaglio canale | RT: `channel:workspace:update` |
| `['messages', chId]` | Messaggi del canale | RT: rich events su message |
| `['members', wsId]` | Membri workspace | RT: `member:workspace:join/leave` |

### Zustand — client state

```ts
// workspaceStore
{
  unreadCounts: Record<string, number>  // channelId → count, aggiornato da sidebar hint
}

// wsStore
{
  status: 'connecting' | 'connected' | 'disconnected'
  subscriptions: Set<string>            // topic WS attivi
}
```

### URL — navigation state (TanStack Router)

```
/                              → redirect a WorkspaceListPage
/:workspaceId                  → workspace attivo
/:workspaceId/:channelId       → canale attivo
```

`workspaceId` e `channelId` si leggono da `useParams()` — non servono in Zustand.

---

## 4. Routing RT events

Il WebSocket riceve tre tipi di `RTEvent`. Ogni tipo ha una destinazione diversa.

```
WS message ricevuto
    │
    ├── type: "rich"
    │   └── queryClient.setQueryData(...)   → update immediato cache
    │
    ├── type: "invalidation"
    │   └── queryClient.invalidateQueries(...)  → refetch HTTP
    │
    └── type: "sidebar"
        └── workspaceStore.setState({ unreadCounts: ... })  → Zustand
```

Il routing avviene in `useRTEvents.ts` tramite una dispatch table (action key → handler).

---

## 5. Optimistic updates

Applicato a operazioni dove la latenza è percepibile: invio messaggio, aggiunta reaction.

### Workflow

```
1. onMutate → genera message_id (UUID) client-side
             → snapshot cache attuale
             → inserisce optimistic item con _pending: true

2. HTTP 202  → onSuccess: niente (non invalidare, aspetta WS)
             → timer di rollback attivo (8s default)

3. WS rich   → clearTimeout(timer)
             → setQueryData: replace optimistic con dati reali (stesso message_id)

4. Fallback  → timeout scaduto o WS error event
             → rollback a snapshot, toast errore utente
```

### Perché funziona senza riconciliazione

Il client genera l'UUID **prima** di inviare il Command (pattern già definito in backend).
L'optimistic item e il messaggio reale hanno lo stesso `message_id` → il replace è un `Array.map` atomico.

---

## 6. WebSocket — lifecycle sottoscrizioni

```
Al connect:
  subscribe  user:<userId>          (automatico, sempre)

Entra in workspace:
  subscribe  workspace:<workspaceId>
  fetch HTTP channels + members     (allineamento stato iniziale)

Apre canale:
  subscribe  channel:<channelId>
  fetch HTTP messages               (allineamento stato iniziale)

Chiude canale:
  unsubscribe  channel:<channelId>

Lascia workspace:
  unsubscribe  workspace:<workspaceId>
```

---

## 7. Autenticazione (MVP)

Auth gestita con JWT. Il token viene conservato in memoria (non localStorage per sicurezza XSS).
Ogni request HTTP include `Authorization: Bearer <token>` via axios interceptor o fetch wrapper.

Per il refresh del token: silent refresh via endpoint dedicato prima della scadenza.

---

## 8. Convenzioni codice

- **Nessuna business logic nei componenti** — solo presentazione. La logica vive nei custom hook.
- **Un file per dominio in `api/`** — le funzioni di fetch sono pure (input → Promise<T>).
- **Query key come costanti** — definite vicino alla funzione fetch, non inline nei componenti.
- **Tipi Go-mirrored** — i tipi in `types/entities.ts` rispecchiano esattamente gli struct Go del backend. Nessun tipo inventato lato frontend.
