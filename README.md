# Progress Tracker

## Overview

Progress Tracker is a mobile-first web app for building consistent learning habits. Users create long-term goals, complete timed daily sessions, maintain streaks, and review progress in history, statistics, and a calendar.

Current version: **0.1.0 Beta**.

## Обзор

Progress Tracker — мобильное веб-приложение для формирования стабильных учебных привычек. Пользователи создают долгосрочные цели, выполняют ежедневные сессии по таймеру, поддерживают серии и анализируют историю, статистику и календарь.

Текущая версия: **0.1.0 Beta**.

## Main Features

- Registration, email verification, sign-in, password recovery, global logout, and account deletion.
- Private goals and server-managed timers with pause, resume, automatic completion, notes, and tags.
- Daily and total progress, streaks, calendar, completion rates, weekly comparison, and monthly statistics.
- Goal templates, session history search, JSON/CSV data export, themes, accent colors, font sizes, and English/Russian interfaces.

## Основные возможности

- Регистрация, подтверждение email, вход, восстановление пароля, выход на всех устройствах и удаление аккаунта.
- Личные цели и серверный таймер с паузой, продолжением, автоматическим завершением, заметками и тегами.
- Дневной и общий прогресс, серии, календарь, процент выполнения, сравнение недель и месячная статистика.
- Шаблоны целей, поиск по истории, экспорт JSON/CSV, темы, цвета акцента, размеры шрифта и интерфейс на английском/русском языках.

## Progress Rules

- A streak grows only when the daily target is reached; partial practice is still saved.
- Sessions from the same calendar day are merged into one daily result.
- Missing a required day resets the current streak. Calendar days follow the account timezone detected from the browser.

## Правила прогресса

- Серия растёт только после выполнения дневной нормы; частичный прогресс сохраняется.
- Сессии одного календарного дня объединяются в единый дневной результат.
- Пропуск обязательного дня сбрасывает серию. Календарный день определяется часовым поясом аккаунта, полученным из браузера.

## Technology

- Go REST API, SQLite, React, TypeScript, and Vite.
- Argon2id password hashing, hashed session/action tokens, secure cookie settings, Origin checks, rate limiting, strict validation, and account-level data isolation.
- Go unit/integration tests, Vitest, Playwright, GitHub Actions, Docker Compose, Nginx, health/readiness checks, and structured request logs.

## Технологии

- REST API на Go, SQLite, React, TypeScript и Vite.
- Argon2id-хеширование паролей, хешированные токены, защищённые cookie, проверка Origin, ограничение запросов, строгая валидация и изоляция данных аккаунтов.
- Модульные и интеграционные Go-тесты, Vitest, Playwright, GitHub Actions, Docker Compose, Nginx, health/readiness-проверки и структурированные журналы запросов.

## Running

- Local backend: `http://127.0.0.1:8080`.
- Local frontend: `http://127.0.0.1:5173` or `http://localhost:5173`.
- Docker Compose frontend: `http://127.0.0.1:8088`.
- SQLite backups are created with the dedicated `backend/cmd/backup` command or the backup binary included in the backend image.

## Запуск

- Локальный backend: `http://127.0.0.1:8080`.
- Локальный frontend: `http://127.0.0.1:5173` или `http://localhost:5173`.
- Frontend через Docker Compose: `http://127.0.0.1:8088`.
- Резервные копии SQLite создаются командой `backend/cmd/backup` или отдельным backup-бинарником в backend-образе.

## Production Notes

Production requires HTTPS, secure cookies, an explicit public origin, SMTP for account emails, persistent SQLite storage, and regular off-host backups. Temporary timer acceleration is available only in development. Background Web Push is not implemented yet.

## Production-заметки

Для production необходимы HTTPS, защищённые cookie, явный публичный origin, SMTP для писем аккаунта, постоянное хранилище SQLite и регулярные внешние резервные копии. Временное ускорение таймера доступно только в режиме разработки. Фоновый Web Push пока не реализован.
