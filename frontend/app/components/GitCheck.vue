<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import {
  WarningOutlined,
  DownloadOutlined,
  FolderOpenOutlined,
  CheckCircleOutlined,
  LoadingOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons-vue'
import { AppService } from '../../bindings/github.com/zhuy1228/GitPilot/internal/app'
import { Events } from '@wailsio/runtime'

const emit = defineEmits(['ready'])

const checking = ref(true)
const gitInstalled = ref(false)
const gitVersion = ref('')
const gitPath = ref('')

// 安装相关
const installDir = ref('C:\\Program Files\\Git')
const installing = ref(false)
const installPhase = ref('')    // downloading | installing | done | error
const installPercent = ref(0)
const installMessage = ref('')

let unsubscribe = null

onMounted(async () => {
  // 监听安装进度事件
  unsubscribe = Events.On('git-install-progress', (event) => {
    const data = event.data?.[0] || event.data
    if (data) {
      installPhase.value = data.phase
      installPercent.value = data.percent
      installMessage.value = data.message
      if (data.phase === 'done') {
        installing.value = false
        gitInstalled.value = true
        setTimeout(() => emit('ready'), 1500)
      } else if (data.phase === 'error') {
        installing.value = false
      }
    }
  })

  // 检查 Git
  try {
    const status = await AppService.CheckGitInstalled()
    gitInstalled.value = status.installed
    gitVersion.value = status.version || ''
    gitPath.value = status.path || ''
    if (status.installed) {
      emit('ready')
    }
  } catch (e) {
    console.error('检查 Git 失败:', e)
    gitInstalled.value = false
  } finally {
    checking.value = false
  }
})

onBeforeUnmount(() => {
  if (unsubscribe) unsubscribe()
})

async function selectInstallDir() {
  try {
    const dir = await AppService.SelectGitInstallDir()
    if (dir) installDir.value = dir
  } catch (e) {
    console.error('选择目录失败:', e)
  }
}

async function startInstall() {
  installing.value = true
  installPhase.value = 'downloading'
  installPercent.value = 0
  installMessage.value = '准备下载...'
  try {
    await AppService.InstallGit(installDir.value)
  } catch (e) {
    console.error('安装 Git 失败:', e)
    installPhase.value = 'error'
    installMessage.value = '安装失败: ' + String(e)
    installing.value = false
  }
}
</script>

<template>
  <!-- 检查中 -->
  <div v-if="checking" class="git-check-overlay">
    <div class="git-check-card">
      <LoadingOutlined :style="{ fontSize: '32px', color: 'var(--accent, #89b4fa)' }" spin />
      <div class="git-check-title">正在检查 Git 环境...</div>
    </div>
  </div>

  <!-- Git 未安装 -->
  <div v-else-if="!gitInstalled" class="git-check-overlay">
    <div class="git-check-card install-card">
      <div class="git-check-icon">
        <WarningOutlined :style="{ fontSize: '48px', color: '#fab387' }" />
      </div>
      <div class="git-check-title">未检测到 Git</div>
      <div class="git-check-desc">GitPilot 需要 Git 才能正常工作，请安装 Git 后使用。</div>

      <!-- 安装进度 -->
      <template v-if="installing || installPhase === 'done'">
        <div class="install-progress">
          <div v-if="installPhase === 'downloading'" class="progress-section">
            <div class="progress-bar-bg">
              <div class="progress-bar-fill" :style="{ width: installPercent + '%' }"></div>
            </div>
            <div class="progress-text">
              <DownloadOutlined /> {{ installMessage }}
            </div>
          </div>
          <div v-else-if="installPhase === 'installing'" class="progress-section">
            <div class="progress-bar-bg">
              <div class="progress-bar-fill installing-anim" style="width: 100%"></div>
            </div>
            <div class="progress-text">
              <LoadingOutlined spin /> {{ installMessage }}
            </div>
          </div>
          <div v-else-if="installPhase === 'done'" class="progress-section done">
            <CheckCircleOutlined :style="{ fontSize: '24px', color: '#a6e3a1' }" />
            <div class="progress-text success">{{ installMessage }}</div>
          </div>
          <div v-else-if="installPhase === 'error'" class="progress-section error">
            <CloseCircleOutlined :style="{ fontSize: '24px', color: '#f38ba8' }" />
            <div class="progress-text error-text">{{ installMessage }}</div>
          </div>
        </div>
      </template>

      <!-- 安装表单 -->
      <template v-else>
        <div class="install-form">
          <div class="install-field">
            <label>安装路径</label>
            <div class="install-dir-input">
              <input
                v-model="installDir"
                class="dir-input"
                placeholder="C:\Program Files\Git"
              />
              <button class="dir-btn" @click="selectInstallDir" title="选择目录">
                <FolderOpenOutlined />
              </button>
            </div>
          </div>
        </div>
        <div class="install-actions">
          <button class="install-btn primary" @click="startInstall">
            <DownloadOutlined /> 自动下载并安装 Git
          </button>
        </div>
        <div class="install-hint">
          将从 GitHub 下载 Git for Windows 安装包（约 65MB），需保持网络连接。
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.git-check-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-primary, #1e1e2e);
}

.git-check-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 40px;
  max-width: 480px;
  width: 100%;
}

.install-card {
  background: var(--bg-secondary, #181825);
  border: 1px solid var(--border-color, #313244);
  border-radius: 12px;
  padding: 40px 36px;
}

.git-check-icon {
  margin-bottom: 4px;
}

.git-check-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary, #cdd6f4);
}

.git-check-desc {
  font-size: 14px;
  color: var(--text-muted, #6c7086);
  text-align: center;
  line-height: 1.6;
}

.install-form {
  width: 100%;
  margin-top: 8px;
}

.install-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.install-field label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary, #a6adc8);
}

.install-dir-input {
  display: flex;
  gap: 6px;
}

.dir-input {
  flex: 1;
  padding: 8px 12px;
  background: var(--bg-primary, #1e1e2e);
  border: 1px solid var(--border-color, #313244);
  border-radius: 6px;
  color: var(--text-primary, #cdd6f4);
  font-size: 13px;
  font-family: 'Consolas', 'Courier New', monospace;
  outline: none;
  transition: border-color 0.15s;
}

.dir-input:focus {
  border-color: var(--accent, #89b4fa);
}

.dir-btn {
  padding: 8px 12px;
  background: var(--bg-primary, #1e1e2e);
  border: 1px solid var(--border-color, #313244);
  border-radius: 6px;
  color: var(--accent, #89b4fa);
  cursor: pointer;
  font-size: 15px;
  display: flex;
  align-items: center;
  transition: all 0.15s;
}

.dir-btn:hover {
  background: var(--bg-hover, #313244);
}

.install-actions {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 4px;
}

.install-btn {
  padding: 10px 20px;
  border-radius: 8px;
  border: 1px solid var(--border-color, #313244);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.15s;
}

.install-btn.primary {
  background: var(--accent, #89b4fa);
  color: #1e1e2e;
  border-color: var(--accent, #89b4fa);
}

.install-btn.primary:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

.install-hint {
  font-size: 12px;
  color: var(--text-muted, #6c7086);
  text-align: center;
  line-height: 1.5;
}

/* 安装进度 */
.install-progress {
  width: 100%;
  margin-top: 8px;
}

.progress-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: center;
}

.progress-section.done,
.progress-section.error {
  gap: 12px;
}

.progress-bar-bg {
  width: 100%;
  height: 6px;
  background: var(--bg-primary, #1e1e2e);
  border-radius: 3px;
  overflow: hidden;
}

.progress-bar-fill {
  height: 100%;
  background: var(--accent, #89b4fa);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.progress-bar-fill.installing-anim {
  background: linear-gradient(90deg, var(--accent, #89b4fa) 0%, #b4befe 50%, var(--accent, #89b4fa) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.progress-text {
  font-size: 13px;
  color: var(--text-secondary, #a6adc8);
  display: flex;
  align-items: center;
  gap: 6px;
}

.progress-text.success {
  color: #a6e3a1;
  font-weight: 600;
}

.progress-text.error-text {
  color: #f38ba8;
  font-size: 12px;
  text-align: center;
  word-break: break-all;
}
</style>
