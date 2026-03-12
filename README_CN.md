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

### 项目与分组管理
- 管理来自 **GitHub**、**Gitee**、**Gitea** 及自建 Git 平台的仓库
- 按自定义**分组**组织项目（如 `github/user`、`gitea`、`work`）
- 按平台管理**凭据**（GitHub / Gitee / Gitea，支持自定义 Base URL）
- 直接在界面中**克隆**远程仓库并分配分组
- 添加 / 删除 / 移动项目到不同分组

### VS Code 风格的 Git 操作
- 以可折叠的**目录树结构**查看变更文件
- **暂存/取消暂存** — 支持单文件、整个目录或全部操作
- 内置提交信息编辑器，一键**提交更改**
- **丢弃更改** — 同时支持已跟踪和未跟踪文件
- **Diff 查看器** — 语法高亮显示增删内容
- **二进制文件检测** — 自动跳过图片、可执行文件等的预览

### 多远程仓库支持
- 每个项目支持管理多个远程仓库（origin、upstream 等）
- **远程仓库选择器** — 下拉切换当前操作的远程仓库
- 通过界面添加 / 删除远程仓库
- **一键推送到所有远程** — 将当前分支推送到全部远程仓库
- **标签多端推送** — 一键将标签推送到所有远程仓库

### 项目级代理控制
- 每个项目可独立控制是否使用代理
- 三种模式：**跟随全局** / **强制使用代理** / **强制不使用代理**
- 所有网络操作（Pull / Push / Fetch / Push All）均尊重项目级代理设置
- 系统代理自动检测（Windows 注册表 / macOS scutil / Linux gsettings）

### 提交历史
- 浏览提交历史，显示作者、时间和提交信息
- 点击提交查看该版本的**变更文件列表**
- 选中文件查看**单文件 diff**
- **版本回滚** — 支持硬回滚/混合回滚/软回滚，操作前弹出确认对话框
- **撤销提交** — 生成反向提交
- **提交搜索** — 按提交信息关键字和/或作者过滤

### 分支管理
- 查看所有本地分支和远程分支
- **创建 / 删除 / 合并**分支
- 通过下拉菜单**切换分支**
- **检出远程分支**到本地
- **删除远程分支**

### 标签管理
- 查看所有标签（显示哈希、描述、时间戳）
- **创建标签**（轻量标签或附注标签）
- **推送标签**到指定远程或所有远程
- **删除标签**（本地和远程同步删除）

### 贮藏管理
- **保存**当前变更（支持自定义描述）
- **应用 / 弹出 / 删除**贮藏条目
- 查看贮藏列表及描述

### 合并冲突处理
- 自动检测合并状态并显示冲突提示
- 在内置编辑器中查看和编辑冲突文件
- **标记已解决 / 中止合并 / 完成合并**

### 批量操作
- 查看所有项目概览（分支、变更数、推送状态）
- **批量 Pull / Push** 所有项目
- 按项目显示操作结果

### Git 安装检测
- 启动时自动检测 Git 是否已安装
- 未安装时显示全屏引导界面
- 支持**自动下载安装** Git for Windows，实时显示进度

### 桌面体验
- **系统托盘** — 关闭窗口自动隐藏到托盘，点击图标恢复
- **可拖拽调整面板** — 侧边栏、文件列表、查看器之间的分隔条可拖拽
- **暗色主题** — Catppuccin Mocha 配色方案 + Ant Design Vue
- **原生文件夹选择器** — 通过系统对话框选择项目路径
- Git 全局配置编辑（user.name / user.email）

## 🏗️ 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25, Wails v3 |
| 前端 | Nuxt 4, Vue 3.5, Vite 7 |
| UI 组件库 | Ant Design Vue 4, @ant-design/icons-vue |
| Git 操作 | Git CLI 封装，支持项目级代理控制 |
| 配置 | 基于 YAML 的持久化配置 |
| 构建 | Taskfile v3, NSIS 安装包（Windows） |

## 📁 项目结构

```
GitPilot/
├── cmd/main.go              # Wails 应用入口，系统托盘，窗口管理
├── config.yaml              # 项目 / 分组 / 凭据配置文件
├── CHANGELOG.md             # 版本更新日志
├── Taskfile.yml             # 构建任务
├── go.mod
├── config/
│   └── config.go            # YAML 配置类型 & 加载/保存
├── internal/
│   ├── proxy.go             # 系统代理检测（Windows/macOS/Linux）
│   ├── git/
│   │   └── client.go        # Git CLI 封装，支持 RunWithProxy
│   └── app/
│       ├── service.go       # AppService — 72 个方法暴露给前端
│       └── gitsetup.go      # Git 安装检测 & 自动安装
├── frontend/
│   ├── nuxt.config.ts
│   ├── package.json
│   └── app/
│       ├── pages/
│       │   └── index.vue    # 主布局，可调整侧边栏宽度
│       ├── layouts/
│       │   └── default.vue  # Ant Design 暗色主题配置
│       └── components/
│           ├── Sidebar.vue          # 分组/项目树、批量操作、凭据管理
│           ├── ContentArea.vue      # Git 状态、Diff、历史、分支、标签
│           ├── FileTreeNode.vue     # 递归文件树（含暂存/丢弃操作）
│           ├── CommitFileTreeNode.vue  # 提交文件树查看器
│           └── GitCheck.vue         # Git 安装检测界面
└── build/                   # Wails 构建配置，NSIS 安装脚本
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
wails3 generate bindings -d frontend/bindings

# 开发模式运行
wails3 dev -config ./build/config.yml -port 9245
```

### 生产构建

```bash
task build
```

构建产物会输出到 `bin/` 目录。

## 📋 API 概览

GitPilot 通过 Wails 绑定向前端暴露了 **72+ 个后端方法**：

| 类别 | 方法 |
|------|------|
| **项目管理** | GetProjectTree, AddProject, RemoveProject, CloneProject, MoveProjectToGroup, SelectDirectory |
| **分组管理** | GetGroups, AddGroup, UpdateGroup, RemoveGroup |
| **凭据管理** | GetCredentials, AddCredential, UpdateCredential, RemoveCredential |
| **Git 状态** | GetProjectStatus, GetProjectChangedFiles, GetFileContent, GetFileDiff, GetFileDiffStaged |
| **暂存操作** | StageFiles, UnstageFiles, StageAll, UnstageAll |
| **提交操作** | CommitChanges, DiscardFiles |
| **远程操作** | PullProject, PushProject, FetchProject, PushToAllRemotes, PushTagToAllRemotes |
| **远程仓库管理** | GetRemotes, AddRemote, RemoveRemote |
| **代理控制** | GetProjectProxy, SetProjectProxy |
| **提交历史** | GetCommitLog, GetCommitDiff, GetCommitFiles, GetCommitFileDiff, RevertCommit, SearchCommitLog |
| **分支管理** | GetBranches, SwitchBranch, CreateBranch, DeleteBranch, MergeBranch, ResetProject |
| **远程分支** | GetRemoteBranches, CheckoutRemoteBranch, DeleteRemoteBranch |
| **标签管理** | GetTags, CreateTag, DeleteTag, PushTag |
| **贮藏管理** | StashSave, GetStashList, StashApply, StashPop, StashDrop |
| **冲突处理** | IsMerging, GetConflictFiles, GetConflictFileContent, SaveConflictFile, ResolveConflictFile, AbortMerge |
| **批量操作** | GetAllProjectOverview, BatchPull, BatchPush |
| **设置** | GetGitGlobalConfig, SetGitGlobalConfig, GetAppSettings, UpdateAppSettings |
| **Git 安装** | CheckGitInstalled, SelectGitInstallDir, InstallGit |

## 📄 开源协议

[MIT](LICENSE)
