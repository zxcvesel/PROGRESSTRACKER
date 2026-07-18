import { readFile, mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
import { chromium } from '@playwright/test'

const publicDirectory = resolve('public')
const outputDirectory = resolve(publicDirectory, 'icons')
const source = await readFile(resolve(publicDirectory, 'favicon.svg'), 'utf8')
const browser = await chromium.launch({ headless: true })
const page = await browser.newPage()

await mkdir(outputDirectory, { recursive: true })
await page.setContent(`<style>html,body{margin:0;width:100%;height:100%;overflow:hidden}svg{display:block;width:100%;height:100%}</style>${source}`)

for (const [name, size] of [['icon-192.png', 192], ['icon-512.png', 512], ['apple-touch-icon.png', 180]]) {
  await page.setViewportSize({ width: size, height: size })
  await page.screenshot({ path: resolve(outputDirectory, name) })
}

await browser.close()
