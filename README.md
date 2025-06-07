# 📦 CloudBoxIO

CloudBoxIO is a lightweight, self-hosted file storage and sharing service built using Go and Fiber. It supports file uploads, secure JWT-based authentication, shared/public files, and more — all backed by SQLite for simplicity and portability.

---

## 🚀 Features

- 🔐 User authentication (Signup/Login) using JWT
- 📁 Upload, list, and download personal files
- 🌐 Shared file support (public listing)
- 🗑️ File deletion
- 🧠 Filename conflict resolution (e.g., `file(1).txt`)
- 📊 SQLite-based metadata and user storage
- 📂 Optional file logging and server logs
- 🧾 Graceful server shutdown and cleanup
- 🧠 Auto-generated `.env` file with required flags and JWT secret

---

## ⚙️ Tech Stack

- Language: **Go (Golang)**
- Web Framework: **Fiber**
- Database: **SQLite**
- Auth: **JWT**
- Logging: **Standard Library log package**
- Environment Handling: **`godotenv`**

---

## 🧪 Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) 1.24+
- SQLite3 (optional CLI for viewing DB)

---

### 🛠️ Run Locally

```bash
git clone https://github.com/yourusername/cloudboxio.git
cd cloudboxio
go mod tidy
go run main.go
