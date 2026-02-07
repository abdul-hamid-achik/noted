<script setup lang="ts">
import { ref, computed } from 'vue'
import { useNotesStore } from '../stores/notes'
import type { Note } from '../types'

const notesStore = useNotesStore()
const searchQuery = ref('')
const selectedTag = ref<string | null>(null)

const filteredNotes = computed(() => {
  let result = notesStore.sortedNotes
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

function selectNote(note: Note) {
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
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Search -->
    <div class="p-3 border-b border-nord2">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search notes..."
        class="w-full bg-nord0 text-nord4 border border-nord3 rounded px-3 py-1.5 text-sm focus:outline-none focus:border-nord8 placeholder-nord3"
      />
    </div>

    <!-- New Note button -->
    <div class="px-3 py-2 border-b border-nord2">
      <button
        @click="createNewNote"
        class="w-full bg-nord10 hover:bg-nord9 text-nord6 text-sm font-medium py-1.5 px-3 rounded transition-colors"
      >
        + New Note
      </button>
    </div>

    <!-- Tag filters -->
    <div v-if="notesStore.tags.length > 0" class="px-3 py-2 border-b border-nord2 flex flex-wrap gap-1">
      <button
        v-for="tag in notesStore.tags"
        :key="tag.id"
        @click="toggleTag(tag.name)"
        class="text-xs px-2 py-0.5 rounded-full transition-colors"
        :class="
          selectedTag === tag.name
            ? 'bg-nord8 text-nord0'
            : 'bg-nord2 text-nord4 hover:bg-nord3'
        "
      >
        {{ tag.name }}
        <span v-if="tag.note_count" class="ml-0.5 opacity-70">{{ tag.note_count }}</span>
      </button>
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
        class="px-3 py-2.5 border-b border-nord2 cursor-pointer transition-colors"
        :class="
          notesStore.currentNote?.id === note.id
            ? 'bg-nord2'
            : 'hover:bg-nord0'
        "
      >
        <div class="flex items-center justify-between mb-0.5">
          <span class="text-sm font-medium text-nord6 truncate">{{ note.title }}</span>
          <span class="text-xs text-nord3 ml-2 shrink-0">{{ formatDate(note.updated_at) }}</span>
        </div>
        <div class="text-xs text-nord3 truncate mb-1">
          {{ firstLine(note.content) }}
        </div>
        <div v-if="note.tags.length > 0" class="flex gap-1 flex-wrap">
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
  </div>
</template>
