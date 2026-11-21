#!/usr/bin/env node
/**
 * 构建 Mellow 前端，并把产物同步到 theme/dist（GoReleaser 的 before hook）。
 *
 * 为什么需要这一步：theme/theme_embed.go 用 //go:embed all:dist 把 theme/dist
 * 打进二进制（编译时带 -tags embed）。GoReleaser 只负责 go build，
 * 不会自己跑前端构建；没有这一步 -tags embed 会直接编译失败。
 *
 * 用法：
 *   node scripts/prepare-frontend.mjs               # 构建 + 同步（CI / 本地发布用）
 *   node scripts/prepare-frontend.mjs --skip-build  # 只同步已有的 Mellow/dist
 *
 * 说明：
 *   - 已有 Mellow/node_modules 时跳过 npm ci（CI 上首次会装，本地不重复装）；
 *   - 跨平台：Windows 下 npm 是 .cmd，所以走 shell 解析。
 */
import { execFileSync, execSync } from 'node:child_process'
import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const frontDir = resolve(root, 'Mellow')
const buildDir = resolve(frontDir, 'dist')
const embedDir = resolve(root, 'theme', 'dist')
const skipBuild = process.argv.includes('--skip-build')

function run(command, args, cwd) {
  console.log(`[goreleaser] $ ${command} ${args.join(' ')}`)
  // Windows 上 npm 是 npm.cmd：必须经 shell 解析；其它平台直接 execFile，
  // 避免 Node 对「shell:true + args 数组」报 DEP0190
  if (process.platform === 'win32') {
    execSync(`${command} ${args.join(' ')}`, { cwd, stdio: 'inherit' })
  } else {
    execFileSync(command, args, { cwd, stdio: 'inherit' })
  }
}

if (!skipBuild) {
  if (!existsSync(resolve(frontDir, 'node_modules'))) {
    run('npm', ['ci', '--no-fund', '--no-audit'], frontDir)
  }
  run('npm', ['run', 'build'], frontDir)
}

if (!existsSync(resolve(buildDir, 'index.html'))) {
  console.error('[goreleaser] 没找到 Mellow/dist/index.html，前端没有构建成功，终止打包')
  process.exit(1)
}

// 清空再拷：theme/dist 是上一次的产物，残留旧文件会被 //go:embed all:dist 一起打进去
rmSync(embedDir, { recursive: true, force: true })
mkdirSync(embedDir, { recursive: true })
cpSync(buildDir, embedDir, { recursive: true })
console.log(`[goreleaser] 前端产物已同步到 ${embedDir}`)
