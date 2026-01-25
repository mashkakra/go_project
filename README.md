## Описание
Проект разделен на фронт, бэк и базу. Бэкенд — Go‑сервис, который рендерит HTML‑шаблоны и отдаёт API. Фронт для GitHub Pages — статическая витрина (в `frontend/index.html`). Схема БД и сиды — в `db/allbiblios.sql`.

## Структура
- `backend/` — Go‑сервер (HTTP + БД)
- `frontend/` — статический фронт для GitHub Pages
- `frontend/templates/` — HTML‑шаблоны для серверного рендеринга
- `db/` — SQL схема и сиды

## Локальный запуск без Docker
1) Установить PostgreSQL и применить SQL из `db/allbiblios.sql`.
2) Из папки `backend`:
```
go run .
```

Переменные окружения (по умолчанию используются эти значения):
- `DB_HOST` (default: `localhost`)
- `DB_PORT` (default: `5432`)
- `DB_USER` (default: `postgres`)
- `DB_PASSWORD` (default: `00000000`)
- `DB_NAME` (default: `postgres`)

## Локальный запуск через Docker Compose
В корне репозитория:
```
docker compose up --build
```

Бэкенд поднимется на `http://localhost:8080`.

## GitHub Pages (статический фронт)
GitHub Pages публикует содержимое `frontend/`. Для этого добавлен workflow в `.github/workflows/gh-pages.yml`.
Главная страница — `frontend/index.html`.

Примечание: шаблоны в `frontend/templates/` предназначены для серверного рендеринга бэкенда и не являются статическим SPA.
