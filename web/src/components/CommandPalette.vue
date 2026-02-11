<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useNotesStore } from '../stores/notes'
import { useUiStore } from '../stores/ui'
import Fuse from 'fuse.js'

const router = useRouter()
const notesStore = useNotesStore()
const uiStore = useUiStore()
const query = ref('')
const selectedIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)

interface Command {
  id: string
  label: string
  shortcut?: string
  category: 'note' | 'navigation' | 'view' | 'action'
  action: () => void
}

const commands = computed<Command[]>(() => {
  const cmds: Command[] = [
    // Note actions
    { id: 'new-note', label: 'New Note', shortcut: ':new', category: 'note', action: () => {
      notesStore.createNote({ title: 'Untitled', content: '', tags: [] })
    }},
    // Navigation
    { id: 'go-dashboard', label: 'Go to Dashboard', category: 'navigation', action: () => router.push('/dashboard') },
    { id: 'go-settings', label: 'Go to Settings', category: 'navigation', action: () => router.push('/settings') },
    { id: 'go-graph', label: 'Go to Graph', category: 'navigation', action: () => router.push('/graph') },
    { id: 'go-editor', label: 'Go to Editor', category: 'navigation', action: () => router.push('/') },
    // View toggles
    { id: 'toggle-sidebar', label: 'Toggle Sidebar', shortcut: 'Ctrl+B', category: 'view', action: () => uiStore.toggleSidebar() },
    { id: 'toggle-preview', label: 'Toggle Preview', shortcut: 'Ctrl+E', category: 'view', action: () => uiStore.togglePreview() },
    { id: 'toggle-backlinks', label: 'Toggle Backlinks Panel', category: 'view', action: () => uiStore.toggleBacklinks() },
    { id: 'toggle-outline', label: 'Toggle Outline Panel', category: 'view', action: () => uiStore.toggleOutline() },
    // Note search (dynamic)
    ...notesStore.sortedNotes.map((note) => ({
      id: `note-${note.id}`,
      label: note.title,
      category: 'note' as const,
      action: () => {
        notesStore.selectNote(note)
        router.push(`/notes/${note.id}`)
      },
    })),
  ]
  return cmds
})

const fuse = computed(() => new Fuse(commands.value, {
  keys: ['label'],
  threshold: 0.4,
  includeScore: true,
}))

const filteredCommands = computed(() => {
  if (!query.value) return commands.value.slice(0, 20)
  return fuse.value.search(query.value).slice(0, 20).map((r) => r.item)
})

function executeCommand(cmd: Command) {
  uiStore.closeCommandPalette()
  cmd.action()
}

function handleKeydown(e: KeyboardEvent) {
  switch (e.key) {
    case 'Escape':
      e.preventDefault()
      uiStore.closeCommandPalette()
      break
    case 'ArrowDown':
      e.preventDefault()
      selectedIndex.value = Math.min(selectedIndex.value + 1, filteredCommands.value.length - 1)
      break
    case 'ArrowUp':
      e.preventDefault()
      selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
      break
    case 'Enter':
      e.preventDefault()
      if (filteredCommands.value[selectedIndex.value]) {
        executeCommand(filteredCommands.value[selectedIndex.value])
      }
      break
  }
}

function handleOverlayClick(e: MouseEvent) {
  if (e.target === e.currentTarget) {
    uiStore.closeCommandPalette()
  }
}

const categoryIcon: Record<string, string> = {
  note: '#',
  navigation: '>',
  view: '~',
  action: '!',
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
          placeholder="Type a command or search notes..."
          class="w-full bg-nord0 text-nord4 border border-nord3 rounded px-3 py-2 text-sm focus:outline-none focus:border-nord8 placeholder-nord3 font-mono"
        />
      </div>

      <!-- Results -->
      <div class="max-h-[50vh] overflow-y-auto">
        <div
          v-if="filteredCommands.length === 0"
          class="p-4 text-center text-nord3 text-sm"
        >
          No commands found
        </div>
        <div
          v-for="(cmd, index) in filteredCommands"
          :key="cmd.id"
          @click="executeCommand(cmd)"
          class="px-3 py-2 cursor-pointer border-b border-nord2 last:border-b-0 transition-colors flex items-center gap-3"
          :class="index === selectedIndex ? 'bg-nord2' : 'hover:bg-nord0'"
        >
          <span class="text-xs font-mono text-nord8 w-4 text-center shrink-0">{{ categoryIcon[cmd.category] }}</span>
          <span class="text-sm text-nord6 flex-1">{{ cmd.label }}</span>
          <span v-if="cmd.shortcut" class="text-xs text-nord3 font-mono">{{ cmd.shortcut }}</span>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-3 py-2 border-t border-nord2 flex items-center gap-4 text-xs text-nord3">
        <span><kbd class="bg-nord2 px-1 py-px rounded text-nord4">↑↓</kbd> navigate</span>
        <span><kbd class="bg-nord2 px-1 py-px rounded text-nord4">↵</kbd> execute</span>
        <span><kbd class="bg-nord2 px-1 py-px rounded text-nord4">esc</kbd> close</span>
        <span class="ml-auto">{{ filteredCommands.length }} results</span>
      </div>
    </div>
  </div>
</template>
