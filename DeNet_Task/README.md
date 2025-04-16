# HTTP Server for User Management (Go)

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-blue.svg)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-24.0+-blue.svg)](https://www.docker.com/)

Простой HTTP-сервер на Go для управления пользователями и их активностями (реферальные коды, задания, бонусные баллы).

## 🚀 Функционал
- **JWT-авторизация** (Middleware для всех эндпоинтов)
- **API Endpoints**:
  - `GET /users/{id}/status` — информация о пользователе
  - `GET /users/leaderboard` — топ пользователей по балансу
  - `POST /users/{id}/task/complete` — выполнение задания (награда в баллах)
  - `POST /users/{id}/referrer` — ввод реферального кода
- **Хранилище**: PostgreSQL с миграциями (`golang-migrate`)
- **Docker-сборка**: Готовый `docker-compose.yml` для развертывания

## 📦 Установка
### Предварительные требования
- Go 1.21+
- PostgreSQL 15+
- Docker 24.0+ (опционально)

### Запуск локально
1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/ваш-username/название-репозитория.git
   cd название-репозитория
