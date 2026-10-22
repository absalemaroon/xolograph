# Xolograph frontend

React + TypeScript + Tailwind (tokens in `src/styles/index.css`), Zustand state, talks to the Go engine (`backend/cmd/xolographd`).

```
src/
  app/          App root, navigation config
  api/          engine client, types, endpoints
  components/   ui/ (design system) · layout/ (shell, sidebar, topbar)
  features/     overview · cases · evidence · audit · settings · planned
  hooks/ lib/ state/ styles/ types/
```

Dev: start the engine (`cd backend && go run ./cmd/xolographd --port 8787 --token dev --data-dir ./.data`), then
`cd frontend && npm install && npm run dev` (port 3000). `npm test` runs unit tests; `npm run build` typechecks and builds.
