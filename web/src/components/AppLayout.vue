<script setup lang="ts">
import { onMounted } from 'vue'
import { useNotesStore } from '../stores/notes'
import { useUiStore } from '../stores/ui'
import { useWebSocket } from '../composables/useWebSocket'
import NoteSidebar from './NoteSidebar.vue'
import NoteEditor from './NoteEditor.vue'
import MarkdownPreview from './MarkdownPreview.vue'
import StatusBar from './StatusBar.vue'
import FuzzyFinder from './FuzzyFinder.vue'

const notesStore = useNotesStore()
const uiStore = useUiStore()

useWebSocket((event) => {
  notesStore.handleWsEvent(event.type, event.data)
})

onMounted(async () => {
  await Promise.all([notesStore.fetchNotes(), notesStore.fetchTags(), notesStore.fetchFolders()])
})

function handleKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'p') {
    e.preventDefault()
    uiStore.openFuzzyFinder()
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
    e.preventDefault()
    uiStore.toggleSidebar()
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'e') {
    e.preventDefault()
    uiStore.togglePreview()
  }
}
</script>

<template>
  <div
    class="flex flex-col h-screen bg-nord0 text-nord4 overflow-hidden"
    @keydown="handleKeydown"
  >
    <div class="flex flex-1 overflow-hidden">
      <!-- Sidebar -->
      <aside
        v-if="uiStore.sidebarOpen"
        class="w-[250px] min-w-[250px] border-r border-nord2 flex flex-col bg-nord1 overflow-hidden"
      >
        <NoteSidebar />
      </aside>

      <!-- Editor -->
      <main class="flex-1 flex flex-col overflow-hidden">
        <NoteEditor />
      </main>

      <!-- Preview -->
      <aside
        v-if="uiStore.previewOpen"
        class="w-[350px] min-w-[350px] border-l border-nord2 flex flex-col bg-nord0 overflow-hidden"
      >
        <MarkdownPreview />
      </aside>
    </div>

    <!-- Status bar -->
    <StatusBar />

    <!-- Fuzzy finder modal -->
    <FuzzyFinder v-if="uiStore.fuzzyFinderOpen" />
  </div>
</template>
