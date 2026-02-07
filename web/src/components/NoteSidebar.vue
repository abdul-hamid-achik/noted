<script setup lang="ts">
import { ref, computed } from 'vue'
import { useNotesStore } from '../stores/notes'
import FolderTree from './FolderTree.vue'
import type { Note } from '../types'

const notesStore = useNotesStore()
const searchQuery = ref('')
const selectedTag = ref<string | null>(null)
const hoveredNoteId = ref<number | null>(null)
const confirmingDeleteId = ref<number | null>(null)
const confirmingTagId = ref<number | null>(null)
const multiSelectMode = ref(false)
const selectedNoteIds = ref<Set<number>>(new Set())
const batchConfirming = ref(false)

const filteredNotes = computed(() => {
  let result = notesStore.notesInCurrentFolder
  if (selectedTag.value) {
    result = result.filter((n) => n.tags.some((t) => t.name === selectedTag.value))
  }
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(
      (n) =>
        n.title.toLowerCase().includes(q) || n.content.toLowerCase().includes(q)
    )
  }
  return result
})

function onNoteDragStart(e: DragEvent, note: Note) {
  if (e.dataTransfer) {
    e.dataTransfer.setData('text/note-id', String(note.id))
    e.dataTransfer.effectAllowed = 'move'
  }
}

function selectNote(note: Note) {
  if (multiSelectMode.value) {
    toggleNoteSelection(note.id)
    return
  }
  notesStore.selectNote(note)
}

async function createNewNote() {
  await notesStore.createNote({
    title: 'Untitled',
    content: '',
    tags: [],
  })
}

function toggleTag(tag: string) {
  selectedTag.value = selectedTag.value === tag ? null : tag
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  const days = Math.floor(hrs / 24)
  if (days < 7) return `${days}d ago`
  return d.toLocaleDateString()
}

function firstLine(content: string): string {
  const line = content.split('\n').find((l) => l.trim().length > 0) || ''
  return line.length > 50 ? line.slice(0, 50) + '...' : line
}

// Note deletion
function startDeleteNote(id: number, event: Event) {
  event.stopPropagation()
  confirmingDeleteId.value = id
}

function cancelDeleteNote() {
  confirmingDeleteId.value = null
}

async function confirmDeleteNote(id: number, event: Event) {
  event.stopPropagation()
  await notesStore.deleteNote(id)
  confirmingDeleteId.value = null
  selectedNoteIds.value.delete(id)
}

// Tag deletion
function startDeleteTag(tagId: number, event: Event) {
  event.stopPropagation()
  confirmingTagId.value = tagId
}

function cancelDeleteTag() {
  confirmingTagId.value = null
}

async function confirmDeleteTag(tagId: number, event: Event) {
  event.stopPropagation()
  await notesStore.deleteTag(tagId)
  confirmingTagId.value = null
  if (selectedTag.value && !notesStore.tags.some((t) => t.name === selectedTag.value)) {
    selectedTag.value = null
  }
}

// Multi-select
function toggleNoteSelection(id: number) {
  const s = new Set(selectedNoteIds.value)
  if (s.has(id)) {
    s.delete(id)
  } else {
    s.add(id)
  }
  selectedNoteIds.value = s
}

function toggleMultiSelect() {
  multiSelectMode.value = !multiSelectMode.value
  if (!multiSelectMode.value) {
    selectedNoteIds.value = new Set()
    batchConfirming.value = false
  }
}

function startBatchDelete() {
  batchConfirming.value = true
}

function cancelBatchDelete() {
  batchConfirming.value = false
}

async function confirmBatchDelete() {
  const ids = Array.from(selectedNoteIds.value)
  for (const id of ids) {
    await notesStore.deleteNote(id)
  }
  selectedNoteIds.value = new Set()
  batchConfirming.value = false
  multiSelectMode.value = false
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    if (confirmingDeleteId.value !== null) {
      cancelDeleteNote()
    } else if (confirmingTagId.value !== null) {
      cancelDeleteTag()
    } else if (batchConfirming.value) {
      cancelBatchDelete()
    } else if (multiSelectMode.value) {
      toggleMultiSelect()
    }
  }
}
</script>

<template>
  <div class="flex flex-col h-full" @keydown="handleKeydown" tabindex="-1">
    <!-- Search -->
    <div class="p-3 border-b border-nord2">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search notes..."
        class="w-full bg-nord0 text-nord4 border border-nord3 rounded px-3 py-1.5 text-sm focus:outline-none focus:border-nord8 placeholder-nord3"
      />
    </div>

    <!-- New Note button + multi-select toggle -->
    <div class="px-3 py-2 border-b border-nord2 flex gap-2">
      <button
        @click="createNewNote"
        class="flex-1 bg-nord10 hover:bg-nord9 text-nord6 text-sm font-medium py-1.5 px-3 rounded transition-colors"
      >
        + New Note
      </button>
      <button
        @click="toggleMultiSelect"
        class="px-2.5 py-1.5 rounded text-sm transition-colors"
        :class="
          multiSelectMode
            ? 'bg-nord8 text-nord0'
            : 'bg-nord2 text-nord4 hover:bg-nord3'
        "
        :title="multiSelectMode ? 'Exit multi-select' : 'Multi-select'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
          <path d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" />
        </svg>
      </button>
    </div>

    <!-- Tag filters -->
    <div v-if="notesStore.tags.length > 0" class="px-3 py-2 border-b border-nord2 flex flex-wrap gap-1">
      <div
        v-for="tag in notesStore.tags"
        :key="tag.id"
        class="group relative inline-flex items-center"
      >
        <!-- Tag deletion confirmation -->
        <div
          v-if="confirmingTagId === tag.id"
          class="flex items-center gap-1 text-xs animate-slideIn"
        >
          <span class="text-nord4 whitespace-nowrap">Delete tag?</span>
          <button
            @click="confirmDeleteTag(tag.id, $event)"
            class="px-1.5 py-0.5 rounded bg-nord11 text-nord6 hover:bg-red-500 transition-colors"
          >
            Yes
          </button>
          <button
            @click="cancelDeleteTag"
            class="px-1.5 py-0.5 rounded bg-nord2 text-nord4 hover:bg-nord3 transition-colors"
          >
            No
          </button>
        </div>

        <!-- Normal tag pill -->
        <button
          v-else
          @click="toggleTag(tag.name)"
          class="text-xs px-2 py-0.5 rounded-full transition-colors inline-flex items-center gap-0.5"
          :class="
            selectedTag === tag.name
              ? 'bg-nord8 text-nord0'
              : 'bg-nord2 text-nord4 hover:bg-nord3'
          "
        >
          {{ tag.name }}
          <span v-if="tag.note_count" class="opacity-70">{{ tag.note_count }}</span>
          <span
            @click="startDeleteTag(tag.id, $event)"
            class="ml-0.5 opacity-0 group-hover:opacity-100 transition-opacity hover:text-nord11 cursor-pointer"
            title="Delete tag"
          >&times;</span>
        </button>
      </div>
    </div>

    <!-- Folder tree -->
    <div class="border-b border-nord2 py-1">
      <FolderTree />
    </div>

    <!-- Notes list -->
    <div class="flex-1 overflow-y-auto">
      <div
        v-if="filteredNotes.length === 0"
        class="p-4 text-center text-nord3 text-sm"
      >
        {{ searchQuery ? 'No matching notes' : 'No notes yet' }}
      </div>
      <div
        v-for="note in filteredNotes"
        :key="note.id"
        @click="selectNote(note)"
        @mouseenter="hoveredNoteId = note.id"
        @mouseleave="hoveredNoteId = null"
        draggable="true"
        @dragstart="onNoteDragStart($event, note)"
        class="relative px-3 py-2.5 border-b border-nord2 cursor-pointer transition-colors"
        :class="[
          notesStore.currentNote?.id === note.id && !multiSelectMode
            ? 'bg-nord2'
            : 'hover:bg-nord0',
          selectedNoteIds.has(note.id) ? 'bg-nord2/50 border-l-2 border-l-nord8' : '',
        ]"
      >
        <!-- Inline delete confirmation -->
        <div
          v-if="confirmingDeleteId === note.id"
          class="absolute inset-0 bg-nord1 flex items-center justify-center gap-2 z-10 animate-slideIn"
        >
          <span class="text-sm text-nord4">Delete this note?</span>
          <button
            @click="confirmDeleteNote(note.id, $event)"
            class="px-3 py-1 text-xs rounded bg-nord11 text-nord6 hover:brightness-110 transition-colors font-medium"
          >
            Delete
          </button>
          <button
            @click.stop="cancelDeleteNote"
            class="px-3 py-1 text-xs rounded bg-nord3 text-nord4 hover:bg-nord2 transition-colors"
          >
            Cancel
          </button>
        </div>

        <!-- Note content -->
        <div class="flex items-center justify-between mb-0.5">
          <div class="flex items-center gap-2 min-w-0">
            <!-- Multi-select checkbox -->
            <input
              v-if="multiSelectMode"
              type="checkbox"
              :checked="selectedNoteIds.has(note.id)"
              @click.stop="toggleNoteSelection(note.id)"
              class="h-3.5 w-3.5 rounded border-nord3 text-nord8 focus:ring-nord8 shrink-0 accent-[#88C0D0]"
            />
            <span class="text-sm font-medium text-nord6 truncate">{{ note.title }}</span>
          </div>
          <div class="flex items-center gap-1 shrink-0">
            <!-- Delete button on hover -->
            <button
              v-if="hoveredNoteId === note.id && confirmingDeleteId !== note.id && !multiSelectMode"
              @click="startDeleteNote(note.id, $event)"
              class="p-0.5 rounded text-nord3 hover:text-nord11 hover:bg-nord2 transition-colors"
              title="Delete note"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
              </svg>
            </button>
            <span class="text-xs text-nord3 ml-1">{{ formatDate(note.updated_at) }}</span>
          </div>
        </div>
        <div class="text-xs text-nord3 truncate mb-1" :class="multiSelectMode ? 'ml-5.5' : ''">
          {{ firstLine(note.content) }}
        </div>
        <div v-if="note.tags.length > 0" class="flex gap-1 flex-wrap" :class="multiSelectMode ? 'ml-5.5' : ''">
          <span
            v-for="tag in note.tags"
            :key="tag.id"
            class="text-[10px] px-1.5 py-px rounded bg-nord2 text-nord8"
          >
            {{ tag.name }}
          </span>
        </div>
      </div>
    </div>

    <!-- Batch delete bar -->
    <div
      v-if="multiSelectMode && selectedNoteIds.size > 0"
      class="px-3 py-2.5 border-t border-nord2 bg-nord1"
    >
      <div v-if="batchConfirming" class="flex items-center justify-between gap-2 animate-slideIn">
        <span class="text-sm text-nord4">
          Delete {{ selectedNoteIds.size }} note{{ selectedNoteIds.size > 1 ? 's' : '' }}?
        </span>
        <div class="flex gap-2">
          <button
            @click="confirmBatchDelete"
            class="px-3 py-1 text-xs rounded bg-nord11 text-nord6 hover:brightness-110 transition-colors font-medium"
          >
            Delete
          </button>
          <button
            @click="cancelBatchDelete"
            class="px-3 py-1 text-xs rounded bg-nord3 text-nord4 hover:bg-nord2 transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
      <button
        v-else
        @click="startBatchDelete"
        class="w-full py-1.5 text-sm rounded bg-nord11/20 text-nord11 hover:bg-nord11/30 transition-colors font-medium"
      >
        Delete selected ({{ selectedNoteIds.size }})
      </button>
    </div>
  </div>
</template>

<style scoped>
@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateX(-8px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.animate-slideIn {
  animation: slideIn 150ms ease-out;
}
</style>
