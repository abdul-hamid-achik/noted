<script setup lang="ts">
import { ref, computed } from 'vue'
import { useNotesStore } from '../stores/notes'
import FolderTreeNode from './FolderTreeNode.vue'
import type { Folder } from '../types'

type FolderNode = Folder & { children: FolderNode[] }

const notesStore = useNotesStore()
const expandedFolders = ref<Set<number>>(new Set())
const editingFolderId = ref<number | null>(null)
const editingName = ref('')
const creatingFolder = ref(false)
const newFolderName = ref('')
const newFolderParentId = ref<number | null>(null)
const contextMenu = ref<{ x: number; y: number; folder: FolderNode } | null>(null)

const allNotesCount = computed(() => notesStore.notes.length)

function toggleExpand(id: number) {
  if (expandedFolders.value.has(id)) {
    expandedFolders.value.delete(id)
  } else {
    expandedFolders.value.add(id)
  }
}

function selectFolder(id: number | null) {
  notesStore.selectFolder(id)
}

function startCreate(parentId: number | null = null) {
  creatingFolder.value = true
  newFolderName.value = ''
  newFolderParentId.value = parentId
  if (parentId !== null) {
    expandedFolders.value.add(parentId)
  }
}

async function confirmCreate() {
  const name = newFolderName.value.trim()
  if (!name) {
    creatingFolder.value = false
    return
  }
  try {
    await notesStore.createFolder({
      name,
      parent_id: newFolderParentId.value,
    })
  } catch (e) {
    console.error('Failed to create folder:', e)
  }
  creatingFolder.value = false
  newFolderName.value = ''
  newFolderParentId.value = null
}

function cancelCreate() {
  creatingFolder.value = false
  newFolderName.value = ''
  newFolderParentId.value = null
}

function startRename(folder: FolderNode) {
  editingFolderId.value = folder.id
  editingName.value = folder.name
  contextMenu.value = null
}

async function confirmRename() {
  if (editingFolderId.value === null) return
  const name = editingName.value.trim()
  if (!name) {
    editingFolderId.value = null
    return
  }
  try {
    await notesStore.updateFolder(editingFolderId.value, { name })
  } catch (e) {
    console.error('Failed to rename folder:', e)
  }
  editingFolderId.value = null
  editingName.value = ''
}

function cancelRename() {
  editingFolderId.value = null
  editingName.value = ''
}

async function handleDeleteFolder(folder: FolderNode) {
  contextMenu.value = null
  try {
    await notesStore.deleteFolder(folder.id)
  } catch (e) {
    console.error('Failed to delete folder:', e)
  }
}

function showContextMenu(e: MouseEvent, folder: FolderNode) {
  e.preventDefault()
  contextMenu.value = { x: e.clientX, y: e.clientY, folder }
}

function closeContextMenu() {
  contextMenu.value = null
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'move'
  }
}

async function onDropOnFolder(e: DragEvent, folderId: number | null) {
  e.preventDefault()
  const noteIdStr = e.dataTransfer?.getData('text/note-id')
  if (!noteIdStr) return
  const noteId = parseInt(noteIdStr, 10)
  if (isNaN(noteId)) return
  try {
    await notesStore.moveNoteToFolder(noteId, folderId)
  } catch (err) {
    console.error('Failed to move note:', err)
  }
}
</script>

<template>
  <div class="select-none" @click="closeContextMenu">
    <!-- Header -->
    <div class="flex items-center justify-between px-3 py-1.5">
      <span class="text-[10px] font-semibold uppercase tracking-wider text-nord3">Folders</span>
      <button
        @click.stop="startCreate(null)"
        class="text-nord3 hover:text-nord4 text-xs px-1"
        title="New folder"
      >
        +
      </button>
    </div>

    <!-- All Notes -->
    <div
      @click="selectFolder(null)"
      @dragover="onDragOver"
      @drop="onDropOnFolder($event, null)"
      class="flex items-center gap-1.5 px-3 py-1 cursor-pointer text-sm transition-colors"
      :class="notesStore.currentFolder === null ? 'bg-nord2 text-nord6' : 'text-nord4 hover:bg-nord0'"
    >
      <svg class="w-3.5 h-3.5 shrink-0" viewBox="0 0 16 16" fill="currentColor">
        <path d="M1 3.5A1.5 1.5 0 012.5 2h3.879a1.5 1.5 0 011.06.44l1.122 1.12A1.5 1.5 0 009.62 4H13.5A1.5 1.5 0 0115 5.5v7a1.5 1.5 0 01-1.5 1.5h-11A1.5 1.5 0 011 12.5v-9z"/>
      </svg>
      <span class="truncate">All Notes</span>
      <span class="ml-auto text-[10px] text-nord3">{{ allNotesCount }}</span>
    </div>

    <!-- Folder tree -->
    <FolderTreeNode
      v-for="folder in notesStore.folderTree"
      :key="folder.id"
      :folder="(folder as FolderNode)"
      :depth="0"
      :expanded-folders="expandedFolders"
      :editing-folder-id="editingFolderId"
      :editing-name="editingName"
      :creating-folder="creatingFolder"
      :new-folder-parent-id="newFolderParentId"
      :new-folder-name="newFolderName"
      @toggle-expand="toggleExpand"
      @select="selectFolder"
      @context-menu="showContextMenu"
      @confirm-rename="confirmRename"
      @cancel-rename="cancelRename"
      @update:editing-name="editingName = $event"
      @update:new-folder-name="newFolderName = $event"
      @confirm-create="confirmCreate"
      @cancel-create="cancelCreate"
      @drop="onDropOnFolder"
    />

    <!-- Inline create at root level -->
    <div v-if="creatingFolder && newFolderParentId === null" class="px-3 py-1">
      <input
        v-model="newFolderName"
        @keydown.enter="confirmCreate"
        @keydown.escape="cancelCreate"
        @blur="confirmCreate"
        type="text"
        placeholder="Folder name..."
        class="w-full bg-nord0 text-nord4 border border-nord8 rounded px-2 py-0.5 text-sm focus:outline-none"
        autofocus
      />
    </div>

    <!-- Context menu -->
    <Teleport to="body">
      <div
        v-if="contextMenu"
        class="fixed bg-nord2 border border-nord3 rounded shadow-lg py-1 z-50 min-w-[140px]"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        @click.stop
      >
        <button
          @click="startRename(contextMenu!.folder)"
          class="w-full text-left px-3 py-1 text-sm text-nord4 hover:bg-nord3 transition-colors"
        >
          Rename
        </button>
        <button
          @click="startCreate(contextMenu!.folder.id)"
          class="w-full text-left px-3 py-1 text-sm text-nord4 hover:bg-nord3 transition-colors"
        >
          New Subfolder
        </button>
        <div class="border-t border-nord3 my-0.5" />
        <button
          @click="handleDeleteFolder(contextMenu!.folder)"
          class="w-full text-left px-3 py-1 text-sm text-nord11 hover:bg-nord3 transition-colors"
        >
          Delete
        </button>
      </div>
    </Teleport>
  </div>
</template>
