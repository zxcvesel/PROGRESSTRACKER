# Progress Tracker

## Overview

Progress Tracker is a mobile-first web app for building consistent learning habits. Users create long-term goals, complete focused sessions, maintain daily streaks, and review progress through statistics, history, and a calendar.

Current version: **0.1.0 Beta**.

## Обзор

Progress Tracker — мобильное веб-приложение для формирования стабильных учебных привычек. Пользователи создают долгосрочные цели, проводят сфокусированные сессии, поддерживают ежедневные серии и анализируют прогресс через статистику, историю и календарь.

Текущая версия: **0.1.0 Beta**.

## Main Features

- Account registration, sign-in, sign-out, profile name editing, and password change.
- Private goals, sessions, history, calendar, and statistics for each account.
- Long-term goals with total duration and a daily time target.
- Session timer with pause, resume, automatic target completion, and recovery after page reload.
- Session notes, tags, editing, deletion, and one combined daily result.
- Daily and total progress, current and longest streaks, completion rates, weekly comparison, monthly totals, and goal distribution.
- Dark and light themes, accent colors, font sizes, English and Russian interfaces.
- Installable PWA foundation and optional browser notifications while the app is open.

## Основные возможности

- Регистрация, вход, выход, изменение имени профиля и смена пароля.
- Личные цели, сессии, история, календарь и статистика для каждого аккаунта.
- Долгосрочные цели с общей длительностью и ежедневной нормой времени.
- Таймер с паузой, продолжением, автоматическим завершением нормы и восстановлением после перезагрузки страницы.
- Заметки, теги, редактирование и удаление сессий с единым результатом за день.
- Дневной и общий прогресс, текущая и лучшая серии, completion rate, сравнение недель, итог за месяц и распределение времени по целям.
- Тёмная и светлая темы, цвета акцента, размеры шрифта, английский и русский интерфейсы.
- Основа устанавливаемого PWA и необязательные браузерные уведомления при открытом приложении.

## Progress Rules

- A day increases the streak only after its daily target is reached.
- Partial practice is saved but does not increase the streak.
- Additional practice on the same calendar day is merged into one daily session.
- Missing a required calendar day resets the current streak.
- Total goal progress is based on completed daily targets.

## Правила прогресса

- День увеличивает серию только после выполнения дневной нормы.
- Частичная практика сохраняется, но не увеличивает серию.
- Дополнительные занятия в тот же календарный день объединяются в одну дневную сессию.
- Пропуск обязательного календарного дня сбрасывает текущую серию.
- Общий прогресс цели считается по выполненным дневным нормам.

## Technology

- Backend: Go REST API.
- Database: SQLite.
- Frontend: React, TypeScript, and Vite.
- Security: Argon2id password hashes, hashed session tokens, secure cookie options, Origin checks, bounded rate limiting, strict request validation, and hardened HTTP/SQLite defaults.
- Tests: backend unit and HTTP integration tests; frontend lint and production build checks.

## Технологии

- Backend: REST API на Go.
- База данных: SQLite.
- Frontend: React, TypeScript и Vite.
- Безопасность: Argon2id-хэши паролей, хэши токенов сессий, защищенные настройки cookie, проверка Origin, ограничение запросов, строгая валидация и усиленные настройки HTTP/SQLite.
- Тесты: модульные и HTTP-интеграционные тесты backend, lint и production build frontend.

## Local Addresses

- Backend: `http://127.0.0.1:8080`.
- Frontend: `http://127.0.0.1:5173` or `http://localhost:5173`.
- The Vite development server proxies frontend `/api` requests to the backend.

## Production Configuration

- Use Go **1.25.12** or newer within the 1.25 release line.
- Set `PROGRESS_TRACKER_HOST`, `PROGRESS_TRACKER_PORT`, and `PROGRESS_TRACKER_DB_PATH` for the deployment environment.
- Set `PROGRESS_TRACKER_ALLOWED_ORIGINS` to the public frontend origin and enable `PROGRESS_TRACKER_SECURE_COOKIES=true` only behind HTTPS.
- Enable `PROGRESS_TRACKER_TRUST_PROXY=true` only when requests pass through a trusted reverse proxy that overwrites forwarding headers.
- GitHub Actions checks backend tests, formatting, vet, vulnerabilities, frontend lint, build, and dependency audit.

## Конфигурация production

- Используйте Go **1.25.12** или более новую версию линейки 1.25.
- Задайте `PROGRESS_TRACKER_HOST`, `PROGRESS_TRACKER_PORT` и `PROGRESS_TRACKER_DB_PATH` для среды развертывания.
- В `PROGRESS_TRACKER_ALLOWED_ORIGINS` укажите публичный адрес frontend, а `PROGRESS_TRACKER_SECURE_COOKIES=true` включайте только при работе через HTTPS.
- Включайте `PROGRESS_TRACKER_TRUST_PROXY=true` только за доверенным reverse proxy, который перезаписывает forwarding-заголовки.
- GitHub Actions проверяет тесты backend, форматирование, vet, уязвимости, lint и сборку frontend, а также зависимости.

## Локальные адреса

- Backend: `http://127.0.0.1:8080`.
- Frontend: `http://127.0.0.1:5173` или `http://localhost:5173`.
- Vite development server перенаправляет frontend-запросы `/api` на backend.

## Project Structure

- `backend/cmd/api`: API routes, authentication, goals, sessions, statistics, security helpers, and tests.
- `backend/data`: local SQLite database, excluded from Git.
- `frontend/src`: application state, components, screens, and styles.
- `frontend/public`: PWA manifest and public visual assets.

## Структура проекта

- `backend/cmd/api`: маршруты API, авторизация, цели, сессии, статистика, безопасность и тесты.
- `backend/data`: локальная база SQLite, исключённая из Git.
- `frontend/src`: состояние приложения, компоненты, экраны и стили.
- `frontend/public`: PWA manifest и публичные визуальные ресурсы.

## Current Limitations

Background Web Push is not implemented yet. On iPhone, background notifications will require an installed Home Screen web app, a service worker, push subscriptions, and backend delivery. Email change, password reset by email, account deletion, and deployment infrastructure are also planned.

The timer speed selector is a temporary development control for manual testing.

## Текущие ограничения

Фоновый Web Push пока не реализован. На iPhone для фоновых уведомлений потребуются установка приложения на экран «Домой», service worker, push-подписки и отправка уведомлений с backend. Также запланированы смена email, сброс пароля через email, удаление аккаунта и инфраструктура deployment.

Переключатель скорости таймера — временный инструмент разработки для ручного тестирования.
