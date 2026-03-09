<script setup>
import { ref, onBeforeUnmount } from 'vue'

const selectedProject = ref(null)

function onSelectProject(project) {
  selectedProject.value = project
}

// ---- 侧栏拖拽调整宽度 ----
const sidebarWidth = ref(280)
const MIN_SIDEBAR = 180
const MAX_SIDEBAR = 500
let draggingSidebar = false

function startSidebarResize(e) {
  e.preventDefault()
  draggingSidebar = true
  const startX = e.clientX
  const startW = sidebarWidth.value

  function onMove(ev) {
    if (!draggingSidebar) return
    const delta = ev.clientX - startX
    sidebarWidth.value = Math.min(MAX_SIDEBAR, Math.max(MIN_SIDEBAR, startW + delta))
  }
  function onUp() {
    draggingSidebar = false
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
  draggingSidebar = false
})
</script>

<template>
  <div class="main-layout">
    <Sidebar :selected-project="selectedProject" :style="{ width: sidebarWidth + 'px' }" @select-project="onSelectProject" />
    <div class="resize-handle" @mousedown="startSidebarResize"></div>
    <ContentArea :project="selectedProject" />
  </div>
</template>

<style scoped>
.main-layout {
  display: flex;
  height: 100vh;
  width: 100%;
}

.resize-handle {
  width: 4px;
  cursor: col-resize;
  background: transparent;
  flex-shrink: 0;
  position: relative;
  z-index: 10;
  transition: background 0.15s;
}

.resize-handle:hover,
.resize-handle:active {
  background: var(--accent, #89b4fa);
}
</style>
