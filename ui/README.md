# Scrum Poker UI

A lightweight, real-time scrum poker web app for estimating user stories with your team. Built with Vue 3, TypeScript, and Bootstrap 5.

## Features

- **Create & join rooms** - one click to start a session, share the link with your team
- **Real-time updates** - votes and reveals sync instantly via Server-Sent Events (SSE)
- **Reveal & reset** - show all votes at once, then reset for the next story
- **Dark / light theme** - choose your favorite
- **Copy room link** - quickly share the current room URL via clipboard
- **Persistent config** - username and theme selection is stored in localStorage across sessions

## Tech Stack

- Vue 3 (Composition API)
- TypeScript
- Bootstrap 5
- pnpm

### Install dependencies

```sh
pnpm install
```

### Development

Starts the Vite dev server with hot-reload. Expects the backend API on [localhost:1323](http://localhost:1323).

```sh
pnpm dev
```

### Production build

Runs type-checking and builds optimized static assets to `dist/`.

```sh
pnpm build
```
