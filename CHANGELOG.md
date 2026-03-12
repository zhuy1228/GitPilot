# Changelog

## v0.4.1 (2026-03-12)

### ✨ 新功能
- **GitLab 平台支持**：新增 GitLab 作为第四个平台（GitHub / Gitee / Gitea / GitLab）
  - 完整实现 PlatformAPI 全部 16 个接口方法（CreateRepo / ListRepos / ListReleases / CreateRelease / UploadAsset 等）
  - 实现 7 个扩展迁移方法（Labels / Milestones / Issues / PullRequests）
  - 前端新增 GitLab 图标、平台选择项、BaseURL 输入支持
- **Release 同步 Token 输入**：同步发布时可手动输入源/目标 Token，不再依赖配置文件中的凭证

### 🐛 Bug 修复
- **凭证匹配修复**：自建平台（Gitea/GitLab）优先按 BaseURL 精确匹配，解决自建 GitLab 被误识别为 Gitea 的问题
- **Gitea 迁移 API 修复**：`service` 字段从整数改为字符串，修复 HTTP 422 错误
- **配置文件覆盖修复**：构建时仅在 `bin/config.yaml` 不存在时复制初始模板，避免每次 `wails3 dev` 丢失已有配置
- **错误信息优化**：迁移相关错误信息显示实际 URL 而非推断的平台名

### 🔨 改进
- **智能迁移路径**：源平台开启代理时自动跳过 Gitea 原生迁移，改用本地代理中转（解决国内服务器无法访问 GitHub 的问题）
- **推送超时优化**：在线迁移的 push 操作超时从 30 秒提升到 10 分钟
- **大仓库推送**：设置 `http.postBuffer=500MB`，push --all 失败时自动回退为逐分支推送，避免 HTTP 413
- **Git 操作**：新增 `RunWithProxyTimeout` 方法，支持自定义超时的 git 命令执行

---

## v0.4.0 (2026-03-12)

### 🔧 架构重构
- **配置架构重构**：从 `Platform → User → Project` 层级结构重构为扁平的 `Projects + Groups + Credentials` 模型
  - 项目支持多远程仓库指向不同平台，不再受限于单一平台层级
  - 新增分组（Group）管理，支持自定义分组和图标
  - 新增凭据（Credential）管理，支持 GitHub / Gitee / Gitea 等多平台凭据
  - 侧边栏完全重写，适配新的分组→项目树结构

### ✨ 新功能
- **一键多端推送**：Push All 按钮，一键将当前分支推送到所有远程仓库
  - 顶部操作栏新增 "Push All" 按钮（仅在有多个远程仓库时启用）
  - 标签管理新增 "推送到所有远程" 按钮
  - 推送结果逐一展示每个远程仓库的成功/失败状态
- **项目代理开关**：每个项目可独立控制是否使用系统代理
  - 三种模式：跟随全局 / 强制使用代理 / 强制不使用代理
  - 顶部操作栏新增代理设置下拉菜单
  - Pull / Push / Fetch / Push All 等所有网络操作均尊重项目级代理设置

### 🔨 后端改进
- `GitClient` 新增 `RunWithProxy()` 方法，支持按调用级别覆盖全局代理设置
- 所有网络操作（Pull/Push/Fetch/PushToAllRemotes/PushTagToAllRemotes）均支持项目级代理
- 配置管理方法重构：Group CRUD / Credential CRUD / Project 分组管理
- 新增 `PushToAllRemotes()` / `PushTagToAllRemotes()` 服务方法
- 新增 `GetProjectProxy()` / `SetProjectProxy()` 服务方法

---

## v0.3.1 (2026-02-27)

### ✨ 新功能
- **多 Remote 远程仓库支持**
  - 新增 RemoteList / AddRemote / RemoveRemote 远程仓库管理
  - 所有远程操作支持指定 remote（Push/Pull/Fetch/PushTag/DeleteTag 等）
  - 顶部信息栏新增 remote 选择器下拉菜单
  - 支持切换当前操作的远程仓库、添加/删除远程仓库

---

## v0.3.0 (2026-02-25)

### ✨ 新功能
- **Git 安装检测**：启动时检测 Git 是否已安装，支持自动下载安装 Git for Windows
- **冲突处理**：合并冲突检测、冲突文件编辑器、标记已解决、中止/完成合并
- **提交搜索**：支持按提交信息关键字和作者搜索
- **批量操作**：侧边栏批量 Pull/Push 所有项目

---

## v0.2.0 (2026-02-20)

### ✨ 新功能
- 分支管理（创建/删除/合并）
- 远程分支（查看/检出/删除）
- Stash 贮藏管理
- Git 全局配置设置（user.name / user.email）
- Tag 标签管理
- CI/CD 发布流水线
- NSIS 安装包

---

## v0.1.0

- 初始版本
- 基本 Git 操作（Clone / Pull / Push / Commit）
- 文件变更查看和 Diff 展示
- 系统代理自动检测
