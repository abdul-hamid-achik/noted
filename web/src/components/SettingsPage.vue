<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
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
const deletingAll = ref(false)
const resetting = ref(false)
const actionMessage = ref('')
const actionError = ref('')

// Danger zone confirmations
const deleteConfirmStep = ref(0) // 0=idle, 1=first confirm, 2=second confirm
const resetConfirmStep = ref(0)

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

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return `${h}h ${m}m`
}

const searchSyncPercent = computed(() => {
  if (!settings.value) return 100
  const total = settings.value.search_indexed + settings.value.search_pending
  if (total === 0) return 100
  return Math.round((settings.value.search_indexed / total) * 100)
})

function clearActionMessages() {
  actionMessage.value = ''
  actionError.value = ''
}

async function handleVacuum() {
  clearActionMessages()
  vacuuming.value = true
  try {
    const result = await api.vacuumDB()
    actionMessage.value = result.message
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
    actionMessage.value = result.message
    await loadSettings()
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Checkpoint failed'
  } finally {
    checkpointing.value = false
  }
}

function initiateDeleteAll() {
  clearActionMessages()
  resetConfirmStep.value = 0
  deleteConfirmStep.value = 1
}

function cancelDeleteAll() {
  deleteConfirmStep.value = 0
}

async function confirmDeleteAll() {
  if (deleteConfirmStep.value === 1) {
    deleteConfirmStep.value = 2
    return
  }
  clearActionMessages()
  deletingAll.value = true
  try {
    const result = await api.deleteAllNotes()
    actionMessage.value = result.message
    deleteConfirmStep.value = 0
    await loadSettings()
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Delete failed'
  } finally {
    deletingAll.value = false
  }
}

function initiateReset() {
  clearActionMessages()
  deleteConfirmStep.value = 0
  resetConfirmStep.value = 1
}

function cancelReset() {
  resetConfirmStep.value = 0
}

async function confirmReset() {
  if (resetConfirmStep.value === 1) {
    resetConfirmStep.value = 2
    return
  }
  clearActionMessages()
  resetting.value = true
  try {
    const result = await api.resetDatabase()
    actionMessage.value = result.message
    resetConfirmStep.value = 0
    await loadSettings()
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Reset failed'
  } finally {
    resetting.value = false
  }
}

function goToDashboard() {
  router.push('/dashboard')
}

function goToEditor() {
  router.push('/')
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
          @click="goToDashboard"
          class="bg-nord2 hover:bg-nord3 text-nord6 text-sm font-medium py-1.5 px-4 rounded transition-colors"
        >
          Dashboard
        </button>
        <button
          @click="goToEditor"
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
            <div class="text-xs text-nord3 mb-0.5">Path</div>
            <div class="text-sm text-nord4 font-mono truncate" :title="settings.db_path">{{ settings.db_path }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Size</div>
            <div class="text-sm text-nord4">{{ formatBytes(settings.db_size_bytes) }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Journal Mode</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.journal_mode }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">SQLite Version</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.sqlite_version }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Notes</div>
            <div class="text-sm text-nord4">{{ settings.total_notes }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Tags</div>
            <div class="text-sm text-nord4">{{ settings.total_tags }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Memories</div>
            <div class="text-sm text-nord4">{{ settings.total_memories }}</div>
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
        <h2 class="text-sm font-medium text-nord6 mb-4 uppercase tracking-wider">Application</h2>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <div class="text-xs text-nord3 mb-0.5">Version</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.app_version || 'dev' }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Go Version</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.go_version }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Platform</div>
            <div class="text-sm text-nord4 font-mono">{{ settings.platform }}</div>
          </div>
          <div>
            <div class="text-xs text-nord3 mb-0.5">Uptime</div>
            <div class="text-sm text-nord4">{{ formatUptime(settings.uptime_seconds) }}</div>
          </div>
        </div>
      </section>

      <!-- Search Index -->
      <section class="bg-nord1 rounded-lg p-5">
        <h2 class="text-sm font-medium text-nord6 mb-4 uppercase tracking-wider">Search Index</h2>
        <div class="space-y-4">
          <div>
            <div class="flex justify-between text-sm mb-1">
              <span class="text-nord4">Embedding sync</span>
              <span class="text-nord3">{{ searchSyncPercent }}%</span>
            </div>
            <div class="w-full bg-nord2 rounded-full h-2">
              <div
                class="h-2 rounded-full transition-all duration-500"
                :class="searchSyncPercent === 100 ? 'bg-nord14' : 'bg-nord13'"
                :style="{ width: `${searchSyncPercent}%` }"
              />
            </div>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <div class="text-xs text-nord3">Indexed</div>
              <div class="text-lg font-semibold text-nord14">{{ settings.search_indexed }}</div>
            </div>
            <div>
              <div class="text-xs text-nord3">Pending</div>
              <div class="text-lg font-semibold" :class="settings.search_pending > 0 ? 'text-nord13' : 'text-nord3'">
                {{ settings.search_pending }}
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Danger Zone -->
      <section class="bg-nord1 rounded-lg p-5 border border-nord11">
        <h2 class="text-sm font-medium text-nord11 mb-4 uppercase tracking-wider">Danger Zone</h2>
        <div class="space-y-4">
          <!-- Delete All Notes -->
          <div class="flex items-center justify-between py-3 border-b border-nord2">
            <div>
              <div class="text-sm text-nord4">Delete All Notes</div>
              <div class="text-xs text-nord3">Permanently delete all notes and their tag associations.</div>
            </div>
            <div class="flex items-center gap-2 shrink-0 ml-4">
              <template v-if="deleteConfirmStep === 0">
                <button
                  @click="initiateDeleteAll"
                  class="bg-nord11 hover:brightness-110 text-nord6 text-sm py-1.5 px-4 rounded transition-all"
                >
                  Delete All Notes
                </button>
              </template>
              <template v-else-if="deleteConfirmStep === 1">
                <span class="text-xs text-nord13 mr-2">Are you sure?</span>
                <button
                  @click="confirmDeleteAll"
                  class="bg-nord11 hover:brightness-110 text-nord6 text-sm py-1.5 px-4 rounded transition-all"
                >
                  Yes, Continue
                </button>
                <button
                  @click="cancelDeleteAll"
                  class="bg-nord2 hover:bg-nord3 text-nord4 text-sm py-1.5 px-3 rounded transition-colors"
                >
                  Cancel
                </button>
              </template>
              <template v-else>
                <span class="text-xs text-nord11 mr-2 font-bold">This cannot be undone!</span>
                <button
                  @click="confirmDeleteAll"
                  :disabled="deletingAll"
                  class="bg-nord11 hover:brightness-110 disabled:opacity-50 text-nord6 text-sm py-1.5 px-4 rounded transition-all flex items-center gap-2"
                >
                  <svg v-if="deletingAll" class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  {{ deletingAll ? 'Deleting...' : 'CONFIRM DELETE ALL' }}
                </button>
                <button
                  @click="cancelDeleteAll"
                  class="bg-nord2 hover:bg-nord3 text-nord4 text-sm py-1.5 px-3 rounded transition-colors"
                >
                  Cancel
                </button>
              </template>
            </div>
          </div>

          <!-- Reset Database -->
          <div class="flex items-center justify-between py-3">
            <div>
              <div class="text-sm text-nord4">Reset Database</div>
              <div class="text-xs text-nord3">Drop and recreate all tables. All data will be lost.</div>
            </div>
            <div class="flex items-center gap-2 shrink-0 ml-4">
              <template v-if="resetConfirmStep === 0">
                <button
                  @click="initiateReset"
                  class="bg-nord11 hover:brightness-110 text-nord6 text-sm py-1.5 px-4 rounded transition-all"
                >
                  Reset Database
                </button>
              </template>
              <template v-else-if="resetConfirmStep === 1">
                <span class="text-xs text-nord13 mr-2">Are you sure?</span>
                <button
                  @click="confirmReset"
                  class="bg-nord11 hover:brightness-110 text-nord6 text-sm py-1.5 px-4 rounded transition-all"
                >
                  Yes, Continue
                </button>
                <button
                  @click="cancelReset"
                  class="bg-nord2 hover:bg-nord3 text-nord4 text-sm py-1.5 px-3 rounded transition-colors"
                >
                  Cancel
                </button>
              </template>
              <template v-else>
                <span class="text-xs text-nord11 mr-2 font-bold">ALL DATA WILL BE LOST!</span>
                <button
                  @click="confirmReset"
                  :disabled="resetting"
                  class="bg-nord11 hover:brightness-110 disabled:opacity-50 text-nord6 text-sm py-1.5 px-4 rounded transition-all flex items-center gap-2"
                >
                  <svg v-if="resetting" class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  {{ resetting ? 'Resetting...' : 'CONFIRM FULL RESET' }}
                </button>
                <button
                  @click="cancelReset"
                  class="bg-nord2 hover:bg-nord3 text-nord4 text-sm py-1.5 px-3 rounded transition-colors"
                >
                  Cancel
                </button>
              </template>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
