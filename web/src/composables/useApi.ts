import type { Note, NoteCreateRequest, NoteUpdateRequest, Tag, Memory, MemoryCreateRequest, Stats } from '../types'

const BASE = '/api'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`API error ${res.status}: ${text}`)
  }
  if (res.status === 204) {
    return undefined as T
  }
  return res.json()
}

export function useApi() {
  // Notes
  async function getNotes(): Promise<Note[]> {
    return request<Note[]>('/notes')
  }

  async function getNote(id: number): Promise<Note> {
    return request<Note>(`/notes/${id}`)
  }

  async function createNote(data: NoteCreateRequest): Promise<Note> {
    return request<Note>('/notes', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async function updateNote(id: number, data: NoteUpdateRequest): Promise<Note> {
    return request<Note>(`/notes/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async function deleteNote(id: number): Promise<void> {
    return request<void>(`/notes/${id}`, {
      method: 'DELETE',
    })
  }

  async function searchNotes(q: string, limit = 20): Promise<Note[]> {
    const params = new URLSearchParams({ q, limit: String(limit) })
    return request<Note[]>(`/notes/search?${params}`)
  }

  // Memories
  async function getMemories(): Promise<Memory[]> {
    return request<Memory[]>('/memories')
  }

  async function createMemory(data: MemoryCreateRequest): Promise<Memory> {
    return request<Memory>('/memories', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async function deleteMemory(id: number): Promise<void> {
    return request<void>(`/memories/${id}`, {
      method: 'DELETE',
    })
  }

  async function recallMemories(q: string, limit = 10, category?: string): Promise<Memory[]> {
    const params = new URLSearchParams({ q, limit: String(limit) })
    if (category) params.set('category', category)
    return request<Memory[]>(`/memories/recall?${params}`)
  }

  // Tags
  async function getTags(): Promise<Tag[]> {
    return request<Tag[]>('/tags')
  }

  // Stats
  async function getStats(): Promise<Stats> {
    return request<Stats>('/stats')
  }

  return {
    getNotes,
    getNote,
    createNote,
    updateNote,
    deleteNote,
    searchNotes,
    getMemories,
    createMemory,
    deleteMemory,
    recallMemories,
    getTags,
    getStats,
  }
}
