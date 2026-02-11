import { ref, onMounted, onUnmounted } from 'vue'

export interface SSEEvent {
  type: string
  data: unknown
}

export function useSSE(onEvent: (event: SSEEvent) => void) {
  const connected = ref(false)
  let eventSource: EventSource | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectDelay = 1000
  const maxReconnectDelay = 30000

  function connect() {
    const url = `${window.location.origin}/api/events`
    eventSource = new EventSource(url)

    eventSource.onopen = () => {
      connected.value = true
      reconnectDelay = 1000
    }

    eventSource.onmessage = (event) => {
      try {
        const parsed = JSON.parse(event.data) as SSEEvent
        onEvent(parsed)
      } catch {
        // ignore malformed messages
      }
    }

    eventSource.onerror = () => {
      connected.value = false
      eventSource?.close()
      scheduleReconnect()
    }
  }

  function scheduleReconnect() {
    if (reconnectTimer) return
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay)
      connect()
    }, reconnectDelay)
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    connected.value = false
  }

  onMounted(() => {
    connect()
  })

  onUnmounted(() => {
    disconnect()
  })

  return { connected }
}
