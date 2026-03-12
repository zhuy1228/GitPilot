# GitPilot

[中文文档](README_CN.md)

A cross-platform desktop Git repository management tool built with **Go + Wails v3 + Nuxt 4 + Vue 3**.

Manage multiple Git repositories across different platforms (GitHub, Gitee, Gitea, etc.) from a single unified interface — like a lightweight, standalone version of VS Code’s Git panel.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v3-red?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vue.js&logoColor=white)
![Nuxt](https://img.shields.io/badge/Nuxt-4-00DC82?logo=nuxt.js&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue)

## ✨ Features

### Project & Group Management
- Manage repositories from **GitHub**, **Gitee**, **Gitea**, and custom Git platforms
- Organize projects into custom **Groups** (e.g. `github/user`, `gitea`, `work`)
- Manage **Credentials** per platform (GitHub / Gitee / Gitea with custom base URL)
- **Clone** remote repositories directly from the GUI with group assignment
- Add / remove / move projects between groups

### VS Code-Style Git Operations
- **View changed files** with a collapsible directory tree structure
- **Stage / Unstage** files individually, by directory, or all at once
- **Commit changes** with a built-in message editor
- **Discard changes** for tracked and untracked files
- **Diff viewer** with syntax-highlighted additions and deletions
- **Binary file detection** — skips preview for images, executables, etc.

### Multi-Remote Support
- Manage multiple remotes per project (origin, upstream, etc.)
- **Remote selector** dropdown to switch active remote
- Add / remove remotes through the GUI
- **Push to All Remotes** — one-click push current branch to every remote
- **Push Tag to All Remotes** — push tags to all remotes at once

### Per-Project Proxy Control
- Each project can independently control proxy usage
- Three modes: **Follow Global** / **Force Proxy On** / **Force Proxy Off**
- All network operations (Pull / Push / Fetch / Push All) respect per-project proxy settings
- System proxy auto-detection from Windows registry / macOS scutil / Linux gsettings

### Commit History
- Browse commit history with author, time, and message
- Click a commit to view its **changed file list**
- Select a file to view its **per-file diff**
- **Version rollback** — hard / mixed / soft reset with confirmation dialog
- **Revert commit** — create a reverse commit
- **Search commits** — filter by message keyword and/or author

### Branch Management
- View all local and remote branches
- **Create / Delete / Merge** branches
- **Switch branches** via dropdown selector
- **Checkout remote branches** to local
- **Delete remote branches**

### Tag Management
- View all tags with hash, message, and timestamp
- **Create tags** (lightweight or annotated)
- **Push tags** to selected remote or all remotes
- **Delete tags** (local and remote)

### Stash Management
- **Save** current changes with optional message
- **Apply / Pop / Drop** stash entries
- View stash list with descriptions

### Merge Conflict Resolution
- Detect merge state and display conflict indicator
- View and edit conflict files in a built-in editor
- **Mark as resolved / Abort merge / Complete merge**

### Batch Operations
- View all projects overview (branch, changes, push status)
- **Batch Pull / Push** all projects at once
- Results displayed per project

### Git Installation Detection
- Auto-detect Git installation on startup
- If not installed, show a full-screen guide
- **Auto-download and install** Git for Windows with progress tracking

### Desktop Experience
- **System tray** — close window hides to tray, click tray icon to restore
- **Resizable panels** — drag splitters between sidebar, file list, and viewer
- **Dark theme** — Catppuccin Mocha color scheme with Ant Design Vue
- **Native folder picker** — select project paths via OS file dialog
- Git global config editor (user.name / user.email)

## 🏗️ Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25, Wails v3 |
| Frontend | Nuxt 4, Vue 3.5, Vite 7 |
| UI Library | Ant Design Vue 4, @ant-design/icons-vue |
| Git Operations | Git CLI wrapper with per-project proxy support |
| Config | YAML-based persistent configuration |
| Build | Taskfile v3, NSIS installer (Windows) |

## 📁 Project Structure

```
GitPilot/
├── cmd/main.go              # Wails app entry, system tray, window management
├── config.yaml              # Projects / Groups / Credentials configuration
├── CHANGELOG.md             # Version history
├── Taskfile.yml             # Build tasks
├── go.mod
├── config/
│   └── config.go            # YAML config types & loader/saver
├── internal/
│   ├── proxy.go             # System proxy detection (Windows/macOS/Linux)
│   ├── git/
│   │   └── client.go        # Git CLI wrapper with RunWithProxy support
│   └── app/
│       ├── service.go       # AppService — 72 methods exposed to frontend
│       └── gitsetup.go      # Git installation detection & auto-install
├── frontend/
│   ├── nuxt.config.ts
│   ├── package.json
│   └── app/
│       ├── pages/
│       │   └── index.vue    # Main layout with resizable sidebar
│       ├── layouts/
│       │   └── default.vue  # Ant Design dark theme provider
│       └── components/
│           ├── Sidebar.vue          # Group/project tree, batch ops, credentials
│           ├── ContentArea.vue      # Git status, diff, history, branches, tags
│           ├── FileTreeNode.vue     # Recursive file tree with stage/discard
│           ├── CommitFileTreeNode.vue  # Commit file tree viewer
│           └── GitCheck.vue         # Git installation check screen
└── build/                   # Wails build config, NSIS installer scripts
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
wails3 generate bindings -d frontend/bindings

# Run in development mode
wails3 dev -config ./build/config.yml -port 9245
```

### Build for Production

```bash
task build
```

The binary will be output to the `bin/` directory.

## 📋 API Overview

GitPilot exposes **72+ backend methods** to the frontend via Wails bindings:

| Category | Methods |
|----------|---------|
| **Project Management** | GetProjectTree, AddProject, RemoveProject, CloneProject, MoveProjectToGroup, SelectDirectory |
| **Group Management** | GetGroups, AddGroup, UpdateGroup, RemoveGroup |
| **Credential Management** | GetCredentials, AddCredential, UpdateCredential, RemoveCredential |
| **Git Status** | GetProjectStatus, GetProjectChangedFiles, GetFileContent, GetFileDiff, GetFileDiffStaged |
| **Staging** | StageFiles, UnstageFiles, StageAll, UnstageAll |
| **Commit** | CommitChanges, DiscardFiles |
| **Remote Operations** | PullProject, PushProject, FetchProject, PushToAllRemotes, PushTagToAllRemotes |
| **Remote Management** | GetRemotes, AddRemote, RemoveRemote |
| **Proxy Control** | GetProjectProxy, SetProjectProxy |
| **Commit History** | GetCommitLog, GetCommitDiff, GetCommitFiles, GetCommitFileDiff, RevertCommit, SearchCommitLog |
| **Branch Management** | GetBranches, SwitchBranch, CreateBranch, DeleteBranch, MergeBranch, ResetProject |
| **Remote Branches** | GetRemoteBranches, CheckoutRemoteBranch, DeleteRemoteBranch |
| **Tags** | GetTags, CreateTag, DeleteTag, PushTag |
| **Stash** | StashSave, GetStashList, StashApply, StashPop, StashDrop |
| **Conflict Resolution** | IsMerging, GetConflictFiles, GetConflictFileContent, SaveConflictFile, ResolveConflictFile, AbortMerge |
| **Batch Operations** | GetAllProjectOverview, BatchPull, BatchPush |
| **Settings** | GetGitGlobalConfig, SetGitGlobalConfig, GetAppSettings, UpdateAppSettings |
| **Git Setup** | CheckGitInstalled, SelectGitInstallDir, InstallGit |

## 📄 License

[MIT](LICENSE)
