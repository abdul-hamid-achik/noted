<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import * as d3 from 'd3'
import { useApi } from '../composables/useApi'
import type { GraphData, GraphNode } from '../types'

const api = useApi()
const router = useRouter()

const svgRef = ref<SVGSVGElement | null>(null)
const loading = ref(true)
const error = ref('')
const graphData = ref<GraphData | null>(null)
const searchQuery = ref('')
const hoveredNode = ref<number | null>(null)

// D3 simulation types
interface SimNode extends GraphNode {
  x: number
  y: number
  fx: number | null
  fy: number | null
}

interface SimEdge {
  source: SimNode
  target: SimNode
  label: string
}

let simulation: d3.Simulation<SimNode, SimEdge> | null = null

onMounted(async () => {
  await fetchGraph()
})

onUnmounted(() => {
  if (simulation) {
    simulation.stop()
    simulation = null
  }
})

async function fetchGraph() {
  loading.value = true
  error.value = ''
  try {
    graphData.value = await api.getGraph()
    if (graphData.value.nodes.length > 0) {
      await renderGraph()
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load graph'
  } finally {
    loading.value = false
  }
}

function getConnectedNodeIds(nodeId: number): Set<number> {
  const connected = new Set<number>()
  if (!graphData.value) return connected
  for (const edge of graphData.value.edges) {
    if (edge.source === nodeId || (edge.source as unknown as SimNode).id === nodeId) {
      const targetId = typeof edge.target === 'number' ? edge.target : (edge.target as unknown as SimNode).id
      connected.add(targetId)
    }
    if (edge.target === nodeId || (edge.target as unknown as SimNode).id === nodeId) {
      const sourceId = typeof edge.source === 'number' ? edge.source : (edge.source as unknown as SimNode).id
      connected.add(sourceId)
    }
  }
  return connected
}

function nodeRadius(linkCount: number): number {
  return Math.max(6, Math.min(20, 6 + linkCount * 2))
}

async function renderGraph() {
  if (!svgRef.value || !graphData.value) return

  const svg = d3.select(svgRef.value)
  svg.selectAll('*').remove()

  const width = svgRef.value.clientWidth
  const height = svgRef.value.clientHeight

  const nodes: SimNode[] = graphData.value.nodes.map((n) => ({
    ...n,
    x: width / 2 + (Math.random() - 0.5) * 200,
    y: height / 2 + (Math.random() - 0.5) * 200,
    fx: null,
    fy: null,
  }))

  const nodeMap = new Map(nodes.map((n) => [n.id, n]))
  const edges: SimEdge[] = graphData.value.edges
    .filter((e) => nodeMap.has(e.source) && nodeMap.has(e.target))
    .map((e) => ({
      source: nodeMap.get(e.source)!,
      target: nodeMap.get(e.target)!,
      label: e.label,
    }))

  // Container for zoom
  const g = svg.append('g')

  // Zoom behavior
  const zoom = d3.zoom<SVGSVGElement, unknown>()
    .scaleExtent([0.1, 4])
    .on('zoom', (event) => {
      g.attr('transform', event.transform)
    })

  svg.call(zoom)

  // Center the view
  svg.call(zoom.transform, d3.zoomIdentity.translate(width / 2, height / 2).scale(0.8).translate(-width / 2, -height / 2))

  // Edges
  const edgeGroup = g.append('g').attr('class', 'edges')
  const edgeLines = edgeGroup
    .selectAll('line')
    .data(edges)
    .enter()
    .append('line')
    .attr('stroke', '#4C566A')
    .attr('stroke-width', 1.5)
    .attr('stroke-opacity', 0.6)

  // Node group
  const nodeGroup = g.append('g').attr('class', 'nodes')
  const nodeGs = nodeGroup
    .selectAll('g')
    .data(nodes)
    .enter()
    .append('g')
    .attr('cursor', 'pointer')
    .call(
      d3.drag<SVGGElement, SimNode>()
        .on('start', (event, d) => {
          if (!event.active) simulation?.alphaTarget(0.3).restart()
          d.fx = d.x
          d.fy = d.y
        })
        .on('drag', (event, d) => {
          d.fx = event.x
          d.fy = event.y
        })
        .on('end', (event, d) => {
          if (!event.active) simulation?.alphaTarget(0)
          d.fx = null
          d.fy = null
        })
    )

  // Node circles
  const circles = nodeGs
    .append('circle')
    .attr('r', (d) => nodeRadius(d.link_count))
    .attr('fill', '#88C0D0')
    .attr('stroke', '#81A1C1')
    .attr('stroke-width', 1.5)

  // Node labels
  const labels = nodeGs
    .append('text')
    .text((d) => d.title.length > 24 ? d.title.slice(0, 22) + '...' : d.title)
    .attr('dy', (d) => nodeRadius(d.link_count) + 14)
    .attr('text-anchor', 'middle')
    .attr('fill', '#D8DEE9')
    .attr('font-size', '11px')
    .attr('font-family', 'Inter, sans-serif')
    .attr('pointer-events', 'none')

  // Hover and click events
  nodeGs
    .on('mouseenter', (_event, d) => {
      hoveredNode.value = d.id
      const connected = getConnectedNodeIds(d.id)
      connected.add(d.id)

      circles
        .attr('fill', (n) => connected.has(n.id) ? '#88C0D0' : '#434C5E')
        .attr('stroke', (n) => n.id === d.id ? '#ECEFF4' : connected.has(n.id) ? '#81A1C1' : '#434C5E')
        .attr('stroke-width', (n) => n.id === d.id ? 2.5 : 1.5)

      labels
        .attr('fill', (n) => connected.has(n.id) ? '#ECEFF4' : '#4C566A')

      edgeLines
        .attr('stroke', (e) => {
          const src = (e.source as SimNode).id
          const tgt = (e.target as SimNode).id
          return (src === d.id || tgt === d.id) ? '#88C0D0' : '#3B4252'
        })
        .attr('stroke-opacity', (e) => {
          const src = (e.source as SimNode).id
          const tgt = (e.target as SimNode).id
          return (src === d.id || tgt === d.id) ? 1 : 0.2
        })
        .attr('stroke-width', (e) => {
          const src = (e.source as SimNode).id
          const tgt = (e.target as SimNode).id
          return (src === d.id || tgt === d.id) ? 2 : 1
        })
    })
    .on('mouseleave', () => {
      hoveredNode.value = null
      circles
        .attr('fill', '#88C0D0')
        .attr('stroke', '#81A1C1')
        .attr('stroke-width', 1.5)
      labels
        .attr('fill', '#D8DEE9')
      edgeLines
        .attr('stroke', '#4C566A')
        .attr('stroke-opacity', 0.6)
        .attr('stroke-width', 1.5)
    })
    .on('click', (_event, d) => {
      router.push(`/notes/${d.id}`)
    })

  // Search highlighting
  watch(searchQuery, (q) => {
    const lower = q.toLowerCase().trim()
    if (!lower) {
      circles.attr('fill', '#88C0D0').attr('stroke', '#81A1C1').attr('stroke-width', 1.5)
      labels.attr('fill', '#D8DEE9')
      return
    }
    circles
      .attr('fill', (d) => d.title.toLowerCase().includes(lower) ? '#EBCB8B' : '#434C5E')
      .attr('stroke', (d) => d.title.toLowerCase().includes(lower) ? '#EBCB8B' : '#434C5E')
      .attr('stroke-width', (d) => d.title.toLowerCase().includes(lower) ? 2.5 : 1.5)
    labels
      .attr('fill', (d) => d.title.toLowerCase().includes(lower) ? '#ECEFF4' : '#4C566A')
  })

  // Force simulation
  simulation = d3.forceSimulation<SimNode>(nodes)
    .force('link', d3.forceLink<SimNode, SimEdge>(edges).id((d) => d.id).distance(120))
    .force('charge', d3.forceManyBody().strength(-300))
    .force('center', d3.forceCenter(width / 2, height / 2))
    .force('collision', d3.forceCollide<SimNode>().radius((d) => nodeRadius(d.link_count) + 10))
    .on('tick', () => {
      edgeLines
        .attr('x1', (d) => (d.source as SimNode).x)
        .attr('y1', (d) => (d.source as SimNode).y)
        .attr('x2', (d) => (d.target as SimNode).x)
        .attr('y2', (d) => (d.target as SimNode).y)

      nodeGs.attr('transform', (d) => `translate(${d.x},${d.y})`)
    })
}
</script>

<template>
  <div class="min-h-screen bg-nord0 text-nord4 flex flex-col">
    <!-- Header -->
    <header class="flex items-center justify-between px-6 py-4 border-b border-nord2 shrink-0">
      <div class="flex items-center gap-3">
        <h1 class="text-xl font-semibold text-nord6 font-mono">noted</h1>
        <span class="text-sm text-nord3">graph</span>
      </div>
      <div class="flex items-center gap-2">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Filter nodes..."
          class="bg-nord1 text-nord4 border border-nord3 rounded px-3 py-1.5 text-sm focus:outline-none focus:border-nord8 placeholder-nord3 font-mono w-48"
        />
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

    <!-- Loading -->
    <div v-if="loading" class="flex-1 flex items-center justify-center">
      <div class="text-nord3">Loading graph...</div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="flex-1 flex items-center justify-center">
      <div class="text-center">
        <div class="text-nord11 mb-2">{{ error }}</div>
        <button
          @click="fetchGraph"
          class="bg-nord10 hover:bg-nord9 text-nord6 text-sm py-1.5 px-4 rounded transition-colors"
        >
          Retry
        </button>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="graphData && graphData.nodes.length === 0"
      class="flex-1 flex items-center justify-center"
    >
      <div class="text-center">
        <div class="text-nord3 mb-2">No notes with links</div>
        <div class="text-nord3 text-sm mb-4">Create [[wikilinks]] between notes to see the graph</div>
        <button
          @click="router.push('/')"
          class="bg-nord10 hover:bg-nord9 text-nord6 text-sm py-1.5 px-4 rounded transition-colors"
        >
          Go to Editor
        </button>
      </div>
    </div>

    <!-- Graph SVG -->
    <svg
      v-else
      ref="svgRef"
      class="flex-1 w-full"
      style="min-height: 0"
    />
  </div>
</template>
