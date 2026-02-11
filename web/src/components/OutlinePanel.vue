<script setup lang="ts">
import { computed } from 'vue'
import { useNotesStore } from '../stores/notes'

const notesStore = useNotesStore()

interface Heading {
  level: number
  text: string
  id: string
}

const headings = computed<Heading[]>(() => {
  const content = notesStore.currentNote?.content || ''
  if (!content) return []

  const lines = content.split('\n')
  const result: Heading[] = []

  for (const line of lines) {
    const match = line.match(/^(#{1,6})\s+(.+)$/)
    if (match) {
      const level = match[1].length
      const text = match[2].trim()
      const id = text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
      result.push({ level, text, id })
    }
  }

  return result
})

const minLevel = computed(() => {
  if (headings.value.length === 0) return 1
  return Math.min(...headings.value.map((h) => h.level))
})

function scrollToHeading(heading: Heading) {
  // Search in the CodeMirror editor for the heading text
  const editor = document.querySelector('.cm-content')
  if (!editor) return

  const lines = editor.querySelectorAll('.cm-line')
  for (const line of lines) {
    if (line.textContent?.includes(heading.text)) {
      line.scrollIntoView({ behavior: 'smooth', block: 'center' })
      break
    }
  }
}
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex items-center h-9 px-3 bg-nord1 border-b border-nord2 text-sm">
      <span class="text-nord3">Outline</span>
      <span class="ml-auto text-xs text-nord3">{{ headings.length }}</span>
    </div>

    <div class="flex-1 overflow-y-auto py-2">
      <div v-if="headings.length === 0" class="px-3 py-2 text-xs text-nord3">
        No headings found
      </div>
      <button
        v-for="heading in headings"
        :key="heading.id"
        @click="scrollToHeading(heading)"
        class="w-full text-left px-3 py-1 text-sm text-nord4 hover:bg-nord2 hover:text-nord6 transition-colors truncate"
        :style="{ paddingLeft: `${(heading.level - minLevel) * 12 + 12}px` }"
      >
        <span class="text-nord8 mr-1 text-xs opacity-50">{{ '#'.repeat(heading.level) }}</span>
        {{ heading.text }}
      </button>
    </div>
  </div>
</template>
