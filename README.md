# GitPilot

[中文文档](README_CN.md)

A cross-platform desktop Git repository management tool built with **Go + Wails v3 + Nuxt 4 + Vue 3**.

Manage multiple Git repositories across different platforms (GitHub, Gitee, Gitea, etc.) from a single unified interface — like a lightweight, standalone version of VS Code's Git panel.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v3-red?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vue.js&logoColor=white)
![Nuxt](https://img.shields.io/badge/Nuxt-4-00DC82?logo=nuxt.js&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue)

## ✨ Features

### Multi-Platform Management
- Manage repositories from **GitHub**, **Gitee**, **Gitea**, and custom Git platforms
- Organize projects by **Platform → User → Project** hierarchy
- Add / remove platforms, users, and projects through the GUI

### VS Code-Style Git Operations
- **View changed files** with a collapsible directory tree structure
- **Stage / Unstage** files individually, by directory, or all at once
- **Commit changes** with a built-in message editor
- **Discard changes** for tracked and untracked files
- **Diff viewer** with syntax-highlighted additions and deletions
- **Binary file detection** — skips preview for images, executables, etc.

### Commit History
- Browse commit history with author, time, and message
- Click a commit to view its **changed file list** (like VS Code)
- Select a file to view its **per-file diff**
- **Version rollback** — hard / mixed / soft reset with confirmation dialog

### Branch Management
- View all local branches with current branch indicator
- **Switch branches** via dropdown selector
- **Pull / Push / Fetch** with current branch awareness

### Desktop Experience
- **System tray** — close window hides to tray, click tray icon to restore
- **Resizable panels** — drag splitters between sidebar, file list, and viewer
- **Dark theme** — Catppuccin Mocha color scheme with Ant Design Vue
- **Native folder picker** — select project paths via OS file dialog

## 🏗️ Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25, Wails v3 (alpha.74) |
| Frontend | Nuxt 4, Vue 3.5, Vite 7 |
| UI Library | Ant Design Vue 4, @ant-design/icons-vue |
| Git Operations | Git CLI with proxy support, 30s timeout |
| Config | YAML-based persistent configuration |
| Build | Taskfile v3 |

## 📁 Project Structure

```
GitPilot/
├── main.go                  # Wails app entry, system tray, window management
├── config.yaml              # Platform/user/project configuration
├── Taskfile.yml             # Build tasks
├── go.mod
├── cmd/                     # CLI utilities
├── config/
│   └── config.go            # YAML config loader/saver
├── internal/
│   ├── proxy.go             # System proxy detection
│   ├── git/
│   │   └── client.go        # Git CLI wrapper (34 operations)
│   └── app/
│       └── service.go       # AppService — 34 methods exposed to frontend
├── frontend/
│   ├── nuxt.config.ts
│   ├── package.json
│   └── app/
│       ├── pages/
│       │   └── index.vue    # Main layout with resizable sidebar
│       ├── layouts/
│       │   └── default.vue  # Ant Design dark theme provider
│       └── components/
│           ├── Sidebar.vue      # Platform/user/project tree sidebar
│           ├── ContentArea.vue  # Git status, diff, history, branches
│           └── FileTreeNode.vue # Recursive file tree with actions
└── build/                   # Wails build configuration
```

## 🚀 Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.25+
- [Node.js](https://nodejs.org/) 18+
- [Git](https://git-scm.com/)
- [Task](https://taskfile.dev/) (optional, for build tasks)
- [Wails v3 CLI](https://v3alpha.wails.io/)

### Development

```bash
# Clone the repository
git clone https://github.com/zhuy1228/GitPilot.git
cd GitPilot

# Install frontend dependencies
cd frontend && npm install && cd ..

# Generate Wails bindings
wails3 generate bindings

# Build frontend
cd frontend && npx nuxi generate && cd ..

# Run in development mode
task dev
# or
wails3 dev -config ./build/config.yml -port 9245
```

### Build for Production

```bash
task build
```

The binary will be output to the `bin/` directory.

## 📋 API Overview

GitPilot exposes **34 backend methods** to the frontend via Wails bindings:

| Category | Methods |
|----------|---------|
| **Projects** | GetProjectTree, AddProject, RemoveProject, SelectDirectory |
| **Platforms** | AddPlatform, UpdatePlatform, RemovePlatform, GetPlatformInfo |
| **Users** | AddUser, UpdateUser, RemoveUser, GetUserInfo |
| **Git Status** | GetProjectStatus, GetProjectChangedFiles, GetFileContent, GetFileDiff, GetFileDiffStaged |
| **Staging** | StageFiles, UnstageFiles, StageAll, UnstageAll |
| **Commit** | CommitChanges, DiscardFiles |
| **Remote** | PullProject, PushProject, FetchProject |
| **History** | GetCommitLog, GetCommitDiff, GetCommitFiles, GetCommitFileDiff |
| **Branches** | GetBranches, SwitchBranch, ResetProject |

## 📄 License

[MIT](LICENSE)
