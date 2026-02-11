<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useApi } from '../composables/useApi'
import type { SettingsInfo } from '../types'

const api = useApi()
const router = useRouter()

const settings = ref<SettingsInfo | null>(null)
const loading = ref(true)
const error = ref('')

// Action states
const vacuuming = ref(false)
const checkpointing = ref(false)
const actionMessage = ref('')
const actionError = ref('')

onMounted(async () => {
  await loadSettings()
})

async function loadSettings() {
  loading.value = true
  error.value = ''
  try {
    settings.value = await api.getSettings()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load settings'
  } finally {
    loading.value = false
  }
}

function clearActionMessages() {
  actionMessage.value = ''
  actionError.value = ''
}

async function handleVacuum() {
  clearActionMessages()
  vacuuming.value = true
  try {
    const result = await api.vacuumDB()
    actionMessage.value = result.status === 'ok' ? 'Database vacuumed successfully' : 'Vacuum completed'
    await loadSettings()
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Vacuum failed'
  } finally {
    vacuuming.value = false
  }
}

async function handleCheckpoint() {
  clearActionMessages()
  checkpointing.value = true
  try {
    const result = await api.walCheckpoint()
    actionMessage.value = result.status === 'ok' ? 'WAL checkpoint completed' : 'Checkpoint completed'
    await loadSettings()
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Checkpoint failed'
  } finally {
    checkpointing.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-nord0 text-nord4">
    <!-- Header -->
    <header class="flex items-center justify-between px-6 py-4 border-b border-nord2">
      <div class="flex items-center gap-3">
        <h1 class="text-xl font-semibold text-nord6 font-mono">noted</h1>
        <span class="text-sm text-nord3">settings</span>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="router.push('/dashboard')"
          class="bg-nord2 hover:bg-nord3 text-nord6 text-sm font-medium py-1.5 px-4 rounded transition-colors"
        >
          Dashboard
        </button>
        <button
          @click="router.push('/')"
          class="bg-nord10 hover:bg-nord9 text-nord6 text-sm font-medium py-1.5 px-4 rounded transition-colors"
        >
          Back to Editor
        </button>
      </div>
    </header>

    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center h-64">
      <div class="text-nord3">Loading settings...</div>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="flex items-center justify-center h-64">
      <div class="text-center">
        <div class="text-nord11 mb-2">{{ error }}</div>
        <button
          @click="loadSettings"
          class="bg-nord10 hover:bg-nord9 text-nord6 text-sm py-1.5 px-4 rounded transition-colors"
        >
          Retry
        </button>
      </div>
    </div>

    <div v-else-if="settings" class="p-6 space-y-6 max-w-4xl mx-auto">
      <!-- Action feedback -->
      <div
        v-if="actionMessage"
        class="bg-nord1 border border-nord14 rounded-lg p-3 flex items-center justify-between"
      >
        <span class="text-nord14 text-sm">{{ actionMessage }}</span>
        <button @click="actionMessage = ''" class="text-nord3 hover:text-nord4 text-xs ml-4">dismiss</button>
      </div>
      <div
        v-if="actionError"
        class="bg-nord1 border border-nord11 rounded-lg p-3 flex items-center justify-between"
      >
        <span class="text-nord11 text-sm">{{ actionError }}</span>
        <button @click="actionError = ''" class="text-nord3 hover:text-nord4 text-xs ml-4">dismiss</button>
      </div>

      <!-- Database Info -->
      <section class="bg-nord1 rounded-lg p-5">
        <h2 class="text-sm font-medium text-nord6 mb-4 uppercase tracking-wider">Database</h2>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <div class="text-xs text-nord3 mb-0.5">Journal Mode</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.db.journal_mode }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Page Size</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.db.page_size }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Cache Size</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.db.cache_size }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Busy Timeout</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.db.busy_timeout }}ms</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Foreign Keys</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.db.foreign_keys ? 'ON' : 'OFF' }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">WAL Pages</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.db.wal_pages }}</div>
          </div>
        </div>

        <!-- DB Actions -->
        <div class="mt-5 pt-4 border-t border-nord2 flex flex-wrap gap-3">
          <button
            @click="handleVacuum"
            :disabled="vacuuming"
            class="bg-nord3 hover:bg-nord10 disabled:opacity-50 disabled:cursor-not-allowed text-nord6 text-sm py-1.5 px-4 rounded transition-colors flex items-center gap-2"
          >
            <svg v-if="vacuuming" class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            {{ vacuuming ? 'Vacuuming...' : 'Vacuum Database' }}
          </button>
          <button
            @click="handleCheckpoint"
            :disabled="checkpointing"
            class="bg-nord3 hover:bg-nord10 disabled:opacity-50 disabled:cursor-not-allowed text-nord6 text-sm py-1.5 px-4 rounded transition-colors flex items-center gap-2"
          >
            <svg v-if="checkpointing" class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            {{ checkpointing ? 'Checkpointing...' : 'WAL Checkpoint' }}
          </button>
        </div>
      </section>

      <!-- Application Info -->
      <section class="bg-nord1 rounded-lg p-5">
        <h2 class="text-sm font-medium text-nord6 mb-4 uppercase tracking-wider">Runtime</h2>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <div class="text-xs text-nord3 mb-0.5">App Version</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.app.version }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Go Version</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.runtime.go_version }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Platform</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.runtime.goos }}/{{ settings.runtime.goarch }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Goroutines</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.runtime.num_goroutine }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">CPUs</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.runtime.num_cpu }}</div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
