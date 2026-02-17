import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUiStore } from '../stores/ui'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => { store[key] = value }),
    removeItem: vi.fn((key: string) => { delete store[key] }),
    clear: vi.fn(() => { store = {} }),
  }
})()

Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock })

describe('useUiStore', () => {
  let store: ReturnType<typeof useUiStore>

  beforeEach(() => {
    localStorageMock.clear()
    setActivePinia(createPinia())
    store = useUiStore()
  })

  describe('initial state', () => {
    it('has default values', () => {
      expect(store.sidebarOpen).toBe(true)
      expect(store.previewOpen).toBe(false)
      expect(store.backlinksOpen).toBe(false)
      expect(store.outlineOpen).toBe(false)
      expect(store.currentView).toBe('editor')
      expect(store.vimMode).toBe('normal')
      expect(store.fuzzyFinderOpen).toBe(false)
      expect(store.commandPaletteOpen).toBe(false)
    })
  })

  describe('toggleSidebar', () => {
    it('toggles sidebar state', () => {
      expect(store.sidebarOpen).toBe(true)
      store.toggleSidebar()
      expect(store.sidebarOpen).toBe(false)
      store.toggleSidebar()
      expect(store.sidebarOpen).toBe(true)
    })
  })

  describe('togglePreview', () => {
    it('toggles preview and sets active panel', () => {
      store.togglePreview()
      expect(store.previewOpen).toBe(true)
      expect(store.activeRightPanel).toBe('preview')
    })
  })

  describe('toggleBacklinks', () => {
    it('toggles backlinks and sets active panel', () => {
      store.toggleBacklinks()
      expect(store.backlinksOpen).toBe(true)
      expect(store.activeRightPanel).toBe('backlinks')
    })
  })

  describe('toggleOutline', () => {
    it('toggles outline and sets active panel', () => {
      store.toggleOutline()
      expect(store.outlineOpen).toBe(true)
      expect(store.activeRightPanel).toBe('outline')
    })
  })

  describe('vim mode', () => {
    it('sets vim mode', () => {
      store.setVimMode('insert')
      expect(store.vimMode).toBe('insert')
      store.setVimMode('visual')
      expect(store.vimMode).toBe('visual')
      store.setVimMode('command')
      expect(store.vimMode).toBe('command')
      store.setVimMode('normal')
      expect(store.vimMode).toBe('normal')
    })
  })

  describe('fuzzy finder', () => {
    it('opens and closes command palette on fuzzy open', () => {
      store.commandPaletteOpen = true
      store.openFuzzyFinder()
      expect(store.fuzzyFinderOpen).toBe(true)
      expect(store.commandPaletteOpen).toBe(false)
    })

    it('closes fuzzy finder', () => {
      store.fuzzyFinderOpen = true
      store.closeFuzzyFinder()
      expect(store.fuzzyFinderOpen).toBe(false)
    })
  })

  describe('command palette', () => {
    it('opens and closes fuzzy finder on palette open', () => {
      store.fuzzyFinderOpen = true
      store.openCommandPalette()
      expect(store.commandPaletteOpen).toBe(true)
      expect(store.fuzzyFinderOpen).toBe(false)
    })

    it('closes command palette', () => {
      store.commandPaletteOpen = true
      store.closeCommandPalette()
      expect(store.commandPaletteOpen).toBe(false)
    })
  })

  describe('cursor tracking', () => {
    it('tracks cursor position', () => {
      store.cursorLine = 10
      store.cursorCol = 25
      expect(store.cursorLine).toBe(10)
      expect(store.cursorCol).toBe(25)
    })
  })
})
