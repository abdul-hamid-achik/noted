<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useNotesStore } from '../stores/notes'
import { useUiStore } from '../stores/ui'
import type { Note } from '../types'
import Fuse from 'fuse.js'

const router = useRouter()
const notesStore = useNotesStore()
const uiStore = useUiStore()
const query = ref('')
const selectedIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)

const fuse = computed(() => new Fuse(notesStore.sortedNotes, {
  keys: [
    { name: 'title', weight: 2 },
    { name: 'content', weight: 1 },
    { name: 'tags.name', weight: 1.5 },
  ],
  threshold: 0.4,
  includeScore: true,
  includeMatches: true,
}))

const filteredNotes = computed(() => {
  if (!query.value) return notesStore.sortedNotes.slice(0, 30)
  return fuse.value.search(query.value).slice(0, 30).map((r) => r.item)
})

function selectNote(note: Note) {
  notesStore.selectNote(note)
  router.push(`/notes/${note.id}`)
  uiStore.closeFuzzyFinder()
}

function handleKeydown(e: KeyboardEvent) {
  switch (e.key) {
    case 'Escape':
      e.preventDefault()
      uiStore.closeFuzzyFinder()
      break
    case 'ArrowDown':
      e.preventDefault()
      selectedIndex.value = Math.min(selectedIndex.value + 1, filteredNotes.value.length - 1)
      break
    case 'ArrowUp':
      e.preventDefault()
      selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
      break
    case 'Enter':
      e.preventDefault()
      if (filteredNotes.value[selectedIndex.value]) {
        selectNote(filteredNotes.value[selectedIndex.value])
      }
      break
  }
}

function handleOverlayClick(e: MouseEvent) {
  if (e.target === e.currentTarget) {
    uiStore.closeFuzzyFinder()
  }
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

onMounted(async () => {
  document.addEventListener('keydown', handleKeydown)
  await nextTick()
  inputRef.value?.focus()
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div
    class="fixed inset-0 bg-black/60 flex items-start justify-center pt-[15vh] z-50"
    @click="handleOverlayClick"
  >
    <div class="bg-nord1 rounded-lg shadow-2xl w-full max-w-lg border border-nord2 overflow-hidden">
      <!-- Search input -->
      <div class="p-3 border-b border-nord2">
        <input
          ref="inputRef"
          v-model="query"
          type="text"
          placeholder="Search notes... (fuzzy match)"
          class="w-full bg-nord0 text-nord4 border border-nord3 rounded px-3 py-2 text-sm focus:outline-none focus:border-nord8 placeholder-nord3 font-mono"
        />
      </div>

      <!-- Results -->
      <div class="max-h-[50vh] overflow-y-auto">
        <div
          v-if="filteredNotes.length === 0"
          class="p-4 text-center text-nord3 text-sm"
        >
          No notes found
        </div>
        <div
          v-for="(note, index) in filteredNotes"
          :key="note.id"
          @click="selectNote(note)"
          class="px-3 py-2 cursor-pointer border-b border-nord2 last:border-b-0 transition-colors"
          :class="index === selectedIndex ? 'bg-nord2' : 'hover:bg-nord0'"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 min-w-0">
              <span v-if="note.pinned" class="text-nord13 text-xs shrink-0" title="Pinned">*</span>
              <span class="text-sm font-medium text-nord6 truncate">{{ note.title }}</span>
            </div>
            <span class="text-xs text-nord3 ml-2 shrink-0">{{ formatDate(note.updated_at) }}</span>
          </div>
          <div class="flex items-center gap-1 mt-0.5">
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

      <!-- Footer -->
      <div class="px-3 py-2 border-t border-nord2 flex items-center gap-4 text-xs text-nord3">
        <span><kbd class="bg-nord2 px-1 py-px rounded text-nord4">↑↓</kbd> navigate</span>
        <span><kbd class="bg-nord2 px-1 py-px rounded text-nord4">↵</kbd> select</span>
        <span><kbd class="bg-nord2 px-1 py-px rounded text-nord4">esc</kbd> close</span>
        <span class="ml-auto">{{ filteredNotes.length }} notes</span>
      </div>
    </div>
  </div>
</template>
