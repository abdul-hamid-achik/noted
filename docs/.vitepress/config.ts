import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'noted',
  description: 'A terminal-only, agents-first Obsidian alternative.',
  lang: 'en-US',
  lastUpdated: true,
  cleanUrls: true,
  metaChunk: true,

  themeConfig: {
    logo: '/noted.svg',
    nav: [
      { text: 'Guide', link: '/guide/what-is-noted' },
      { text: 'Reference', link: '/reference/commands' },
      { text: 'TUI', link: '/reference/tui' },
      { text: 'MCP', link: '/reference/mcp' },
    ],
    sidebar: {
      '/getting-started/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Installation', link: '/getting-started/installation' },
            { text: 'Quick start', link: '/getting-started/quick-start' },
            { text: 'Vault setup', link: '/getting-started/vault-setup' },
          ],
        },
      ],
      '/guide/': [
        {
          text: 'Guide',
          items: [
            { text: 'What is noted?', link: '/guide/what-is-noted' },
            { text: 'Capturing notes', link: '/guide/capturing-notes' },
            { text: 'Tags and folders', link: '/guide/tags-and-folders' },
            { text: 'Daily notes', link: '/guide/daily-notes' },
            { text: 'Templates', link: '/guide/templates' },
            { text: 'Wikilinks', link: '/guide/wikilinks' },
            { text: 'Version history', link: '/guide/version-history' },
            { text: 'Semantic search', link: '/guide/semantic-search' },
          ],
        },
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'CLI commands', link: '/reference/commands' },
            { text: 'TUI keybindings', link: '/reference/tui' },
            { text: 'MCP server', link: '/reference/mcp' },
            { text: 'Configuration', link: '/reference/configuration' },
            { text: 'Vault format', link: '/reference/vault-format' },
          ],
        },
      ],
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/abdul-hamid-achik/noted' },
    ],
    editLink: {
      pattern: 'https://github.com/abdul-hamid-achik/noted/edit/main/docs/:path',
    },
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 Abdul Hamid Achik',
    },
  },
})
