import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type VimMode = 'normal' | 'insert' | 'visual' | 'command'
export type RightPanel = 'preview' | 'backlinks' | 'outline'

function loadUiState() {
  try {
    const raw = localStorage.getItem('noted-ui-state')
    if (raw) return JSON.parse(raw)
  } catch { /* ignore */ }
  return null
}

export const useUiStore = defineStore('ui', () => {
  const saved = loadUiState()

  const sidebarOpen = ref(saved?.sidebarOpen ?? true)
  const previewOpen = ref(saved?.previewOpen ?? false)
  const backlinksOpen = ref(saved?.backlinksOpen ?? false)
  const outlineOpen = ref(saved?.outlineOpen ?? false)
  const activeRightPanel = ref<RightPanel>(saved?.activeRightPanel ?? 'preview')
  const currentView = ref<'editor' | 'dashboard'>('editor')
  const vimMode = ref<VimMode>('normal')
  const cursorLine = ref(1)
  const cursorCol = ref(1)
  const fuzzyFinderOpen = ref(false)
  const commandPaletteOpen = ref(false)

  function toggleSidebar() {
    sidebarOpen.value = !sidebarOpen.value
  }

  function togglePreview() {
    previewOpen.value = !previewOpen.value
    if (previewOpen.value) activeRightPanel.value = 'preview'
  }

  function toggleBacklinks() {
    backlinksOpen.value = !backlinksOpen.value
    if (backlinksOpen.value) activeRightPanel.value = 'backlinks'
  }

  function toggleOutline() {
    outlineOpen.value = !outlineOpen.value
    if (outlineOpen.value) activeRightPanel.value = 'outline'
  }

  function setVimMode(mode: VimMode) {
    vimMode.value = mode
  }

  function openFuzzyFinder() {
    fuzzyFinderOpen.value = true
    commandPaletteOpen.value = false
  }

  function closeFuzzyFinder() {
    fuzzyFinderOpen.value = false
  }

  function openCommandPalette() {
    commandPaletteOpen.value = true
    fuzzyFinderOpen.value = false
  }

  function closeCommandPalette() {
    commandPaletteOpen.value = false
  }

  // Persist UI state to localStorage
  watch(
    [sidebarOpen, previewOpen, backlinksOpen, outlineOpen, activeRightPanel],
    () => {
      localStorage.setItem('noted-ui-state', JSON.stringify({
        sidebarOpen: sidebarOpen.value,
        previewOpen: previewOpen.value,
        backlinksOpen: backlinksOpen.value,
        outlineOpen: outlineOpen.value,
        activeRightPanel: activeRightPanel.value,
      }))
    }
  )

  return {
    sidebarOpen,
    previewOpen,
    backlinksOpen,
    outlineOpen,
    activeRightPanel,
    currentView,
    vimMode,
    cursorLine,
    cursorCol,
    fuzzyFinderOpen,
    commandPaletteOpen,
    toggleSidebar,
    togglePreview,
    toggleBacklinks,
    toggleOutline,
    setVimMode,
    openFuzzyFinder,
    closeFuzzyFinder,
    openCommandPalette,
    closeCommandPalette,
  }
})
