<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { marked } from 'marked'
import { useNotesStore } from '../stores/notes'

const notesStore = useNotesStore()
const previewContainer = ref<HTMLElement | null>(null)

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  const content = notesStore.currentNote?.content || ''
  if (!content) return '<p class="text-nord3">No content to preview</p>'
  return marked.parse(content) as string
})

// Basic scroll sync: listen for scroll events on a hypothetical editor scroll
// This is a simple percentage-based approach
const scrollPercent = ref(0)

function handleScroll(e: Event) {
  const el = e.target as HTMLElement
  const maxScroll = el.scrollHeight - el.clientHeight
  if (maxScroll > 0) {
    scrollPercent.value = el.scrollTop / maxScroll
  }
}

// Expose scroll sync for external use
function syncScrollTo(percent: number) {
  if (!previewContainer.value) return
  const el = previewContainer.value
  const maxScroll = el.scrollHeight - el.clientHeight
  el.scrollTop = maxScroll * percent
}

watch(scrollPercent, (p) => {
  syncScrollTo(p)
})

onMounted(() => {
  // placeholder for scroll sync setup
})

onUnmounted(() => {
  // cleanup
})
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
      @scroll="handleScroll"
      v-html="renderedContent"
    />
  </div>
</template>
