import { describe, it, expect } from 'vitest'
import router from '../router'

describe('router', () => {
  it('has editor route at /', () => {
    const route = router.getRoutes().find((r) => r.path === '/')
    expect(route).toBeDefined()
    expect(route?.name).toBe('editor')
  })

  it('has note route at /notes/:id', () => {
    const route = router.getRoutes().find((r) => r.path === '/notes/:id')
    expect(route).toBeDefined()
    expect(route?.name).toBe('note')
  })

  it('has dashboard route', () => {
    const route = router.getRoutes().find((r) => r.path === '/dashboard')
    expect(route).toBeDefined()
    expect(route?.name).toBe('dashboard')
  })

  it('has settings route', () => {
    const route = router.getRoutes().find((r) => r.path === '/settings')
    expect(route).toBeDefined()
    expect(route?.name).toBe('settings')
  })

  it('has graph route', () => {
    const route = router.getRoutes().find((r) => r.path === '/graph')
    expect(route).toBeDefined()
    expect(route?.name).toBe('graph')
  })

  it('has catch-all redirect to /', () => {
    const route = router.getRoutes().find((r) => r.path === '/:pathMatch(.*)*')
    expect(route).toBeDefined()
  })

  it('has correct number of routes', () => {
    // 5 named routes + 1 catch-all
    expect(router.getRoutes().length).toBe(6)
  })
})
