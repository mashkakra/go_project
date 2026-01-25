FROM golang:1.25.3-alpine AS builder

WORKDIR /app

COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

COPY backend ./backend
COPY frontend ./frontend

WORKDIR /app/backend
RUN go build -o /app/bin/server .

FROM alpine:3.20

WORKDIR /app/backend

COPY --from=builder /app/bin/server /app/backend/server
COPY --from=builder /app/frontend /app/frontend

EXPOSE 8080

CMD ["./server"]
