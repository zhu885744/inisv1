#!/usr/bin/env node
/**
 * 从 CHANGELOG.md 提取某个版本的发布说明，供 GoReleaser 的 --release-notes 使用。
 *
 * 约定：CHANGELOG.md 里每个版本以二级标题开头，例如
 *   ## INIS-cms v1.0.8
 *   ### 新增
 *   - ...
 *   ### 修复
 *   - ...
 * 脚本会截取「该标题 → 下一个 ## 标题之前」的整段内容。
 *
 * 用法：
 *   node scripts/extract-release-notes.mjs                 # 版本取 GITHUB_REF_NAME / GORELEASER_CURRENT_TAG
 *   node scripts/extract-release-notes.mjs v1.0.8          # 显式指定版本
 *   node scripts/extract-release-notes.mjs v1.0.8 -o x.md  # 指定输出文件
 *   node scripts/extract-release-notes.mjs v1.0.8 --allow-missing  # 找不到时退化为占位说明（默认直接失败）
 *
 * 说明：
 *   - 默认写到仓库根目录 release-notes.md（已在 .gitignore 排除，不会弄脏工作区）；
 *   - 找不到对应版本时默认退出码 1：发布说明必须与版本一起维护，
 *     宁可让流水线失败，也不要发出一个没有更新内容的 Release。
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')

const args = process.argv.slice(2)
let version = ''
let outFile = resolve(root, 'release-notes.md')
let allowMissing = false

for (let i = 0; i < args.length; i++) {
  const arg = args[i]
  if (arg === '-o' || arg === '--output') {
    outFile = resolve(root, args[++i] || 'release-notes.md')
  } else if (arg === '--allow-missing') {
    allowMissing = true
  } else if (!arg.startsWith('-')) {
    version = arg
  }
}

// tag 形如 v1.0.8；去掉前缀 v 后只比对数字部分
version = String(version || process.env.GITHUB_REF_NAME || process.env.GORELEASER_CURRENT_TAG || '')
  .trim()
  .replace(/^v/i, '')

if (!version) {
  console.error('[release-notes] 没拿到版本号：请传参，或设置 GITHUB_REF_NAME / GORELEASER_CURRENT_TAG')
  process.exit(1)
}

const changelogPath = resolve(root, 'CHANGELOG.md')
let changelog = ''
try {
  changelog = readFileSync(changelogPath, 'utf8')
} catch {
  console.error(`[release-notes] 读取不到 ${changelogPath}`)
  process.exit(1)
}

const lines = changelog.split(/\r?\n/)
const escape = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
// 版本号后面不能再跟数字或点，避免 1.0.8 误匹配 1.0.80
const headingRe = new RegExp(`^##\\s+.*?v?${escape(version)}(?![\\d.])`, 'i')
const isH2 = (line) => /^##\s+/.test(line)

let start = -1
for (let i = 0; i < lines.length; i++) {
  if (isH2(lines[i]) && headingRe.test(lines[i])) {
    start = i
    break
  }
}

if (start === -1) {
  const headings = lines.filter(isH2).map((line) => line.trim())
  console.error(`[release-notes] CHANGELOG.md 里没有找到 v${version} 的段落`)
  console.error(`[release-notes] 现有版本标题：\n  ${headings.join('\n  ') || '(无)'}`)

  if (!allowMissing) {
    console.error('[release-notes] 请在 CHANGELOG.md 补上该版本段落，或加 --allow-missing 走占位说明')
    process.exit(1)
  }

  writeFileSync(
    outFile,
    `## INIS-cms v${version}\n\n> 本版本的详细更新内容请查看仓库中的 CHANGELOG.md。\n`,
    'utf8'
  )
  console.warn(`[release-notes] 已写入占位说明 → ${outFile}`)
  process.exit(0)
}

let end = lines.length
for (let i = start + 1; i < lines.length; i++) {
  if (isH2(lines[i])) {
    end = i
    break
  }
}

// 去掉段尾多余空行，保留段内格式
const body = lines.slice(start, end).join('\n').replace(/\s+$/, '')

writeFileSync(outFile, `${body}\n`, 'utf8')

const count = body.split('\n').length
console.log(`[release-notes] 已提取 v${version}（${count} 行）→ ${outFile}`)
