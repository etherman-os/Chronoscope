# Chronoscope Web

Frontend dashboard for the Chronoscope session replay platform.

## Tech Stack

- React 18
- TypeScript 5
- Vite 5
- React Router 6
- Axios

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
| `VITE_API_KEY` | API key for authentication | `dev-api-key-12345` |

Create a `.env.local` file to override defaults:

```
VITE_API_URL=http://localhost:8080/v1
VITE_API_KEY=your-api-key
```

## Build

```bash
npm run build
```

## Scripts

- `npm run dev` — Start development server
- `npm run build` — Build for production
- `npm run preview` — Preview production build
- `npm run lint` — Run ESLint
