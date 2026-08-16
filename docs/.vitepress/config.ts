import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Vervet',
  description: 'A desktop MongoDB explorer.',
  base: '/vervet/',
  // cleanUrls stays off: /vervet/privacy.html is an existing published URL.
  cleanUrls: false,
  lastUpdated: true,
  head: [['link', { rel: 'icon', href: '/vervet/logo.svg' }]],
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Guide', link: '/guide/install' },
      { text: 'Download', link: '/guide/install' },
      { text: 'GitHub', link: 'https://github.com/blacktau/vervet' },
    ],
    sidebar: [
      {
        text: 'Guide',
        items: [
          { text: 'Install', link: '/guide/install' },
          { text: 'Connecting to a server', link: '/guide/connecting' },
          { text: 'Browsing your data', link: '/guide/browsing' },
          { text: 'Querying', link: '/guide/querying' },
          { text: 'Scripts', link: '/guide/scripts' },
          { text: 'Indexes and statistics', link: '/guide/indexes-and-stats' },
        ],
      },
      {
        text: 'About',
        items: [{ text: 'Privacy policy', link: '/privacy' }],
      },
    ],
    search: { provider: 'local' },
    socialLinks: [{ icon: 'github', link: 'https://github.com/blacktau/vervet' }],
    editLink: {
      pattern: 'https://github.com/blacktau/vervet/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },
    footer: {
      message: 'Released under the GPL-3.0 licence.',
      copyright: 'Copyright © Sean Garrett',
    },
  },
})
