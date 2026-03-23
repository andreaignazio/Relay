# Relay — Frontend Implementation Pipeline

Ogni checkpoint è testabile end-to-end: si avvia il backend, si avvia il frontend, si verifica il comportamento descritto nella sezione "Test manuale".

Auth rinviata a post-MVP. Per ora tutte le request HTTP non richiedono token.

---

## Setup iniziale (pre-checkpoint)

Prima di iniziare i checkpoint, installare le dipendenze mancanti.

### Dipendenze da installare

```bash
# Routing
npm install @tanstack/react-router @tanstack/router-devtools

# Server state
npm install @tanstack/react-query @tanstack/react-query-devtools

# Client state
npm install zustand

# HTTP client
npm install axios

# Utility
npm install uuid
npm install -D @types/uuid
```

### Obiettivo setup

- `queryClient.ts` configurato e `QueryClientProvider` montato in `main.tsx`
- TanStack Router configurato con route root
- Zustand store scheletro creato
- `api/` con funzione base `apiClient` (axios instance con baseURL)

---

## Checkpoint 1 — Crea workspace · visualizza lista

**Obiettivo:** L'utente apre l'app, vede la lista dei workspace a cui appartiene, può creare un nuovo workspace, e il nuovo workspace appare nella lista senza ricaricare la pagina.

### Cosa buildare

#### Backend (verificare che esista)
- `GET /workspaces` → lista workspace dell'utente
- `POST /workspaces` → crea workspace (ritorna 202 + `message_id`)

#### Frontend

**Tipi** (`src/types/entities.ts`)
```ts
type Workspace = {
  workspace_id: string
  name: string
  slug: string
  created_at: string
}
```

**API** (`src/api/workspaces.ts`)
```ts
getWorkspaces()         // GET /workspaces → Workspace[]
createWorkspace(name)   // POST /workspaces → { message_id }
```

**Query** (`src/api/workspaces.ts`)
```ts
workspacesQueryKey = ['workspaces']
useWorkspacesQuery()
```

**Mutation con optimistic update**
```ts
useCreateWorkspaceMutation()
// onMutate  → inserisce workspace ottimistico in lista
// onSuccess → non invalidare (attende conferma WS in futuro)
// onError   → rollback
// Per ora: invalidateQueries onSuccess (WS non ancora integrato)
```

**Pages / Components**
```
WorkspaceListPage
└── WorkspaceList
    ├── WorkspaceItem (per ogni workspace)
    └── CreateWorkspaceDialog
        └── form: nome workspace → submit → mutation
```

**Route**
```
/ → WorkspaceListPage
```

### Test manuale ✓

1. `npm run dev` + backend avviato
2. Apri `localhost:5173` → vedi lista workspace (vuota o con dati seed)
3. Clicca "Crea workspace" → inserisci nome → conferma
4. Il workspace appare nella lista **immediatamente** (optimistic)
5. Ricarica la pagina → il workspace è ancora presente (persistito su backend)

---

## Checkpoint 2 — Crea canale · visualizza in sidebar

**Obiettivo:** L'utente clicca su un workspace, vede la sidebar con i canali, può creare un nuovo canale, e il canale appare nella sidebar senza ricaricare la pagina.

### Prerequisito

Checkpoint 1 completato e funzionante.

### Cosa buildare

#### Backend (verificare che esista)
- `GET /workspaces/:workspaceId/channels` → lista canali del workspace
- `POST /workspaces/:workspaceId/channels` → crea canale (202 + `message_id`)

#### Frontend

**Tipi** (`src/types/entities.ts`)
```ts
type Channel = {
  channel_id: string
  workspace_id: string
  name: string
  type: 'public' | 'private' | 'dm'
  created_at: string
}
```

**API** (`src/api/channels.ts`)
```ts
getChannels(workspaceId)          // GET /workspaces/:id/channels → Channel[]
createChannel(workspaceId, name)  // POST /workspaces/:id/channels → { message_id }
```

**Query + Mutation**
```ts
channelsQueryKey = (wsId: string) => ['channels', wsId]
useChannelsQuery(workspaceId)
useCreateChannelMutation(workspaceId)
// stesso pattern optimistic del checkpoint 1
```

**Layout shell**
```
WorkspacePage
├── Sidebar
│   ├── WorkspaceHeader (nome workspace + menu)
│   ├── ChannelList
│   │   ├── ChannelItem × N   (data-active, data-unread)
│   │   └── CreateChannelButton → CreateChannelDialog
│   └── UserFooter (avatar + nome utente placeholder)
└── <Outlet />  (area contenuto — per ora placeholder)
```

**Routes**
```
/:workspaceId           → WorkspacePage (layout con sidebar)
/:workspaceId/channels  → placeholder "Seleziona un canale"
```

**Navigation**
- Click su WorkspaceItem in lista → naviga a `/:workspaceId`
- `WorkspacePage` fa fetch `useChannelsQuery(workspaceId)` al mount

### Test manuale ✓

1. Da WorkspaceListPage, clicca su un workspace → navighi a `/:workspaceId`
2. Sidebar mostra i canali del workspace (anche lista vuota)
3. Clicca "Crea canale" → inserisci nome → conferma
4. Il canale appare nella sidebar **immediatamente** (optimistic)
5. Ricarica la pagina → il canale è ancora presente

---

## Roadmap checkpoint successivi (non implementare ora)

| Checkpoint | Feature |
|---|---|
| 3 | Apri canale → vedi messaggi (fetch + lista) |
| 4 | Invia messaggio → optimistic update + conferma WS |
| 5 | WebSocket setup → connect, subscribe user/workspace/channel |
| 6 | RT rich events → messaggi in tempo reale |
| 7 | RT invalidation → canali/membri aggiornati in tempo reale |
| 8 | RT sidebar hint → badge unread |
| 9 | Thread panel |
| 10 | Auth (JWT, login, register) |
