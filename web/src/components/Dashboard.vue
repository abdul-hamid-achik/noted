<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Doughnut, Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  ArcElement,
  BarElement,
  CategoryScale,
  LinearScale,
  Tooltip,
  Legend,
} from 'chart.js'
import { useApi } from '../composables/useApi'
import type { Stats, Note, Tag } from '../types'

ChartJS.register(ArcElement, BarElement, CategoryScale, LinearScale, Tooltip, Legend)

const api = useApi()
const router = useRouter()

const stats = ref<Stats>({
  total_notes: 0,
  total_tags: 0,
  db_size_bytes: 0,
  db_size: '0 B',
  unsynced_notes: 0,
})
const tags = ref<Tag[]>([])
const recentNotes = ref<Note[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    const [s, t, n] = await Promise.all([
      api.getStats(),
      api.getTags(),
      api.getNotes(),
    ])
    stats.value = s
    tags.value = t
    recentNotes.value = n.slice(0, 10)
  } finally {
    loading.value = false
  }
})

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const syncedPercent = computed(() => {
  const total = stats.value.total_notes
  if (total === 0) return 100
  const synced = total - stats.value.unsynced_notes
  return Math.round((synced / total) * 100)
})

// Nord palette colors for charts
const chartColors = [
  '#8FBCBB', '#88C0D0', '#81A1C1', '#5E81AC',
  '#BF616A', '#D08770', '#EBCB8B', '#A3BE8C', '#B48EAD',
]

const tagChartData = computed(() => ({
  labels: tags.value.map((t) => t.name),
  datasets: [
    {
      data: tags.value.map((t) => t.note_count || 0),
      backgroundColor: tags.value.map((_, i) => chartColors[i % chartColors.length]),
      borderWidth: 0,
    },
  ],
}))

// Note activity by day (last 7 days)
const noteActivityData = computed(() => {
  const days: string[] = []
  const counts: number[] = []
  for (let i = 6; i >= 0; i--) {
    const d = new Date()
    d.setDate(d.getDate() - i)
    const label = d.toLocaleDateString(undefined, { weekday: 'short' })
    const dayStr = d.toISOString().split('T')[0]
    days.push(label)
    counts.push(recentNotes.value.filter((n) => n.updated_at.startsWith(dayStr)).length)
  }
  return {
    labels: days,
    datasets: [{
      label: 'Notes Updated',
      data: counts,
      backgroundColor: chartColors.slice(0, 7),
      borderWidth: 0,
      borderRadius: 4,
    }],
  }
})

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      labels: {
        color: '#D8DEE9',
        font: { family: 'Inter', size: 12 },
      },
    },
    tooltip: {
      backgroundColor: '#3B4252',
      titleColor: '#ECEFF4',
      bodyColor: '#D8DEE9',
      borderColor: '#434C5E',
      borderWidth: 1,
    },
  },
}

const barChartOptions = {
  ...chartOptions,
  scales: {
    x: {
      ticks: { color: '#D8DEE9', font: { family: 'Inter', size: 11 } },
      grid: { color: '#3B4252' },
    },
    y: {
      ticks: { color: '#D8DEE9', font: { family: 'Inter', size: 11 }, stepSize: 1 },
      grid: { color: '#3B4252' },
    },
  },
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
        <span class="text-sm text-nord3">dashboard</span>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="router.push('/settings')"
          class="p-1.5 rounded hover:bg-nord2 text-nord3 hover:text-nord4 transition-colors"
          title="Settings"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd" />
          </svg>
        </button>
        <button
          @click="goToEditor"
          class="bg-nord10 hover:bg-nord9 text-nord6 text-sm font-medium py-1.5 px-4 rounded transition-colors"
        >
          Back to Editor
        </button>
      </div>
    </header>

    <div v-if="loading" class="flex items-center justify-center h-64">
      <div class="text-nord3">Loading...</div>
    </div>

    <div v-else class="p-6 space-y-6 max-w-7xl mx-auto">
      <!-- Stats cards -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="bg-nord1 rounded-lg p-4 border-l-4 border-nord8">
          <div class="text-xs text-nord3 uppercase tracking-wider mb-1">Total Notes</div>
          <div class="text-2xl font-bold text-nord6">{{ stats.total_notes }}</div>
        </div>
        <div class="bg-nord1 rounded-lg p-4 border-l-4 border-nord7">
          <div class="text-xs text-nord3 uppercase tracking-wider mb-1">DB Size</div>
          <div class="text-2xl font-bold text-nord6">{{ stats.db_size }}</div>
        </div>
        <div class="bg-nord1 rounded-lg p-4 border-l-4 border-nord9">
          <div class="text-xs text-nord3 uppercase tracking-wider mb-1">Tags</div>
          <div class="text-2xl font-bold text-nord6">{{ stats.total_tags }}</div>
        </div>
        <div class="bg-nord1 rounded-lg p-4 border-l-4 border-nord10">
          <div class="text-xs text-nord3 uppercase tracking-wider mb-1">Unsynced</div>
          <div class="text-2xl font-bold text-nord6">{{ stats.unsynced_notes }}</div>
        </div>
      </div>

      <!-- Charts row -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- Tag distribution -->
        <div class="bg-nord1 rounded-lg p-4">
          <h2 class="text-sm font-medium text-nord6 mb-4">Tag Distribution</h2>
          <div v-if="tags.length > 0" class="h-64">
            <Doughnut :data="tagChartData" :options="chartOptions" />
          </div>
          <div v-else class="h-64 flex items-center justify-center text-nord3 text-sm">
            No tags yet
          </div>
        </div>

        <!-- Note Activity -->
        <div class="bg-nord1 rounded-lg p-4">
          <h2 class="text-sm font-medium text-nord6 mb-4">Recent Activity</h2>
          <div class="h-64">
            <Bar :data="noteActivityData" :options="barChartOptions" />
          </div>
        </div>
      </div>

      <!-- Bottom row -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- Recent activity -->
        <div class="bg-nord1 rounded-lg p-4">
          <h2 class="text-sm font-medium text-nord6 mb-4">Recent Notes</h2>
          <div class="space-y-2">
            <div
              v-for="note in recentNotes"
              :key="note.id"
              class="flex items-center justify-between py-1.5 px-2 rounded hover:bg-nord2 cursor-pointer transition-colors"
              @click="router.push('/'); /* will select on editor page */"
            >
              <div class="flex items-center gap-2 min-w-0">
                <span class="text-sm text-nord4 truncate">{{ note.title }}</span>
                <span
                  v-for="tag in note.tags.slice(0, 2)"
                  :key="tag.id"
                  class="text-[10px] px-1.5 py-px rounded bg-nord2 text-nord8 shrink-0"
                >
                  {{ tag.name }}
                </span>
              </div>
              <span class="text-xs text-nord3 shrink-0 ml-2">{{ formatDate(note.updated_at) }}</span>
            </div>
            <div v-if="recentNotes.length === 0" class="text-sm text-nord3 text-center py-4">
              No notes yet
            </div>
          </div>
        </div>

        <!-- Search index status -->
        <div class="bg-nord1 rounded-lg p-4">
          <h2 class="text-sm font-medium text-nord6 mb-4">Search Index</h2>
          <div class="space-y-4">
            <div>
              <div class="flex justify-between text-sm mb-1">
                <span class="text-nord4">Embedding sync</span>
                <span class="text-nord3">{{ syncedPercent }}%</span>
              </div>
              <div class="w-full bg-nord2 rounded-full h-2">
                <div
                  class="h-2 rounded-full transition-all duration-500"
                  :class="syncedPercent === 100 ? 'bg-nord14' : 'bg-nord13'"
                  :style="{ width: `${syncedPercent}%` }"
                />
              </div>
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <div class="text-xs text-nord3">Synced</div>
                <div class="text-lg font-semibold text-nord14">
                  {{ stats.total_notes - stats.unsynced_notes }}
                </div>
              </div>
              <div>
                <div class="text-xs text-nord3">Pending</div>
                <div class="text-lg font-semibold" :class="stats.unsynced_notes > 0 ? 'text-nord13' : 'text-nord3'">
                  {{ stats.unsynced_notes }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
