import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useApi } from '../composables/useApi'
import type { Note, NoteCreateRequest, NoteUpdateRequest, Tag, Memory } from '../types'

export const useNotesStore = defineStore('notes', () => {
  const api = useApi()

  // State
  const notes = ref<Note[]>([])
  const currentNote = ref<Note | null>(null)
  const tags = ref<Tag[]>([])
  const memories = ref<Memory[]>([])
  const searchResults = ref<Note[]>([])
  const loading = ref(false)

  // Getters
  const sortedNotes = computed(() => {
    return [...notes.value].sort(
      (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    )
  })

  const recentNotes = computed(() => sortedNotes.value.slice(0, 10))

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
    searchResults,
    loading,
    sortedNotes,
    recentNotes,
    notesByTag,
    fetchNotes,
    fetchNote,
    createNote,
    updateNote,
    deleteNote,
    searchNotes,
    fetchTags,
    fetchMemories,
    recallMemories,
    selectNote,
    handleWsEvent,
  }
})
