<script setup>
import { ref, computed, watch, h, onErrorCaptured, onBeforeUnmount } from 'vue'
import {
  SyncOutlined,
  CloudDownloadOutlined,
  CloudUploadOutlined,
  ReloadOutlined,
  FileOutlined,
  CheckCircleOutlined,
  PlusOutlined,
  MinusOutlined,
  UndoOutlined,
  CaretDownOutlined,
  CaretRightOutlined,
  BranchesOutlined,
  HistoryOutlined,
  EditOutlined,
  SwapOutlined,
  UserOutlined,
  RollbackOutlined,
  ExclamationCircleOutlined,
  CloseCircleOutlined,
  TagOutlined,
  TagsOutlined,
  CloudUploadOutlined as PushTagIcon,
  DeleteOutlined,
  SendOutlined,
  MergeCellsOutlined,
  InboxOutlined,
  SettingOutlined,
  GlobalOutlined,
  SaveOutlined,
  DownloadOutlined,
  SearchOutlined,
  WarningOutlined,
  StopOutlined,
} from '@ant-design/icons-vue'
import { Modal, message } from 'ant-design-vue'
import { AppService } from '../../bindings/github.com/zhuy1228/GitPilot/internal/app'
import FileTreeNode from './FileTreeNode.vue'
import CommitFileTreeNode from './CommitFileTreeNode.vue'

const props = defineProps({
  project: { type: Object, default: null }
})

const status = ref(null)
const changedFiles = ref([])
const loadingBase = ref(false)
const loadingFiles = ref(false)
const errorMsg = ref('')
const renderError = ref('')
const selectedFile = ref(null)
const fileContent = ref('')
const fileDiff = ref('')
const viewMode = ref('diff')
const actionLoading = ref('')
const collapsedDirs = ref(new Set())
const commitMessage = ref('')
const commitLoading = ref(false)
const stagedCollapsed = ref(false)
const unstagedCollapsed = ref(false)

// ---- 分支 & 提交历史 ----
const activeTab = ref('changes') // 'changes' | 'history'
const branches = ref([])
const branchLoading = ref(false)
const showBranchDropdown = ref(false)
const commitLogs = ref([])
const commitLogsLoading = ref(false)
const selectedCommit = ref(null)
const commitDiff = ref('')
const resetLoading = ref(false)
const commitFiles = ref([])
const commitFilesLoading = ref(false)
const selectedCommitFile = ref(null)
const commitFileDiff = ref('')
const commitCollapsedDirs = ref(new Set())

// ---- 标签管理 ----
const tags = ref([])
const tagsLoading = ref(false)
const showCreateTag = ref(false)
const newTagName = ref('')
const newTagMessage = ref('')
const createTagLoading = ref(false)

// ---- 分支管理 ----
const newBranchName = ref('')
const createBranchLoading = ref(false)
const remoteBranches = ref([])

// ---- Remote 远程仓库 ----
const remotes = ref([])
const currentRemote = ref('origin')
const showRemoteDropdown = ref(false)
const showAddRemote = ref(false)
const newRemoteName = ref('')
const newRemoteUrl = ref('')
const addRemoteLoading = ref(false)

// ---- Stash 贮藏管理 ----
const stashList = ref([])
const stashLoading = ref(false)
const showStashSave = ref(false)
const stashMessage = ref('')
const stashSaveLoading = ref(false)
const showStashPanel = ref(false)

// ---- 设置 ----
const showSettings = ref(false)
const settingsLoading = ref(false)
const gitConfig = ref({ userName: '', userEmail: '' })

// ---- 冲突处理 ----
const isMerging = ref(false)
const conflictFiles = ref([])
const selectedConflictFile = ref(null)
const conflictContent = ref('')
const conflictSaving = ref(false)

// ---- 提交搜索 ----
const searchKeyword = ref('')
const searchAuthor = ref('')
const searchLoading = ref(false)
const isSearchMode = ref(false)
const searchResults = ref([])
const displayLogs = computed(() => isSearchMode.value ? searchResults.value : commitLogs.value)

// ---- 文件列表拖拽调整宽度 ----
const fileListWidth = ref(300)
const MIN_FILELIST = 180
const MAX_FILELIST = 600
let draggingFileList = false

function startFileListResize(e) {
  e.preventDefault()
  draggingFileList = true
  const startX = e.clientX
  const startW = fileListWidth.value

  function onMove(ev) {
    if (!draggingFileList) return
    const delta = ev.clientX - startX
    fileListWidth.value = Math.min(MAX_FILELIST, Math.max(MIN_FILELIST, startW + delta))
  }
  function onUp() {
    draggingFileList = false
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

onBeforeUnmount(() => {
  draggingFileList = false
})

// 分离暂存区和工作区文件
const stagedFiles = computed(() => changedFiles.value.filter(f => f.staged))
const unstagedFiles = computed(() => changedFiles.value.filter(f => !f.staged))

// 将扁平文件列表构建为目录树
function buildTree(files) {
  if (!files.length) return []

  const root = { name: '', children: {}, files: [] }

  for (const file of files) {
    const parts = file.filePath.split('/')
    let current = root
    for (let i = 0; i < parts.length - 1; i++) {
      const dirName = parts[i]
      if (!current.children[dirName]) {
        current.children[dirName] = { name: dirName, children: {}, files: [] }
      }
      current = current.children[dirName]
    }
    current.files.push(file)
  }

  function toArray(node, parentPath) {
    const result = []
    const dirs = Object.values(node.children).sort((a, b) => a.name.localeCompare(b.name))
    for (const dir of dirs) {
      const dirPath = parentPath ? parentPath + '/' + dir.name : dir.name
      const children = toArray(dir, dirPath)
      const fileCount = countFiles(dir)
      result.push({ type: 'dir', name: dir.name, path: dirPath, children, fileCount })
    }
    const sortedFiles = [...node.files].sort((a, b) => {
      const nameA = a.filePath.split('/').pop()
      const nameB = b.filePath.split('/').pop()
      return nameA.localeCompare(nameB)
    })
    for (const f of sortedFiles) {
      const fileName = f.filePath.split('/').pop()
      result.push({ type: 'file', name: fileName, data: f })
    }
    return result
  }

  function countFiles(node) {
    let count = node.files.length
    for (const child of Object.values(node.children)) {
      count += countFiles(child)
    }
    return count
  }

  return toArray(root, '')
}

const stagedTree = computed(() => buildTree(stagedFiles.value))
const unstagedTree = computed(() => buildTree(unstagedFiles.value))

function toggleDir(dirPath) {
  const s = new Set(collapsedDirs.value)
  if (s.has(dirPath)) {
    s.delete(dirPath)
  } else {
    s.add(dirPath)
  }
  collapsedDirs.value = s
}

// 捕获子组件/自身渲染错误
onErrorCaptured((err) => {
  renderError.value = String(err)
  console.error('[ContentArea] 渲染错误:', err)
  return false
})

// 分步异步加载
async function loadStatus() {
  if (!props.project?.path) return
  loadingBase.value = true
  loadingFiles.value = false
  errorMsg.value = ''
  renderError.value = ''
  status.value = null
  changedFiles.value = []
  selectedFile.value = null
  fileContent.value = ''
  fileDiff.value = ''

  // 第一步：快速获取分支+远程URL
  try {
    const result = await AppService.GetProjectStatus(props.project.path)
    // 防御 Wails 绑定可能返回 null/undefined
    if (result && typeof result === 'object') {
      status.value = {
        branch: result.branch || '',
        remoteUrl: result.remoteUrl || result.remoteURL || '',
        remotes: Array.isArray(result.remotes) ? result.remotes : [],
        currentRemote: result.currentRemote || 'origin',
        changedFiles: [],
      }
      // 同步 remotes
      remotes.value = status.value.remotes
      if (remotes.value.length && !remotes.value.find(r => r.name === currentRemote.value)) {
        currentRemote.value = remotes.value[0].name
      }
    } else {
      errorMsg.value = '获取项目状态返回空值: ' + JSON.stringify(result)
      loadingBase.value = false
      return
    }
  } catch (e) {
    errorMsg.value = '获取项目状态失败: ' + String(e)
    loadingBase.value = false
    return
  }
  loadingBase.value = false

  // 第二步：异步加载变更文件（可能较慢）
  loadingFiles.value = true
  try {
    const files = await AppService.GetProjectChangedFiles(props.project.path)
    changedFiles.value = Array.isArray(files) ? files : []
  } catch (e) {
    console.error('获取变更文件失败:', e)
    // 不阻塞主渲染，仅文件列表显示异常
  } finally {
    loadingFiles.value = false
  }

  // 检查合并冲突状态
  try {
    isMerging.value = await AppService.IsMerging(props.project.path)
    if (isMerging.value) {
      const cFiles = await AppService.GetConflictFiles(props.project.path)
      conflictFiles.value = Array.isArray(cFiles) ? cFiles : []
    } else {
      conflictFiles.value = []
      selectedConflictFile.value = null
    }
  } catch (e) {
    console.error('检查合并状态失败:', e)
  }

  // 第三步：并行加载分支列表 + 提交历史
  loadBranches()
  if (activeTab.value === 'history') {
    loadCommitLog()
  }
}

function isNewFile(file) {
  return file && (file.status === 'A' || file.status === '?')
}

const binaryExts = new Set([
  '.png', '.jpg', '.jpeg', '.gif', '.bmp', '.ico', '.webp', '.svg',
  '.mp3', '.mp4', '.avi', '.mov', '.wav', '.flac', '.ogg',
  '.zip', '.tar', '.gz', '.bz2', '.7z', '.rar', '.xz',
  '.exe', '.dll', '.so', '.dylib', '.bin', '.dat',
  '.pdf', '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx',
  '.woff', '.woff2', '.ttf', '.otf', '.eot',
  '.class', '.pyc', '.o', '.a', '.lib',
  '.sqlite', '.db', '.mdb',
  '.icns',
])

function isBinaryFile(filePath) {
  if (!filePath) return false
  const ext = '.' + filePath.split('.').pop().toLowerCase()
  return binaryExts.has(ext)
}

async function selectFile(file) {
  selectedFile.value = file
  fileContent.value = ''
  fileDiff.value = ''

  // 二进制文件直接跳过读取
  if (isBinaryFile(file.filePath)) {
    viewMode.value = 'content'
    fileContent.value = '[二进制文件，不支持预览]'
    fileDiff.value = '[二进制文件，不支持预览]'
    return
  }

  if (isNewFile(file)) {
    // 新增/未跟踪文件：先获取全部内容，用内容模拟 diff
    viewMode.value = 'content'
    try {
      const content = await AppService.GetFileContent(props.project.path, file.filePath)
      fileContent.value = content
      const lines = content.split('\n')
      fileDiff.value = `--- /dev/null\n+++ b/${file.filePath}\n@@ -0,0 +1,${lines.length} @@\n` + lines.map(l => '+' + l).join('\n')
    } catch (e) {
      fileContent.value = '读取文件失败: ' + e
      fileDiff.value = '(新增文件，读取失败)'
    }
  } else {
    // 已有文件：根据是否暂存选择不同的 diff API
    viewMode.value = 'diff'
    try {
      const diff = file.staged
        ? await AppService.GetFileDiffStaged(props.project.path, file.filePath)
        : await AppService.GetFileDiff(props.project.path, file.filePath)
      fileDiff.value = diff || '(无差异)'
    } catch (e) {
      fileDiff.value = '获取 diff 失败: ' + e
    }
    try {
      const content = await AppService.GetFileContent(props.project.path, file.filePath)
      fileContent.value = content
    } catch (e) {
      fileContent.value = '读取文件失败: ' + e
    }
  }
}

// ---- 暂存/取消暂存/丢弃操作 ----

// 静默刷新变更文件列表（不触发 loading / 骨架屏，保留选中状态）
async function refreshFiles() {
  try {
    const files = await AppService.GetProjectChangedFiles(props.project.path)
    changedFiles.value = Array.isArray(files) ? files : []

    // 保持选中文件的引用同步：用最新数据中的同路径+同staged替换
    if (selectedFile.value) {
      const sel = selectedFile.value
      const match = changedFiles.value.find(
        f => f.filePath === sel.filePath && f.staged === sel.staged
      )
      if (!match) {
        // 如果原来选中的文件 staged 态变了（比如刚暂存），尝试找另一种态
        const altMatch = changedFiles.value.find(f => f.filePath === sel.filePath)
        if (altMatch) {
          selectedFile.value = altMatch
        } else {
          // 文件已不在变更列表中（比如被丢弃了），清空右侧
          selectedFile.value = null
          fileContent.value = ''
          fileDiff.value = ''
        }
      } else {
        selectedFile.value = match
      }
    }
  } catch (e) {
    console.error('刷新变更文件失败:', e)
  }
}

// 从文件列表中收集指定目录下的所有文件路径
function collectFilesInDir(dirPath, fileList) {
  return fileList.filter(f => f.filePath.startsWith(dirPath + '/') || f.filePath === dirPath).map(f => f.filePath)
}

async function stageFile(file) {
  try {
    await AppService.StageFiles(props.project.path, [file.filePath])
    await refreshFiles()
  } catch (e) {
    console.error('暂存失败:', e)
  }
}

async function unstageFile(file) {
  try {
    await AppService.UnstageFiles(props.project.path, [file.filePath])
    await refreshFiles()
  } catch (e) {
    console.error('取消暂存失败:', e)
  }
}

async function discardFile(file) {
  try {
    await AppService.DiscardFiles(props.project.path, [file.filePath])
    await refreshFiles()
  } catch (e) {
    console.error('丢弃更改失败:', e)
  }
}

async function stageDir(dirPath) {
  const paths = collectFilesInDir(dirPath, unstagedFiles.value)
  if (!paths.length) return
  try {
    await AppService.StageFiles(props.project.path, paths)
    await refreshFiles()
  } catch (e) {
    console.error('暂存目录失败:', e)
  }
}

async function unstageDir(dirPath) {
  const paths = collectFilesInDir(dirPath, stagedFiles.value)
  if (!paths.length) return
  try {
    await AppService.UnstageFiles(props.project.path, paths)
    await refreshFiles()
  } catch (e) {
    console.error('取消暂存目录失败:', e)
  }
}

async function discardDir(dirPath) {
  const paths = collectFilesInDir(dirPath, unstagedFiles.value)
  if (!paths.length) return
  try {
    await AppService.DiscardFiles(props.project.path, paths)
    await refreshFiles()
  } catch (e) {
    console.error('丢弃目录更改失败:', e)
  }
}

async function stageAll() {
  try {
    await AppService.StageAll(props.project.path)
    await refreshFiles()
  } catch (e) {
    console.error('暂存全部失败:', e)
  }
}

async function unstageAll() {
  try {
    await AppService.UnstageAll(props.project.path)
    await refreshFiles()
  } catch (e) {
    console.error('取消暂存全部失败:', e)
  }
}

async function discardAll() {
  // 丢弃所有未暂存的更改
  const paths = unstagedFiles.value.map(f => f.filePath)
  if (!paths.length) return
  try {
    await AppService.DiscardFiles(props.project.path, paths)
    await refreshFiles()
  } catch (e) {
    console.error('丢弃全部更改失败:', e)
  }
}

async function commitChanges() {
  if (!commitMessage.value.trim()) return
  commitLoading.value = true
  try {
    await AppService.CommitChanges(props.project.path, commitMessage.value.trim())
    commitMessage.value = ''
    await refreshFiles()
    // 刷新提交历史
    loadCommitLog()
  } catch (e) {
    console.error('提交失败:', e)
  } finally {
    commitLoading.value = false
  }
}

async function gitAction(action) {
  if (!props.project?.path) return
  actionLoading.value = action
  try {
    if (action === 'pull') {
      await AppService.PullProject(props.project.path, currentRemote.value)
    } else if (action === 'push') {
      await AppService.PushProject(props.project.path, currentRemote.value)
    } else {
      await AppService.FetchProject(props.project.path, currentRemote.value)
    }
    await loadStatus()
  } catch (e) {
    console.error(`${action} 失败:`, e)
  } finally {
    actionLoading.value = ''
  }
}

function getStatusColor(status) {
  const colors = { 'M': 'orange', 'A': 'green', 'D': 'red', '?': 'default' }
  return colors[status] || 'default'
}

function formatDiff(diff) {
  if (!diff) return ''
  return diff.split('\n').map(line => {
    if (line.startsWith('+') && !line.startsWith('+++')) {
      return `<span class="diff-add">${escapeHtml(line)}</span>`
    } else if (line.startsWith('-') && !line.startsWith('---')) {
      return `<span class="diff-del">${escapeHtml(line)}</span>`
    } else if (line.startsWith('@@')) {
      return `<span class="diff-hunk">${escapeHtml(line)}</span>`
    }
    return escapeHtml(line)
  }).join('\n')
}

function escapeHtml(text) {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

watch(() => props.project, loadStatus, { immediate: true })

// 切换到历史标签时加载提交历史
watch(activeTab, (tab) => {
  if (tab === 'history' && !commitLogs.value.length && !commitLogsLoading.value) {
    loadCommitLog()
  }
})

// ---- 分支管理 ----
async function loadBranches() {
  if (!props.project?.path) return
  try {
    const list = await AppService.GetBranches(props.project.path)
    branches.value = Array.isArray(list) ? list : []
  } catch (e) {
    console.error('获取分支失败:', e)
  }
  loadRemoteBranches()
  loadStashList()
}

async function switchBranch(branchName) {
  if (!props.project?.path) return
  branchLoading.value = true
  showBranchDropdown.value = false
  try {
    await AppService.SwitchBranch(props.project.path, branchName)
    await loadStatus()
  } catch (e) {
    console.error('切换分支失败:', e)
  } finally {
    branchLoading.value = false
  }
}

async function createBranch() {
  if (!props.project?.path || !newBranchName.value.trim()) return
  createBranchLoading.value = true
  try {
    await AppService.CreateBranch(props.project.path, newBranchName.value.trim())
    newBranchName.value = ''
    await loadBranches()
    message.success('分支创建成功')
  } catch (e) {
    console.error('创建分支失败:', e)
    Modal.error({ title: '创建分支失败', content: String(e) })
  } finally {
    createBranchLoading.value = false
  }
}

async function deleteBranch(branchName, force = false) {
  if (!props.project?.path) return
  Modal.confirm({
    title: '确认删除分支',
    icon: h(ExclamationCircleOutlined),
    content: h('div', [
      h('p', `确定要删除分支吗？`),
      h('p', { style: 'font-family: monospace; color: #89b4fa; font-size: 15px;' }, branchName),
    ]),
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { danger: true },
    async onOk() {
      try {
        await AppService.DeleteBranch(props.project.path, branchName, force)
        await loadBranches()
        message.success(`分支 ${branchName} 已删除`)
      } catch (e) {
        // 如果安全删除失败，提示是否强制删除
        if (!force && String(e).includes('not fully merged')) {
          Modal.confirm({
            title: '分支未完全合并',
            icon: h(ExclamationCircleOutlined),
            content: h('div', [
              h('p', `分支 ${branchName} 尚未完全合并到当前分支。`),
              h('p', { style: 'color: #f38ba8;' }, '是否强制删除？'),
            ]),
            okText: '强制删除',
            cancelText: '取消',
            okButtonProps: { danger: true },
            async onOk() {
              await deleteBranch(branchName, true)
            },
          })
        } else {
          Modal.error({ title: '删除分支失败', content: String(e) })
        }
      }
    },
  })
}

async function mergeBranch(branchName) {
  if (!props.project?.path) return
  Modal.confirm({
    title: '确认合并分支',
    icon: h(MergeCellsOutlined),
    content: h('div', [
      h('p', [
        '将 ',
        h('span', { style: 'font-family: monospace; color: #89b4fa;' }, branchName),
        ' 合并到 ',
        h('span', { style: 'font-family: monospace; color: #a6e3a1;' }, status.value?.branch || '当前分支'),
      ]),
    ]),
    okText: '合并',
    cancelText: '取消',
    async onOk() {
      try {
        const result = await AppService.MergeBranch(props.project.path, branchName)
        message.success('合并成功')
        await loadStatus()
      } catch (e) {
        console.error('合并分支失败:', e)
        Modal.error({ title: '合并失败', content: String(e) })
      }
    },
  })
}

// ---- 远程分支 ----
async function loadRemoteBranches() {
  if (!props.project?.path) return
  try {
    const list = await AppService.GetRemoteBranches(props.project.path, currentRemote.value)
    remoteBranches.value = Array.isArray(list) ? list : []
  } catch (e) {
    console.error('获取远程分支失败:', e)
  }
}

async function checkoutRemoteBranch(remoteBranch) {
  if (!props.project?.path) return
  showBranchDropdown.value = false
  try {
    await AppService.CheckoutRemoteBranch(props.project.path, remoteBranch)
    message.success('检出成功')
    await loadStatus()
  } catch (e) {
    console.error('检出远程分支失败:', e)
    Modal.error({ title: '检出失败', content: String(e) })
  }
}

async function deleteRemoteBranch(remoteBranch) {
  if (!props.project?.path) return
  Modal.confirm({
    title: '确认删除远程分支',
    icon: h(ExclamationCircleOutlined),
    content: h('div', [
      h('p', '确定要删除远程分支吗？此操作不可逆！'),
      h('p', { style: 'font-family: monospace; color: #f38ba8; font-size: 15px;' }, remoteBranch),
    ]),
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { danger: true },
    async onOk() {
      try {
        await AppService.DeleteRemoteBranch(props.project.path, remoteBranch, currentRemote.value)
        await loadRemoteBranches()
        message.success(`远程分支 ${remoteBranch} 已删除`)
      } catch (e) {
        Modal.error({ title: '删除远程分支失败', content: String(e) })
      }
    },
  })
}

// ---- Stash 贮藏管理 ----
async function loadStashList() {
  if (!props.project?.path) return
  stashLoading.value = true
  try {
    const list = await AppService.GetStashList(props.project.path)
    stashList.value = Array.isArray(list) ? list : []
  } catch (e) {
    console.error('获取贮藏列表失败:', e)
    stashList.value = []
  } finally {
    stashLoading.value = false
  }
}

async function stashSave() {
  if (!props.project?.path) return
  stashSaveLoading.value = true
  try {
    await AppService.StashSave(props.project.path, stashMessage.value.trim())
    stashMessage.value = ''
    showStashSave.value = false
    message.success('已贮藏当前变更')
    await refreshFiles()
    await loadStashList()
  } catch (e) {
    console.error('贮藏失败:', e)
    Modal.error({ title: '贮藏失败', content: String(e) })
  } finally {
    stashSaveLoading.value = false
  }
}

async function stashApply(index) {
  if (!props.project?.path) return
  try {
    await AppService.StashApply(props.project.path, index)
    message.success('已应用贮藏')
    await refreshFiles()
  } catch (e) {
    Modal.error({ title: '应用贮藏失败', content: String(e) })
  }
}

async function stashPop(index) {
  if (!props.project?.path) return
  try {
    await AppService.StashPop(props.project.path, index)
    message.success('已应用并删除贮藏')
    await refreshFiles()
    await loadStashList()
  } catch (e) {
    Modal.error({ title: '应用贮藏失败', content: String(e) })
  }
}

async function stashDrop(index) {
  if (!props.project?.path) return
  Modal.confirm({
    title: '确认删除贮藏',
    icon: h(ExclamationCircleOutlined),
    content: `确定要删除 stash@{${index}} 吗？`,
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { danger: true },
    async onOk() {
      try {
        await AppService.StashDrop(props.project.path, index)
        message.success('已删除贮藏')
        await loadStashList()
      } catch (e) {
        Modal.error({ title: '删除贮藏失败', content: String(e) })
      }
    },
  })
}

// ---- 设置 ----
async function openSettings() {
  showSettings.value = true
  settingsLoading.value = true
  try {
    const config = await AppService.GetGitGlobalConfig()
    gitConfig.value = {
      userName: config?.userName || '',
      userEmail: config?.userEmail || '',
    }
  } catch (e) {
    console.error('获取 Git 配置失败:', e)
  } finally {
    settingsLoading.value = false
  }
}

async function saveSettings() {
  settingsLoading.value = true
  try {
    await AppService.SetGitGlobalConfig(gitConfig.value.userName, gitConfig.value.userEmail)
    message.success('设置已保存')
    showSettings.value = false
  } catch (e) {
    Modal.error({ title: '保存设置失败', content: String(e) })
  } finally {
    settingsLoading.value = false
  }
}

// ---- 冲突处理 ----
async function selectConflictFile(file) {
  selectedConflictFile.value = file
  conflictContent.value = ''
  try {
    const content = await AppService.GetConflictFileContent(props.project.path, file.filePath)
    conflictContent.value = content
  } catch (e) {
    conflictContent.value = '读取冲突文件失败: ' + e
  }
}

async function saveConflictContent() {
  if (!selectedConflictFile.value) return
  conflictSaving.value = true
  try {
    await AppService.SaveConflictFile(props.project.path, selectedConflictFile.value.filePath, conflictContent.value)
    message.success('冲突文件已保存')
  } catch (e) {
    Modal.error({ title: '保存失败', content: String(e) })
  } finally {
    conflictSaving.value = false
  }
}

async function resolveConflictFiles(files) {
  if (!props.project?.path) return
  try {
    const paths = files.map(f => f.filePath)
    await AppService.ResolveConflictFile(props.project.path, paths)
    message.success('已标记为已解决')
    isMerging.value = await AppService.IsMerging(props.project.path)
    if (isMerging.value) {
      const cFiles = await AppService.GetConflictFiles(props.project.path)
      conflictFiles.value = Array.isArray(cFiles) ? cFiles : []
    } else {
      conflictFiles.value = []
    }
    selectedConflictFile.value = null
    await refreshFiles()
  } catch (e) {
    Modal.error({ title: '解决冲突失败', content: String(e) })
  }
}

async function completeMerge() {
  if (!props.project?.path) return
  const msg = commitMessage.value.trim() || 'Merge completed'
  commitLoading.value = true
  try {
    await AppService.CommitChanges(props.project.path, msg)
    commitMessage.value = ''
    isMerging.value = false
    conflictFiles.value = []
    selectedConflictFile.value = null
    await loadStatus()
  } catch (e) {
    Modal.error({ title: '完成合并失败', content: String(e) })
  } finally {
    commitLoading.value = false
  }
}

async function handleAbortMerge() {
  if (!props.project?.path) return
  Modal.confirm({
    title: '确认中止合并',
    icon: h(ExclamationCircleOutlined),
    content: '中止合并将丢弃所有合并更改，回到合并之前的状态。确定继续？',
    okText: '中止合并',
    cancelText: '取消',
    okButtonProps: { danger: true },
    async onOk() {
      try {
        await AppService.AbortMerge(props.project.path)
        message.success('已中止合并')
        isMerging.value = false
        conflictFiles.value = []
        selectedConflictFile.value = null
        await loadStatus()
      } catch (e) {
        Modal.error({ title: '中止合并失败', content: String(e) })
      }
    },
  })
}

// ---- 提交搜索 ----
async function searchCommits() {
  if (!props.project?.path) return
  if (!searchKeyword.value.trim() && !searchAuthor.value.trim()) return
  searchLoading.value = true
  isSearchMode.value = true
  try {
    const results = await AppService.SearchCommitLog(
      props.project.path,
      searchKeyword.value.trim(),
      searchAuthor.value.trim(),
      100
    )
    searchResults.value = Array.isArray(results) ? results : []
  } catch (e) {
    console.error('搜索提交失败:', e)
    searchResults.value = []
  } finally {
    searchLoading.value = false
  }
}

function clearSearch() {
  isSearchMode.value = false
  searchKeyword.value = ''
  searchAuthor.value = ''
  searchResults.value = []
}

// ---- 提交历史 ----
async function loadCommitLog() {
  if (!props.project?.path) return
  commitLogsLoading.value = true
  try {
    const logs = await AppService.GetCommitLog(props.project.path, 100)
    commitLogs.value = Array.isArray(logs) ? logs : []
  } catch (e) {
    console.error('获取提交历史失败:', e)
  } finally {
    commitLogsLoading.value = false
  }
}

async function selectCommit(commit) {
  selectedCommit.value = commit
  selectedCommitFile.value = null
  commitFileDiff.value = ''
  commitDiff.value = ''
  commitFiles.value = []
  commitCollapsedDirs.value = new Set()
  commitFilesLoading.value = true
  try {
    const files = await AppService.GetCommitFiles(props.project.path, commit.hash)
    commitFiles.value = Array.isArray(files) ? files : []
  } catch (e) {
    console.error('获取提交文件列表失败:', e)
  } finally {
    commitFilesLoading.value = false
  }
}

async function selectCommitFile(file) {
  if (!selectedCommit.value) return
  selectedCommitFile.value = file
  commitFileDiff.value = ''
  viewMode.value = 'diff'
  try {
    const diff = await AppService.GetCommitFileDiff(props.project.path, selectedCommit.value.hash, file.filePath)
    commitFileDiff.value = diff || '(无差异)'
  } catch (e) {
    commitFileDiff.value = '获取 diff 失败: ' + e
  }
}

function toggleCommitDir(dirPath) {
  const s = new Set(commitCollapsedDirs.value)
  if (s.has(dirPath)) s.delete(dirPath)
  else s.add(dirPath)
  commitCollapsedDirs.value = s
}

// 为提交文件构建树
const commitFileTree = computed(() => {
  if (!commitFiles.value.length) return []
  // 复用 buildTree 的逻辑，但需要适配 CommitFileInfo 结构
  const files = commitFiles.value.map(f => ({ ...f, staged: false }))
  return buildTree(files)
})

function getCommitStatusLabel(status) {
  const map = { M: '修改', A: '新增', D: '删除', R: '重命名', C: '复制' }
  return map[status] || status
}
function getCommitStatusColor(status) {
  const map = { M: 'orange', A: 'green', D: 'red', R: 'blue', C: 'cyan' }
  return map[status] || 'default'
}

async function resetToCommit(commit, mode = 'hard') {
  if (!props.project?.path || !commit) return
  const modeLabel = { hard: '硬回滚（丢弃所有更改）', soft: '软回滚（保留到暂存区）', mixed: '混合回滚（保留到工作区）' }
  Modal.confirm({
    title: '确认版本回滚',
    icon: h(ExclamationCircleOutlined),
    content: h('div', [
      h('p', `确定要回滚到以下提交吗？`),
      h('p', { style: 'font-family: monospace; color: #89b4fa;' }, `${commit.shortHash} - ${commit.message}`),
      h('p', { style: 'color: #f38ba8; font-size: 12px; margin-top: 8px;' }, `模式：${modeLabel[mode] || modeLabel.hard}`),
      mode === 'hard' ? h('p', { style: 'color: #f38ba8; font-size: 12px; font-weight: bold;' }, '⚠ 此操作不可逆，所有未提交的更改将被丢弃！') : null,
    ]),
    okText: '确认回滚',
    cancelText: '取消',
    okButtonProps: { danger: true },
    async onOk() {
      resetLoading.value = true
      try {
        await AppService.ResetProject(props.project.path, commit.hash, mode)
        // 回滚成功后刷新所有状态
        await loadStatus()
        selectedCommit.value = null
        commitDiff.value = ''
      } catch (e) {
        console.error('版本回滚失败:', e)
        Modal.error({ title: '回滚失败', content: String(e) })
      } finally {
        resetLoading.value = false
      }
    },
  })
}

async function revertCommit(commit) {
  if (!props.project?.path || !commit) return
  Modal.confirm({
    title: '确认撤回提交',
    icon: h(ExclamationCircleOutlined),
    content: h('div', [
      h('p', '将创建一个新提交来撤回以下提交的更改：'),
      h('p', { style: 'font-family: monospace; color: #89b4fa;' }, `${commit.shortHash} - ${commit.message}`),
      h('p', { style: 'color: var(--text-muted); font-size: 12px; margin-top: 8px;' }, '此操作会生成一个新的反向提交，不会丢失历史记录。'),
    ]),
    okText: '确认撤回',
    cancelText: '取消',
    okButtonProps: { danger: true },
    async onOk() {
      try {
        await AppService.RevertCommit(props.project.path, commit.hash)
        await loadStatus()
        selectedCommit.value = null
        commitDiff.value = ''
      } catch (e) {
        console.error('撤回提交失败:', e)
        Modal.error({ title: '撤回失败', content: String(e) })
      }
    },
  })
}

function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  const now = new Date()
  const diff = Math.floor((now - d) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return Math.floor(diff / 60) + ' 分钟前'
  if (diff < 86400) return Math.floor(diff / 3600) + ' 小时前'
  if (diff < 604800) return Math.floor(diff / 86400) + ' 天前'
  return d.toLocaleDateString('zh-CN')
}

// ---- 标签管理 ----
async function loadTags() {
  if (!props.project?.path) return
  tagsLoading.value = true
  try {
    const list = await AppService.GetTags(props.project.path)
    tags.value = Array.isArray(list) ? list : []
  } catch (e) {
    console.error('获取标签失败:', e)
    tags.value = []
  } finally {
    tagsLoading.value = false
  }
}

async function createTag() {
  if (!props.project?.path || !newTagName.value.trim()) return
  createTagLoading.value = true
  try {
    await AppService.CreateTag(props.project.path, newTagName.value.trim(), newTagMessage.value.trim() || newTagName.value.trim())
    newTagName.value = ''
    newTagMessage.value = ''
    showCreateTag.value = false
    await loadTags()
  } catch (e) {
    console.error('创建标签失败:', e)
    Modal.error({ title: '创建标签失败', content: String(e) })
  } finally {
    createTagLoading.value = false
  }
}

async function deleteTag(tag) {
  if (!props.project?.path) return
  Modal.confirm({
    title: '确认删除标签',
    icon: h(ExclamationCircleOutlined),
    content: h('div', [
      h('p', `确定要删除标签吗？`),
      h('p', { style: 'font-family: monospace; color: #89b4fa; font-size: 15px;' }, tag.name),
      h('p', { style: 'color: var(--text-muted); font-size: 12px; margin-top: 8px;' }, '将同时删除本地和远程标签。'),
    ]),
    okText: '确认删除',
    cancelText: '取消',
    okButtonProps: { danger: true },
    async onOk() {
      try {
        await AppService.DeleteTag(props.project.path, tag.name, currentRemote.value)
        await loadTags()
      } catch (e) {
        console.error('删除标签失败:', e)
        Modal.error({ title: '删除失败', content: String(e) })
      }
    },
  })
}

async function pushTag(tag) {
  if (!props.project?.path) return
  try {
    await AppService.PushTag(props.project.path, tag.name, currentRemote.value)
    Modal.success({ title: '推送成功', content: `标签 ${tag.name} 已推送到远程` })
  } catch (e) {
    console.error('推送标签失败:', e)
    Modal.error({ title: '推送失败', content: String(e) })
  }
}

// 切换到标签 Tab 时加载标签
watch(activeTab, (tab) => {
  if (tab === 'tags' && !tags.value.length && !tagsLoading.value) {
    loadTags()
  }
})

// ---- Remote 远程仓库管理 ----
async function switchRemote(remoteName) {
  currentRemote.value = remoteName
  showRemoteDropdown.value = false
  // 切换 remote 后刷新远程分支
  await loadRemoteBranches()
}

async function addRemote() {
  if (!props.project?.path || !newRemoteName.value.trim() || !newRemoteUrl.value.trim()) return
  addRemoteLoading.value = true
  try {
    await AppService.AddRemote(props.project.path, newRemoteName.value.trim(), newRemoteUrl.value.trim())
    newRemoteName.value = ''
    newRemoteUrl.value = ''
    showAddRemote.value = false
    message.success('远程仓库添加成功')
    await loadStatus()
  } catch (e) {
    Modal.error({ title: '添加远程仓库失败', content: String(e) })
  } finally {
    addRemoteLoading.value = false
  }
}

async function removeRemote(remoteName) {
  if (!props.project?.path) return
  Modal.confirm({
    title: '确认删除远程仓库',
    icon: h(ExclamationCircleOutlined),
    content: h('div', [
      h('p', '确定要删除远程仓库吗？'),
      h('p', { style: 'font-family: monospace; color: #f38ba8; font-size: 15px;' }, remoteName),
    ]),
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { danger: true },
    async onOk() {
      try {
        await AppService.RemoveRemote(props.project.path, remoteName)
        message.success(`远程仓库 ${remoteName} 已删除`)
        if (currentRemote.value === remoteName) {
          currentRemote.value = 'origin'
        }
        await loadStatus()
      } catch (e) {
        Modal.error({ title: '删除远程仓库失败', content: String(e) })
      }
    },
  })
}

const currentRemoteUrl = computed(() => {
  const r = remotes.value.find(r => r.name === currentRemote.value)
  return r ? r.url : status.value?.remoteUrl || ''
})
</script>

<template>
  <div class="content-area">
    <!-- 未选择项目 -->
    <div v-if="!project" class="empty-state">
      <FileOutlined :style="{ fontSize: '48px', color: 'var(--text-muted)' }" />
      <span style="color: var(--text-muted)">从左侧栏选择项目查看详情</span>
    </div>

    <!-- 项目内容 -->
    <div v-else class="project-view">
      <!-- 渲染错误提示 -->
      <div v-if="renderError" style="background: #f5222d; color: #fff; padding: 8px 16px; font-size: 13px;">
        渲染错误: {{ renderError }}
      </div>

      <!-- 错误状态 -->
      <div v-if="errorMsg" style="padding: 24px; text-align: center; color: var(--text-muted);">
        <div style="font-size: 36px; margin-bottom: 12px;">⚠️</div>
        <div>{{ errorMsg }}</div>
        <a-button style="margin-top: 12px;" @click="loadStatus">重试</a-button>
      </div>

      <!-- 正常内容 -->
      <template v-else>
        <!-- 顶部信息栏 -->
        <div class="project-header">
          <div class="project-info">
            <span class="project-name">{{ project.label }}</span>
            <template v-if="loadingBase">
              <a-spin size="small" />
            </template>
            <template v-else-if="status">
              <a-dropdown :open="showBranchDropdown" @openChange="v => showBranchDropdown = v" :trigger="['click']">
                <a-tag color="green" class="branch-tag" @click.prevent="showBranchDropdown = !showBranchDropdown">
                  <BranchesOutlined /> {{ status.branch }}
                  <CaretDownOutlined style="font-size: 10px; margin-left: 2px;" />
                </a-tag>
                <template #overlay>
                  <div class="branch-dropdown">
                    <!-- 创建新分支 -->
                    <div class="branch-create-box">
                      <a-input
                        v-model:value="newBranchName"
                        placeholder="新分支名称"
                        size="small"
                        style="flex:1"
                        @pressEnter="createBranch"
                      />
                      <a-button
                        type="primary"
                        size="small"
                        :loading="createBranchLoading"
                        :disabled="!newBranchName.trim()"
                        @click="createBranch"
                      >
                        <template #icon><PlusOutlined /></template>
                      </a-button>
                    </div>
                    <!-- 本地分支 -->
                    <div class="branch-dropdown-title">本地分支</div>
                    <div class="branch-dropdown-list">
                      <div
                        v-for="b in branches"
                        :key="b.name"
                        class="branch-dropdown-item"
                        :class="{ active: b.current }"
                      >
                        <div class="branch-item-main" @click="switchBranch(b.name)">
                          <BranchesOutlined style="font-size: 12px; margin-right: 6px;" />
                          <span class="branch-item-name">{{ b.name }}</span>
                          <span v-if="b.current" style="margin-left: auto; color: var(--success, #a6e3a1);">✓</span>
                        </div>
                        <div v-if="!b.current" class="branch-item-actions" @click.stop>
                          <a-tooltip title="合并到当前分支">
                            <span class="branch-action-btn" @click="mergeBranch(b.name)"><MergeCellsOutlined /></span>
                          </a-tooltip>
                          <a-tooltip title="删除分支">
                            <span class="branch-action-btn danger" @click="deleteBranch(b.name)"><DeleteOutlined /></span>
                          </a-tooltip>
                        </div>
                      </div>
                      <div v-if="!branches.length" style="padding: 12px; text-align: center; color: var(--text-muted);">
                        无分支数据
                      </div>
                    </div>
                    <!-- 远程分支 -->
                    <template v-if="remoteBranches.length">
                      <div class="branch-dropdown-title">远程分支</div>
                      <div class="branch-dropdown-list" style="max-height: 150px;">
                        <div
                          v-for="rb in remoteBranches"
                          :key="rb.name"
                          class="branch-dropdown-item remote"
                        >
                          <div class="branch-item-main" @click="checkoutRemoteBranch(rb.name)">
                            <GlobalOutlined style="font-size: 12px; margin-right: 6px; color: var(--text-muted);" />
                            <span class="branch-item-name">{{ rb.name }}</span>
                          </div>
                          <div class="branch-item-actions" @click.stop>
                            <a-tooltip title="检出到本地">
                              <span class="branch-action-btn" @click="checkoutRemoteBranch(rb.name)"><DownloadOutlined /></span>
                            </a-tooltip>
                            <a-tooltip title="删除远程分支">
                              <span class="branch-action-btn danger" @click="deleteRemoteBranch(rb.name)"><DeleteOutlined /></span>
                            </a-tooltip>
                          </div>
                        </div>
                      </div>
                    </template>
                  </div>
                </template>
              </a-dropdown>
              <span v-if="status.remoteUrl" class="remote-url" :title="currentRemoteUrl">
                <a-dropdown :open="showRemoteDropdown" @openChange="v => showRemoteDropdown = v" :trigger="['click']">
                  <span class="remote-selector" @click.prevent="showRemoteDropdown = !showRemoteDropdown">
                    <GlobalOutlined style="margin-right: 4px;" />
                    {{ currentRemote }}
                    <CaretDownOutlined style="font-size: 10px; margin-left: 2px;" />
                  </span>
                  <template #overlay>
                    <div class="remote-dropdown">
                      <div class="branch-dropdown-title">远程仓库</div>
                      <div class="branch-dropdown-list">
                        <div
                          v-for="r in remotes"
                          :key="r.name"
                          class="branch-dropdown-item"
                          :class="{ active: r.name === currentRemote }"
                        >
                          <div class="branch-item-main" @click="switchRemote(r.name)">
                            <GlobalOutlined style="font-size: 12px; margin-right: 6px;" />
                            <span class="branch-item-name">{{ r.name }}</span>
                            <span v-if="r.name === currentRemote" style="margin-left: auto; color: var(--success, #a6e3a1);">✓</span>
                          </div>
                          <div class="branch-item-actions" @click.stop>
                            <a-tooltip :title="r.url">
                              <span class="branch-action-btn" style="cursor: default; opacity: 0.6; font-size: 10px;">URL</span>
                            </a-tooltip>
                            <a-tooltip title="删除远程仓库" v-if="r.name !== 'origin'">
                              <span class="branch-action-btn danger" @click="removeRemote(r.name)"><DeleteOutlined /></span>
                            </a-tooltip>
                          </div>
                        </div>
                        <div v-if="!remotes.length" style="padding: 12px; text-align: center; color: var(--text-muted);">
                          无远程仓库
                        </div>
                      </div>
                      <!-- 添加远程仓库 -->
                      <div v-if="!showAddRemote" class="remote-add-btn" @click="showAddRemote = true">
                        <PlusOutlined /> 添加远程仓库
                      </div>
                      <div v-else class="remote-add-form">
                        <a-input v-model:value="newRemoteName" placeholder="名称 (如 upstream)" size="small" style="margin-bottom: 4px;" />
                        <a-input v-model:value="newRemoteUrl" placeholder="仓库地址" size="small" style="margin-bottom: 4px;" />
                        <div style="display: flex; gap: 4px;">
                          <a-button size="small" type="primary" :loading="addRemoteLoading" :disabled="!newRemoteName.trim() || !newRemoteUrl.trim()" @click="addRemote" style="flex: 1;">添加</a-button>
                          <a-button size="small" @click="showAddRemote = false; newRemoteName = ''; newRemoteUrl = ''">取消</a-button>
                        </div>
                      </div>
                    </div>
                  </template>
                </a-dropdown>
                <span class="remote-url-text" :title="currentRemoteUrl">{{ currentRemoteUrl }}</span>
              </span>
            </template>
          </div>
          <a-space class="project-actions">
            <a-button size="small" :loading="actionLoading === 'fetch'" :disabled="loadingBase" @click="gitAction('fetch')">
              <template #icon><SyncOutlined /></template>
              Fetch
            </a-button>
            <a-button size="small" :loading="actionLoading === 'pull'" :disabled="loadingBase" @click="gitAction('pull')">
              <template #icon><CloudDownloadOutlined /></template>
              Pull
            </a-button>
            <a-button size="small" :loading="actionLoading === 'push'" :disabled="loadingBase" @click="gitAction('push')">
              <template #icon><CloudUploadOutlined /></template>
              Push
            </a-button>
            <a-button size="small" :disabled="loadingBase" @click="loadStatus">
              <template #icon><ReloadOutlined /></template>
            </a-button>
            <a-button size="small" @click="openSettings">
              <template #icon><SettingOutlined /></template>
            </a-button>
          </a-space>
        </div>

        <div class="project-body">
          <!-- 文件列表 -->
          <div class="file-list" :style="{ width: fileListWidth + 'px' }">
            <!-- Tab 切换：变更 / 历史 -->
            <div class="file-list-tabs">
              <div class="file-list-tab" :class="{ active: activeTab === 'changes' }" @click="activeTab = 'changes'">
                <EditOutlined /> 变更
                <a-badge v-if="changedFiles.length" :count="changedFiles.length" :number-style="{ backgroundColor: 'var(--accent)', fontSize: '10px', height: '16px', lineHeight: '16px', minWidth: '16px', padding: '0 4px' }" />
              </div>
              <div class="file-list-tab" :class="{ active: activeTab === 'history' }" @click="activeTab = 'history'">
                <HistoryOutlined /> 历史
              </div>
              <div class="file-list-tab" :class="{ active: activeTab === 'tags' }" @click="activeTab = 'tags'">
                <TagsOutlined /> 标签
              </div>
            </div>

            <!-- ===== 变更面板 ===== -->
            <template v-if="activeTab === 'changes'">
              <!-- 提交输入区 -->
              <div class="commit-box">
              <div class="commit-input-row">
                <a-input
                  v-model:value="commitMessage"
                  placeholder="提交信息"
                  size="small"
                  :disabled="!stagedFiles.length"
                  @pressEnter="commitChanges"
                />
                <a-button
                  type="primary"
                  size="small"
                  :loading="commitLoading"
                  :disabled="!commitMessage.trim() || !stagedFiles.length"
                  @click="commitChanges"
                  title="提交暂存的更改"
                >
                  Commit
                </a-button>
                <a-dropdown :trigger="['click']">
                  <a-button size="small" :disabled="!unstagedFiles.length && !stagedFiles.length" title="贮藏">
                    <template #icon><InboxOutlined /></template>
                  </a-button>
                  <template #overlay>
                    <div class="stash-dropdown">
                      <div class="stash-save-section">
                        <a-input
                          v-model:value="stashMessage"
                          placeholder="贮藏描述 (可选)"
                          size="small"
                          style="flex:1"
                          @pressEnter="stashSave"
                        />
                        <a-button type="primary" size="small" :loading="stashSaveLoading" @click="stashSave">
                          <template #icon><SaveOutlined /></template>
                          Stash
                        </a-button>
                      </div>
                      <div class="stash-list-title" v-if="stashList.length">
                        <InboxOutlined /> 贮藏列表 ({{ stashList.length }})
                      </div>
                      <div class="stash-list" v-if="stashList.length">
                        <div v-for="s in stashList" :key="s.index" class="stash-item">
                          <div class="stash-item-main">
                            <span class="stash-ref">{{ s.ref }}</span>
                            <span class="stash-msg">{{ s.message }}</span>
                          </div>
                          <div class="stash-item-actions">
                            <a-tooltip title="应用并删除">
                              <span class="stash-action-btn" @click="stashPop(s.index)"><CheckCircleOutlined /></span>
                            </a-tooltip>
                            <a-tooltip title="应用（保留）">
                              <span class="stash-action-btn" @click="stashApply(s.index)"><DownloadOutlined /></span>
                            </a-tooltip>
                            <a-tooltip title="删除">
                              <span class="stash-action-btn danger" @click="stashDrop(s.index)"><DeleteOutlined /></span>
                            </a-tooltip>
                          </div>
                        </div>
                      </div>
                      <div v-else style="padding: 12px; text-align: center; color: var(--text-muted); font-size: 12px;">
                        暂无贮藏
                      </div>
                    </div>
                  </template>
                </a-dropdown>
              </div>
            </div>

            <!-- 合并冲突提示栏 -->
            <div v-if="isMerging" class="conflict-banner">
              <div class="conflict-banner-header">
                <WarningOutlined style="color: #fab387;" />
                <span style="font-weight: 600;">合并冲突</span>
                <span v-if="conflictFiles.length" style="color: var(--text-muted); font-size: 12px;">
                  {{ conflictFiles.length }} 个文件需要解决
                </span>
                <span style="flex:1"></span>
                <a-space :size="4">
                  <a-button size="small" danger @click="handleAbortMerge">
                    <template #icon><StopOutlined /></template>
                    中止
                  </a-button>
                  <a-button
                    size="small"
                    type="primary"
                    :disabled="conflictFiles.length > 0"
                    :loading="commitLoading"
                    @click="completeMerge"
                  >
                    完成合并
                  </a-button>
                </a-space>
              </div>
              <div v-if="conflictFiles.length" class="conflict-file-list">
                <div
                  v-for="cf in conflictFiles"
                  :key="cf.filePath"
                  class="conflict-file-item"
                  :class="{ active: selectedConflictFile?.filePath === cf.filePath }"
                  @click="selectConflictFile(cf)"
                >
                  <WarningOutlined style="color: #fab387; font-size: 12px;" />
                  <span class="conflict-file-name">{{ cf.filePath }}</span>
                  <div class="conflict-file-actions" @click.stop>
                    <a-tooltip title="标记为已解决">
                      <span class="conflict-resolve-btn" @click="resolveConflictFiles([cf])">
                        <CheckCircleOutlined />
                      </span>
                    </a-tooltip>
                  </div>
                </div>
              </div>
              <div v-if="conflictFiles.length > 1" style="padding: 4px 8px; text-align: right; border-top: 1px solid var(--border-color);">
                <a-button size="small" type="link" @click="resolveConflictFiles(conflictFiles)">
                  <CheckCircleOutlined /> 全部标记为已解决
                </a-button>
              </div>
            </div>

            <div class="file-list-content">
              <!-- 骨架屏 -->
              <template v-if="loadingBase || loadingFiles">
                <div v-for="i in 6" :key="i" style="padding: 10px 12px;">
                  <div style="height: 14px; background: var(--bg-hover, #333); border-radius: 4px; animation: pulse 1.5s infinite;" :style="{ width: (50 + i * 8) + '%' }"></div>
                </div>
              </template>
              <!-- 无变更 -->
              <div v-else-if="!changedFiles.length" style="padding: 24px; text-align: center; color: var(--text-muted);">
                <CheckCircleOutlined /> 工作区干净，无变更文件
              </div>
              <template v-else>
                <!-- ===== 已暂存的更改 ===== -->
                <div v-if="stagedFiles.length" class="section-group">
                  <div class="section-header" @click="stagedCollapsed = !stagedCollapsed">
                    <CaretDownOutlined v-if="!stagedCollapsed" class="section-arrow" />
                    <CaretRightOutlined v-else class="section-arrow" />
                    <span class="section-title">已暂存的更改</span>
                    <a-badge :count="stagedFiles.length" :number-style="{ backgroundColor: 'var(--success, #a6e3a1)', color: '#1e1e2e' }" />
                    <span style="flex:1"></span>
                    <span class="section-actions" @click.stop>
                      <span class="section-action-btn" title="取消暂存全部" @click="unstageAll"><MinusOutlined /></span>
                    </span>
                  </div>
                  <div v-show="!stagedCollapsed">
                    <FileTreeNode
                      v-for="node in stagedTree"
                      :key="'staged-' + node.type + '-' + (node.path || node.data?.filePath)"
                      :node="node"
                      :collapsed-dirs="collapsedDirs"
                      :selected-file="selectedFile"
                      :depth="0"
                      mode="staged"
                      @select-file="selectFile"
                      @toggle-dir="toggleDir"
                      @unstage-file="unstageFile"
                      @unstage-dir="unstageDir"
                    />
                  </div>
                </div>

                <!-- ===== 更改 (未暂存) ===== -->
                <div v-if="unstagedFiles.length" class="section-group">
                  <div class="section-header" @click="unstagedCollapsed = !unstagedCollapsed">
                    <CaretDownOutlined v-if="!unstagedCollapsed" class="section-arrow" />
                    <CaretRightOutlined v-else class="section-arrow" />
                    <span class="section-title">更改</span>
                    <a-badge :count="unstagedFiles.length" :number-style="{ backgroundColor: 'var(--accent, #89b4fa)', color: '#1e1e2e' }" />
                    <span style="flex:1"></span>
                    <span class="section-actions" @click.stop>
                      <span class="section-action-btn" title="暂存全部" @click="stageAll"><PlusOutlined /></span>
                      <span class="section-action-btn danger" title="丢弃全部更改" @click="discardAll"><UndoOutlined /></span>
                    </span>
                  </div>
                  <div v-show="!unstagedCollapsed">
                    <FileTreeNode
                      v-for="node in unstagedTree"
                      :key="'unstaged-' + node.type + '-' + (node.path || node.data?.filePath)"
                      :node="node"
                      :collapsed-dirs="collapsedDirs"
                      :selected-file="selectedFile"
                      :depth="0"
                      mode="unstaged"
                      @select-file="selectFile"
                      @toggle-dir="toggleDir"
                      @stage-file="stageFile"
                      @stage-dir="stageDir"
                      @discard-file="discardFile"
                      @discard-dir="discardDir"
                    />
                  </div>
                </div>
              </template>
            </div>
            </template>

            <!-- ===== 提交历史面板 ===== -->
            <template v-else-if="activeTab === 'history'">
              <div class="file-list-content history-panel">
                <!-- 搜索栏 -->
                <div class="search-bar">
                  <a-input
                    v-model:value="searchKeyword"
                    placeholder="搜索提交信息..."
                    size="small"
                    allow-clear
                    @pressEnter="searchCommits"
                  >
                    <template #prefix><SearchOutlined style="color: var(--text-muted);" /></template>
                  </a-input>
                  <a-input
                    v-model:value="searchAuthor"
                    placeholder="作者"
                    size="small"
                    style="width: 100px;"
                    allow-clear
                    @pressEnter="searchCommits"
                  >
                    <template #prefix><UserOutlined style="color: var(--text-muted);" /></template>
                  </a-input>
                  <a-button size="small" type="primary" :loading="searchLoading" @click="searchCommits">
                    <template #icon><SearchOutlined /></template>
                  </a-button>
                  <a-button v-if="isSearchMode" size="small" @click="clearSearch">清除</a-button>
                </div>
                <!-- 提交列表区域 -->
                <div class="commit-list-section" :class="{ 'has-selected': selectedCommit }">
                  <template v-if="commitLogsLoading || searchLoading">
                    <div v-for="i in 8" :key="i" style="padding: 10px 12px;">
                      <div style="height: 14px; background: var(--bg-hover, #333); border-radius: 4px; animation: pulse 1.5s infinite;" :style="{ width: (40 + i * 7) + '%' }"></div>
                      <div style="height: 10px; background: var(--bg-hover, #333); border-radius: 4px; animation: pulse 1.5s infinite; margin-top: 4px; width: 40%;"></div>
                    </div>
                  </template>
                  <div v-else-if="!displayLogs.length" style="padding: 24px; text-align: center; color: var(--text-muted);">
                    {{ isSearchMode ? '未找到匹配的提交' : '暂无提交历史' }}
                  </div>
                  <template v-else>
                    <div
                      v-for="log in displayLogs"
                      :key="log.hash"
                      class="commit-item"
                      :class="{ active: selectedCommit?.hash === log.hash, unpushed: !log.pushed }"
                      @click="selectCommit(log)"
                    >
                      <div class="commit-msg-row">
                        <span v-if="!log.pushed" class="commit-unpushed-dot" title="未推送"></span>
                        <div class="commit-msg">{{ log.message }}</div>
                        <div class="commit-action-btns" @click.stop>
                          <!-- 撤回按钮 -->
                          <a-button
                            type="text"
                            size="small"
                            class="commit-action-btn"
                            title="撤回此提交"
                            @click.stop="revertCommit(log)"
                          >
                            <template #icon><CloseCircleOutlined /></template>
                          </a-button>
                          <!-- 回滚下拉 -->
                          <a-dropdown :trigger="['click']">
                            <a-button
                              type="text"
                              size="small"
                              class="commit-action-btn"
                              :loading="resetLoading"
                              @click.stop
                            >
                              <template #icon><RollbackOutlined /></template>
                            </a-button>
                            <template #overlay>
                              <a-menu @click="({ key }) => resetToCommit(log, key)">
                                <a-menu-item key="hard">
                                  <span style="color: #f38ba8;">🔴 硬回滚</span>
                                  <span style="font-size: 11px; color: var(--text-muted); margin-left: 8px;">丢弃所有更改</span>
                                </a-menu-item>
                                <a-menu-item key="mixed">
                                  <span style="color: #fab387;">🟡 混合回滚</span>
                                  <span style="font-size: 11px; color: var(--text-muted); margin-left: 8px;">保留到工作区</span>
                                </a-menu-item>
                                <a-menu-item key="soft">
                                  <span style="color: #a6e3a1;">🟢 软回滚</span>
                                  <span style="font-size: 11px; color: var(--text-muted); margin-left: 8px;">保留到暂存区</span>
                                </a-menu-item>
                              </a-menu>
                            </template>
                          </a-dropdown>
                        </div>
                      </div>
                      <div class="commit-meta">
                        <span class="commit-hash">{{ log.shortHash }}</span>
                        <span v-if="!log.pushed" class="commit-push-tag">未推送</span>
                        <span class="commit-author"><UserOutlined /> {{ log.author }}</span>
                        <span class="commit-time">{{ formatTime(log.timestamp) }}</span>
                      </div>
                    </div>
                  </template>
                </div>

                <!-- 选中提交的变更文件列表 -->
                <template v-if="selectedCommit">
                  <div class="commit-files-divider">
                    <span class="commit-files-title">
                      <EditOutlined /> {{ selectedCommit.shortHash }} 变更文件
                      <a-badge v-if="commitFiles.length" :count="commitFiles.length" :number-style="{ backgroundColor: 'var(--accent, #89b4fa)', color: '#1e1e2e', marginLeft: '6px' }" />
                    </span>
                  </div>
                  <div class="commit-files-section">
                    <template v-if="commitFilesLoading">
                      <div v-for="i in 4" :key="i" style="padding: 6px 12px;">
                        <div style="height: 14px; background: var(--bg-hover, #333); border-radius: 4px; animation: pulse 1.5s infinite;" :style="{ width: (30 + i * 12) + '%' }"></div>
                      </div>
                    </template>
                    <div v-else-if="!commitFiles.length" style="padding: 12px; text-align: center; color: var(--text-muted); font-size: 12px;">
                      无变更文件
                    </div>
                    <template v-else>
                      <CommitFileTreeNode
                        v-for="cf in commitFileTree"
                        :key="cf.type + '-' + (cf.path || cf.data?.filePath)"
                        :node="cf"
                        :collapsed-dirs="commitCollapsedDirs"
                        :selected-file="selectedCommitFile"
                        @select-file="selectCommitFile($event)"
                        @toggle-dir="toggleCommitDir($event)"
                      />
                    </template>
                  </div>
                </template>
              </div>
            </template>

            <!-- ===== 标签管理面板 ===== -->
            <template v-else-if="activeTab === 'tags'">
              <div class="file-list-content tags-panel">
                <!-- 创建标签区域 -->
                <div class="tag-create-box">
                  <div v-if="!showCreateTag" style="display: flex; justify-content: flex-end; padding: 6px 0;">
                    <a-button size="small" type="primary" @click="showCreateTag = true">
                      <template #icon><PlusOutlined /></template>
                      新建标签
                    </a-button>
                  </div>
                  <template v-else>
                    <a-input v-model:value="newTagName" placeholder="标签名 (例: v1.0.0)" size="small" style="margin-bottom: 6px;" @pressEnter="createTag" />
                    <a-input v-model:value="newTagMessage" placeholder="标签描述 (可选)" size="small" style="margin-bottom: 6px;" />
                    <div style="display: flex; gap: 6px; justify-content: flex-end;">
                      <a-button size="small" @click="showCreateTag = false; newTagName = ''; newTagMessage = ''">取消</a-button>
                      <a-button size="small" type="primary" :loading="createTagLoading" :disabled="!newTagName.trim()" @click="createTag">创建</a-button>
                    </div>
                  </template>
                </div>

                <!-- 标签列表 -->
                <div class="tag-list-section">
                  <template v-if="tagsLoading">
                    <div v-for="i in 5" :key="i" style="padding: 10px 12px;">
                      <div style="height: 14px; background: var(--bg-hover, #333); border-radius: 4px; animation: pulse 1.5s infinite;" :style="{ width: (40 + i * 8) + '%' }"></div>
                    </div>
                  </template>
                  <div v-else-if="!tags.length" style="padding: 24px; text-align: center; color: var(--text-muted);">
                    <TagOutlined :style="{ fontSize: '28px', marginBottom: '8px' }" />
                    <div>暂无标签</div>
                  </div>
                  <template v-else>
                    <div
                      v-for="tag in tags"
                      :key="tag.name"
                      class="tag-item"
                    >
                      <div class="tag-main-row">
                        <TagOutlined class="tag-icon" />
                        <span class="tag-name">{{ tag.name }}</span>
                        <div class="tag-action-btns" @click.stop>
                          <a-tooltip title="推送到远程">
                            <a-button type="text" size="small" class="tag-action-btn push" @click="pushTag(tag)">
                              <template #icon><SendOutlined /></template>
                            </a-button>
                          </a-tooltip>
                          <a-tooltip title="删除标签">
                            <a-button type="text" size="small" class="tag-action-btn delete" @click="deleteTag(tag)">
                              <template #icon><DeleteOutlined /></template>
                            </a-button>
                          </a-tooltip>
                        </div>
                      </div>
                      <div class="tag-meta">
                        <span class="tag-hash">{{ tag.hash }}</span>
                        <span v-if="tag.message" class="tag-message">{{ tag.message }}</span>
                        <span class="tag-time">{{ formatTime(tag.timestamp) }}</span>
                      </div>
                    </div>
                  </template>
                </div>

                <!-- 刷新按钮 -->
                <div style="padding: 8px 10px; border-top: 1px solid var(--border-color); text-align: center;">
                  <a-button size="small" :loading="tagsLoading" @click="loadTags" block>
                    <template #icon><ReloadOutlined /></template>
                    刷新标签
                  </a-button>
                </div>
              </div>
            </template>
          </div>

          <!-- 拖拽分隔条 -->
          <div class="resize-handle-inner" @mousedown="startFileListResize"></div>

          <!-- 文件内容/Diff 展示 -->
          <div class="file-viewer">
            <!-- 变更模式的文件查看器 -->
            <template v-if="activeTab === 'changes'">
              <!-- 冲突文件编辑器 -->
              <template v-if="selectedConflictFile">
                <div class="viewer-header">
                  <WarningOutlined style="color: #fab387; margin-right: 6px;" />
                  <span class="viewer-path">{{ selectedConflictFile.filePath }}</span>
                  <a-tag color="warning" size="small" style="margin-left: 8px;">冲突</a-tag>
                  <span style="flex:1"></span>
                  <a-space :size="6">
                    <a-button size="small" :loading="conflictSaving" @click="saveConflictContent">
                      <template #icon><SaveOutlined /></template>
                      保存
                    </a-button>
                    <a-button size="small" type="primary" @click="resolveConflictFiles([selectedConflictFile])">
                      <template #icon><CheckCircleOutlined /></template>
                      标记已解决
                    </a-button>
                  </a-space>
                </div>
                <div class="viewer-content conflict-editor">
                  <textarea
                    v-model="conflictContent"
                    class="conflict-textarea"
                    spellcheck="false"
                  ></textarea>
                </div>
              </template>
              <!-- 普通文件查看器 -->
              <div v-else-if="!selectedFile" class="empty-state small">
                <span style="color: var(--text-muted)">点击左侧文件查看详情</span>
              </div>
              <template v-else>
                <div class="viewer-header">
                  <span class="viewer-path">{{ selectedFile.filePath }}</span>
                  <a-tag v-if="isNewFile(selectedFile)" color="green" size="small" style="margin-left: 8px;">新增文件</a-tag>
                  <span style="flex:1"></span>
                  <a-segmented v-model:value="viewMode" size="small" :options="[
                    { value: 'diff', label: 'Diff' },
                    { value: 'content', label: '内容' }
                  ]" />
                </div>
                <div class="viewer-content">
                  <pre v-if="viewMode === 'diff'" class="code-block diff-block"><code v-html="formatDiff(fileDiff)"></code></pre>
                  <pre v-else class="code-block"><code>{{ fileContent }}</code></pre>
                </div>
              </template>
            </template>
            <!-- 历史模式的文件 Diff -->
            <template v-else-if="activeTab === 'history'">
              <div v-if="!selectedCommit" class="empty-state small">
                <span style="color: var(--text-muted)">点击左侧提交查看详情</span>
              </div>
              <div v-else-if="!selectedCommitFile" class="empty-state small">
                <div style="text-align: center;">
                  <HistoryOutlined :style="{ fontSize: '32px', color: 'var(--text-muted)', marginBottom: '8px' }" />
                  <div style="color: var(--text-muted); font-size: 13px;">{{ selectedCommit.shortHash }} · {{ selectedCommit.message }}</div>
                  <div style="color: var(--text-muted); font-size: 12px; margin-top: 4px;">{{ selectedCommit.author }} · {{ formatTime(selectedCommit.timestamp) }}</div>
                  <div style="color: var(--text-muted); font-size: 12px; margin-top: 12px;">点击左侧文件查看变更详情</div>
                </div>
              </div>
              <template v-else>
                <div class="viewer-header">
                  <span class="viewer-path">{{ selectedCommitFile.filePath }}</span>
                  <a-tag :color="getCommitStatusColor(selectedCommitFile.status)" size="small" style="margin-left: 8px;">{{ getCommitStatusLabel(selectedCommitFile.status) }}</a-tag>
                  <span style="flex:1"></span>
                  <span style="font-size: 12px; color: var(--text-muted);">{{ selectedCommit.shortHash }} · {{ selectedCommit.author }}</span>
                </div>
                <div class="viewer-content">
                  <pre class="code-block diff-block"><code v-html="formatDiff(commitFileDiff)"></code></pre>
                </div>
              </template>
            </template>
            <!-- 标签模式 -->
            <template v-else-if="activeTab === 'tags'">
              <div class="empty-state small">
                <TagsOutlined :style="{ fontSize: '48px', color: 'var(--text-muted)', marginBottom: '8px' }" />
                <span style="color: var(--text-muted)">在左侧管理项目标签</span>
                <span style="color: var(--text-muted); font-size: 12px; margin-top: 4px;">共 {{ tags.length }} 个标签</span>
              </div>
            </template>
          </div>
        </div>
      </template>
    </div>

    <!-- 设置弹窗 -->
    <a-modal
      v-model:open="showSettings"
      title="设置"
      :width="480"
      @ok="saveSettings"
      ok-text="保存"
      cancel-text="取消"
      :ok-button-props="{ loading: settingsLoading }"
    >
      <a-form layout="vertical" :style="{ marginTop: '16px' }">
        <div style="margin-bottom: 16px; font-weight: 600; color: var(--text-secondary); font-size: 13px;">
          <UserOutlined /> Git 全局配置
        </div>
        <a-form-item label="user.name">
          <a-input v-model:value="gitConfig.userName" placeholder="Your Name" />
        </a-form-item>
        <a-form-item label="user.email">
          <a-input v-model:value="gitConfig.userEmail" placeholder="you@example.com" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.content-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-muted);
  gap: 8px;
}

.empty-state.small {
  font-size: 13px;
}

/* 项目头部 */
.project-view {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.project-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
  gap: 12px;
  --wails-draggable: drag;
}

.project-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
}

.project-name {
  font-size: 15px;
  font-weight: 600;
  white-space: nowrap;
}

.remote-url {
  font-size: 12px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.remote-selector {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  padding: 1px 6px;
  border-radius: 4px;
  color: var(--accent, #89b4fa);
  font-weight: 500;
  transition: background 0.2s;
}
.remote-selector:hover {
  background: var(--hover-bg, rgba(137, 180, 250, 0.1));
}

.remote-url-text {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.remote-dropdown {
  background: var(--dropdown-bg, #1e1e2e);
  border: 1px solid var(--border-color, #313244);
  border-radius: 8px;
  padding: 8px 0;
  min-width: 280px;
  max-width: 400px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
}
.remote-add-btn {
  padding: 8px 12px;
  color: var(--accent, #89b4fa);
  cursor: pointer;
  font-size: 12px;
  border-top: 1px solid var(--border-color, #313244);
  margin-top: 4px;
}
.remote-add-btn:hover {
  background: var(--hover-bg, rgba(137, 180, 250, 0.08));
}
.remote-add-form {
  padding: 8px 12px;
  border-top: 1px solid var(--border-color, #313244);
  margin-top: 4px;
}

.project-actions {
  flex-shrink: 0;
}

/* 主体区域 */
.project-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* 文件列表 */
.file-list {
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  min-width: 180px;
}

.resize-handle-inner {
  width: 4px;
  cursor: col-resize;
  background: transparent;
  flex-shrink: 0;
  position: relative;
  z-index: 10;
  transition: background 0.15s;
}

.resize-handle-inner:hover,
.resize-handle-inner:active {
  background: var(--accent, #89b4fa);
}

/* 提交区域 */
.commit-box {
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-color);
}

.commit-input-row {
  display: flex;
  gap: 6px;
  align-items: center;
}

/* 分组区域 */
.section-group {
  border-bottom: 1px solid var(--border-color, rgba(255,255,255,0.06));
}

.section-group:last-child {
  border-bottom: none;
}

.section-header {
  display: flex;
  align-items: center;
  padding: 5px 10px;
  cursor: pointer;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.3px;
  background: var(--bg-secondary, #181825);
  transition: background 0.12s;
  user-select: none;
}

.section-header:hover {
  background: var(--bg-hover);
}

.section-arrow {
  font-size: 10px;
  color: var(--text-muted);
  flex-shrink: 0;
}

.section-title {
  white-space: nowrap;
}

.section-actions {
  display: flex;
  visibility: hidden;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.section-header:hover .section-actions {
  visibility: visible;
}

.section-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 4px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.12s;
}

.section-action-btn:hover {
  background: var(--bg-active, rgba(255,255,255,0.12));
  color: var(--text-primary);
}

.section-action-btn.danger:hover {
  background: rgba(243, 139, 168, 0.2);
  color: var(--danger, #f38ba8);
}

.file-list-content {
  flex: 1;
  overflow-y: auto;
  padding: 0;
}

/* 文件查看器 */
.file-viewer {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.viewer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.viewer-path {
  font-size: 13px;
  color: var(--text-secondary);
  font-family: 'Consolas', 'Courier New', monospace;
}

.viewer-content {
  flex: 1;
  overflow: auto;
  padding: 0;
}

.code-block {
  margin: 0;
  padding: 12px;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre;
  tab-size: 4;
  color: var(--text-primary);
  user-select: text;
  -webkit-user-select: text;
}

:deep(.diff-add) {
  background: rgba(166, 227, 161, 0.15);
  color: var(--success);
  display: inline-block;
  width: 100%;
}

:deep(.diff-del) {
  background: rgba(243, 139, 168, 0.15);
  color: var(--danger);
  display: inline-block;
  width: 100%;
}

:deep(.diff-hunk) {
  color: var(--accent);
  display: inline-block;
  width: 100%;
}

@keyframes pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 0.8; }
}

/* 分支选择器 */
.branch-tag {
  cursor: pointer;
  user-select: none;
}

.branch-dropdown {
  background: var(--bg-surface, #252536);
  border: 1px solid var(--border-color, #313244);
  border-radius: 6px;
  min-width: 260px;
  max-height: 400px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0,0,0,0.4);
}

.branch-create-box {
  display: flex;
  gap: 6px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-color);
}

.branch-dropdown-title {
  padding: 8px 12px 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.branch-dropdown-list {
  max-height: 200px;
  overflow-y: auto;
}

.branch-dropdown-item {
  display: flex;
  align-items: center;
  padding: 0 4px 0 0;
  font-size: 13px;
  color: var(--text-primary);
  transition: background 0.12s;
}

.branch-dropdown-item:hover {
  background: var(--bg-hover);
}

.branch-dropdown-item.active {
  background: var(--bg-active);
  color: var(--success, #a6e3a1);
}

.branch-item-main {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
  padding: 7px 8px 7px 12px;
  cursor: pointer;
}

.branch-item-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.branch-item-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  visibility: hidden;
}

.branch-dropdown-item:hover .branch-item-actions {
  visibility: visible;
}

.branch-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  font-size: 12px;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.12s;
}

.branch-action-btn:hover {
  background: var(--bg-active, rgba(255,255,255,0.12));
  color: var(--accent, #89b4fa);
}

.branch-action-btn.danger:hover {
  background: rgba(243, 139, 168, 0.2);
  color: var(--danger, #f38ba8);
}

/* Stash 下拉 */
.stash-dropdown {
  background: var(--bg-surface, #252536);
  border: 1px solid var(--border-color, #313244);
  border-radius: 6px;
  min-width: 300px;
  max-height: 360px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0,0,0,0.4);
}

.stash-save-section {
  display: flex;
  gap: 6px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-color);
}

.stash-list-title {
  padding: 8px 12px 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 4px;
}

.stash-list {
  max-height: 240px;
  overflow-y: auto;
}

.stash-item {
  display: flex;
  align-items: center;
  padding: 6px 12px;
  font-size: 12px;
  border-bottom: 1px solid var(--border-color, rgba(255,255,255,0.04));
  transition: background 0.12s;
}

.stash-item:hover {
  background: var(--bg-hover);
}

.stash-item-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stash-ref {
  font-family: 'Consolas', 'Courier New', monospace;
  color: var(--accent, #89b4fa);
  font-size: 11px;
}

.stash-msg {
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.stash-item-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  visibility: hidden;
}

.stash-item:hover .stash-item-actions {
  visibility: visible;
}

.stash-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  font-size: 12px;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.12s;
}

.stash-action-btn:hover {
  background: var(--bg-active, rgba(255,255,255,0.12));
  color: var(--accent, #89b4fa);
}

.stash-action-btn.danger:hover {
  background: rgba(243, 139, 168, 0.2);
  color: var(--danger, #f38ba8);
}

/* Tab 切换 */
.file-list-tabs {
  display: flex;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary, #181825);
  flex-shrink: 0;
}

.file-list-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 7px 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.15s;
  user-select: none;
}

.file-list-tab:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
}

.file-list-tab.active {
  color: var(--accent, #89b4fa);
  border-bottom-color: var(--accent, #89b4fa);
}

/* 提交历史项 */
.commit-item {
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-color, rgba(255,255,255,0.04));
  transition: background 0.12s;
}

.commit-item:hover {
  background: var(--bg-hover);
}

.commit-item.active {
  background: var(--bg-active);
}

.commit-msg-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.commit-msg-row .commit-msg {
  flex: 1;
  min-width: 0;
}

.commit-action-btns {
  display: flex;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
  visibility: hidden;
}

.commit-action-btn {
  color: var(--text-muted) !important;
  width: 24px;
  height: 24px;
}

.commit-action-btn:hover {
  color: #f38ba8 !important;
}

.commit-item:hover .commit-action-btns {
  visibility: visible;
}

/* 未推送标识 */
.commit-unpushed-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #a6e3a1;
  flex-shrink: 0;
  margin-right: 2px;
}

.commit-push-tag {
  font-size: 10px;
  padding: 0 5px;
  border-radius: 3px;
  background: rgba(166, 227, 161, 0.15);
  color: #a6e3a1;
  line-height: 16px;
  flex-shrink: 0;
}

.commit-item.unpushed {
  border-left: 2px solid #a6e3a1;
}

.commit-msg {
  font-size: 13px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
}

.commit-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 2px;
}

.commit-hash {
  font-family: 'Consolas', 'Courier New', monospace;
  color: var(--accent, #89b4fa);
}

.commit-author {
  display: flex;
  align-items: center;
  gap: 3px;
}

.commit-time {
  margin-left: auto;
}

/* 提交历史面板布局 */
.history-panel {
  display: flex;
  flex-direction: column;
}

.commit-list-section {
  overflow-y: auto;
  flex: 1;
  min-height: 80px;
}

.commit-list-section.has-selected {
  flex: none;
  max-height: 45%;
  border-bottom: none;
}

.commit-files-divider {
  padding: 6px 12px;
  background: var(--bg-secondary, #1e1e2e);
  border-top: 1px solid var(--border-color, rgba(255,255,255,0.06));
  border-bottom: 1px solid var(--border-color, rgba(255,255,255,0.06));
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.commit-files-title {
  display: flex;
  align-items: center;
  gap: 6px;
}

.commit-files-section {
  overflow-y: auto;
  flex: 1;
  min-height: 60px;
}

.commit-file-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary);
  transition: background 0.12s;
}

.commit-file-item:hover {
  background: var(--bg-hover);
}

.commit-file-item.active {
  background: var(--bg-active);
}

.commit-file-item.dir {
  color: var(--text-secondary);
  font-weight: 500;
}

.commit-file-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
  flex: 1;
}

.commit-file-count {
  font-size: 11px;
  color: var(--text-muted);
  margin-left: 4px;
  flex-shrink: 0;
}

/* 标签管理面板 */
.tags-panel {
  display: flex;
  flex-direction: column;
}

.tag-create-box {
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-color);
}

.tag-list-section {
  flex: 1;
  overflow-y: auto;
}

.tag-item {
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-color, rgba(255,255,255,0.04));
  transition: background 0.12s;
}

.tag-item:hover {
  background: var(--bg-hover);
}

.tag-main-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.tag-icon {
  color: var(--accent, #89b4fa);
  font-size: 14px;
  flex-shrink: 0;
}

.tag-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tag-action-btns {
  display: flex;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
  visibility: hidden;
}

.tag-item:hover .tag-action-btns {
  visibility: visible;
}

.tag-action-btn {
  width: 24px;
  height: 24px;
  color: var(--text-muted) !important;
}

.tag-action-btn.push:hover {
  color: var(--accent, #89b4fa) !important;
}

.tag-action-btn.delete:hover {
  color: var(--danger, #f38ba8) !important;
}

.tag-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 2px;
  padding-left: 20px;
}

.tag-hash {
  font-family: 'Consolas', 'Courier New', monospace;
  color: var(--accent, #89b4fa);
}

.tag-message {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tag-time {
  margin-left: auto;
  flex-shrink: 0;
}

/* 冲突处理 */
.conflict-banner {
  border-bottom: 1px solid var(--border-color);
  background: rgba(250, 179, 135, 0.08);
}

.conflict-banner-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  font-size: 13px;
}

.conflict-file-list {
  max-height: 150px;
  overflow-y: auto;
  border-top: 1px solid var(--border-color);
}

.conflict-file-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.12s;
}

.conflict-file-item:hover {
  background: var(--bg-hover);
}

.conflict-file-item.active {
  background: var(--bg-active);
}

.conflict-file-name {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: 'Consolas', 'Courier New', monospace;
  color: var(--text-secondary);
}

.conflict-file-actions {
  visibility: hidden;
  flex-shrink: 0;
}

.conflict-file-item:hover .conflict-file-actions {
  visibility: visible;
}

.conflict-resolve-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  font-size: 13px;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.12s;
}

.conflict-resolve-btn:hover {
  background: rgba(166, 227, 161, 0.2);
  color: var(--success, #a6e3a1);
}

.conflict-editor {
  display: flex;
  flex-direction: column;
}

.conflict-textarea {
  flex: 1;
  width: 100%;
  background: var(--bg-primary, #1e1e2e);
  color: var(--text-primary);
  border: none;
  outline: none;
  resize: none;
  padding: 12px;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  tab-size: 4;
}

/* 提交搜索 */
.search-bar {
  display: flex;
  gap: 6px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
</style>
