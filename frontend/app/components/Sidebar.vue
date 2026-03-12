<script setup>
import { ref, onMounted, computed, h } from 'vue'
import {
  FolderOutlined,
  FolderOpenOutlined,
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
  KeyOutlined,
  GroupOutlined,
  SwapOutlined,
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
const addForm = ref({ name: '', path: '', group: '' })

// 克隆项目弹窗
const showCloneDialog = ref(false)
const cloneForm = ref({ repoURL: '', parentDir: '', name: '', group: '' })
const cloneLoading = ref(false)

// 分组弹窗
const showGroupDialog = ref(false)
const groupDialogMode = ref('add')
const groupForm = ref({ name: '', icon: '', oldName: '' })

// 凭证弹窗
const showCredentialDialog = ref(false)
const credentialDialogMode = ref('add')
const credentialForm = ref({ platform: '', baseUrl: '', username: '', token: '', oldPlatform: '', oldUsername: '' })

// 凭证列表弹窗
const showCredentialList = ref(false)
const credentials = ref([])

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

// 获取所有分组名（用于下拉选择）
const groupNames = computed(() => {
  const names = []
  ;(tree.value || []).forEach(node => {
    if (node.type === 'group') {
      names.push(node.label)
    }
  })
  return names
})

// 构建 key → 原始节点 的映射
const nodeMap = computed(() => {
  const map = {}
  ;(tree.value || []).forEach(group => {
    group.children?.forEach(proj => {
      map[proj.key] = proj
    })
  })
  return map
})

// 转换后端树数据为 a-tree 格式
const treeData = computed(() => {
  return (tree.value || []).map(group => ({
    key: group.key,
    title: group.label,
    type: 'group',
    icon: group.icon,
    isLeaf: false,
    children: (group.children || []).map(proj => ({
      key: proj.key,
      title: proj.label,
      type: 'project',
      path: proj.path,
      isLeaf: true,
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
    })
    expandedKeys.value = keys
  } catch (e) {
    console.error('加载项目树失败:', e)
  }
}

function onTreeSelect(keys, { node }) {
  if (node.type === 'project') {
    selectedKeys.value = keys
    const raw = nodeMap.value[node.key]
    const path = raw?.path || node.path || ''
    emit('select-project', { key: node.key, label: node.title || raw?.label, path, type: 'project' })
  }
}

// --- 右键菜单 ---
function onRightClick({ event, node }) {
  event.preventDefault()
  const type = node.type
  const data = {}
  if (type === 'group') {
    data.groupName = node.title
  } else if (type === 'project') {
    data.name = node.title
    data.path = node.path
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
    case 'add-group': openAddGroupDialog(); break
    case 'edit-group': openEditGroupDialog(data.groupName); break
    case 'remove-group': removeGroup(data.groupName); break
    case 'add-project': openAddDialog(data.groupName); break
    case 'clone-project': openCloneDialog(data.groupName); break
    case 'remove-project': removeProject(data.path); break
    case 'manage-credentials': openCredentialList(); break
  }
}

const contextMenuItems = computed(() => {
  const { type } = contextMenu.value
  if (type === 'empty') {
    return [
      { key: 'add-group', label: '添加分组', icon: h(GroupOutlined) },
      { key: 'add-project', label: '添加项目', icon: h(FolderOutlined) },
      { key: 'clone-project', label: '克隆项目', icon: h(CloudDownloadOutlined) },
      { type: 'divider' },
      { key: 'manage-credentials', label: '凭证管理', icon: h(KeyOutlined) },
    ]
  }
  if (type === 'group') {
    return [
      { key: 'add-project', label: '添加项目', icon: h(FolderOutlined) },
      { key: 'clone-project', label: '克隆项目', icon: h(CloudDownloadOutlined) },
      { type: 'divider' },
      { key: 'edit-group', label: '编辑分组', icon: h(EditOutlined) },
      { key: 'add-group', label: '添加分组', icon: h(GroupOutlined) },
      { type: 'divider' },
      { key: 'remove-group', label: '删除分组', danger: true, icon: h(DeleteOutlined) },
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

// --- 分组操作 ---
function openAddGroupDialog() {
  groupDialogMode.value = 'add'
  groupForm.value = { name: '', icon: 'folder', oldName: '' }
  showGroupDialog.value = true
}

function openEditGroupDialog(groupName) {
  groupDialogMode.value = 'edit'
  groupForm.value = { name: groupName, icon: '', oldName: groupName }
  showGroupDialog.value = true
}

async function saveGroup() {
  if (!groupForm.value.name) return
  try {
    if (groupDialogMode.value === 'add') {
      await AppService.AddGroup(groupForm.value.name, groupForm.value.icon || 'folder')
    } else {
      await AppService.UpdateGroup(groupForm.value.oldName, groupForm.value.name, groupForm.value.icon)
    }
    showGroupDialog.value = false
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    message.error('操作失败: ' + String(e))
  }
}

async function removeGroup(name) {
  if (!confirm(`确定删除分组 "${name}" 吗？分组下的项目将移到"未分组"。`)) return
  try {
    await AppService.RemoveGroup(name)
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    message.error('删除失败: ' + String(e))
  }
}

// --- 项目操作 ---
function openAddDialog(groupName) {
  addForm.value = { name: '', path: '', group: groupName || '' }
  showAddDialog.value = true
}

async function pickDirectory() {
  try {
    const path = await AppService.SelectDirectory()
    if (path) {
      addForm.value.path = path
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
    await AppService.AddProject(addForm.value.name, addForm.value.path, addForm.value.group)
    showAddDialog.value = false
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    message.error('添加项目失败: ' + String(e))
  }
}

// --- 克隆项目 ---
function openCloneDialog(groupName) {
  cloneForm.value = { repoURL: '', parentDir: '', name: '', group: groupName || '' }
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
      cloneForm.value.repoURL,
      cloneForm.value.parentDir,
      cloneForm.value.name,
      cloneForm.value.group
    )
    showCloneDialog.value = false
    message.success('克隆成功')
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    message.error('克隆失败: ' + String(e))
  } finally {
    cloneLoading.value = false
  }
}

async function removeProject(path) {
  const name = path.split(/[/\\]/).pop()
  if (!confirm(`确定删除项目 "${name}" 吗？`)) return
  try {
    await AppService.RemoveProject(path)
    await loadTree()
    emit('tree-updated')
  } catch (e) {
    message.error('删除项目失败: ' + String(e))
  }
}

// --- 凭证管理 ---
async function openCredentialList() {
  try {
    const list = await AppService.GetCredentials()
    credentials.value = list || []
  } catch (e) {
    credentials.value = []
  }
  showCredentialList.value = true
}

function openAddCredentialDialog() {
  credentialDialogMode.value = 'add'
  credentialForm.value = { platform: '', baseUrl: '', username: '', token: '', oldPlatform: '', oldUsername: '' }
  showCredentialDialog.value = true
}

function openEditCredentialDialog(cred) {
  credentialDialogMode.value = 'edit'
  credentialForm.value = {
    platform: cred.platform,
    baseUrl: cred.baseUrl || '',
    username: cred.username,
    token: cred.token || '',
    oldPlatform: cred.platform,
    oldUsername: cred.username
  }
  showCredentialDialog.value = true
}

async function saveCredential() {
  if (!credentialForm.value.platform || !credentialForm.value.username) return
  try {
    if (credentialDialogMode.value === 'add') {
      await AppService.AddCredential(
        credentialForm.value.platform,
        credentialForm.value.baseUrl,
        credentialForm.value.username,
        credentialForm.value.token
      )
    } else {
      await AppService.UpdateCredential(
        credentialForm.value.oldPlatform,
        credentialForm.value.oldUsername,
        credentialForm.value.baseUrl,
        credentialForm.value.token
      )
    }
    showCredentialDialog.value = false
    await openCredentialList()
  } catch (e) {
    message.error('操作失败: ' + String(e))
  }
}

async function removeCredential(platform, username) {
  if (!confirm(`确定删除凭证 "${platform}/${username}" 吗？`)) return
  try {
    await AppService.RemoveCredential(platform, username)
    await openCredentialList()
  } catch (e) {
    message.error('删除失败: ' + String(e))
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
    const results = await AppService.BatchPull(selectedBatchPaths.value, 'origin')
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
    const results = await AppService.BatchPush(selectedBatchPaths.value, 'origin')
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

// 分组图标
const groupIcons = {
  github: `<svg viewBox="0 0 16 16" fill="currentColor"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>`,
  gitee: `<svg viewBox="0 0 1024 1024" fill="currentColor"><path d="M512 1024C229.222 1024 0 794.778 0 512S229.222 0 512 0s512 229.222 512 512-229.222 512-512 512z m259.149-568.883h-290.74a25.293 25.293 0 0 0-25.292 25.293l-0.026 63.206c0 13.952 11.315 25.293 25.267 25.293h177.024c13.978 0 25.293 11.315 25.293 25.267v12.646a75.853 75.853 0 0 1-75.853 75.853h-240.23a25.293 25.293 0 0 1-25.267-25.293V417.203a75.853 75.853 0 0 1 75.827-75.853h353.946a25.293 25.293 0 0 0 25.267-25.292l0.077-63.207a25.293 25.293 0 0 0-25.268-25.293H417.152a189.62 189.62 0 0 0-189.62 189.645V771.15c0 13.977 11.316 25.293 25.294 25.293h372.94a170.65 170.65 0 0 0 170.65-170.65V480.384a25.293 25.293 0 0 0-25.293-25.267z"/></svg>`,
  gitea: `<svg viewBox="0 0 640 640" fill="currentColor"><path d="M395.022 297.778c-13.158 0-23.822 10.664-23.822 23.822s10.664 23.822 23.822 23.822 23.822-10.664 23.822-23.822-10.664-23.822-23.822-23.822zM243.2 297.778c-13.158 0-23.822 10.664-23.822 23.822s10.664 23.822 23.822 23.822 23.822-10.664 23.822-23.822-10.664-23.822-23.822-23.822zM319.111 22.756C158.578 22.756 28.444 152.889 28.444 313.422s130.133 290.667 290.667 290.667 290.667-130.133 290.667-290.667S479.644 22.756 319.111 22.756zm165.689 361.244c0 5.333-0.711 10.667-1.778 15.644-20.267 93.867-148.089 167.111-305.067 167.111a360.604 360.604 0 0 1-65.778-5.689c-21.333 18.133-56.889 37.689-101.333 49.422-7.822 2.133-16 3.911-24.889 5.333h-0.711c-4.267 0-7.822-3.556-8.889-7.822v-0.356c-1.067-4.622 2.133-7.467 4.978-10.667 17.067-18.844 36.622-34.844 48-71.467-51.911-36.267-83.911-85.067-83.911-139.733 0-106.311 107.378-192.356 239.644-192.356 118.4 0 220.089 68.267 238.578 160.356 1.778 6.4 3.556 14.578 3.556 22.4-0.356 2.489-0.356 5.333-0.356 7.822z"/></svg>`,
}

function getGroupIcon(icon) {
  return groupIcons[icon] || null
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
        <a-button type="text" size="small" @click="openAddGroupDialog" title="添加分组">
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
        <template #title="{ title, type, icon }">
          <div class="tree-node-title">
            <span
              v-if="type === 'group' && getGroupIcon(icon)"
              class="group-logo"
              v-html="getGroupIcon(icon)"
            ></span>
            <GroupOutlined v-else-if="type === 'group'" class="node-icon" />
            <FolderOutlined v-else class="node-icon" />
            <span class="node-label">{{ title }}</span>
          </div>
        </template>
      </a-tree>
      <div v-else class="empty-tip">右键添加分组或项目</div>
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
        <a-form-item label="分组">
          <a-select v-model:value="addForm.group" placeholder="选择分组（可选）" allow-clear>
            <a-select-option v-for="g in groupNames" :key="g" :value="g">{{ g }}</a-select-option>
          </a-select>
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
        <a-form-item label="分组">
          <a-select v-model:value="cloneForm.group" placeholder="选择分组（可选）" allow-clear>
            <a-select-option v-for="g in groupNames" :key="g" :value="g">{{ g }}</a-select-option>
          </a-select>
        </a-form-item>
        <div v-if="cloneForm.parentDir && cloneForm.name" style="font-size: 12px; color: var(--text-muted); margin-top: -8px; margin-bottom: 8px;">
          <span>目标路径：</span>
          <span style="color: var(--accent, #89b4fa);">{{ cloneForm.parentDir }}/{{ cloneForm.name }}</span>
        </div>
      </a-form>
    </a-modal>

    <!-- 分组弹窗 -->
    <a-modal
      v-model:open="showGroupDialog"
      :title="groupDialogMode === 'add' ? '添加分组' : '编辑分组'"
      @ok="saveGroup"
      :ok-text="groupDialogMode === 'add' ? '添加' : '保存'"
      cancel-text="取消"
      :width="420"
    >
      <a-form layout="vertical" :style="{ marginTop: '16px' }">
        <a-form-item label="分组名称">
          <a-input v-model:value="groupForm.name" placeholder="我的项目" />
        </a-form-item>
        <a-form-item label="图标">
          <a-select v-model:value="groupForm.icon" placeholder="选择图标">
            <a-select-option value="folder">📁 文件夹</a-select-option>
            <a-select-option value="github">🐙 GitHub</a-select-option>
            <a-select-option value="gitee">🟠 Gitee</a-select-option>
            <a-select-option value="gitea">🍵 Gitea</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 凭证列表弹窗 -->
    <a-modal
      v-model:open="showCredentialList"
      title="凭证管理"
      :footer="null"
      :width="520"
    >
      <div style="margin-bottom: 12px;">
        <a-button size="small" @click="openAddCredentialDialog">
          <template #icon><PlusOutlined /></template>
          添加凭证
        </a-button>
      </div>
      <div v-if="!credentials.length" style="padding: 24px; text-align: center; color: var(--text-muted);">
        暂无凭证
      </div>
      <div v-else class="credential-list">
        <div v-for="cred in credentials" :key="cred.platform + '/' + cred.username" class="credential-item">
          <div class="credential-info">
            <div class="credential-name">
              <a-tag :color="cred.platform === 'github' ? 'default' : cred.platform === 'gitee' ? 'red' : 'green'" size="small">
                {{ cred.platform }}
              </a-tag>
              <span>{{ cred.username }}</span>
            </div>
            <div v-if="cred.baseUrl" class="credential-url">{{ cred.baseUrl }}</div>
          </div>
          <a-space :size="4">
            <a-button type="text" size="small" @click="openEditCredentialDialog(cred)">
              <template #icon><EditOutlined /></template>
            </a-button>
            <a-button type="text" size="small" danger @click="removeCredential(cred.platform, cred.username)">
              <template #icon><DeleteOutlined /></template>
            </a-button>
          </a-space>
        </div>
      </div>
    </a-modal>

    <!-- 凭证编辑弹窗 -->
    <a-modal
      v-model:open="showCredentialDialog"
      :title="credentialDialogMode === 'add' ? '添加凭证' : '编辑凭证'"
      @ok="saveCredential"
      :ok-text="credentialDialogMode === 'add' ? '添加' : '保存'"
      cancel-text="取消"
      :width="420"
    >
      <a-form layout="vertical" :style="{ marginTop: '16px' }">
        <a-form-item label="平台">
          <a-select
            v-model:value="credentialForm.platform"
            placeholder="选择平台"
            :disabled="credentialDialogMode === 'edit'"
          >
            <a-select-option value="github">GitHub</a-select-option>
            <a-select-option value="gitee">Gitee</a-select-option>
            <a-select-option value="gitea">Gitea</a-select-option>
            <a-select-option value="gitlab">GitLab</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="用户名">
          <a-input
            v-model:value="credentialForm.username"
            placeholder="your-username"
            :disabled="credentialDialogMode === 'edit'"
          />
        </a-form-item>
        <a-form-item>
          <template #label>
            Base URL <span class="form-hint">（自建平台需要填写）</span>
          </template>
          <a-input v-model:value="credentialForm.baseUrl" placeholder="http://192.168.1.10:3000" />
        </a-form-item>
        <a-form-item>
          <template #label>
            Token <span class="form-hint">（可选，用于 API 认证）</span>
          </template>
          <a-input-password v-model:value="credentialForm.token" placeholder="ghp_xxxx / Bearer token" />
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

.group-logo {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.group-logo :deep(svg) {
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

/* 凭证列表 */
.credential-list {
  max-height: 400px;
  overflow-y: auto;
}

.credential-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 4px;
  border-bottom: 1px solid var(--border-color, rgba(255,255,255,0.04));
}

.credential-item:hover {
  background: var(--bg-hover, rgba(255,255,255,0.04));
}

.credential-info {
  flex: 1;
  min-width: 0;
}

.credential-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
}

.credential-url {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 2px;
}
</style>
