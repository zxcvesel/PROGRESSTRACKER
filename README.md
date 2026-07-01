# Progress Tracker

## Overview

Progress Tracker is a mobile-first web app for tracking long-term learning goals. It helps users create goals, practice with a session timer, save notes, and follow progress through streaks, history, and statistics.

## Обзор

Progress Tracker — мобильное веб-приложение для отслеживания долгосрочных учебных целей. Оно помогает создавать цели, заниматься по таймеру, сохранять заметки и следить за прогрессом через стрики, историю и статистику.

## Tech Stack

- Backend: Go
- API: REST
- Database: SQLite
- Frontend: React
- Language: TypeScript
- Build tool: Vite
- Styling: custom CSS
- Design: mobile-first dark UI with cyan accent

## Технологический стек

- Backend: Go
- API: REST
- Database: SQLite
- Frontend: React
- Language: TypeScript
- Build tool: Vite
- Styling: custom CSS
- Design: мобильный темный интерфейс с бирюзовым акцентом

## Current Features

- Create, edit, and delete long-term goals.
- Set total goal duration and daily target time.
- Start, pause, resume, and finish practice sessions.
- Automatically stop the timer when the daily target is reached.
- Save session notes and tags.
- View, edit, and delete session history.
- Track today's progress, total goal progress, current streak, and longest streak.
- View basic statistics with weekly activity, monthly total, and goal time distribution.

## Текущий функционал

- Создание, редактирование и удаление долгосрочных целей.
- Настройка общей длительности цели и ежедневной нормы времени.
- Запуск, пауза, продолжение и завершение сессий практики.
- Автоматическая остановка таймера после достижения дневной нормы.
- Сохранение заметок и тегов к сессиям.
- Просмотр, редактирование и удаление истории сессий.
- Отслеживание прогресса за сегодня, общего прогресса цели, текущего и лучшего стрика.
- Базовая статистика с недельной активностью, итогом за месяц и распределением времени по целям.

## Progress Rules

- A day counts only when the daily target is completed.
- Partial progress is saved, but it does not increase the streak.
- The streak starts from the first completed daily target.
- Extra practice on the same day is added to the same daily result.
- Total goal progress is based on completed daily targets.

## Правила прогресса

- День засчитывается только после выполнения дневной нормы.
- Частичный прогресс сохраняется, но не увеличивает стрик.
- Стрик начинается с первого дня, когда дневная норма выполнена.
- Дополнительная практика в тот же день добавляется к дневному результату.
- Общий прогресс цели считается по выполненным дневным нормам.

## Project Structure

- backend: Go REST API, SQLite storage, business logic, and tests
- backend/cmd/api: backend entry point and backend tests
- backend/data: local SQLite storage
- frontend: React, TypeScript, and Vite app
- frontend/src: main frontend source files and styles
- frontend/public: public icons and static assets

## Структура проекта

- backend: REST API на Go, SQLite, бизнес-логика и тесты
- backend/cmd/api: точка входа backend-приложения и backend-тесты
- backend/data: локальное хранилище SQLite
- frontend: приложение на React, TypeScript и Vite
- frontend/src: основные frontend-файлы и стили
- frontend/public: публичные иконки и статические файлы

## Development Status

Implemented: goals, session timer, history, streaks, basic statistics, and mobile dark UI.

Not implemented yet: user accounts, authentication, settings, language switching, deployment setup, and final statistics polish.

The temporary timer speed control is used only for manual testing during development.

## Статус разработки

Реализовано: цели, таймер сессий, история, стрики, базовая статистика и темный мобильный интерфейс.

Пока не реализовано: аккаунты пользователей, авторизация, настройки, смена языка, настройка деплоя и финальная доработка статистики.

Временное управление скоростью таймера используется только для ручного тестирования во время разработки.
