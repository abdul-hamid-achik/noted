import { defineStore } from 'pinia'
import { ref } from 'vue'

export type VimMode = 'normal' | 'insert' | 'visual' | 'command'

export const useUiStore = defineStore('ui', () => {
  const sidebarOpen = ref(true)
  const previewOpen = ref(false)
  const currentView = ref<'editor' | 'dashboard'>('editor')
  const vimMode = ref<VimMode>('normal')
  const cursorLine = ref(1)
  const cursorCol = ref(1)
  const fuzzyFinderOpen = ref(false)

  function toggleSidebar() {
    sidebarOpen.value = !sidebarOpen.value
  }

  function togglePreview() {
    previewOpen.value = !previewOpen.value
  }

  function setVimMode(mode: VimMode) {
    vimMode.value = mode
  }

  function openFuzzyFinder() {
    fuzzyFinderOpen.value = true
  }

  function closeFuzzyFinder() {
    fuzzyFinderOpen.value = false
  }

  return {
    sidebarOpen,
    previewOpen,
    currentView,
    vimMode,
    cursorLine,
    cursorCol,
    fuzzyFinderOpen,
    toggleSidebar,
    togglePreview,
    setVimMode,
    openFuzzyFinder,
    closeFuzzyFinder,
  }
})
