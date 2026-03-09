# GitPilot

[English](README.md)

一款跨平台桌面 Git 仓库管理工具，基于 **Go + Wails v3 + Nuxt 4 + Vue 3** 构建。

在统一界面中管理来自不同平台（GitHub、Gitee、Gitea 等）的多个 Git 仓库 —— 类似一个独立的、轻量级的 VS Code Git 面板。

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v3-red?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vue.js&logoColor=white)
![Nuxt](https://img.shields.io/badge/Nuxt-4-00DC82?logo=nuxt.js&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue)

## ✨ 功能特性

### 多平台管理
- 管理来自 **GitHub**、**Gitee**、**Gitea** 及自建 Git 平台的仓库
- 按 **平台 → 用户 → 项目** 层级组织
- 通过图形界面添加/删除平台、用户和项目

### VS Code 风格的 Git 操作
- 以可折叠的**目录树结构**查看变更文件
- **暂存/取消暂存** — 支持单文件、整个目录或全部操作
- 内置提交信息编辑器，一键**提交更改**
- **丢弃更改** — 同时支持已跟踪和未跟踪文件
- **Diff 查看器** — 语法高亮显示增删内容
- **二进制文件检测** — 自动跳过图片、可执行文件等的预览

### 提交历史
- 浏览提交历史，显示作者、时间和提交信息
- 点击提交查看该版本的**变更文件列表**（与 VS Code 一致）
- 选中文件查看**单文件 diff**
- **版本回滚** — 支持硬回滚/混合回滚/软回滚，操作前弹出确认对话框

### 分支管理
- 查看所有本地分支，标记当前分支
- 通过下拉菜单**切换分支**
- **拉取/推送/获取远程信息**，自动识别当前分支

### 桌面体验
- **系统托盘** — 关闭窗口自动隐藏到托盘，点击托盘图标恢复窗口
- **可拖拽调整面板** — 侧边栏、文件列表、查看器之间的分隔条可拖拽
- **暗色主题** — Catppuccin Mocha 配色方案 + Ant Design Vue
- **原生文件夹选择器** — 通过系统对话框选择项目路径

## 🏗️ 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25, Wails v3 (alpha.74) |
| 前端 | Nuxt 4, Vue 3.5, Vite 7 |
| UI 组件库 | Ant Design Vue 4, @ant-design/icons-vue |
| Git 操作 | Git CLI 封装，支持代理，30 秒超时 |
| 配置 | 基于 YAML 的持久化配置 |
| 构建 | Taskfile v3 |

## 📁 项目结构

```
GitPilot/
├── main.go                  # Wails 应用入口，系统托盘，窗口管理
├── config.yaml              # 平台/用户/项目配置文件
├── Taskfile.yml             # 构建任务
├── go.mod
├── cmd/                     # 命令行工具
├── config/
│   └── config.go            # YAML 配置加载/保存
├── internal/
│   ├── proxy.go             # 系统代理检测
│   ├── git/
│   │   └── client.go        # Git CLI 封装（34 个操作）
│   └── app/
│       └── service.go       # AppService — 34 个方法暴露给前端
├── frontend/
│   ├── nuxt.config.ts
│   ├── package.json
│   └── app/
│       ├── pages/
│       │   └── index.vue    # 主布局，可调整侧边栏宽度
│       ├── layouts/
│       │   └── default.vue  # Ant Design 暗色主题配置
│       └── components/
│           ├── Sidebar.vue      # 平台/用户/项目树形侧边栏
│           ├── ContentArea.vue  # Git 状态、Diff、历史、分支
│           └── FileTreeNode.vue # 递归文件树组件（含操作按钮）
└── build/                   # Wails 构建配置
```

## 🚀 快速开始

### 环境要求

- [Go](https://golang.org/dl/) 1.25+
- [Node.js](https://nodejs.org/) 18+
- [Git](https://git-scm.com/)
- [Task](https://taskfile.dev/)（可选，用于构建任务）
- [Wails v3 CLI](https://v3alpha.wails.io/)

### 开发运行

```bash
# 克隆仓库
git clone https://github.com/zhuy1228/GitPilot.git
cd GitPilot

# 安装前端依赖
cd frontend && npm install && cd ..

# 生成 Wails 绑定
wails3 generate bindings

# 构建前端
cd frontend && npx nuxi generate && cd ..

# 开发模式运行
task dev
# 或者
wails3 dev -config ./build/config.yml -port 9245
```

### 生产构建

```bash
task build
```

构建产物会输出到 `bin/` 目录。

## 📋 API 概览

GitPilot 通过 Wails 绑定向前端暴露了 **34 个后端方法**：

| 类别 | 方法 |
|------|------|
| **项目管理** | GetProjectTree, AddProject, RemoveProject, SelectDirectory |
| **平台管理** | AddPlatform, UpdatePlatform, RemovePlatform, GetPlatformInfo |
| **用户管理** | AddUser, UpdateUser, RemoveUser, GetUserInfo |
| **Git 状态** | GetProjectStatus, GetProjectChangedFiles, GetFileContent, GetFileDiff, GetFileDiffStaged |
| **暂存操作** | StageFiles, UnstageFiles, StageAll, UnstageAll |
| **提交操作** | CommitChanges, DiscardFiles |
| **远程操作** | PullProject, PushProject, FetchProject |
| **提交历史** | GetCommitLog, GetCommitDiff, GetCommitFiles, GetCommitFileDiff |
| **分支管理** | GetBranches, SwitchBranch, ResetProject |

## 📄 开源协议

[MIT](LICENSE)
