import { readdir, readFile, writeFile } from 'node:fs/promises'
import { resolve, relative, sep } from 'node:path'
import { createHash } from 'node:crypto'

const distDirectory = resolve('dist')
const assetsDirectory = resolve(distDirectory, 'assets')
const workerPath = resolve(distDirectory, 'sw.js')
const assetFiles = await readdir(assetsDirectory, { recursive: true, withFileTypes: true })
const assetPaths = assetFiles
  .filter((entry) => entry.isFile())
  .map((entry) => `/${relative(distDirectory, resolve(entry.parentPath, entry.name)).split(sep).join('/')}`)
  .sort()
const buildID = createHash('sha256').update(JSON.stringify(assetPaths)).digest('hex').slice(0, 12)

const workerSource = await readFile(workerPath, 'utf8')
const marker = 'const BUILD_ASSETS = []'
const cacheMarker = '__BUILD_ID__'
if (!workerSource.includes(marker) || !workerSource.includes(cacheMarker)) {
  throw new Error('PWA build markers were not found in dist/sw.js')
}

await writeFile(
  workerPath,
  workerSource
    .replace(cacheMarker, buildID)
    .replace(marker, `const BUILD_ASSETS = ${JSON.stringify(assetPaths)}`),
)
