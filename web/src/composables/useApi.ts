import type {
  Note,
  NoteCreateRequest,
  NoteUpdateRequest,
  Tag,
  Stats,
  SettingsInfo,
  ActionResult,
  Folder,
  FolderCreateRequest,
  FolderUpdateRequest,
  GraphData,
} from '../types'

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

  async function pinNote(id: number): Promise<Note> {
    return request<Note>(`/notes/${id}/pin`, { method: 'POST' })
  }

  async function unpinNote(id: number): Promise<Note> {
    return request<Note>(`/notes/${id}/pin`, { method: 'DELETE' })
  }

  // Backlinks
  async function getBacklinks(id: number): Promise<Note[]> {
    return request<Note[]>(`/notes/${id}/backlinks`)
  }

  // Graph
  async function getGraph(): Promise<GraphData> {
    return request<GraphData>('/graph')
  }

  // Tags
  async function getTags(): Promise<Tag[]> {
    return request<Tag[]>('/tags')
  }

  async function deleteTag(id: number): Promise<void> {
    return request<void>(`/tags/${id}`, {
      method: 'DELETE',
    })
  }

  async function removeTagFromNote(tagId: number, noteId: number): Promise<void> {
    return request<void>(`/tags/${tagId}/notes/${noteId}`, {
      method: 'DELETE',
    })
  }

  // Folders
  async function getFolders(): Promise<Folder[]> {
    return request<Folder[]>('/folders')
  }

  async function createFolder(data: FolderCreateRequest): Promise<Folder> {
    return request<Folder>('/folders', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async function updateFolder(id: number, data: FolderUpdateRequest): Promise<Folder> {
    return request<Folder>(`/folders/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async function deleteFolder(id: number): Promise<void> {
    return request<void>(`/folders/${id}`, {
      method: 'DELETE',
    })
  }

  async function moveNoteToFolder(noteId: number, folderId: number | null): Promise<Note> {
    return request<Note>(`/notes/${noteId}/move`, {
      method: 'PUT',
      body: JSON.stringify({ folder_id: folderId }),
    })
  }

  // Stats
  async function getStats(): Promise<Stats> {
    return request<Stats>('/stats')
  }

  // Settings
  async function getSettings(): Promise<SettingsInfo> {
    return request<SettingsInfo>('/settings')
  }

  async function vacuumDB(): Promise<ActionResult> {
    return request<ActionResult>('/settings/vacuum', { method: 'POST' })
  }

  async function walCheckpoint(): Promise<ActionResult> {
    return request<ActionResult>('/settings/wal-checkpoint', { method: 'POST' })
  }

  return {
    getNotes,
    getNote,
    createNote,
    updateNote,
    deleteNote,
    searchNotes,
    pinNote,
    unpinNote,
    getBacklinks,
    getGraph,
    getTags,
    deleteTag,
    removeTagFromNote,
    getFolders,
    createFolder,
    updateFolder,
    deleteFolder,
    moveNoteToFolder,
    getStats,
    getSettings,
    vacuumDB,
    walCheckpoint,
  }
}
