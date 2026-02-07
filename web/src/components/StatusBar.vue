<script setup lang="ts">
import { computed } from 'vue'
import { useNotesStore } from '../stores/notes'
import { useUiStore } from '../stores/ui'

const notesStore = useNotesStore()
const uiStore = useUiStore()

const modeLabel = computed(() => {
  switch (uiStore.vimMode) {
    case 'insert':
      return '-- INSERT --'
    case 'visual':
      return '-- VISUAL --'
    case 'command':
      return ':'
    default:
      return '-- NORMAL --'
  }
})

const modeClass = computed(() => {
  switch (uiStore.vimMode) {
    case 'insert':
      return 'bg-nord14 text-nord0'
    case 'visual':
      return 'bg-nord15 text-nord0'
    case 'command':
      return 'bg-nord13 text-nord0'
    default:
      return 'bg-nord9 text-nord0'
  }
})

const wordCount = computed(() => {
  const content = notesStore.currentNote?.content || ''
  if (!content.trim()) return 0
  return content.trim().split(/\s+/).length
})

const tagCount = computed(() => notesStore.currentNote?.tags?.length || 0)
</script>

<template>
  <div class="flex items-center h-6 bg-nord1 border-t border-nord2 text-xs font-mono select-none">
    <!-- Vim mode indicator -->
    <div :class="[modeClass, 'px-2 h-full flex items-center font-bold']">
      {{ modeLabel }}
    </div>

    <!-- Note title + modified -->
    <div class="flex items-center px-3 text-nord4 truncate">
      <template v-if="notesStore.currentNote">
        <span>{{ notesStore.currentNote.title }}</span>
      </template>
      <template v-else>
        <span class="text-nord3">no file</span>
      </template>
    </div>

    <div class="flex-1" />

    <!-- Right side info -->
    <div class="flex items-center gap-3 px-3 text-nord3">
      <template v-if="notesStore.currentNote">
        <span>{{ tagCount }} tag{{ tagCount !== 1 ? 's' : '' }}</span>
        <span>{{ wordCount }} word{{ wordCount !== 1 ? 's' : '' }}</span>
        <span>{{ uiStore.cursorLine }}:{{ uiStore.cursorCol }}</span>
      </template>
    </div>
  </div>
</template>
