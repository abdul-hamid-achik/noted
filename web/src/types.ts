export interface Note {
  id: number
  title: string
  content: string
  tags: Tag[]
  folder_id?: number | null
  created_at: string
  updated_at: string
  expires_at?: string
  source?: string
  source_ref?: string
  embedding_synced?: boolean
}

export interface Folder {
  id: number
  name: string
  parent_id: number | null
  note_count?: number
  created_at: string
  updated_at: string
}

export interface FolderCreateRequest {
  name: string
  parent_id?: number | null
}

export interface FolderUpdateRequest {
  name?: string
  parent_id?: number | null
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
  folder_id?: number | null
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

export interface SettingsInfo {
  db_path: string
  db_size_bytes: number
  db_size: string
  journal_mode: string
  sqlite_version: string
  total_notes: number
  total_tags: number
  total_memories: number
  app_version: string
  go_version: string
  platform: string
  uptime_seconds: number
  search_indexed: number
  search_pending: number
}

export interface ActionResult {
  success: boolean
  message: string
}

export interface WebSocketEvent {
  type: 'note_created' | 'note_updated' | 'note_deleted' | 'memory_created' | 'memory_deleted'
  data: unknown
}
