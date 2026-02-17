import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useApi } from '../composables/useApi'

// Mock global fetch
const mockFetch = vi.fn()
globalThis.fetch = mockFetch

function jsonResponse(data: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  }
}

function emptyResponse(status = 204) {
  return {
    ok: true,
    status,
    json: () => Promise.resolve(undefined),
    text: () => Promise.resolve(''),
  }
}

describe('useApi', () => {
  let api: ReturnType<typeof useApi>

  beforeEach(() => {
    vi.clearAllMocks()
    api = useApi()
  })

  describe('getNotes', () => {
    it('fetches notes', async () => {
      mockFetch.mockResolvedValue(jsonResponse([{ id: 1, title: 'Test' }]))

      const notes = await api.getNotes()

      expect(mockFetch).toHaveBeenCalledWith('/api/notes', expect.objectContaining({
        headers: { 'Content-Type': 'application/json' },
      }))
      expect(notes).toEqual([{ id: 1, title: 'Test' }])
    })
  })

  describe('getNote', () => {
    it('fetches single note by id', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ id: 5, title: 'Note 5' }))

      const note = await api.getNote(5)

      expect(mockFetch).toHaveBeenCalledWith('/api/notes/5', expect.anything())
      expect(note.id).toBe(5)
    })
  })

  describe('createNote', () => {
    it('posts new note', async () => {
      const newNote = { id: 10, title: 'New', content: 'body' }
      mockFetch.mockResolvedValue(jsonResponse(newNote, 201))

      const result = await api.createNote({ title: 'New', content: 'body' })

      expect(mockFetch).toHaveBeenCalledWith('/api/notes', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ title: 'New', content: 'body' }),
      }))
      expect(result.id).toBe(10)
    })
  })

  describe('updateNote', () => {
    it('puts updated note', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ id: 1, title: 'Updated' }))

      await api.updateNote(1, { title: 'Updated' })

      expect(mockFetch).toHaveBeenCalledWith('/api/notes/1', expect.objectContaining({
        method: 'PUT',
      }))
    })
  })

  describe('deleteNote', () => {
    it('sends DELETE request', async () => {
      mockFetch.mockResolvedValue(emptyResponse())

      await api.deleteNote(1)

      expect(mockFetch).toHaveBeenCalledWith('/api/notes/1', expect.objectContaining({
        method: 'DELETE',
      }))
    })
  })

  describe('searchNotes', () => {
    it('passes query and limit as search params', async () => {
      mockFetch.mockResolvedValue(jsonResponse([]))

      await api.searchNotes('test', 10)

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/notes/search?'),
        expect.anything()
      )
      const url = mockFetch.mock.calls[0][0] as string
      expect(url).toContain('q=test')
      expect(url).toContain('limit=10')
    })
  })

  describe('pinNote / unpinNote', () => {
    it('pins a note', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ id: 1, pinned: true }))

      await api.pinNote(1)

      expect(mockFetch).toHaveBeenCalledWith('/api/notes/1/pin', expect.objectContaining({
        method: 'POST',
      }))
    })

    it('unpins a note', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ id: 1, pinned: false }))

      await api.unpinNote(1)

      expect(mockFetch).toHaveBeenCalledWith('/api/notes/1/pin', expect.objectContaining({
        method: 'DELETE',
      }))
    })
  })

  describe('getBacklinks', () => {
    it('fetches backlinks for a note', async () => {
      mockFetch.mockResolvedValue(jsonResponse([{ id: 2, title: 'Linker' }]))

      const result = await api.getBacklinks(1)

      expect(mockFetch).toHaveBeenCalledWith('/api/notes/1/backlinks', expect.anything())
      expect(result).toHaveLength(1)
    })
  })

  describe('tags', () => {
    it('fetches tags', async () => {
      mockFetch.mockResolvedValue(jsonResponse([{ id: 1, name: 'go' }]))

      const tags = await api.getTags()

      expect(tags).toHaveLength(1)
    })

    it('deletes a tag', async () => {
      mockFetch.mockResolvedValue(emptyResponse())

      await api.deleteTag(1)

      expect(mockFetch).toHaveBeenCalledWith('/api/tags/1', expect.objectContaining({
        method: 'DELETE',
      }))
    })

    it('removes tag from note', async () => {
      mockFetch.mockResolvedValue(emptyResponse())

      await api.removeTagFromNote(5, 10)

      expect(mockFetch).toHaveBeenCalledWith('/api/tags/5/notes/10', expect.objectContaining({
        method: 'DELETE',
      }))
    })
  })

  describe('folders', () => {
    it('fetches folders', async () => {
      mockFetch.mockResolvedValue(jsonResponse([{ id: 1, name: 'Folder' }]))

      const folders = await api.getFolders()

      expect(folders).toHaveLength(1)
    })

    it('creates a folder', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ id: 2, name: 'New' }, 201))

      await api.createFolder({ name: 'New' })

      expect(mockFetch).toHaveBeenCalledWith('/api/folders', expect.objectContaining({
        method: 'POST',
      }))
    })

    it('deletes a folder', async () => {
      mockFetch.mockResolvedValue(emptyResponse())

      await api.deleteFolder(1)

      expect(mockFetch).toHaveBeenCalledWith('/api/folders/1', expect.objectContaining({
        method: 'DELETE',
      }))
    })

    it('moves note to folder', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ id: 1, folder_id: 2 }))

      await api.moveNoteToFolder(1, 2)

      expect(mockFetch).toHaveBeenCalledWith('/api/notes/1/move', expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ folder_id: 2 }),
      }))
    })
  })

  describe('stats and settings', () => {
    it('fetches stats', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ total_notes: 5 }))

      const stats = await api.getStats()

      expect(stats.total_notes).toBe(5)
    })

    it('fetches settings', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ db: {}, runtime: {}, app: {} }))

      const settings = await api.getSettings()

      expect(settings).toHaveProperty('db')
    })

    it('vacuum calls POST', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ status: 'ok' }))

      await api.vacuumDB()

      expect(mockFetch).toHaveBeenCalledWith('/api/settings/vacuum', expect.objectContaining({
        method: 'POST',
      }))
    })

    it('wal checkpoint calls POST', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ status: 'ok' }))

      await api.walCheckpoint()

      expect(mockFetch).toHaveBeenCalledWith('/api/settings/wal-checkpoint', expect.objectContaining({
        method: 'POST',
      }))
    })
  })

  describe('error handling', () => {
    it('throws on non-ok response', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        text: () => Promise.resolve('not found'),
      })

      await expect(api.getNote(999)).rejects.toThrow('API error 404: not found')
    })
  })

  describe('graph', () => {
    it('fetches graph data', async () => {
      mockFetch.mockResolvedValue(jsonResponse({ nodes: [], edges: [] }))

      const graph = await api.getGraph()

      expect(graph.nodes).toEqual([])
      expect(graph.edges).toEqual([])
    })
  })
})
