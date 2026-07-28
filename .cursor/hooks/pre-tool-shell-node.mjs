// preToolUse Shell: Win → PowerShell (chcp 65001 + UTF-8 + registry Path) then cmd; Unix → path_helper. Debug: CURSOR_HOOK_DEBUG=1.

import { Buffer } from 'node:buffer'
import { execSync } from 'node:child_process'
import process from 'node:process'

const W = process.platform === 'win32'

const stdin = () =>
  new Promise((r) => {
    let d = ''
    process.stdin.setEncoding('utf8')
    process.stdin.on('data', (c) => (d += c))
    process.stdin.on('end', () => r(d))
    process.stdin.on('error', () => r(d))
  })

const out = (o) => void process.stdout.write(`${JSON.stringify(o)}\n`)
const log = (m) => {
  if (process.env.CURSOR_HOOK_DEBUG) process.stderr.write(`[pre-tool-shell-node] ${m}\n`)
}

function toWin(s) {
  s = String(s ?? '').trim()
  if (!s || !W) return s
  const m = s.match(/^\/([a-zA-Z]):\/(.*)$/)
  if (m) return `${m[1].toUpperCase()}:\\${m[2].replace(/\//g, '\\')}`
  return /^[a-zA-Z]:\//.test(s) ? s.replace(/\//g, '\\') : s
}

function cwdOf(inp, ti) {
  for (const v of [
    ti.cwd,
    ti.working_directory,
    ti.workingDirectory,
    inp.cwd,
    inp.working_directory,
    inp.workingDirectory,
    inp.workspace_roots?.[0],
    inp.workspaceRoots?.[0],
  ]) {
    const t = v == null ? '' : String(v).trim()
    if (t) return toWin(t)
  }
  try {
    return toWin(process.cwd())
  } catch {
    return ''
  }
}

function wrapWin(cd, cmd) {
  const q = cd.replace(/'/g, "''")
  const ps = [
    'try{cmd.exe /c "chcp 65001>nul"|Out-Null}catch{}',
    '$u=[System.Text.UTF8Encoding]::new($false)',
    '[Console]::OutputEncoding=$u;[Console]::InputEncoding=$u;$OutputEncoding=$u',
    "$env:PYTHONUTF8='1';$env:PYTHONIOENCODING='utf-8'",
    "$env:Path=[Environment]::GetEnvironmentVariable('Path','Machine')+';'+[Environment]::GetEnvironmentVariable('Path','User')",
    `Set-Location -LiteralPath '${q}'`,
    cmd,
  ].join(';')
  return `powershell.exe -NoProfile -NonInteractive -EncodedCommand ${Buffer.from(ps, 'utf16le').toString('base64')}`
}

function wrapUnix(cd, cmd) {
  const q = (s) => s.replace(/'/g, "'\\''")
  try {
    const o = execSync('/usr/libexec/path_helper -s', { encoding: 'utf8', timeout: 3000 })
    const p = o.match(/PATH="([^"]+)"/)?.[1]
    if (p) return `export PATH='${q(p)}' && cd '${q(cd)}' && ${cmd}`
  } catch {}
  return `cd '${q(cd)}' && ${cmd}`
}

let inp = {}
try {
  inp = JSON.parse((await stdin()).replace(/^\uFEFF/, '').trim() || '{}')
} catch {}

const name = inp.tool_name ?? inp.toolName ?? ''
if (!/^shell$/i.test(name) && name !== 'Bash') {
  log(`skip ${name || '?'}`)
  out({ permission: 'allow' })
  process.exit(0)
}

const ti = inp.tool_input ?? inp.toolInput ?? {}
const cmd = ti.command ?? inp.command ?? ''
const cd = cwdOf(inp, ti)
if (!cmd || !cd) {
  log(`skip !cmd=${!cmd} !cd=${!cd}`)
  out({ permission: 'allow' })
  process.exit(0)
}

log(`${W ? 'win' : 'unix'} ${cd}`)
out({
  permission: 'allow',
  updated_input: { ...ti, command: W ? wrapWin(cd, cmd) : wrapUnix(cd, cmd), cwd: cd, working_directory: cd },
})