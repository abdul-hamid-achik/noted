export interface Note {
  id: number
  title: string
  content: string
  tags: Tag[]
  folder_id?: number | null
  pinned?: boolean
  pinned_at?: string | null
  created_at: string
  updated_at: string
  expires_at?: string
  source?: string
  source_ref?: string
  embedding_synced?: boolean
}

export interface NoteLink {
  id: number
  source_note_id: number
  target_note_id: number
  link_text: string
  created_at: string
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

export interface Stats {
  total_notes: number
  total_tags: number
  unsynced_notes: number
  db_size_bytes: number
  db_size: string
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

export interface SettingsInfo {
  db: {
    journal_mode: string
    page_size: number
    cache_size: number
    busy_timeout: number
    foreign_keys: boolean
    wal_pages: number
  }
  runtime: {
    goos: string
    goarch: string
    num_goroutine: number
    num_cpu: number
    go_version: string
  }
  app: {
    version: string
  }
}

export interface ActionResult {
  status: string
  [key: string]: unknown
}

export interface GraphData {
  nodes: GraphNode[]
  edges: GraphEdge[]
}

export interface GraphNode {
  id: number
  title: string
  folder_id?: number | null
  link_count: number
}

export interface GraphEdge {
  source: number
  target: number
  label: string
}

export interface SSEEvent {
  type: string
  data: unknown
}
