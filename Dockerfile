FROM node:22-alpine AS ui-build
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app/ui
COPY ui/package.json ui/pnpm-lock.yaml ui/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY ui/ .
RUN pnpm build

FROM golang:1.25-alpine AS go-build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-build /app/ui/dist ./ui/dist
RUN CGO_ENABLED=0 go build -v -o /scrumpoker ./cmd/scrumpoker

FROM gcr.io/distroless/static-debian12
COPY --from=go-build /scrumpoker /scrumpoker
EXPOSE 1323
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD ["/scrumpoker", "healthcheck"]
ENTRYPOINT ["/scrumpoker"]
