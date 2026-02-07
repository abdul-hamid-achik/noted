<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine, drawSelection } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { vim, Vim, getCM } from '@replit/codemirror-vim'
import { markdown } from '@codemirror/lang-markdown'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { syntaxHighlighting, HighlightStyle } from '@codemirror/language'
import { tags } from '@lezer/highlight'
import { useNotesStore } from '../stores/notes'
import { useUiStore } from '../stores/ui'
import { useRouter } from 'vue-router'

const notesStore = useNotesStore()
const uiStore = useUiStore()
const router = useRouter()
const editorContainer = ref<HTMLElement | null>(null)
const editorView = shallowRef<EditorView | null>(null)
const modified = ref(false)
const docCompartment = new Compartment()
const confirmingDelete = ref(false)
const confirmingTagRemove = ref<number | null>(null)

// Nord CodeMirror theme
const nordTheme = EditorView.theme({
  '&': {
    backgroundColor: '#2E3440',
    color: '#D8DEE9',
  },
  '.cm-content': {
    caretColor: '#D8DEE9',
    fontFamily: "'JetBrains Mono', monospace",
    fontSize: '14px',
    lineHeight: '1.6',
  },
  '.cm-cursor, .cm-dropCursor': {
    borderLeftColor: '#D8DEE9',
  },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: '#434C5E',
  },
  '.cm-activeLine': {
    backgroundColor: '#3B425220',
  },
  '.cm-gutters': {
    backgroundColor: '#2E3440',
    color: '#4C566A',
    border: 'none',
    borderRight: '1px solid #3B4252',
  },
  '.cm-activeLineGutter': {
    backgroundColor: '#3B425220',
    color: '#D8DEE9',
  },
  '.cm-lineNumbers .cm-gutterElement': {
    padding: '0 8px',
  },
  '.cm-matchingBracket': {
    backgroundColor: '#434C5E',
    color: '#88C0D0',
    outline: 'none',
  },
  '.cm-searchMatch': {
    backgroundColor: '#434C5E',
  },
  '.cm-searchMatch.cm-searchMatch-selected': {
    backgroundColor: '#5E81AC40',
  },
  '.cm-panels': {
    backgroundColor: '#3B4252',
    color: '#D8DEE9',
  },
  '.cm-panels.cm-panels-top': {
    borderBottom: '1px solid #434C5E',
  },
  '.cm-panels.cm-panels-bottom': {
    borderTop: '1px solid #434C5E',
  },
  // Vim status bar in CodeMirror
  '.cm-vim-panel': {
    backgroundColor: '#3B4252',
    color: '#D8DEE9',
    padding: '2px 8px',
    fontFamily: "'JetBrains Mono', monospace",
    fontSize: '13px',
  },
  '.cm-vim-panel input': {
    backgroundColor: '#3B4252',
    color: '#D8DEE9',
    border: 'none',
    outline: 'none',
    fontFamily: "'JetBrains Mono', monospace",
    fontSize: '13px',
  },
  // Normal mode: block cursor (nord9 blue)
  '.cm-fat-cursor': {
    backgroundColor: '#81A1C1cc !important',
    color: '#2E3440 !important',
  },
  '&:not(.cm-focused) .cm-fat-cursor': {
    backgroundColor: '#81A1C140 !important',
    outline: '1px solid #81A1C180',
  },
}, { dark: true })

// Nord syntax highlighting
const nordHighlightStyle = HighlightStyle.define([
  { tag: tags.keyword, color: '#81A1C1' },
  { tag: tags.operator, color: '#81A1C1' },
  { tag: tags.variableName, color: '#D8DEE9' },
  { tag: tags.function(tags.variableName), color: '#88C0D0' },
  { tag: tags.string, color: '#A3BE8C' },
  { tag: tags.number, color: '#B48EAD' },
  { tag: tags.bool, color: '#81A1C1' },
  { tag: tags.comment, color: '#4C566A', fontStyle: 'italic' },
  { tag: tags.lineComment, color: '#4C566A', fontStyle: 'italic' },
  { tag: tags.blockComment, color: '#4C566A', fontStyle: 'italic' },
  { tag: tags.meta, color: '#5E81AC' },
  { tag: tags.link, color: '#88C0D0', textDecoration: 'underline' },
  { tag: tags.url, color: '#88C0D0', textDecoration: 'underline' },
  { tag: tags.heading, color: '#8FBCBB', fontWeight: 'bold' },
  { tag: tags.heading1, color: '#8FBCBB', fontWeight: 'bold', fontSize: '1.3em' },
  { tag: tags.heading2, color: '#88C0D0', fontWeight: 'bold', fontSize: '1.2em' },
  { tag: tags.heading3, color: '#81A1C1', fontWeight: 'bold', fontSize: '1.1em' },
  { tag: tags.emphasis, fontStyle: 'italic', color: '#ECEFF4' },
  { tag: tags.strong, fontWeight: 'bold', color: '#ECEFF4' },
  { tag: tags.strikethrough, textDecoration: 'line-through' },
  { tag: tags.typeName, color: '#8FBCBB' },
  { tag: tags.className, color: '#8FBCBB' },
  { tag: tags.definition(tags.variableName), color: '#D8DEE9' },
  { tag: tags.tagName, color: '#81A1C1' },
  { tag: tags.attributeName, color: '#8FBCBB' },
  { tag: tags.attributeValue, color: '#A3BE8C' },
  { tag: tags.processingInstruction, color: '#5E81AC' },
  { tag: tags.quote, color: '#E5E9F0', fontStyle: 'italic' },
  { tag: tags.monospace, color: '#D08770' },
])

let saveTimeout: ReturnType<typeof setTimeout> | null = null

function extractTitle(content: string): string {
  for (const line of content.split('\n')) {
    const trimmed = line.trim()
    if (!trimmed) continue
    // Use first markdown heading if present
    const heading = trimmed.match(/^#{1,6}\s+(.+)/)
    if (heading) return heading[1].trim()
    // Otherwise use first non-empty line
    return trimmed.slice(0, 120)
  }
  return 'Untitled'
}

async function saveCurrentNote() {
  const note = notesStore.currentNote
  const view = editorView.value
  if (!note || !view) return

  const content = view.state.doc.toString()
  const title = extractTitle(content)
  await notesStore.updateNote(note.id, { title, content })
  modified.value = false
}

function promptDeleteNote() {
  if (!notesStore.currentNote) return
  confirmingDelete.value = true
}

async function confirmDeleteCurrentNote() {
  const note = notesStore.currentNote
  if (!note) return
  if (saveTimeout) clearTimeout(saveTimeout)
  await notesStore.deleteNote(note.id)
  confirmingDelete.value = false
}

function cancelDelete() {
  confirmingDelete.value = false
}

function startRemoveTag(tagId: number) {
  confirmingTagRemove.value = tagId
}

async function confirmRemoveTag(tagId: number) {
  const note = notesStore.currentNote
  if (!note) return
  await notesStore.removeTagFromNote(note.id, tagId)
  confirmingTagRemove.value = null
}

function cancelRemoveTag() {
  confirmingTagRemove.value = null
}

function handleEditorKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    if (confirmingDelete.value) {
      cancelDelete()
    } else if (confirmingTagRemove.value !== null) {
      cancelRemoveTag()
    }
  }
}

function registerVimCommands() {
  // :w - save
  Vim.defineEx('w', 'w', () => {
    saveCurrentNote()
  })

  // :q - close/deselect
  Vim.defineEx('q', 'q', () => {
    notesStore.selectNote(null)
  })

  // :wq - save and close
  Vim.defineEx('wq', 'wq', () => {
    saveCurrentNote().then(() => {
      notesStore.selectNote(null)
    })
  })

  // :new <title> - create new note
  Vim.defineEx('new', 'new', (_cm: unknown, params: { args?: string[] }) => {
    const title = params.args?.join(' ') || 'Untitled'
    notesStore.createNote({ title, content: '' })
  })

  // :tag <name> - add tag to current note
  Vim.defineEx('tag', 'tag', (_cm: unknown, params: { args?: string[] }) => {
    const note = notesStore.currentNote
    if (!note || !params.args?.[0]) return
    const tagNames = note.tags.map((t) => t.name)
    if (!tagNames.includes(params.args[0])) {
      tagNames.push(params.args[0])
      notesStore.updateNote(note.id, { tags: tagNames })
    }
  })

  // :untag <name> - remove tag
  Vim.defineEx('untag', 'untag', (_cm: unknown, params: { args?: string[] }) => {
    const note = notesStore.currentNote
    if (!note || !params.args?.[0]) return
    const tagNames = note.tags.map((t) => t.name).filter((t) => t !== params.args![0])
    notesStore.updateNote(note.id, { tags: tagNames })
  })

  // :search <query> - search notes
  Vim.defineEx('search', 'search', (_cm: unknown, params: { args?: string[] }) => {
    const query = params.args?.join(' ')
    if (query) notesStore.searchNotes(query)
  })

  // :dash - go to dashboard
  Vim.defineEx('dash', 'dash', () => {
    router.push('/dashboard')
  })

  // :preview - toggle preview
  Vim.defineEx('preview', 'preview', () => {
    uiStore.togglePreview()
  })

  // :sidebar - toggle sidebar
  Vim.defineEx('sidebar', 'sidebar', () => {
    uiStore.toggleSidebar()
  })

  // :files - open fuzzy finder
  Vim.defineEx('files', 'files', () => {
    uiStore.openFuzzyFinder()
  })

  // :del / :delete - delete current note
  Vim.defineEx('del', 'del', () => {
    promptDeleteNote()
  })
  Vim.defineEx('delete', 'delete', () => {
    promptDeleteNote()
  })

  // :mkdir <name> - create a new folder
  Vim.defineEx('mkdir', 'mkdir', (_cm: unknown, params: { args?: string[] }) => {
    const name = params.args?.join(' ')
    if (!name) return
    notesStore.createFolder({ name }).catch((e: Error) => {
      console.error('Failed to create folder:', e)
    })
  })

  // :mv <folder-name> - move current note to a folder by name
  Vim.defineEx('mv', 'mv', (_cm: unknown, params: { args?: string[] }) => {
    const note = notesStore.currentNote
    if (!note) return
    const folderName = params.args?.join(' ')
    if (!folderName) {
      // Move to root (no folder)
      notesStore.moveNoteToFolder(note.id, null).catch((e: Error) => {
        console.error('Failed to move note:', e)
      })
      return
    }
    const folder = notesStore.folders.find(
      (f) => f.name.toLowerCase() === folderName.toLowerCase()
    )
    if (!folder) {
      console.error(`Folder "${folderName}" not found`)
      return
    }
    notesStore.moveNoteToFolder(note.id, folder.id).catch((e: Error) => {
      console.error('Failed to move note:', e)
    })
  })
}

function createEditor(container: HTMLElement, content: string) {
  const updateListener = EditorView.updateListener.of((update) => {
    if (update.docChanged) {
      modified.value = true
      // Auto-save after 2 seconds of inactivity
      if (saveTimeout) clearTimeout(saveTimeout)
      saveTimeout = setTimeout(() => {
        saveCurrentNote()
      }, 2000)
    }

    // Update cursor position
    const cursor = update.state.selection.main.head
    const line = update.state.doc.lineAt(cursor)
    uiStore.cursorLine = line.number
    uiStore.cursorCol = cursor - line.from + 1
  })

  const state = EditorState.create({
    doc: content,
    extensions: [
      vim(),
      lineNumbers(),
      highlightActiveLine(),
      drawSelection(),
      history(),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      markdown(),
      nordTheme,
      syntaxHighlighting(nordHighlightStyle),
      updateListener,
      EditorView.lineWrapping,
      docCompartment.of([]),
    ],
  })

  const view = new EditorView({ state, parent: container })

  // Track vim mode using the cm instance event
  const cm = getCM(view)
  if (cm) {
    cm.on('vim-mode-change', (ev: { mode: string; subMode?: string }) => {
      if (ev.mode === 'insert' || ev.mode === 'replace') {
        uiStore.setVimMode('insert')
        container.dataset.vimMode = 'insert'
      } else if (ev.mode === 'visual') {
        uiStore.setVimMode('visual')
        container.dataset.vimMode = 'visual'
      } else {
        uiStore.setVimMode('normal')
        container.dataset.vimMode = 'normal'
      }
    })
  }

  return view
}

onMounted(() => {
  registerVimCommands()
  if (editorContainer.value) {
    editorView.value = createEditor(editorContainer.value, notesStore.currentNote?.content || '')
  }
})

onUnmounted(() => {
  if (saveTimeout) clearTimeout(saveTimeout)
  editorView.value?.destroy()
})

watch(
  () => notesStore.currentNote,
  (note) => {
    const view = editorView.value
    if (!view) return
    const newContent = note?.content || ''
    const currentContent = view.state.doc.toString()
    if (newContent !== currentContent) {
      view.dispatch({
        changes: {
          from: 0,
          to: view.state.doc.length,
          insert: newContent,
        },
      })
      modified.value = false
    }
  }
)
</script>

<template>
  <div class="flex flex-col h-full" @keydown="handleEditorKeydown">
    <!-- Tab bar / note title -->
    <div
      v-if="notesStore.currentNote"
      class="flex items-center h-9 px-3 bg-nord1 border-b border-nord2 text-sm"
    >
      <span
        v-if="notesStore.currentNote.folder_id"
        class="text-nord3 text-xs mr-1"
      >{{ notesStore.folders.find(f => f.id === notesStore.currentNote!.folder_id)?.name || '' }} /</span>
      <span class="text-nord4">{{ notesStore.currentNote.title }}</span>
      <span v-if="modified" class="ml-1 text-nord13">[+]</span>
      <div class="flex-1" />
      <div class="flex items-center gap-1.5">
        <!-- Tags with remove buttons -->
        <span
          v-for="tag in notesStore.currentNote.tags"
          :key="tag.id"
          class="group text-[10px] px-1.5 py-px rounded bg-nord2 text-nord8 inline-flex items-center gap-0.5"
        >
          <!-- Tag removal confirmation -->
          <template v-if="confirmingTagRemove === tag.id">
            <span class="text-nord4">Remove?</span>
            <button
              @click="confirmRemoveTag(tag.id)"
              class="text-nord11 hover:text-red-400 font-bold"
            >Y</button>
            <button
              @click="cancelRemoveTag"
              class="text-nord4 hover:text-nord6"
            >N</button>
          </template>
          <template v-else>
            {{ tag.name }}
            <button
              @click="startRemoveTag(tag.id)"
              class="opacity-0 group-hover:opacity-100 transition-opacity hover:text-nord11 ml-0.5"
              title="Remove tag from note"
            >&times;</button>
          </template>
        </span>

        <!-- Delete note button -->
        <button
          @click="promptDeleteNote"
          class="ml-2 p-1 rounded text-nord3 hover:text-nord11 hover:bg-nord2 transition-colors"
          title="Delete note (:del)"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Delete confirmation bar -->
    <div
      v-if="confirmingDelete && notesStore.currentNote"
      class="flex items-center justify-center gap-3 px-3 py-2 bg-nord11/10 border-b border-nord11/30 animate-slideDown"
    >
      <span class="text-sm text-nord4">
        Delete "<span class="text-nord6 font-medium">{{ notesStore.currentNote.title }}</span>"?
      </span>
      <button
        @click="confirmDeleteCurrentNote"
        class="px-3 py-1 text-xs rounded bg-nord11 text-nord6 hover:brightness-110 transition-colors font-medium"
      >
        Delete
      </button>
      <button
        @click="cancelDelete"
        class="px-3 py-1 text-xs rounded bg-nord3 text-nord4 hover:bg-nord2 transition-colors"
      >
        Cancel
      </button>
    </div>

    <!-- Editor area -->
    <div class="flex-1 overflow-hidden relative">
      <div
        v-if="!notesStore.currentNote"
        class="flex items-center justify-center h-full text-nord3"
      >
        <div class="text-center">
          <div class="text-4xl mb-4 font-mono">noted</div>
          <div class="text-sm mb-6">Select a note or create a new one</div>
          <div class="text-xs text-nord3 space-y-1">
            <div><kbd class="bg-nord2 px-1.5 py-0.5 rounded text-nord4">Ctrl+P</kbd> fuzzy finder</div>
            <div><kbd class="bg-nord2 px-1.5 py-0.5 rounded text-nord4">Ctrl+B</kbd> toggle sidebar</div>
            <div><kbd class="bg-nord2 px-1.5 py-0.5 rounded text-nord4">Ctrl+E</kbd> toggle preview</div>
            <div><kbd class="bg-nord2 px-1.5 py-0.5 rounded text-nord4">:new title</kbd> new note</div>
            <div><kbd class="bg-nord2 px-1.5 py-0.5 rounded text-nord4">:del</kbd> delete note</div>
            <div><kbd class="bg-nord2 px-1.5 py-0.5 rounded text-nord4">:mkdir name</kbd> create folder</div>
            <div><kbd class="bg-nord2 px-1.5 py-0.5 rounded text-nord4">:mv folder</kbd> move to folder</div>
            <div><kbd class="bg-nord2 px-1.5 py-0.5 rounded text-nord4">:dash</kbd> dashboard</div>
          </div>
        </div>
      </div>
      <div ref="editorContainer" class="h-full" data-vim-mode="normal" />
    </div>
  </div>
</template>

<style>
@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-100%);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.animate-slideDown {
  animation: slideDown 150ms ease-out;
}

/* Normal mode: blue block cursor */
[data-vim-mode="normal"] .cm-fat-cursor {
  background-color: #81A1C1cc !important;
  color: #2E3440 !important;
}
[data-vim-mode="normal"]:not(.cm-focused) .cm-fat-cursor {
  background-color: #81A1C140 !important;
  outline: 1px solid #81A1C180;
}

/* Insert mode: green line cursor, thicker for visibility */
[data-vim-mode="insert"] .cm-cursor,
[data-vim-mode="insert"] .cm-dropCursor {
  border-left-color: #A3BE8C !important;
  border-left-width: 2.5px !important;
}

/* Visual mode: purple block cursor */
[data-vim-mode="visual"] .cm-fat-cursor {
  background-color: #B48EADcc !important;
  color: #2E3440 !important;
}
[data-vim-mode="visual"] .cm-selectionBackground,
[data-vim-mode="visual"] .cm-content ::selection {
  background-color: #B48EAD30 !important;
}
</style>
