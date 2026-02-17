import { describe, it, expect, beforeEach, vi } from 'vitest'

// Reset module state between tests
let useToast: typeof import('../composables/useToast').useToast

describe('useToast', () => {
  beforeEach(async () => {
    vi.useFakeTimers()
    // Re-import to reset module-level state (nextId, toasts)
    vi.resetModules()
    const mod = await import('../composables/useToast')
    useToast = mod.useToast
  })

  it('adds a success toast', () => {
    const { toasts, success } = useToast()
    success('Saved!')
    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0].type).toBe('success')
    expect(toasts.value[0].message).toBe('Saved!')
  })

  it('adds an error toast with longer duration', () => {
    const { toasts, error } = useToast()
    error('Failed!')
    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0].type).toBe('error')
    expect(toasts.value[0].duration).toBe(5000)
  })

  it('adds an info toast', () => {
    const { toasts, info } = useToast()
    info('Note updated')
    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0].type).toBe('info')
  })

  it('auto-removes toast after duration', () => {
    const { toasts, success } = useToast()
    success('Auto remove')
    expect(toasts.value).toHaveLength(1)

    vi.advanceTimersByTime(3000)
    expect(toasts.value).toHaveLength(0)
  })

  it('manually removes a toast', () => {
    const { toasts, info, remove } = useToast()
    info('First')
    info('Second')
    expect(toasts.value).toHaveLength(2)

    const id = toasts.value[0].id
    remove(id)
    expect(toasts.value).toHaveLength(1)
  })

  it('handles multiple toasts', () => {
    const { toasts, success, error, info } = useToast()
    success('one')
    error('two')
    info('three')
    expect(toasts.value).toHaveLength(3)
  })
})
