<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useNotesStore } from '../stores/notes'
import { useApi } from '../composables/useApi'
import type { Note } from '../types'

const router = useRouter()
const notesStore = useNotesStore()
const api = useApi()
const backlinks = ref<Note[]>([])
const loading = ref(false)

async function fetchBacklinks(noteId: number) {
  loading.value = true
  try {
    backlinks.value = await api.getBacklinks(noteId)
  } catch {
    backlinks.value = []
  } finally {
    loading.value = false
  }
}

watch(
  () => notesStore.currentNote?.id,
  (id) => {
    if (id) fetchBacklinks(id)
    else backlinks.value = []
  },
  { immediate: true }
)

function navigateToNote(note: Note) {
  notesStore.selectNote(note)
  router.push(`/notes/${note.id}`)
}

function firstLine(content: string): string {
  const line = content.split('\n').find((l) => l.trim().length > 0) || ''
  return line.length > 60 ? line.slice(0, 60) + '...' : line
}
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex items-center h-9 px-3 bg-nord1 border-b border-nord2 text-sm">
      <span class="text-nord3">Backlinks</span>
      <span class="ml-auto text-xs text-nord3">{{ backlinks.length }}</span>
    </div>

    <div class="flex-1 overflow-y-auto">
      <div v-if="loading" class="px-3 py-4 text-xs text-nord3 text-center">Loading...</div>
      <div v-else-if="!notesStore.currentNote" class="px-3 py-4 text-xs text-nord3 text-center">
        No note selected
      </div>
      <div v-else-if="backlinks.length === 0" class="px-3 py-4 text-xs text-nord3 text-center">
        No notes link to this note
      </div>
      <button
        v-for="note in backlinks"
        :key="note.id"
        @click="navigateToNote(note)"
        class="w-full text-left px-3 py-2 border-b border-nord2 hover:bg-nord2 transition-colors"
      >
        <div class="text-sm text-nord6 font-medium truncate">{{ note.title }}</div>
        <div class="text-xs text-nord3 truncate mt-0.5">{{ firstLine(note.content) }}</div>
      </button>
    </div>
  </div>
</template>
