import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useApi } from '../composables/useApi'
import type { Note, NoteCreateRequest, NoteUpdateRequest, Tag, Folder, FolderCreateRequest, FolderUpdateRequest } from '../types'

export const useNotesStore = defineStore('notes', () => {
  const api = useApi()

  // State
  const notes = ref<Note[]>([])
  const currentNote = ref<Note | null>(null)
  const tags = ref<Tag[]>([])
  const folders = ref<Folder[]>([])
  const currentFolder = ref<number | null>(null) // null = All Notes
  const searchResults = ref<Note[]>([])
  const loading = ref(false)

  // Getters
  const sortedNotes = computed(() => {
    const sorted = [...notes.value].sort(
      (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    )
    // Pinned notes first
    return sorted.sort((a, b) => {
      if (a.pinned && !b.pinned) return -1
      if (!a.pinned && b.pinned) return 1
      return 0
    })
  })

  const recentNotes = computed(() => sortedNotes.value.slice(0, 10))

  const notesInCurrentFolder = computed(() => {
    if (currentFolder.value === null) return sortedNotes.value
    return sortedNotes.value.filter((n) => n.folder_id === currentFolder.value)
  })

  const folderTree = computed(() => {
    type FolderNode = Folder & { children: FolderNode[] }
    const rootFolders: FolderNode[] = []
    const childMap = new Map<number, FolderNode[]>()

    for (const f of folders.value) {
      const node: FolderNode = { ...f, children: [] }
      if (f.parent_id === null) {
        rootFolders.push(node)
      } else {
        const siblings = childMap.get(f.parent_id) || []
        siblings.push(node)
        childMap.set(f.parent_id, siblings)
      }
    }

    function attachChildren(nodes: FolderNode[]) {
      for (const node of nodes) {
        node.children = childMap.get(node.id) || []
        attachChildren(node.children)
      }
    }
    attachChildren(rootFolders)
    return rootFolders
  })

  const notesByTag = computed(() => {
    const map = new Map<string, Note[]>()
    for (const note of notes.value) {
      for (const tag of note.tags) {
        const list = map.get(tag.name) || []
        list.push(note)
        map.set(tag.name, list)
      }
    }
    return map
  })

  // Actions
  async function fetchNotes() {
    loading.value = true
    try {
      notes.value = await api.getNotes()
    } finally {
      loading.value = false
    }
  }

  async function fetchNote(id: number) {
    loading.value = true
    try {
      currentNote.value = await api.getNote(id)
    } finally {
      loading.value = false
    }
  }

  async function createNote(data: NoteCreateRequest): Promise<Note> {
    const note = await api.createNote(data)
    notes.value.unshift(note)
    currentNote.value = note
    return note
  }

  async function updateNote(id: number, data: NoteUpdateRequest): Promise<Note> {
    const updated = await api.updateNote(id, data)
    const idx = notes.value.findIndex((n) => n.id === id)
    if (idx !== -1) {
      notes.value.splice(idx, 1, updated)
    }
    if (currentNote.value?.id === id) {
      currentNote.value = updated
    }
    return updated
  }

  async function deleteNote(id: number) {
    await api.deleteNote(id)
    notes.value = notes.value.filter((n) => n.id !== id)
    if (currentNote.value?.id === id) {
      currentNote.value = null
    }
  }

  async function pinNote(id: number) {
    const updated = await api.pinNote(id)
    const idx = notes.value.findIndex((n) => n.id === id)
    if (idx !== -1) {
      notes.value.splice(idx, 1, updated)
    }
    if (currentNote.value?.id === id) {
      currentNote.value = updated
    }
  }

  async function unpinNote(id: number) {
    const updated = await api.unpinNote(id)
    const idx = notes.value.findIndex((n) => n.id === id)
    if (idx !== -1) {
      notes.value.splice(idx, 1, updated)
    }
    if (currentNote.value?.id === id) {
      currentNote.value = updated
    }
  }

  async function deleteTag(id: number) {
    await api.deleteTag(id)
    tags.value = tags.value.filter((t) => t.id !== id)
    for (const note of notes.value) {
      note.tags = note.tags.filter((t) => t.id !== id)
    }
    if (currentNote.value) {
      currentNote.value = {
        ...currentNote.value,
        tags: currentNote.value.tags.filter((t) => t.id !== id),
      }
    }
  }

  async function removeTagFromNote(noteId: number, tagId: number) {
    await api.removeTagFromNote(tagId, noteId)
    const idx = notes.value.findIndex((n) => n.id === noteId)
    if (idx !== -1) {
      notes.value.splice(idx, 1, {
        ...notes.value[idx],
        tags: notes.value[idx].tags.filter((t) => t.id !== tagId),
      })
    }
    if (currentNote.value?.id === noteId) {
      currentNote.value = {
        ...currentNote.value,
        tags: currentNote.value.tags.filter((t) => t.id !== tagId),
      }
    }
    await fetchTags()
  }

  async function searchNotes(query: string, limit = 20) {
    loading.value = true
    try {
      searchResults.value = await api.searchNotes(query, limit)
    } finally {
      loading.value = false
    }
  }

  async function fetchTags() {
    tags.value = await api.getTags()
  }

  // Folder actions
  async function fetchFolders() {
    try {
      folders.value = await api.getFolders()
    } catch {
      folders.value = []
    }
  }

  async function createFolder(data: FolderCreateRequest): Promise<Folder> {
    const folder = await api.createFolder(data)
    folders.value.push(folder)
    return folder
  }

  async function updateFolder(id: number, data: FolderUpdateRequest): Promise<Folder> {
    const updated = await api.updateFolder(id, data)
    const idx = folders.value.findIndex((f) => f.id === id)
    if (idx !== -1) {
      folders.value.splice(idx, 1, updated)
    }
    return updated
  }

  async function deleteFolder(id: number) {
    await api.deleteFolder(id)
    folders.value = folders.value.filter((f) => f.id !== id)
    if (currentFolder.value === id) {
      currentFolder.value = null
    }
    for (const note of notes.value) {
      if (note.folder_id === id) {
        note.folder_id = null
      }
    }
  }

  async function moveNoteToFolder(noteId: number, folderId: number | null): Promise<Note> {
    const updated = await api.moveNoteToFolder(noteId, folderId)
    const idx = notes.value.findIndex((n) => n.id === noteId)
    if (idx !== -1) {
      notes.value.splice(idx, 1, updated)
    }
    if (currentNote.value?.id === noteId) {
      currentNote.value = updated
    }
    await fetchFolders()
    return updated
  }

  function selectFolder(folderId: number | null) {
    currentFolder.value = folderId
  }

  function selectNote(note: Note | null) {
    currentNote.value = note
  }

  function handleSSEEvent(type: string, data: unknown) {
    switch (type) {
      case 'note_created': {
        const note = data as Note
        const exists = notes.value.find((n) => n.id === note.id)
        if (!exists) notes.value.unshift(note)
        break
      }
      case 'note_updated': {
        const note = data as Note
        const idx = notes.value.findIndex((n) => n.id === note.id)
        if (idx !== -1) notes.value.splice(idx, 1, note)
        if (currentNote.value?.id === note.id) currentNote.value = note
        break
      }
      case 'note_deleted': {
        const { id } = data as { id: number }
        notes.value = notes.value.filter((n) => n.id !== id)
        if (currentNote.value?.id === id) currentNote.value = null
        break
      }
      case 'folder_created': {
        const folder = data as Folder
        const exists = folders.value.find((f) => f.id === folder.id)
        if (!exists) folders.value.push(folder)
        break
      }
      case 'folder_updated': {
        const folder = data as Folder
        const idx = folders.value.findIndex((f) => f.id === folder.id)
        if (idx !== -1) folders.value.splice(idx, 1, folder)
        break
      }
      case 'folder_deleted': {
        const { id } = data as { id: number }
        folders.value = folders.value.filter((f) => f.id !== id)
        break
      }
    }
  }

  return {
    notes,
    currentNote,
    tags,
    folders,
    currentFolder,
    searchResults,
    loading,
    sortedNotes,
    recentNotes,
    notesInCurrentFolder,
    folderTree,
    notesByTag,
    fetchNotes,
    fetchNote,
    createNote,
    updateNote,
    deleteNote,
    pinNote,
    unpinNote,
    deleteTag,
    removeTagFromNote,
    searchNotes,
    fetchTags,
    fetchFolders,
    createFolder,
    updateFolder,
    deleteFolder,
    moveNoteToFolder,
    selectFolder,
    selectNote,
    handleSSEEvent,
  }
})
