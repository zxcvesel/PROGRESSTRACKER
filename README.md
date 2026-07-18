# Progress Tracker

## Product

Progress Tracker is a mobile-first web application for long-term learning goals. Users complete timed daily sessions, maintain streaks, and review progress through history, statistics, and a calendar.

## Продукт

Progress Tracker — мобильное веб-приложение для долгосрочных учебных целей. Пользователи выполняют ежедневные сессии по таймеру, поддерживают серии и анализируют историю, статистику и календарь активности.

## Main features

- Registration, email verification, sign-in, password recovery, and account deletion.
- Personal goals with daily targets and a server-managed timer with pause and resume.
- Session notes and tags, history search, calendar, streaks, weekly comparisons, and monthly statistics.
- JSON/CSV export, light and dark themes, accent colors, font sizes, and English/Russian interfaces.
- Installable PWA shell with mobile icons, safe-area layout, offline startup, and session alerts while the app is active.

## Основные возможности

- Регистрация, подтверждение email, вход, восстановление пароля и удаление аккаунта.
- Личные цели с дневной нормой и серверным таймером с паузой и продолжением.
- Заметки и теги сессий, поиск по истории, календарь, серии, сравнение недель и месячная статистика.
- Экспорт JSON/CSV, светлая и тёмная темы, цвета оформления, размеры шрифта и русский/английский интерфейс.
- Устанавливаемая PWA-оболочка с мобильными иконками, safe area, офлайн-запуском и уведомлениями при активном приложении.

## Progress rules

A streak day is counted only when the daily target is reached. Partial practice is saved, sessions from the same calendar day are combined, and a missed required day resets the streak. Calendar days use the account timezone.

## Правила прогресса

День засчитывается в серию только после выполнения дневной нормы. Частичный прогресс сохраняется, сессии одного календарного дня объединяются, а пропущенный обязательный день сбрасывает серию. Дни определяются по часовому поясу аккаунта.

## Technology

Go REST API, SQLite, React, TypeScript, Vite, Docker Compose, Nginx, Vitest, Playwright, and GitHub Actions. Passwords use Argon2id; account data is isolated by user; sessions and action tokens are stored as hashes.

## Технологии

REST API на Go, SQLite, React, TypeScript, Vite, Docker Compose, Nginx, Vitest, Playwright и GitHub Actions. Пароли защищены Argon2id; данные аккаунтов изолированы; токены сессий и действий хранятся в виде хешей.

## Running

- Backend: `http://127.0.0.1:8080`.
- Vite frontend: `http://127.0.0.1:5173`.
- Local Docker Compose: `http://127.0.0.1:8088` with `docker compose up --build`.
- Staging: create `.env.staging` from `.env.staging.example`, provision TLS files in `deploy/certs`, then use `docker compose --env-file .env.staging -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.staging.yml up -d --build`.

## Запуск

- Backend: `http://127.0.0.1:8080`.
- Frontend Vite: `http://127.0.0.1:5173`.
- Локальный Docker Compose: `http://127.0.0.1:8088`, команда `docker compose up --build`.
- Staging: создать `.env.staging` из `.env.staging.example`, разместить TLS-файлы в `deploy/certs` и выполнить `docker compose --env-file .env.staging -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.staging.yml up -d --build`.

## Operations

Production mode requires HTTPS, secure cookies, an explicit public origin, real SMTP credentials with STARTTLS, persistent SQLite storage, and off-host backups. The production Compose stack creates and verifies daily SQLite backups and keeps them in the `progress-backups` volume according to the configured retention period.

## Эксплуатация

Production-режим требует HTTPS, защищённых cookie, явного публичного origin, реального SMTP с STARTTLS, постоянного хранилища SQLite и внешних резервных копий. Production Compose ежедневно создаёт и проверяет SQLite-backup, сохраняя копии в томе `progress-backups` в пределах настроенного срока хранения.
