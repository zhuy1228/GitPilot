<script setup>
import {
  FileOutlined,
  FolderOutlined,
  FolderOpenOutlined,
  CaretRightOutlined,
  CaretDownOutlined,
} from '@ant-design/icons-vue'

const props = defineProps({
  node: { type: Object, required: true },
  collapsedDirs: { type: Set, default: () => new Set() },
  selectedFile: { type: Object, default: null },
  depth: { type: Number, default: 0 },
})

const emit = defineEmits(['select-file', 'toggle-dir'])

function getStatusColor(status) {
  const map = { M: 'orange', A: 'green', D: 'red', R: 'blue', C: 'cyan' }
  return map[status] || 'default'
}
function getStatusLabel(status) {
  const map = { M: '修改', A: '新增', D: '删除', R: '重命名', C: '复制' }
  return map[status] || status
}
</script>

<template>
  <!-- 目录节点 -->
  <div v-if="node.type === 'dir'" class="commit-tree-dir">
    <div class="commit-tree-dir-header" :style="{ paddingLeft: (8 + depth * 16) + 'px' }" @click="emit('toggle-dir', node.path)">
      <CaretDownOutlined v-if="!collapsedDirs.has(node.path)" class="commit-tree-arrow" />
      <CaretRightOutlined v-else class="commit-tree-arrow" />
      <FolderOpenOutlined v-if="!collapsedDirs.has(node.path)" class="commit-tree-icon dir-icon" />
      <FolderOutlined v-else class="commit-tree-icon dir-icon" />
      <span class="commit-tree-dir-name">{{ node.name }}</span>
      <span class="commit-tree-dir-count">{{ node.fileCount }}</span>
    </div>
    <div v-show="!collapsedDirs.has(node.path)">
      <CommitFileTreeNode
        v-for="child in node.children"
        :key="child.type + '-' + (child.path || child.data?.filePath)"
        :node="child"
        :collapsed-dirs="collapsedDirs"
        :selected-file="selectedFile"
        :depth="depth + 1"
        @select-file="emit('select-file', $event)"
        @toggle-dir="emit('toggle-dir', $event)"
      />
    </div>
  </div>

  <!-- 文件节点 -->
  <div
    v-else
    class="commit-tree-file-item"
    :style="{ paddingLeft: (8 + depth * 16) + 'px' }"
    :class="{ active: selectedFile?.filePath === node.data.filePath }"
    @click="emit('select-file', node.data)"
  >
    <FileOutlined class="commit-tree-icon file-icon" />
    <span class="commit-tree-file-name">{{ node.name }}</span>
    <a-tag :color="getStatusColor(node.data.status)" size="small" class="commit-tree-status-tag">{{ getStatusLabel(node.data.status) }}</a-tag>
  </div>
</template>

<style scoped>
.commit-tree-dir-header {
  display: flex;
  align-items: center;
  padding: 3px 8px;
  cursor: pointer;
  gap: 4px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  transition: background 0.12s;
}

.commit-tree-dir-header:hover {
  background: var(--bg-hover);
}

.commit-tree-arrow {
  font-size: 10px;
  color: var(--text-muted);
  flex-shrink: 0;
  width: 14px;
  text-align: center;
}

.commit-tree-icon {
  flex-shrink: 0;
  font-size: 14px;
}

.dir-icon {
  color: var(--accent, #89b4fa);
}

.file-icon {
  color: var(--text-muted);
}

.commit-tree-dir-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.commit-tree-dir-count {
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-hover, rgba(255,255,255,0.06));
  border-radius: 8px;
  padding: 0 6px;
  min-width: 18px;
  text-align: center;
  flex-shrink: 0;
}

.commit-tree-file-item {
  display: flex;
  align-items: center;
  padding: 3px 8px;
  cursor: pointer;
  gap: 6px;
  font-size: 13px;
  transition: background 0.12s;
}

.commit-tree-file-item:hover {
  background: var(--bg-hover);
}

.commit-tree-file-item.active {
  background: var(--bg-active);
}

.commit-tree-file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.commit-tree-status-tag {
  flex-shrink: 0;
  font-size: 11px;
  margin-left: auto;
}
</style>
