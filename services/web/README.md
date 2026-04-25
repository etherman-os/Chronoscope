# Chronoscope Web

Frontend dashboard for the Chronoscope session replay platform.

## Tech Stack

- React 18
- TypeScript 5
- Vite 5
- React Router 6
- Axios
- Vitest + React Testing Library

## Getting Started

```bash
npm install
npm run dev
```

The development server will start on `http://localhost:3000`.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_API_URL` | Base URL for the API | `http://localhost:8080/v1` |
| `VITE_PROJECT_ID` | Project identifier for sessions | `22222222-2222-2222-2222-222222222222` |

Copy `.env.example` to `.env.local` and adjust as needed:

```bash
cp .env.example .env.local
```

## Available Scripts

- `npm run dev` — Start development server
- `npm run build` — Build for production
- `npm run preview` — Preview production build
- `npm run lint` — Run ESLint
- `npm run test` — Run unit tests with Vitest

## Testing

Unit and integration tests are written with Vitest and React Testing Library.

```bash
npm test
```

Tests are located next to the components they cover (e.g., `SessionList.test.tsx`).

The dev server proxies API requests to `http://localhost:8080` via `vite.config.ts`.
