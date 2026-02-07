<script setup lang="ts">
import { ref, nextTick, computed } from 'vue'
import { useNotesStore } from '../stores/notes'
import type { Folder } from '../types'

type FolderNode = Folder & { children: FolderNode[] }

const props = defineProps<{
  folder: FolderNode
  depth: number
  expandedFolders: Set<number>
  editingFolderId: number | null
  editingName: string
  creatingFolder: boolean
  newFolderParentId: number | null
  newFolderName: string
}>()

const emit = defineEmits<{
  'toggle-expand': [id: number]
  select: [id: number]
  'context-menu': [e: MouseEvent, folder: FolderNode]
  'confirm-rename': []
  'cancel-rename': []
  'update:editing-name': [value: string]
  'update:new-folder-name': [value: string]
  'confirm-create': []
  'cancel-create': []
  drop: [e: DragEvent, folderId: number]
}>()

const notesStore = useNotesStore()
const renameInput = ref<HTMLInputElement | null>(null)
const createInput = ref<HTMLInputElement | null>(null)

const isEditing = computed(() => props.editingFolderId === props.folder.id)
const isExpanded = computed(() => props.expandedFolders.has(props.folder.id))
const hasChildren = computed(() => props.folder.children.length > 0)
const isSelected = computed(() => notesStore.currentFolder === props.folder.id)
const showCreateInput = computed(() => props.creatingFolder && props.newFolderParentId === props.folder.id)
const noteCount = computed(() => notesStore.notes.filter((n) => n.folder_id === props.folder.id).length)

const paddingLeft = computed(() => `${(props.depth + 1) * 12 + 12}px`)
const childPaddingLeft = computed(() => `${(props.depth + 2) * 12 + 12}px`)

function onClickFolder() {
  emit('select', props.folder.id)
  if (hasChildren.value) emit('toggle-expand', props.folder.id)
}

function onClickChevron(e: MouseEvent) {
  e.stopPropagation()
  emit('toggle-expand', props.folder.id)
}

function onContextMenu(e: MouseEvent) {
  emit('context-menu', e, props.folder)
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
}

function onDrop(e: DragEvent) {
  emit('drop', e, props.folder.id)
}

function onRenameInput(e: Event) {
  emit('update:editing-name', (e.target as HTMLInputElement).value)
}

function onRenameKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') emit('confirm-rename')
  if (e.key === 'Escape') emit('cancel-rename')
}

function onCreateInput(e: Event) {
  emit('update:new-folder-name', (e.target as HTMLInputElement).value)
}

function onCreateKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') emit('confirm-create')
  if (e.key === 'Escape') emit('cancel-create')
}

function focusRenameInput() {
  nextTick(() => {
    renameInput.value?.focus()
  })
}

function focusCreateInput() {
  nextTick(() => {
    createInput.value?.focus()
  })
}

// Watch for editing state changes to auto-focus
import { watch } from 'vue'
watch(isEditing, (val) => { if (val) focusRenameInput() })
watch(showCreateInput, (val) => { if (val) focusCreateInput() })
</script>

<template>
  <div>
    <!-- Folder row: inline rename mode -->
    <div v-if="isEditing" class="py-1" :style="{ paddingLeft }">
      <input
        :value="editingName"
        @input="onRenameInput"
        @keydown="onRenameKeydown"
        @blur="emit('confirm-rename')"
        ref="renameInput"
        type="text"
        class="w-[calc(100%-24px)] bg-nord0 text-nord4 border border-nord8 rounded px-2 py-0.5 text-sm focus:outline-none"
      />
    </div>

    <!-- Folder row: normal mode -->
    <div
      v-else
      @click="onClickFolder"
      @contextmenu.prevent="onContextMenu"
      @dragover="onDragOver"
      @drop="onDrop"
      class="flex items-center gap-1 py-1 cursor-pointer text-sm transition-colors"
      :class="isSelected ? 'bg-nord2 text-nord6' : 'text-nord4 hover:bg-nord0'"
      :style="{ paddingLeft }"
    >
      <!-- Chevron -->
      <span
        @click="onClickChevron"
        class="w-3 h-3 shrink-0 inline-flex items-center justify-center text-[8px] transition-transform"
        :class="hasChildren ? 'text-nord3' : 'text-transparent'"
        :style="isExpanded ? 'transform: rotate(90deg)' : ''"
      >&#x25B6;</span>

      <!-- Folder icon -->
      <svg class="w-3.5 h-3.5 shrink-0" viewBox="0 0 16 16" fill="currentColor">
        <path d="M1 3.5A1.5 1.5 0 012.5 2h3.879a1.5 1.5 0 011.06.44l1.122 1.12A1.5 1.5 0 009.62 4H13.5A1.5 1.5 0 0115 5.5v7a1.5 1.5 0 01-1.5 1.5h-11A1.5 1.5 0 011 12.5v-9z"/>
      </svg>

      <!-- Name -->
      <span class="truncate">{{ folder.name }}</span>

      <!-- Note count -->
      <span class="ml-auto text-[10px] text-nord3 pr-3">{{ noteCount }}</span>
    </div>

    <!-- Children (when expanded) -->
    <template v-if="isExpanded && hasChildren">
      <FolderTreeNode
        v-for="child in folder.children"
        :key="child.id"
        :folder="child"
        :depth="depth + 1"
        :expanded-folders="expandedFolders"
        :editing-folder-id="editingFolderId"
        :editing-name="editingName"
        :creating-folder="creatingFolder"
        :new-folder-parent-id="newFolderParentId"
        :new-folder-name="newFolderName"
        @toggle-expand="emit('toggle-expand', $event)"
        @select="emit('select', $event)"
        @context-menu="(e, f) => emit('context-menu', e, f)"
        @confirm-rename="emit('confirm-rename')"
        @cancel-rename="emit('cancel-rename')"
        @update:editing-name="emit('update:editing-name', $event)"
        @update:new-folder-name="emit('update:new-folder-name', $event)"
        @confirm-create="emit('confirm-create')"
        @cancel-create="emit('cancel-create')"
        @drop="(e, id) => emit('drop', e, id)"
      />
    </template>

    <!-- Inline create input for subfolders -->
    <div v-if="showCreateInput" class="py-1" :style="{ paddingLeft: childPaddingLeft }">
      <input
        :value="newFolderName"
        @input="onCreateInput"
        @keydown="onCreateKeydown"
        @blur="emit('confirm-create')"
        ref="createInput"
        type="text"
        placeholder="Folder name..."
        class="w-[calc(100%-24px)] bg-nord0 text-nord4 border border-nord8 rounded px-2 py-0.5 text-sm focus:outline-none"
      />
    </div>
  </div>
</template>
