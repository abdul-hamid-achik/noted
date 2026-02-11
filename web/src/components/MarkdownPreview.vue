<script setup lang="ts">
import { computed, ref } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useNotesStore } from '../stores/notes'
import { useRouter } from 'vue-router'

const notesStore = useNotesStore()
const router = useRouter()
const previewContainer = ref<HTMLElement | null>(null)

// Custom renderer for wikilinks [[note title]]
const wikilinkExtension = {
  name: 'wikilink',
  level: 'inline' as const,
  start(src: string) { return src.indexOf('[[') },
  tokenizer(src: string) {
    const match = src.match(/^\[\[([^\]]+)\]\]/)
    if (match) {
      return {
        type: 'wikilink',
        raw: match[0],
        text: match[1].trim(),
      }
    }
    return undefined
  },
  renderer(token: { text: string }) {
    const title = token.text
    return `<a class="wikilink" data-wikilink="${title}" href="#" title="Link to: ${title}">${title}</a>`
  },
}

marked.use({
  breaks: true,
  gfm: true,
  extensions: [wikilinkExtension],
})

const renderedContent = computed(() => {
  const content = notesStore.currentNote?.content || ''
  if (!content) return '<p class="text-nord3">No content to preview</p>'
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html, {
    ADD_ATTR: ['data-wikilink'],
  })
})

function handlePreviewClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (target.classList.contains('wikilink')) {
    e.preventDefault()
    const title = target.getAttribute('data-wikilink')
    if (title) {
      const note = notesStore.notes.find(
        (n) => n.title.toLowerCase() === title.toLowerCase()
      )
      if (note) {
        notesStore.selectNote(note)
        router.push(`/notes/${note.id}`)
      }
    }
  }
}
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Preview header -->
    <div class="flex items-center h-9 px-3 bg-nord1 border-b border-nord2 text-sm">
      <span class="text-nord3">Preview</span>
    </div>

    <!-- Preview content -->
    <div
      ref="previewContainer"
      class="flex-1 overflow-y-auto p-4 prose-nord"
      @click="handlePreviewClick"
      v-html="renderedContent"
    />
  </div>
</template>
