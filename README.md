# 📦 CloudBoxIO

> A lightweight, self-hosted file storage and sharing server built with Go and Fiber.

CloudBoxIO allows users to securely upload, share, and manage files with JWT-based authentication and an optional minimal UI. Built for simplicity and portability, it runs as a single binary and stores data using SQLite.

---

## 🚀 Features

- 🔐 JWT-based user authentication and authorization  
- 📁 Upload, list, and download personal files  
- 🌐 Public/shared file support  
- 🗑️ File deletion with name conflict resolution (e.g., `file(1).txt`)  
- 📊 SQLite for user and file metadata  
- 🧾 Optional file and server logging  
- 🎛️ Admin-only user management  
- 📂 Multi-file uploads  
- 🛑 Graceful shutdown  
- 🖥️ Optional built-in UI  
- 🚧 Rate limiting  
- 🧪 Unit testing  

---

## </> UI

> CloudBoxIO includes a clean, responsive UI for file management out of the box.

<p align="center">
  <img src="https://i.postimg.cc/ZRLWYKMC/index.png" alt="Landing page" width="400">
</p>

<p align="center">
  <img src="https://i.postimg.cc/VNnSTF99/dashboard.png" alt="Dashboard page" width="400">
</p>

<p align="center">
  <img src="https://i.postimg.cc/HxmJC73q/mobile-view.png" alt="Mobile view" width="200">
</p>

---

## ⚡ Quick Start

> ✅ Requires [Go](https://golang.org/dl/) 1.24 or higher (Go is only needed if building from source)

```bash
git clone https://github.com/AumSahayata/cloudboxio.git
cd cloudboxio
go mod tidy
go build .
./cloudboxio
```

> 💡 A `.env` file will be generated automatically on first run. You can edit it to change port, file directories, upload size, rate limiting, and more.

---

## 📚 Documentation

See the [Wiki](https://github.com/AumSahayata/cloudboxio/wiki) for full documentation:

- 🛠️ [Setup Guide](https://github.com/AumSahayata/cloudboxio/wiki/Setup-Guide)  
- ⚙️ [Configuration via `.env`](https://github.com/AumSahayata/cloudboxio/wiki/Configurations)  
- 🔐 [User API Reference](https://github.com/AumSahayata/cloudboxio/wiki/User-APIs)  
- 📁 [File API Reference](https://github.com/AumSahayata/cloudboxio/wiki/File-APIs)  

---

## 📄 License

This project is licensed under the [MIT License](https://github.com/AumSahayata/cloudboxio/blob/main/LICENSE)

---

## 💬 Need Help or Want to Contribute?
- Your feedback, ideas, and contributions are always welcome. Whether it’s fixing a bug, improving the docs, or suggesting a new feature — every bit helps make CloudBoxIO better for everyone.
- Ask questions or share ideas in [Discussions](https://github.com/AumSahayata/cloudboxio/discussions)  
- Report bugs via [Issues](https://github.com/AumSahayata/cloudboxio/issues)  
- Suggestions welcome! You can contribute:
  - 🔄 Docker support  
  - 💻 Frontend improvements  
  - 🛠️ CI pipelines or GitHub Actions  
  - 🧪 Integration testing  
  - 🆕 Bring your own idea
---

## 👨‍💻 Author

Made with ❤️ by [Aum Sahayata](https://github.com/AumSahayata)

---