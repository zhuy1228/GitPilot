<script setup>
import {
  FileOutlined,
  FolderOutlined,
  FolderOpenOutlined,
  CaretRightOutlined,
  CaretDownOutlined,
  PlusOutlined,
  MinusOutlined,
  UndoOutlined,
} from '@ant-design/icons-vue'

const props = defineProps({
  node: { type: Object, required: true },
  collapsedDirs: { type: Set, default: () => new Set() },
  selectedFile: { type: Object, default: null },
  depth: { type: Number, default: 0 },
  // 'unstaged' | 'staged'
  mode: { type: String, default: 'unstaged' },
})

const emit = defineEmits([
  'select-file', 'toggle-dir',
  'stage-file', 'unstage-file', 'discard-file',
  'stage-dir', 'unstage-dir', 'discard-dir',
])

function getStatusColor(status) {
  const colors = { 'M': 'orange', 'A': 'green', 'D': 'red', '?': 'default' }
  return colors[status] || 'default'
}
</script>

<template>
  <!-- 目录节点 -->
  <div v-if="node.type === 'dir'" class="tree-dir">
    <div class="tree-dir-header" :style="{ paddingLeft: (8 + depth * 16) + 'px' }" @click="emit('toggle-dir', node.path)">
      <CaretDownOutlined v-if="!collapsedDirs.has(node.path)" class="tree-arrow" />
      <CaretRightOutlined v-else class="tree-arrow" />
      <FolderOpenOutlined v-if="!collapsedDirs.has(node.path)" class="tree-icon dir-icon" />
      <FolderOutlined v-else class="tree-icon dir-icon" />
      <span class="tree-dir-name">{{ node.name }}</span>
      <span class="tree-dir-count">{{ node.fileCount }}</span>
      <!-- 目录级操作按钮 -->
      <span class="tree-actions" @click.stop>
        <template v-if="mode === 'unstaged'">
          <span class="tree-action-btn" title="暂存目录下所有文件" @click="emit('stage-dir', node.path)"><PlusOutlined /></span>
          <span class="tree-action-btn danger" title="丢弃目录下所有更改" @click="emit('discard-dir', node.path)"><UndoOutlined /></span>
        </template>
        <template v-else>
          <span class="tree-action-btn" title="取消暂存目录下所有文件" @click="emit('unstage-dir', node.path)"><MinusOutlined /></span>
        </template>
      </span>
    </div>
    <div v-show="!collapsedDirs.has(node.path)">
      <FileTreeNode
        v-for="child in node.children"
        :key="child.type + '-' + (child.path || child.data?.filePath)"
        :node="child"
        :collapsed-dirs="collapsedDirs"
        :selected-file="selectedFile"
        :depth="depth + 1"
        :mode="mode"
        @select-file="emit('select-file', $event)"
        @toggle-dir="emit('toggle-dir', $event)"
        @stage-file="emit('stage-file', $event)"
        @unstage-file="emit('unstage-file', $event)"
        @discard-file="emit('discard-file', $event)"
        @stage-dir="emit('stage-dir', $event)"
        @unstage-dir="emit('unstage-dir', $event)"
        @discard-dir="emit('discard-dir', $event)"
      />
    </div>
  </div>

  <!-- 文件节点 -->
  <div v-else class="tree-file-item"
    :style="{ paddingLeft: (8 + depth * 16) + 'px' }"
    :class="{ active: selectedFile?.filePath === node.data.filePath && selectedFile?.staged === node.data.staged }"
    @click="emit('select-file', node.data)">
    <FileOutlined class="tree-icon file-icon" />
    <span class="tree-file-name">{{ node.name }}</span>
    <a-tag :color="getStatusColor(node.data.status)" size="small" class="tree-status-tag">{{ node.data.statusText }}</a-tag>
    <!-- 文件级操作按钮 -->
    <span class="tree-actions" @click.stop>
      <template v-if="mode === 'unstaged'">
        <span class="tree-action-btn" title="暂存此文件" @click="emit('stage-file', node.data)"><PlusOutlined /></span>
        <span class="tree-action-btn danger" title="丢弃更改" @click="emit('discard-file', node.data)"><UndoOutlined /></span>
      </template>
      <template v-else>
        <span class="tree-action-btn" title="取消暂存" @click="emit('unstage-file', node.data)"><MinusOutlined /></span>
      </template>
    </span>
  </div>
</template>

<style scoped>
.tree-dir-header {
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

.tree-dir-header:hover {
  background: var(--bg-hover);
}

.tree-arrow {
  font-size: 10px;
  color: var(--text-muted);
  flex-shrink: 0;
  width: 14px;
  text-align: center;
}

.tree-icon {
  flex-shrink: 0;
  font-size: 14px;
}

.dir-icon {
  color: var(--accent, #89b4fa);
}

.file-icon {
  color: var(--text-muted);
}

.tree-dir-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-dir-count {
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-hover, rgba(255,255,255,0.06));
  border-radius: 8px;
  padding: 0 6px;
  min-width: 18px;
  text-align: center;
  flex-shrink: 0;
}

.tree-file-item {
  display: flex;
  align-items: center;
  padding: 3px 8px;
  cursor: pointer;
  gap: 6px;
  font-size: 13px;
  transition: background 0.12s;
}

.tree-file-item:hover {
  background: var(--bg-hover);
}

.tree-file-item.active {
  background: var(--bg-active);
}

.tree-file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-status-tag {
  flex-shrink: 0;
  font-size: 11px;
}

/* 操作按钮 */
.tree-actions {
  display: flex;
  visibility: hidden;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  margin-left: 4px;
}

.tree-dir-header:hover .tree-actions,
.tree-file-item:hover .tree-actions {
  visibility: visible;
}

.tree-action-btn {
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

.tree-action-btn:hover {
  background: var(--bg-active, rgba(255,255,255,0.12));
  color: var(--text-primary);
}

.tree-action-btn.danger:hover {
  background: rgba(243, 139, 168, 0.2);
  color: var(--danger, #f38ba8);
}
</style>
