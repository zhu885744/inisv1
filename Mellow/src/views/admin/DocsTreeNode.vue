<template>
  <div class="tree-node">
    <div
      class="tree-row"
      :class="{ 'is-active': activeId === node.id }"
      @click="$emit('select', node)"
    >
      <i class="bi bi-folder2 tree-icon" aria-hidden="true" />
      <span class="tree-name">{{ node.name }}</span>
      <span v-if="(node.docs || []).length" class="tree-count">{{ (node.docs || []).length }}</span>

      <span class="tree-actions" @click.stop>
        <button type="button" title="新建子目录" @click="$emit('add-child', node)">
          <i class="bi bi-folder-plus" />
        </button>
        <button type="button" title="编辑目录" @click="$emit('edit', node)">
          <i class="bi bi-pencil" />
        </button>
        <button type="button" title="删除目录" @click="$emit('remove', node)">
          <i class="bi bi-trash" />
        </button>
      </span>
    </div>

    <div v-if="(node.children || []).length" class="tree-children">
      <DocsTreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :active-id="activeId"
        @select="$emit('select', $event)"
        @edit="$emit('edit', $event)"
        @remove="$emit('remove', $event)"
        @add-child="$emit('add-child', $event)"
      />
    </div>
  </div>
</template>

<script setup>
defineProps({
  node: { type: Object, required: true },
  activeId: { type: Number, default: 0 }
})

defineEmits(['select', 'edit', 'remove', 'add-child'])
</script>

<style scoped>
.tree-node {
  display: flex;
  flex-direction: column;
}
.tree-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  font-size: 13.5px;
  color: var(--text-soft);
  cursor: pointer;
  transition: background-color 0.15s, color 0.15s;
}
.tree-row:hover {
  background: var(--bg-muted);
  color: var(--text);
}
.tree-row.is-active {
  background: var(--accent-soft);
  color: var(--primary);
  font-weight: 600;
}
.tree-icon {
  flex-shrink: 0;
  font-size: 14px;
  color: var(--text-light);
}
.tree-row.is-active .tree-icon {
  color: var(--primary);
}
.tree-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tree-count {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
}
.tree-actions {
  display: none;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}
.tree-row:hover .tree-actions,
.tree-row.is-active .tree-actions {
  display: inline-flex;
}
.tree-actions button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  font-size: 11px;
  color: var(--text-light);
  background: transparent;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
.tree-actions button:hover {
  color: var(--primary);
  background: var(--bg-card);
}
.tree-children {
  margin-left: 12px;
  padding-left: 8px;
  border-left: 1px dashed var(--border-soft);
}
</style>
