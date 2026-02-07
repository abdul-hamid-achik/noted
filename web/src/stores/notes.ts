import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useApi } from '../composables/useApi'
import type { Note, NoteCreateRequest, NoteUpdateRequest, Tag, Memory, Folder, FolderCreateRequest, FolderUpdateRequest } from '../types'

export const useNotesStore = defineStore('notes', () => {
  const api = useApi()

  // State
  const notes = ref<Note[]>([])
  const currentNote = ref<Note | null>(null)
  const tags = ref<Tag[]>([])
  const memories = ref<Memory[]>([])
  const folders = ref<Folder[]>([])
  const currentFolder = ref<number | null>(null) // null = All Notes
  const searchResults = ref<Note[]>([])
  const loading = ref(false)

  // Getters
  const sortedNotes = computed(() => {
    return [...notes.value].sort(
      (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    )
  })

  const recentNotes = computed(() => sortedNotes.value.slice(0, 10))

  const notesInCurrentFolder = computed(() => {
    if (currentFolder.value === null) return sortedNotes.value
    return sortedNotes.value.filter((n) => n.folder_id === currentFolder.value)
  })

  const folderTree = computed(() => {
    // Build a tree structure from flat folders list
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

    // Attach children
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
      notes.value[idx] = updated
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

  async function deleteTag(id: number) {
    await api.deleteTag(id)
    tags.value = tags.value.filter((t) => t.id !== id)
    // Remove this tag from all notes in local state
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
      notes.value[idx] = {
        ...notes.value[idx],
        tags: notes.value[idx].tags.filter((t) => t.id !== tagId),
      }
    }
    if (currentNote.value?.id === noteId) {
      currentNote.value = {
        ...currentNote.value,
        tags: currentNote.value.tags.filter((t) => t.id !== tagId),
      }
    }
    // Refresh tags to update counts
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

  async function fetchMemories() {
    memories.value = await api.getMemories()
  }

  async function recallMemories(query: string, limit = 10, category?: string) {
    loading.value = true
    try {
      memories.value = await api.recallMemories(query, limit, category)
    } finally {
      loading.value = false
    }
  }

  // Folder actions
  async function fetchFolders() {
    try {
      folders.value = await api.getFolders()
    } catch {
      // Folders API may not be available yet
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
      folders.value[idx] = updated
    }
    return updated
  }

  async function deleteFolder(id: number) {
    await api.deleteFolder(id)
    folders.value = folders.value.filter((f) => f.id !== id)
    if (currentFolder.value === id) {
      currentFolder.value = null
    }
    // Notes in deleted folder move to root (server handles this)
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
      notes.value[idx] = updated
    }
    if (currentNote.value?.id === noteId) {
      currentNote.value = updated
    }
    // Refresh folders to update note counts
    await fetchFolders()
    return updated
  }

  function selectFolder(folderId: number | null) {
    currentFolder.value = folderId
  }

  function selectNote(note: Note | null) {
    currentNote.value = note
  }

  function handleWsEvent(type: string, data: unknown) {
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
        if (idx !== -1) notes.value[idx] = note
        if (currentNote.value?.id === note.id) currentNote.value = note
        break
      }
      case 'note_deleted': {
        const { id } = data as { id: number }
        notes.value = notes.value.filter((n) => n.id !== id)
        if (currentNote.value?.id === id) currentNote.value = null
        break
      }
    }
  }

  return {
    notes,
    currentNote,
    tags,
    memories,
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
    deleteTag,
    removeTagFromNote,
    searchNotes,
    fetchTags,
    fetchMemories,
    recallMemories,
    fetchFolders,
    createFolder,
    updateFolder,
    deleteFolder,
    moveNoteToFolder,
    selectFolder,
    selectNote,
    handleWsEvent,
  }
})
