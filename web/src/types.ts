export interface Note {
  id: number
  title: string
  content: string
  tags: Tag[]
  created_at: string
  updated_at: string
  expires_at?: string
  source?: string
  source_ref?: string
  embedding_synced?: boolean
}

export interface Tag {
  id: number
  name: string
  note_count?: number
}

export interface Memory {
  id: number
  title: string
  content: string
  category: string
  importance: number
  source?: string
  source_ref?: string
  expires_at?: string
  created_at: string
  updated_at: string
  score?: number
}

export interface Stats {
  total_notes: number
  total_memories: number
  total_tags: number
  db_size_bytes: number
  unsynced_notes: number
}

export interface NoteCreateRequest {
  title: string
  content: string
  tags?: string[]
}

export interface NoteUpdateRequest {
  title?: string
  content?: string
  tags?: string[]
}

export interface MemoryCreateRequest {
  title: string
  content: string
  category: string
  importance?: number
  source?: string
  source_ref?: string
  expires_at?: string
}

export interface WebSocketEvent {
  type: 'note_created' | 'note_updated' | 'note_deleted' | 'memory_created' | 'memory_deleted'
  data: unknown
}
