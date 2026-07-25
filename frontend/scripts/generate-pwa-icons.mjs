import { mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { chromium } from '@playwright/test'

const publicDirectory = resolve('public')
const variants = {
  cyan: ['#19f7e8', '#3fc9ff'],
  purple: ['#b45cff', '#7c5cff'],
  green: ['#37f28f', '#9be66d'],
  orange: ['#ff7a3d', '#ffc15a'],
}
const flamePath = 'M33.4 17.2c1.1 5.8-1.5 9.5-4.8 12.8-2.6 2.7-4.9 5.2-4.9 9.1 0 5.2 4 8.9 9.1 8.9 5.2 0 9.2-3.7 9.2-8.9 0-3.6-1.8-6.7-4.9-9.7.2 2.9-.9 5.1-3.2 6.5-.9-5.4 1.6-9.7-.5-18.7Z'

const browser = await chromium.launch({ headless: true })
const page = await browser.newPage()

await mkdir(resolve(publicDirectory, 'favicons'), { recursive: true })
await mkdir(resolve(publicDirectory, 'manifests'), { recursive: true })

for (const [name, [primary, secondary]] of Object.entries(variants)) {
  const favicon = createIconSVG(primary, secondary, false)
  const appIcon = createIconSVG(primary, secondary, true)
  const iconDirectory = resolve(publicDirectory, 'icons', name)

  await mkdir(iconDirectory, { recursive: true })
  await writeFile(resolve(publicDirectory, 'favicons', `icon-${name}.svg`), favicon)
  await writeFile(
    resolve(publicDirectory, 'manifests', `manifest-${name}.webmanifest`),
    `${JSON.stringify(createManifest(name), null, 2)}\n`,
  )

  await page.setContent(`<style>html,body{margin:0;width:100%;height:100%;overflow:hidden}svg{display:block;width:100%;height:100%}</style>${appIcon}`)
  for (const [fileName, size] of [['icon-192.png', 192], ['icon-512.png', 512], ['apple-touch-icon.png', 180]]) {
    await page.setViewportSize({ width: size, height: size })
    await page.screenshot({ path: resolve(iconDirectory, fileName) })
  }

  if (name === 'cyan') {
    await writeFile(resolve(publicDirectory, 'favicon.svg'), favicon)
    await writeFile(resolve(publicDirectory, 'manifest.webmanifest'), `${JSON.stringify(createManifest(name), null, 2)}\n`)
    for (const [fileName, size] of [['icon-192.png', 192], ['icon-512.png', 512], ['apple-touch-icon.png', 180]]) {
      await page.setViewportSize({ width: size, height: size })
      await page.screenshot({ path: resolve(publicDirectory, 'icons', fileName) })
    }
  }
}

await browser.close()

function createIconSVG(primary, secondary, withBackground) {
  const background = withBackground ? '  <rect width="64" height="64" fill="#071014" />\n' : ''

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <defs>
    <linearGradient id="flame" x1="22" y1="17" x2="43" y2="48" gradientUnits="userSpaceOnUse">
      <stop stop-color="${primary}" />
      <stop offset="1" stop-color="${secondary}" />
    </linearGradient>
    <filter id="glow" x="8" y="6" width="48" height="52" filterUnits="userSpaceOnUse">
      <feGaussianBlur stdDeviation="3.2" />
    </filter>
  </defs>
${background}  <path d="${flamePath}" fill="${primary}" opacity=".38" filter="url(#glow)" />
  <path d="${flamePath}" fill="url(#flame)" />
</svg>
`
}

function createManifest(name) {
  return {
    name: 'Sparx',
    short_name: 'Sparx',
    description: 'Build lasting progress through focused daily sessions.',
    id: '/',
    start_url: '/',
    scope: '/',
    display: 'standalone',
    orientation: 'portrait-primary',
    background_color: '#071014',
    theme_color: '#071014',
    categories: ['productivity', 'education'],
    icons: [
      {
        src: `/icons/${name}/icon-192.png`,
        sizes: '192x192',
        type: 'image/png',
        purpose: 'any',
      },
      {
        src: `/icons/${name}/icon-512.png`,
        sizes: '512x512',
        type: 'image/png',
        purpose: 'any maskable',
      },
    ],
  }
}
