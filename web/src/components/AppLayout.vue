<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useNotesStore } from '../stores/notes'
import { useUiStore } from '../stores/ui'
import { useSSE } from '../composables/useSSE'
import NoteSidebar from './NoteSidebar.vue'
import NoteEditor from './NoteEditor.vue'
import MarkdownPreview from './MarkdownPreview.vue'
import BacklinksPanel from './BacklinksPanel.vue'
import OutlinePanel from './OutlinePanel.vue'
import StatusBar from './StatusBar.vue'
import FuzzyFinder from './FuzzyFinder.vue'
import CommandPalette from './CommandPalette.vue'
import ToastContainer from './ToastContainer.vue'

const router = useRouter()
const route = useRoute()
const notesStore = useNotesStore()
const uiStore = useUiStore()

useSSE((event) => {
  notesStore.handleSSEEvent(event.type, event.data)
})

onMounted(async () => {
  await Promise.all([notesStore.fetchNotes(), notesStore.fetchTags(), notesStore.fetchFolders()])

  // Deep link: if URL has a note ID, select it
  const noteId = route.params.id
  if (noteId) {
    const id = Number(noteId)
    const note = notesStore.notes.find((n) => n.id === id)
    if (note) notesStore.selectNote(note)
  }
})

// Update URL when note changes
watch(
  () => notesStore.currentNote?.id,
  (id) => {
    if (id && route.name === 'editor') {
      router.replace(`/notes/${id}`)
    }
  }
)

function handleKeydown(e: KeyboardEvent) {
  const mod = e.ctrlKey || e.metaKey

  // Cmd/Ctrl+P: fuzzy finder (note search)
  if (mod && e.key === 'p' && !e.shiftKey) {
    e.preventDefault()
    uiStore.openFuzzyFinder()
  }
  // Cmd/Ctrl+Shift+P: command palette
  if (mod && e.key.toLowerCase() === 'p' && e.shiftKey) {
    e.preventDefault()
    uiStore.openCommandPalette()
  }
  // Cmd/Ctrl+B: toggle sidebar
  if (mod && e.key === 'b' && !e.shiftKey) {
    e.preventDefault()
    uiStore.toggleSidebar()
  }
  // Cmd/Ctrl+E: toggle preview
  if (mod && e.key === 'e') {
    e.preventDefault()
    uiStore.togglePreview()
  }
  // Cmd/Ctrl+N: new note
  if (mod && e.key === 'n' && !e.shiftKey) {
    e.preventDefault()
    notesStore.createNote({ title: 'Untitled', content: '' })
  }
  // Cmd/Ctrl+Shift+B: toggle backlinks panel
  if (mod && e.key.toLowerCase() === 'b' && e.shiftKey) {
    e.preventDefault()
    uiStore.toggleBacklinks()
  }
  // Cmd/Ctrl+Shift+O: toggle outline panel
  if (mod && e.key.toLowerCase() === 'o' && e.shiftKey) {
    e.preventDefault()
    uiStore.toggleOutline()
  }
  // Cmd/Ctrl+Shift+G: go to graph view
  if (mod && e.key.toLowerCase() === 'g' && e.shiftKey) {
    e.preventDefault()
    router.push('/graph')
  }
}
</script>

<template>
  <div
    class="flex flex-col h-screen bg-nord0 text-nord4 overflow-hidden"
    @keydown="handleKeydown"
    tabindex="0"
  >
    <div class="flex flex-1 overflow-hidden">
      <!-- Sidebar -->
      <aside
        v-if="uiStore.sidebarOpen"
        class="w-[250px] min-w-[250px] border-r border-nord2 flex flex-col bg-nord1 overflow-hidden transition-all duration-200"
      >
        <NoteSidebar />
      </aside>

      <!-- Editor -->
      <main class="flex-1 flex flex-col overflow-hidden">
        <NoteEditor />
      </main>

      <!-- Right panels -->
      <aside
        v-if="uiStore.previewOpen || uiStore.backlinksOpen || uiStore.outlineOpen"
        class="w-[350px] min-w-[350px] border-l border-nord2 flex flex-col bg-nord0 overflow-hidden transition-all duration-200"
      >
        <!-- Panel tabs -->
        <div class="flex border-b border-nord2 bg-nord1">
          <button
            v-if="uiStore.previewOpen"
            @click="uiStore.activeRightPanel = 'preview'"
            class="px-3 py-1.5 text-xs transition-colors"
            :class="uiStore.activeRightPanel === 'preview' ? 'text-nord8 border-b border-nord8' : 'text-nord3 hover:text-nord4'"
          >
            Preview
          </button>
          <button
            v-if="uiStore.backlinksOpen"
            @click="uiStore.activeRightPanel = 'backlinks'"
            class="px-3 py-1.5 text-xs transition-colors"
            :class="uiStore.activeRightPanel === 'backlinks' ? 'text-nord8 border-b border-nord8' : 'text-nord3 hover:text-nord4'"
          >
            Backlinks
          </button>
          <button
            v-if="uiStore.outlineOpen"
            @click="uiStore.activeRightPanel = 'outline'"
            class="px-3 py-1.5 text-xs transition-colors"
            :class="uiStore.activeRightPanel === 'outline' ? 'text-nord8 border-b border-nord8' : 'text-nord3 hover:text-nord4'"
          >
            Outline
          </button>
        </div>

        <!-- Panel content -->
        <div class="flex-1 overflow-hidden">
          <MarkdownPreview v-if="uiStore.activeRightPanel === 'preview'" />
          <BacklinksPanel v-else-if="uiStore.activeRightPanel === 'backlinks'" />
          <OutlinePanel v-else-if="uiStore.activeRightPanel === 'outline'" />
        </div>
      </aside>
    </div>

    <!-- Status bar -->
    <StatusBar />

    <!-- Modals -->
    <FuzzyFinder v-if="uiStore.fuzzyFinderOpen" />
    <CommandPalette v-if="uiStore.commandPaletteOpen" />
    <ToastContainer />
  </div>
</template>
