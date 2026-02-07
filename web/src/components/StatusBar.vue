<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useNotesStore } from '../stores/notes'
import { useUiStore } from '../stores/ui'

const router = useRouter()
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
      <button
        @click="router.push('/settings')"
        class="hover:text-nord4 transition-colors"
        title="Settings"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd" />
        </svg>
      </button>
    </div>
  </div>
</template>
