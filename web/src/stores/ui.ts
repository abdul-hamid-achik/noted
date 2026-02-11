import { defineStore } from 'pinia'
import { ref } from 'vue'

export type VimMode = 'normal' | 'insert' | 'visual' | 'command'

export const useUiStore = defineStore('ui', () => {
  const sidebarOpen = ref(true)
  const previewOpen = ref(false)
  const backlinksOpen = ref(false)
  const outlineOpen = ref(false)
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
  }

  function toggleBacklinks() {
    backlinksOpen.value = !backlinksOpen.value
  }

  function toggleOutline() {
    outlineOpen.value = !outlineOpen.value
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

  return {
    sidebarOpen,
    previewOpen,
    backlinksOpen,
    outlineOpen,
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
