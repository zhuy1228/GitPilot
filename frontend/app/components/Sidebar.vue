<script setup>
import { ref, onMounted, computed, h } from 'vue'
import {
  FolderOutlined,
  FolderOpenOutlined,
  UserOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CloudDownloadOutlined,
  LoadingOutlined,
  AppstoreOutlined,
  CloudUploadOutlined,
  SyncOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  BranchesOutlined,
} from '@ant-design/icons-vue'
import { message, Modal } from 'ant-design-vue'
import { AppService } from '../../bindings/github.com/zhuy1228/GitPilot/internal/app'

const props = defineProps({
  selectedProject: { type: Object, default: null }
})

const emit = defineEmits(['select-project', 'tree-updated'])

const tree = ref([])
const expandedKeys = ref([])
const selectedKeys = ref([])

// 添加项目弹窗
const showAddDialog = ref(false)
const addForm = ref({ platform: '', username: '', name: '', path: '' })

// 克隆项目弹窗
const showCloneDialog = ref(false)
const cloneForm = ref({ platform: '', username: '', repoURL: '', parentDir: '', name: '' })
const cloneLoading = ref(false)

// ---- 批量操作 ----
const showBatchModal = ref(false)
const batchOverviews = ref([])
const batchLoading = ref(false)
const selectedBatchPaths = ref([])
const batchResults = ref([])
const batchActionLoading = ref('')
const batchResultMap = computed(() => {
  const map = {}
  batchResults.value.forEach(r => { map[r.path] = r })
  return map
})

// 右键菜单
const contextMenu = ref({ visible: false, x: 0, y: 0, type: '', data: {} })

// 平台弹窗
const showPlatformDialog = ref(false)
const platformDialogMode = ref('add')
const platformForm = ref({ name: '', baseUrl: '', oldName: '' })

// 用户弹窗
const showUserDialog = ref(false)
const userDialogMode = ref('add')
const userForm = ref({ platform: '', username: '', token: '', oldUsername: '' })

// 构建 key → 原始节点 的映射，避免 a-tree event node 丢失自定义字段
const nodeMap = computed(() => {
  const map = {}
  ;(tree.value || []).forEach(platform => {
    platform.children?.forEach(user => {
      user.children?.forEach(proj => {
        map[proj.key] = proj
      })
    })
  })
  return map
})

// 转换后端树数据为 a-tree 格式
const treeData = computed(() => {
  return (tree.value || []).map(platform => ({
    key: platform.key,
    title: platform.label,
    type: 'platform',
    isLeaf: false,
    children: (platform.children || []).map(user => ({
      key: user.key,
      title: user.label,
      type: 'user',
      platformKey: platform.key,
      isLeaf: false,
      children: (user.children || []).map(proj => ({
        key: proj.key,
        title: proj.label,
        type: 'project',
        path: proj.path,
        platformKey: platform.key,
        username: user.label,
        isLeaf: true,
      }))
    }))
  }))
})

async function loadTree() {
  try {
    const result = await AppService.GetProjectTree()
    tree.value = result || []
    const keys = []
    tree.value.forEach(node => {
      keys.push(node.key)
      if (node.children) {
        node.children.forEach(child => {
          keys.push(child.key)
        })
      }
    })
    expandedKeys.value = keys
  } catch (e) {
    console.error('加载项目树失败:', e)
  }
}

function onTreeSelect(keys, { node }) {
  if (node.type === 'project') {
    selectedKeys.value = keys
    // 从 nodeMap 中取 path，避免 a-tree 事件节点丢失自定义字段
    const raw = nodeMap.value[node.key]
    const path = raw?.path || node.path || ''
    console.log('[Sidebar] select project:', node.key, 'path:', path)
    emit('select-project', { key: node.key, label: node.title || raw?.label, path, type: 'project' })
  }
}

// --- 右键菜单 ---
function onRightClick({ event, node }) {
  event.preventDefault()
  const type = node.type
  const data = {}
  if (type === 'platform') {
    data.key = node.key
  } else if (type === 'user') {
    data.platformKey = node.platformKey
    data.username = node.title
  } else if (type === 'project') {
    data.platformKey = node.platformKey
    data.username = node.username
    data.name = node.title
  }
  contextMenu.value = { visible: true, x: event.clientX, y: event.clientY, type, data }
}

function onSidebarContextMenu(e) {
  e.preventDefault()
  contextMenu.value = { visible: true, x: e.clientX, y: e.clientY, type: 'empty', data: {} }
}

function hideContextMenu() {
  contextMenu.value.visible = false
}

function onContextMenuClick({ key: action }) {
  const { type, data } = contextMenu.value
  hideContextMenu()
  switch (action) {
    case 'add-platform': openAddPlatformDialog(); break
    case 'add-user': openAddUserDialog(data.key || data.platformKey); break
    case 'edit-platform': openEditPlatformDialog(data.key); break
    case 'remove-platform': removePlatform(data.key); break
    case 'add-project': openAddDialog(data.platformKey, data.username); break
    case 'clone-project': openCloneDialog(data.platformKey, data.username); break
    case 'edit-user': openEditUserDialog(data.platformKey, data.username); break
    case 'remove-user': removeUser(data.platformKey, data.username); break
    case 'remove-project': removeProject(data.platformKey, data.username, data.name); break
  }
}

const contextMenuItems = computed(() => {
  const { type } = contextMenu.value
  if (type === 'empty') {
    return [{ key: 'add-platform', label: '添加平台', icon: h(PlusOutlined) }]
  }
  if (type === 'platform') {
    return [
      { key: 'add-user', label: '添加用户', icon: h(UserOutlined) },
      { key: 'edit-platform', label: '编辑平台', icon: h(EditOutlined) },
      { type: 'divider' },
      { key: 'add-platform', label: '添加平台', icon: h(PlusOutlined) },
      { type: 'divider' },
      { key: 'remove-platform', label: '删除平台', danger: true, icon: h(DeleteOutlined) },
    ]
  }
  if (type === 'user') {
    return [
      { key: 'add-project', label: '添加项目', icon: h(FolderOutlined) },
      { key: 'clone-project', label: '克隆项目', icon: h(CloudDownloadOutlined) },
      { key: 'edit-user', label: '编辑用户', icon: h(EditOutlined) },
      { type: 'divider' },
      { key: 'remove-user', label: '删除用户', danger: true, icon: h(DeleteOutlined) },
    ]
  }
  if (type === 'project') {
    return [
      { key: 'add-project', label: '添加项目', icon: h(FolderOutlined) },
      { key: 'clone-project', label: '克隆项目', icon: h(CloudDownloadOutlined) },
      { type: 'divider' },
      { key: 'remove-project', label: '删除项目', danger: true, icon: h(DeleteOutlined) },
    ]
  }
  return []
})

// --- 平台操作 ---
function openAddPlatformDialog() {
  platformDialogMode.value = 'add'
  platformForm.value = { name: '', baseUrl: '', oldName: '' }
  showPlatformDialog.value = true
}

async function openEditPlatformDialog(platformKey) {
  try {
    const info = await AppService.GetPlatformInfo(platformKey)
    platformDialogMode.value = 'edit'
    platformForm.value = { name: info.name, baseUrl: info.baseUrl || '', oldName: info.name }
    showPlatformDialog.value = true
  } catch (e) {
    console.error('获取平台信息失败:', e)
  }
}

async function savePlatform() {
  if (!platformForm.value.name) return
  try {
    if (platformDialogMode.value === 'add') {
      await AppService.AddPlatform(platformForm.value.name, platformForm.value.baseUrl)
    } else {
      await AppService.UpdatePlatform(platformForm.value.oldName, platformForm.value.baseUrl)
    }
    showPlatformDialog.value = false
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    console.error('操作失败:', e)
  }
}

async function removePlatform(name) {
  if (!confirm(`确定删除平台 "${name}" 及其所有用户和项目吗？`)) return
  try {
    await AppService.RemovePlatform(name)
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    console.error('删除失败:', e)
  }
}

// --- 用户操作 ---
function openAddUserDialog(platformKey) {
  userDialogMode.value = 'add'
  userForm.value = { platform: platformKey, username: '', token: '', oldUsername: '' }
  showUserDialog.value = true
}

async function openEditUserDialog(platformKey, username) {
  try {
    const info = await AppService.GetUserInfo(platformKey, username)
    userDialogMode.value = 'edit'
    userForm.value = {
      platform: platformKey,
      username: info.username,
      token: info.token || '',
      oldUsername: info.username
    }
    showUserDialog.value = true
  } catch (e) {
    console.error('获取用户信息失败:', e)
  }
}

async function saveUser() {
  if (!userForm.value.username) return
  try {
    if (userDialogMode.value === 'add') {
      await AppService.AddUser(userForm.value.platform, userForm.value.username, userForm.value.token)
    } else {
      await AppService.UpdateUser(
        userForm.value.platform,
        userForm.value.oldUsername,
        userForm.value.username,
        userForm.value.token
      )
    }
    showUserDialog.value = false
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    console.error('操作失败:', e)
  }
}

async function removeUser(platform, username) {
  if (!confirm(`确定删除用户 "${username}" 及其所有项目吗？`)) return
  try {
    await AppService.RemoveUser(platform, username)
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    console.error('删除失败:', e)
  }
}

// --- 项目操作 ---
function openAddDialog(platformKey, username) {
  addForm.value = { platform: platformKey, username: username, name: '', path: '' }
  showAddDialog.value = true
}

async function pickDirectory() {
  try {
    const path = await AppService.SelectDirectory()
    if (path) {
      addForm.value.path = path
      // 如果项目名称为空，自动用文件夹名称填充
      if (!addForm.value.name) {
        const parts = path.replace(/\\/g, '/').split('/')
        addForm.value.name = parts[parts.length - 1] || ''
      }
    }
  } catch (e) {
    console.error('选择文件夹失败:', e)
  }
}

async function addProject() {
  if (!addForm.value.name || !addForm.value.path) return
  try {
    await AppService.AddProject(
      addForm.value.platform,
      addForm.value.username,
      addForm.value.name,
      addForm.value.path
    )
    showAddDialog.value = false
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    console.error('添加项目失败:', e)
  }
}

// --- 克隆项目 ---
function openCloneDialog(platformKey, username) {
  cloneForm.value = { platform: platformKey, username: username, repoURL: '', parentDir: '', name: '' }
  showCloneDialog.value = true
}

async function pickCloneDirectory() {
  try {
    const path = await AppService.SelectDirectory()
    if (path) {
      cloneForm.value.parentDir = path
    }
  } catch (e) {
    console.error('选择文件夹失败:', e)
  }
}

function onRepoURLChange() {
  // 从仓库 URL 自动提取项目名称
  if (!cloneForm.value.name && cloneForm.value.repoURL) {
    const url = cloneForm.value.repoURL.trim()
    const match = url.match(/\/([^\/]+?)(\.git)?$/)
    if (match) {
      cloneForm.value.name = match[1]
    }
  }
}

async function cloneProject() {
  if (!cloneForm.value.repoURL || !cloneForm.value.parentDir || !cloneForm.value.name) return
  cloneLoading.value = true
  try {
    await AppService.CloneProject(
      cloneForm.value.platform,
      cloneForm.value.username,
      cloneForm.value.repoURL,
      cloneForm.value.parentDir,
      cloneForm.value.name
    )
    showCloneDialog.value = false
    message.success('克隆成功')
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    console.error('克隆项目失败:', e)
    message.error('克隆失败: ' + String(e))
  } finally {
    cloneLoading.value = false
  }
}

async function removeProject(platform, username, name) {
  if (!confirm(`确定删除项目 "${name}" 吗？`)) return
  try {
    await AppService.RemoveProject(platform, username, name)
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    console.error('删除项目失败:', e)
  }
}

// ---- 批量操作 ----
async function openBatchModal() {
  showBatchModal.value = true
  batchResults.value = []
  await loadBatchOverview()
}

async function loadBatchOverview() {
  batchLoading.value = true
  try {
    const overviews = await AppService.GetAllProjectOverview()
    batchOverviews.value = Array.isArray(overviews) ? overviews : []
    selectedBatchPaths.value = batchOverviews.value
      .filter(p => !p.error)
      .map(p => p.path)
  } catch (e) {
    console.error('获取项目概览失败:', e)
  } finally {
    batchLoading.value = false
  }
}

function toggleBatchPath(path) {
  const idx = selectedBatchPaths.value.indexOf(path)
  if (idx >= 0) {
    selectedBatchPaths.value = selectedBatchPaths.value.filter(p => p !== path)
  } else {
    selectedBatchPaths.value = [...selectedBatchPaths.value, path]
  }
}

function toggleAllBatchPaths(e) {
  if (e.target.checked) {
    selectedBatchPaths.value = batchOverviews.value
      .filter(p => !p.error)
      .map(p => p.path)
  } else {
    selectedBatchPaths.value = []
  }
}

async function doBatchPull() {
  if (!selectedBatchPaths.value.length) return
  batchActionLoading.value = 'pull'
  try {
    const results = await AppService.BatchPull(selectedBatchPaths.value)
    batchResults.value = Array.isArray(results) ? results : []
    const successCount = batchResults.value.filter(r => r.success).length
    message.info(`批量 Pull 完成：${successCount}/${batchResults.value.length} 成功`)
    await loadBatchOverview()
  } catch (e) {
    Modal.error({ title: '批量 Pull 失败', content: String(e) })
  } finally {
    batchActionLoading.value = ''
  }
}

async function doBatchPush() {
  if (!selectedBatchPaths.value.length) return
  batchActionLoading.value = 'push'
  try {
    const results = await AppService.BatchPush(selectedBatchPaths.value)
    batchResults.value = Array.isArray(results) ? results : []
    const successCount = batchResults.value.filter(r => r.success).length
    message.info(`批量 Push 完成：${successCount}/${batchResults.value.length} 成功`)
    await loadBatchOverview()
  } catch (e) {
    Modal.error({ title: '批量 Push 失败', content: String(e) })
  } finally {
    batchActionLoading.value = ''
  }
}

// 平台 SVG Logo
const platformLogos = {
  github: `<svg viewBox="0 0 16 16" fill="currentColor"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>`,
  gitee: `<svg viewBox="0 0 1024 1024" fill="currentColor"><path d="M512 1024C229.222 1024 0 794.778 0 512S229.222 0 512 0s512 229.222 512 512-229.222 512-512 512z m259.149-568.883h-290.74a25.293 25.293 0 0 0-25.292 25.293l-0.026 63.206c0 13.952 11.315 25.293 25.267 25.293h177.024c13.978 0 25.293 11.315 25.293 25.267v12.646a75.853 75.853 0 0 1-75.853 75.853h-240.23a25.293 25.293 0 0 1-25.267-25.293V417.203a75.853 75.853 0 0 1 75.827-75.853h353.946a25.293 25.293 0 0 0 25.267-25.292l0.077-63.207a25.293 25.293 0 0 0-25.268-25.293H417.152a189.62 189.62 0 0 0-189.62 189.645V771.15c0 13.977 11.316 25.293 25.294 25.293h372.94a170.65 170.65 0 0 0 170.65-170.65V480.384a25.293 25.293 0 0 0-25.293-25.267z"/></svg>`,
  gitea: `<svg viewBox="0 0 640 640" fill="currentColor"><path d="M395.022 297.778c-13.158 0-23.822 10.664-23.822 23.822s10.664 23.822 23.822 23.822 23.822-10.664 23.822-23.822-10.664-23.822-23.822-23.822zM243.2 297.778c-13.158 0-23.822 10.664-23.822 23.822s10.664 23.822 23.822 23.822 23.822-10.664 23.822-23.822-10.664-23.822-23.822-23.822zM319.111 22.756C158.578 22.756 28.444 152.889 28.444 313.422s130.133 290.667 290.667 290.667 290.667-130.133 290.667-290.667S479.644 22.756 319.111 22.756zm165.689 361.244c0 5.333-0.711 10.667-1.778 15.644-20.267 93.867-148.089 167.111-305.067 167.111a360.604 360.604 0 0 1-65.778-5.689c-21.333 18.133-56.889 37.689-101.333 49.422-7.822 2.133-16 3.911-24.889 5.333h-0.711c-4.267 0-7.822-3.556-8.889-7.822v-0.356c-1.067-4.622 2.133-7.467 4.978-10.667 17.067-18.844 36.622-34.844 48-71.467-51.911-36.267-83.911-85.067-83.911-139.733 0-106.311 107.378-192.356 239.644-192.356 118.4 0 220.089 68.267 238.578 160.356 1.778 6.4 3.556 14.578 3.556 22.4-0.356 2.489-0.356 5.333-0.356 7.822z"/></svg>`,
  default: `<svg viewBox="0 0 16 16" fill="currentColor"><path d="M15 5.6c0-.3-.2-.6-.5-.7L8.3.2a.6.6 0 0 0-.6 0L1.5 4.9c-.3.1-.5.4-.5.7v5.8c0 .3.2.6.5.7l6.2 3.7c.2.1.4.1.6 0l6.2-3.7c.3-.1.5-.4.5-.7V5.6zM8 1.2l5.2 3.1L8 7.5 2.8 4.3 8 1.2zm-6 4.5l5.5 3.2v6.3L2 12V5.7zm7 9.5V9l5.5-3.2V12L9 15.2z"/></svg>`
}

function getPlatformColor(name) {
  const colors = { github: '#e6edf3', gitee: '#c71d23', gitea: '#609926' }
  return colors[name] || 'var(--text-secondary)'
}

onMounted(() => {
  loadTree()
})
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <span class="logo">GitPilot</span>
      <a-space :size="0">
        <a-button type="text" size="small" @click="openBatchModal" title="批量操作">
          <template #icon><AppstoreOutlined /></template>
        </a-button>
        <a-button type="text" size="small" @click="openAddPlatformDialog" title="添加平台">
          <template #icon><PlusOutlined /></template>
        </a-button>
      </a-space>
    </div>

    <div class="sidebar-content" @contextmenu="onSidebarContextMenu">
      <a-tree
        v-if="treeData.length"
        v-model:expandedKeys="expandedKeys"
        v-model:selectedKeys="selectedKeys"
        :tree-data="treeData"
        block-node
        :show-icon="true"
        @select="onTreeSelect"
        @rightClick="onRightClick"
        @contextmenu.stop
      >
        <template #title="{ title, type, key }">
          <div class="tree-node-title">
            <span
              v-if="type === 'platform'"
              class="platform-logo"
              :style="{ color: getPlatformColor(key) }"
              v-html="platformLogos[key] || platformLogos.default"
            ></span>
            <UserOutlined v-else-if="type === 'user'" class="node-icon" />
            <FolderOutlined v-else class="node-icon" />
            <span class="node-label">{{ title }}</span>
          </div>
        </template>
      </a-tree>
      <div v-else class="empty-tip">右键可添加平台</div>
    </div>

    <!-- 右键菜单 -->
    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="context-menu-mask"
        @click="hideContextMenu"
        @contextmenu.prevent="hideContextMenu"
      >
        <div @click.stop @contextmenu.stop>
          <a-menu
            class="context-menu-popup"
            :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
            :items="contextMenuItems"
            @click="onContextMenuClick"
            mode="vertical"
            :selectable="false"
          />
        </div>
      </div>
    </Teleport>

    <!-- 添加项目弹窗 -->
    <a-modal
      v-model:open="showAddDialog"
      title="添加项目"
      @ok="addProject"
      ok-text="添加"
      cancel-text="取消"
      :width="420"
    >
      <a-form layout="vertical" :style="{ marginTop: '16px' }">
        <a-form-item label="平台">
          <a-input :value="addForm.platform" disabled />
        </a-form-item>
        <a-form-item label="用户">
          <a-input :value="addForm.username" disabled />
        </a-form-item>
        <a-form-item label="项目名称">
          <a-input v-model:value="addForm.name" placeholder="my-project" />
        </a-form-item>
        <a-form-item label="本地路径">
          <a-input v-model:value="addForm.path" placeholder="F:/Projects/xxx">
            <template #suffix>
              <FolderOpenOutlined
                style="cursor: pointer; color: var(--accent, #89b4fa);"
                title="选择文件夹"
                @click="pickDirectory"
              />
            </template>
          </a-input>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 克隆项目弹窗 -->
    <a-modal
      v-model:open="showCloneDialog"
      title="克隆项目"
      @ok="cloneProject"
      :ok-text="cloneLoading ? '克隆中...' : '克隆'"
      cancel-text="取消"
      :ok-button-props="{ loading: cloneLoading, disabled: !cloneForm.repoURL || !cloneForm.parentDir || !cloneForm.name }"
      :closable="!cloneLoading"
      :maskClosable="!cloneLoading"
      :width="480"
    >
      <a-form layout="vertical" :style="{ marginTop: '16px' }">
        <a-form-item label="平台">
          <a-input :value="cloneForm.platform" disabled />
        </a-form-item>
        <a-form-item label="用户">
          <a-input :value="cloneForm.username" disabled />
        </a-form-item>
        <a-form-item label="仓库地址" required>
          <a-input
            v-model:value="cloneForm.repoURL"
            placeholder="https://github.com/user/repo.git"
            @blur="onRepoURLChange"
          />
        </a-form-item>
        <a-form-item label="克隆到目录" required>
          <a-input v-model:value="cloneForm.parentDir" placeholder="选择父目录" read-only>
            <template #suffix>
              <FolderOpenOutlined
                style="cursor: pointer; color: var(--accent, #89b4fa);"
                title="选择文件夹"
                @click="pickCloneDirectory"
              />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item label="项目名称" required>
          <a-input v-model:value="cloneForm.name" placeholder="自动从仓库地址提取" />
        </a-form-item>
        <div v-if="cloneForm.parentDir && cloneForm.name" style="font-size: 12px; color: var(--text-muted); margin-top: -8px; margin-bottom: 8px;">
          <span>目标路径：</span>
          <span style="color: var(--accent, #89b4fa);">{{ cloneForm.parentDir }}/{{ cloneForm.name }}</span>
        </div>
      </a-form>
    </a-modal>

    <!-- 平台弹窗 -->
    <a-modal
      v-model:open="showPlatformDialog"
      :title="platformDialogMode === 'add' ? '添加平台' : '编辑平台'"
      @ok="savePlatform"
      :ok-text="platformDialogMode === 'add' ? '添加' : '保存'"
      cancel-text="取消"
      :width="420"
    >
      <a-form layout="vertical" :style="{ marginTop: '16px' }">
        <a-form-item label="平台名称">
          <a-input
            v-model:value="platformForm.name"
            placeholder="github / gitee / gitea"
            :disabled="platformDialogMode === 'edit'"
          />
        </a-form-item>
        <a-form-item>
          <template #label>
            Base URL <span class="form-hint">（自建平台需要填写）</span>
          </template>
          <a-input v-model:value="platformForm.baseUrl" placeholder="http://192.168.1.10:3000" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 用户弹窗 -->
    <a-modal
      v-model:open="showUserDialog"
      :title="userDialogMode === 'add' ? '添加用户' : '编辑用户'"
      @ok="saveUser"
      :ok-text="userDialogMode === 'add' ? '添加' : '保存'"
      cancel-text="取消"
      :width="420"
    >
      <a-form layout="vertical" :style="{ marginTop: '16px' }">
        <a-form-item label="平台">
          <a-input :value="userForm.platform" disabled />
        </a-form-item>
        <a-form-item label="用户名">
          <a-input v-model:value="userForm.username" placeholder="your-username" />
        </a-form-item>
        <a-form-item>
          <template #label>
            Token <span class="form-hint">（可选，用于 API 认证）</span>
          </template>
          <a-input-password v-model:value="userForm.token" placeholder="ghp_xxxx / Bearer token" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 批量操作弹窗 -->
    <a-modal
      v-model:open="showBatchModal"
      title="批量操作"
      :width="600"
      :footer="null"
    >
      <div class="batch-panel">
        <div class="batch-header">
          <a-checkbox
            :checked="selectedBatchPaths.length === batchOverviews.filter(p => !p.error).length && batchOverviews.length > 0"
            :indeterminate="selectedBatchPaths.length > 0 && selectedBatchPaths.length < batchOverviews.filter(p => !p.error).length"
            @change="toggleAllBatchPaths"
          >
            全选
          </a-checkbox>
          <span style="flex:1"></span>
          <a-space :size="4">
            <a-button size="small" :loading="batchActionLoading === 'pull'" :disabled="!selectedBatchPaths.length" @click="doBatchPull">
              <template #icon><CloudDownloadOutlined /></template>
              Pull
            </a-button>
            <a-button size="small" :loading="batchActionLoading === 'push'" :disabled="!selectedBatchPaths.length" @click="doBatchPush">
              <template #icon><CloudUploadOutlined /></template>
              Push
            </a-button>
            <a-button size="small" :loading="batchLoading" @click="loadBatchOverview">
              <template #icon><SyncOutlined /></template>
            </a-button>
          </a-space>
        </div>
        <div class="batch-list">
          <template v-if="batchLoading">
            <div v-for="i in 4" :key="i" style="padding: 10px 12px;">
              <div style="height: 14px; background: var(--bg-hover, #333); border-radius: 4px; animation: pulse 1.5s infinite;" :style="{ width: (50 + i * 8) + '%' }"></div>
            </div>
          </template>
          <div v-else-if="!batchOverviews.length" style="padding: 24px; text-align: center; color: var(--text-muted);">
            暂无项目
          </div>
          <template v-else>
            <div
              v-for="proj in batchOverviews"
              :key="proj.key"
              class="batch-item"
              :class="{ error: !!proj.error }"
            >
              <a-checkbox
                :checked="selectedBatchPaths.includes(proj.path)"
                :disabled="!!proj.error"
                @change="toggleBatchPath(proj.path)"
                style="flex-shrink: 0;"
              />
              <div class="batch-item-info">
                <div class="batch-item-name">{{ proj.name }}</div>
                <div class="batch-item-meta">
                  <a-tag v-if="proj.branch" size="small" color="green">
                    <BranchesOutlined /> {{ proj.branch }}
                  </a-tag>
                  <a-tag v-if="proj.hasChanges" size="small" color="orange">有变更</a-tag>
                  <a-tag v-if="proj.unpushed > 0" size="small" color="blue">{{ proj.unpushed }} 未推送</a-tag>
                  <span v-if="proj.error" style="color: #f38ba8; font-size: 11px;">{{ proj.error }}</span>
                </div>
              </div>
              <div v-if="batchResultMap[proj.path]" class="batch-result-icon">
                <CheckCircleOutlined v-if="batchResultMap[proj.path].success" style="color: #a6e3a1;" />
                <a-tooltip v-else :title="batchResultMap[proj.path].message">
                  <CloseCircleOutlined style="color: #f38ba8;" />
                </a-tooltip>
              </div>
            </div>
          </template>
        </div>
      </div>
    </a-modal>
  </aside>
</template>

<style scoped>
.sidebar {
  height: 100%;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  min-width: 180px;
}

.sidebar-header {
  height: var(--header-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  border-bottom: 1px solid var(--border-color);
  --wails-draggable: drag;
}

.logo {
  font-size: 15px;
  font-weight: 600;
  color: var(--accent);
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.tree-node-title {
  display: flex;
  align-items: center;
  gap: 6px;
}

.platform-logo {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.platform-logo :deep(svg) {
  width: 16px;
  height: 16px;
}

.node-icon {
  font-size: 14px;
  color: var(--text-muted);
  flex-shrink: 0;
}

.node-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-tip {
  padding: 24px 16px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}

.form-hint {
  font-weight: 400;
  color: var(--text-muted);
  font-size: 11px;
}

/* 右键菜单遮罩 */
.context-menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2000;
}

.context-menu-popup {
  position: fixed !important;
  z-index: 2001;
  border-radius: 6px !important;
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.45) !important;
  min-width: 160px;
}

/* Ant Design 暗色主题覆盖 */
:deep(.ant-tree) {
  background: transparent;
  color: var(--text-primary);
}

:deep(.ant-tree .ant-tree-node-content-wrapper) {
  color: var(--text-primary);
}

:deep(.ant-tree .ant-tree-node-content-wrapper:hover) {
  background: var(--bg-hover);
}

:deep(.ant-tree .ant-tree-node-content-wrapper.ant-tree-node-selected) {
  background: var(--bg-active);
  color: var(--text-primary);
}

:deep(.ant-tree .ant-tree-switcher) {
  color: var(--text-muted);
}

:deep(.ant-tree .ant-tree-treenode) {
  padding: 2px 0;
}

:deep(.ant-tree .ant-tree-icon__customize) {
  display: none;
}

/* 批量操作 */
.batch-panel {
  margin-top: 8px;
}

.batch-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-color, #313244);
  margin-bottom: 8px;
}

.batch-list {
  max-height: 400px;
  overflow-y: auto;
}

.batch-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 4px;
  border-bottom: 1px solid var(--border-color, rgba(255,255,255,0.04));
  transition: background 0.12s;
}

.batch-item:hover {
  background: var(--bg-hover, rgba(255,255,255,0.04));
}

.batch-item.error {
  opacity: 0.6;
}

.batch-item-info {
  flex: 1;
  min-width: 0;
}

.batch-item-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.batch-item-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
  flex-wrap: wrap;
}

.batch-result-icon {
  flex-shrink: 0;
  font-size: 16px;
}

@keyframes pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 0.8; }
}
</style>
