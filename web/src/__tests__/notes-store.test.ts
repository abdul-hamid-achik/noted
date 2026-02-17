import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useNotesStore } from '../stores/notes'
import type { Note, Tag, Folder } from '../types'

// Mock the API composable
const mockApi = {
  getNotes: vi.fn(),
  getNote: vi.fn(),
  createNote: vi.fn(),
  updateNote: vi.fn(),
  deleteNote: vi.fn(),
  searchNotes: vi.fn(),
  pinNote: vi.fn(),
  unpinNote: vi.fn(),
  getBacklinks: vi.fn(),
  getGraph: vi.fn(),
  getTags: vi.fn(),
  deleteTag: vi.fn(),
  removeTagFromNote: vi.fn(),
  getFolders: vi.fn(),
  createFolder: vi.fn(),
  updateFolder: vi.fn(),
  deleteFolder: vi.fn(),
  moveNoteToFolder: vi.fn(),
  getStats: vi.fn(),
  getSettings: vi.fn(),
  vacuumDB: vi.fn(),
  walCheckpoint: vi.fn(),
}

vi.mock('../composables/useApi', () => ({
  useApi: () => mockApi,
}))

function makeNote(overrides: Partial<Note> = {}): Note {
  return {
    id: 1,
    title: 'Test Note',
    content: '# Hello',
    tags: [],
    folder_id: null,
    pinned: false,
    pinned_at: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function makeTag(overrides: Partial<Tag> = {}): Tag {
  return { id: 1, name: 'test-tag', ...overrides }
}

function makeFolder(overrides: Partial<Folder> = {}): Folder {
  return {
    id: 1,
    name: 'Test Folder',
    parent_id: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('useNotesStore', () => {
  let store: ReturnType<typeof useNotesStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    store = useNotesStore()
    vi.clearAllMocks()
  })

  describe('fetchNotes', () => {
    it('loads notes from API', async () => {
      const notes = [makeNote({ id: 1 }), makeNote({ id: 2, title: 'Second' })]
      mockApi.getNotes.mockResolvedValue(notes)

      await store.fetchNotes()

      expect(mockApi.getNotes).toHaveBeenCalledOnce()
      expect(store.notes).toEqual(notes)
      expect(store.loading).toBe(false)
    })

    it('sets loading to false even on error', async () => {
      mockApi.getNotes.mockRejectedValue(new Error('fail'))

      await expect(store.fetchNotes()).rejects.toThrow('fail')
      expect(store.loading).toBe(false)
    })
  })

  describe('fetchNote', () => {
    it('sets currentNote from API', async () => {
      const note = makeNote({ id: 5 })
      mockApi.getNote.mockResolvedValue(note)

      await store.fetchNote(5)

      expect(mockApi.getNote).toHaveBeenCalledWith(5)
      expect(store.currentNote).toEqual(note)
    })
  })

  describe('createNote', () => {
    it('adds note to list and sets as current', async () => {
      const note = makeNote({ id: 10 })
      mockApi.createNote.mockResolvedValue(note)

      const result = await store.createNote({ title: 'New', content: 'body' })

      expect(result).toEqual(note)
      expect(store.notes).toContainEqual(note)
      expect(store.currentNote).toEqual(note)
    })
  })

  describe('updateNote', () => {
    it('updates note in list and currentNote', async () => {
      const original = makeNote({ id: 1, title: 'Old' })
      store.notes = [original]
      store.currentNote = original

      const updated = makeNote({ id: 1, title: 'New' })
      mockApi.updateNote.mockResolvedValue(updated)

      await store.updateNote(1, { title: 'New' })

      expect(store.notes[0].title).toBe('New')
      expect(store.currentNote?.title).toBe('New')
    })

    it('does not crash when updating a note not in list', async () => {
      const updated = makeNote({ id: 99, title: 'New' })
      mockApi.updateNote.mockResolvedValue(updated)

      await store.updateNote(99, { title: 'New' })
      // Should not throw
    })
  })

  describe('deleteNote', () => {
    it('removes note from list', async () => {
      store.notes = [makeNote({ id: 1 }), makeNote({ id: 2 })]
      store.currentNote = makeNote({ id: 1 })
      mockApi.deleteNote.mockResolvedValue(undefined)

      await store.deleteNote(1)

      expect(store.notes).toHaveLength(1)
      expect(store.notes[0].id).toBe(2)
      expect(store.currentNote).toBeNull()
    })

    it('keeps currentNote if different note deleted', async () => {
      store.notes = [makeNote({ id: 1 }), makeNote({ id: 2 })]
      store.currentNote = makeNote({ id: 2 })
      mockApi.deleteNote.mockResolvedValue(undefined)

      await store.deleteNote(1)

      expect(store.currentNote?.id).toBe(2)
    })
  })

  describe('pinNote / unpinNote', () => {
    it('updates pinned status in list', async () => {
      store.notes = [makeNote({ id: 1, pinned: false })]
      const pinned = makeNote({ id: 1, pinned: true, pinned_at: '2026-01-01' })
      mockApi.pinNote.mockResolvedValue(pinned)

      await store.pinNote(1)

      expect(store.notes[0].pinned).toBe(true)
    })

    it('unpins note', async () => {
      store.notes = [makeNote({ id: 1, pinned: true })]
      const unpinned = makeNote({ id: 1, pinned: false, pinned_at: null })
      mockApi.unpinNote.mockResolvedValue(unpinned)

      await store.unpinNote(1)

      expect(store.notes[0].pinned).toBe(false)
    })
  })

  describe('sortedNotes', () => {
    it('sorts by updated_at desc, pinned first', () => {
      store.notes = [
        makeNote({ id: 1, updated_at: '2026-01-01T00:00:00Z', pinned: false }),
        makeNote({ id: 2, updated_at: '2026-01-03T00:00:00Z', pinned: false }),
        makeNote({ id: 3, updated_at: '2026-01-02T00:00:00Z', pinned: true }),
      ]

      const sorted = store.sortedNotes
      expect(sorted[0].id).toBe(3) // pinned first
      expect(sorted[1].id).toBe(2) // then newest
      expect(sorted[2].id).toBe(1) // then oldest
    })
  })

  describe('recentNotes', () => {
    it('returns at most 10 notes', () => {
      store.notes = Array.from({ length: 15 }, (_, i) =>
        makeNote({ id: i + 1, updated_at: `2026-01-${String(i + 1).padStart(2, '0')}T00:00:00Z` })
      )

      expect(store.recentNotes).toHaveLength(10)
    })
  })

  describe('notesInCurrentFolder', () => {
    it('returns all notes when no folder selected', () => {
      store.notes = [makeNote({ id: 1, folder_id: 1 }), makeNote({ id: 2, folder_id: null })]
      store.currentFolder = null

      expect(store.notesInCurrentFolder).toHaveLength(2)
    })

    it('filters by selected folder', () => {
      store.notes = [makeNote({ id: 1, folder_id: 1 }), makeNote({ id: 2, folder_id: 2 })]
      store.currentFolder = 1

      expect(store.notesInCurrentFolder).toHaveLength(1)
      expect(store.notesInCurrentFolder[0].id).toBe(1)
    })
  })

  describe('folderTree', () => {
    it('builds nested folder structure', () => {
      store.folders = [
        makeFolder({ id: 1, name: 'Root', parent_id: null }),
        makeFolder({ id: 2, name: 'Child', parent_id: 1 }),
        makeFolder({ id: 3, name: 'Sibling', parent_id: null }),
      ]

      const tree = store.folderTree
      expect(tree).toHaveLength(2) // Root and Sibling
      const root = tree.find((f) => f.id === 1)
      expect(root?.children).toHaveLength(1)
      expect(root?.children[0].id).toBe(2)
    })
  })

  describe('notesByTag', () => {
    it('groups notes by tag name', () => {
      const tag1 = makeTag({ id: 1, name: 'go' })
      const tag2 = makeTag({ id: 2, name: 'vue' })
      store.notes = [
        makeNote({ id: 1, tags: [tag1, tag2] }),
        makeNote({ id: 2, tags: [tag1] }),
      ]

      const map = store.notesByTag
      expect(map.get('go')).toHaveLength(2)
      expect(map.get('vue')).toHaveLength(1)
    })
  })

  describe('tags', () => {
    it('fetches tags', async () => {
      mockApi.getTags.mockResolvedValue([makeTag()])

      await store.fetchTags()

      expect(store.tags).toHaveLength(1)
    })

    it('deleteTag removes tag from all notes', async () => {
      const tag = makeTag({ id: 5, name: 'remove-me' })
      store.tags = [tag]
      store.notes = [makeNote({ id: 1, tags: [tag] })]
      store.currentNote = makeNote({ id: 1, tags: [tag] })
      mockApi.deleteTag.mockResolvedValue(undefined)

      await store.deleteTag(5)

      expect(store.tags).toHaveLength(0)
      expect(store.notes[0].tags).toHaveLength(0)
      expect(store.currentNote?.tags).toHaveLength(0)
    })
  })

  describe('folders', () => {
    it('fetches folders', async () => {
      mockApi.getFolders.mockResolvedValue([makeFolder()])

      await store.fetchFolders()

      expect(store.folders).toHaveLength(1)
    })

    it('createFolder appends to list', async () => {
      const folder = makeFolder({ id: 3 })
      mockApi.createFolder.mockResolvedValue(folder)

      await store.createFolder({ name: 'New' })

      expect(store.folders).toContainEqual(folder)
    })

    it('deleteFolder resets currentFolder and clears folder_id from notes', async () => {
      store.folders = [makeFolder({ id: 1 })]
      store.currentFolder = 1
      store.notes = [makeNote({ id: 1, folder_id: 1 })]
      mockApi.deleteFolder.mockResolvedValue(undefined)

      await store.deleteFolder(1)

      expect(store.currentFolder).toBeNull()
      expect(store.notes[0].folder_id).toBeNull()
      expect(store.folders).toHaveLength(0)
    })

    it('moveNoteToFolder updates note and refreshes folders', async () => {
      const moved = makeNote({ id: 1, folder_id: 2 })
      store.notes = [makeNote({ id: 1, folder_id: null })]
      store.currentNote = makeNote({ id: 1, folder_id: null })
      mockApi.moveNoteToFolder.mockResolvedValue(moved)
      mockApi.getFolders.mockResolvedValue([])

      await store.moveNoteToFolder(1, 2)

      expect(store.notes[0].folder_id).toBe(2)
      expect(store.currentNote?.folder_id).toBe(2)
      expect(mockApi.getFolders).toHaveBeenCalled()
    })
  })

  describe('search', () => {
    it('populates searchResults', async () => {
      const results = [makeNote({ id: 1, title: 'Found' })]
      mockApi.searchNotes.mockResolvedValue(results)

      await store.searchNotes('Found')

      expect(store.searchResults).toEqual(results)
      expect(store.loading).toBe(false)
    })
  })

  describe('handleSSEEvent', () => {
    it('handles note_created', () => {
      const note = makeNote({ id: 5 })
      store.handleSSEEvent('note_created', note)
      expect(store.notes).toContainEqual(note)
    })

    it('ignores duplicate note_created', () => {
      const note = makeNote({ id: 5 })
      store.notes = [note]
      store.handleSSEEvent('note_created', note)
      expect(store.notes).toHaveLength(1)
    })

    it('handles note_updated', () => {
      store.notes = [makeNote({ id: 1, title: 'Old' })]
      store.currentNote = makeNote({ id: 1, title: 'Old' })

      store.handleSSEEvent('note_updated', makeNote({ id: 1, title: 'New' }))

      expect(store.notes[0].title).toBe('New')
      expect(store.currentNote?.title).toBe('New')
    })

    it('handles note_deleted', () => {
      store.notes = [makeNote({ id: 1 })]
      store.currentNote = makeNote({ id: 1 })

      store.handleSSEEvent('note_deleted', { id: 1 })

      expect(store.notes).toHaveLength(0)
      expect(store.currentNote).toBeNull()
    })

    it('handles folder_created', () => {
      const folder = makeFolder({ id: 3 })
      store.handleSSEEvent('folder_created', folder)
      expect(store.folders).toContainEqual(folder)
    })

    it('handles folder_updated', () => {
      store.folders = [makeFolder({ id: 1, name: 'Old' })]
      store.handleSSEEvent('folder_updated', makeFolder({ id: 1, name: 'New' }))
      expect(store.folders[0].name).toBe('New')
    })

    it('handles folder_deleted', () => {
      store.folders = [makeFolder({ id: 1 })]
      store.handleSSEEvent('folder_deleted', { id: 1 })
      expect(store.folders).toHaveLength(0)
    })
  })

  describe('selectFolder / selectNote', () => {
    it('selectFolder sets currentFolder', () => {
      store.selectFolder(3)
      expect(store.currentFolder).toBe(3)
    })

    it('selectNote sets currentNote', () => {
      const note = makeNote({ id: 7 })
      store.selectNote(note)
      expect(store.currentNote).toEqual(note)
    })

    it('selectNote can set null', () => {
      store.currentNote = makeNote()
      store.selectNote(null)
      expect(store.currentNote).toBeNull()
    })
  })
})
