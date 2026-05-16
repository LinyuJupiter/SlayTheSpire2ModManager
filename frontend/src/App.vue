<script setup>
import {onMounted, ref, computed, reactive, nextTick} from 'vue'
import {
  GetUIState,
  DetectSteamGameExe,
  PickGameExe,
  SetGameExe,
  ListMods,
  SaveModEdit,
  PickImportArchive,
  ImportModArchive,
  DeleteModEntry,
  DisableMod,
  OpenModsFolder,
  OpenModFolder,
  ExportModFolderZip,
  ActivateModVersion,
  CurrentVersion,
  AboutMarkdown,
  StartBackgroundUpdate,
  GetUpdateDownloadState,
  InstallUpdate,
} from '../wailsjs/go/app/App'
const gameExePath = ref('')
const modsRoot = ref('')
const overview = ref(null)
const loadError = ref('')
const actionError = ref('')
const currentVersion = ref('')
const updateInfo = ref(null)
const updateStatus = ref('')
const updateError = ref('')
const checkingUpdate = ref(false)
const installingUpdate = ref(false)
const updateDownloadState = ref(null)
const updateReadyPromptShown = ref(false)
const settingsOpen = ref(false)
const aboutOpen = ref(false)
const aboutMarkdown = ref('')
const updatePromptOpen = ref(false)
/** 仅导入 Mod 相关错误，显示在「导入 Mod」卡片内 */
const importError = ref('')
const saving = ref(false)
const importPath = ref('')
const editing = ref(null)
/** 删除确认：null 关闭；含 folderName / manifestFile / modName */
const deleteConfirm = ref(null)
const detail = ref(null)
/** 长 description 展开状态 key = folderName + manifestFile */
const descExpanded = reactive({})

const duplicateBanner = computed(() => {
  if (!overview.value?.duplicateIDs?.length) return ''
  return `以下 mod id 重复出现：${overview.value.duplicateIDs.join('、')}（请手动调整相关 JSON 中的 id）`
})

const renderedAboutHtml = computed(() => renderMarkdown(aboutMarkdown.value))

const updateReady = computed(() => !!updateDownloadState.value?.ready)
const updateDownloading = computed(() => !!updateDownloadState.value?.downloading)

/** 已规范布局：按 slug（第一段路径）聚合；否则仍按完整 folderName 聚合 */
function folderGroupKey(m) {
  if (m.layoutNormalized && m.folderName) {
    const parts = String(m.folderName).replace(/\\/g, '/').split('/')
    return parts[0] || m.folderName
  }
  return m.folderName
}

/** 文件夹列展示名：已规范时只显示 slug */
function displayFolderLabel(m) {
  if (m.layoutNormalized && m.folderName) {
    const parts = String(m.folderName).replace(/\\/g, '/').split('/')
    return parts[0] || m.folderName
  }
  return m.folderName
}

/** 打开/导出 使用的 mods 子路径（已规范时用 slug 目录） */
function openFolderTargetPath(m) {
  return displayFolderLabel(m)
}

function tryParseUintPrefix(s) {
  s = String(s || '').trim()
  let j = 0
  while (j < s.length && s[j] >= '0' && s[j] <= '9') j++
  if (j === 0) return { ok: false, val: 0 }
  const val = parseInt(s.slice(0, j), 10)
  return { ok: true, val: Number.isFinite(val) ? val : 0 }
}

/** 与后端 CompareModVersionStrings 一致：段内数值前缀优先，再字典序 */
function compareModVersionStrings(a, b) {
  a = String(a || '')
    .trim()
    .replace(/^v/i, '')
  b = String(b || '')
    .trim()
    .replace(/^v/i, '')
  if (a === b) return 0
  const partsA = a.split('.')
  const partsB = b.split('.')
  const n = Math.max(partsA.length, partsB.length)
  for (let i = 0; i < n; i++) {
    const sa = partsA[i] ?? ''
    const sb = partsB[i] ?? ''
    const ia = tryParseUintPrefix(sa)
    const ib = tryParseUintPrefix(sb)
    if (ia.ok && ib.ok) {
      if (ia.val !== ib.val) return ia.val > ib.val ? 1 : -1
      continue
    }
    if (sa < sb) return -1
    if (sa > sb) return 1
  }
  if (a < b) return -1
  if (a > b) return 1
  return 0
}

function escapeHtml(s) {
  return String(s || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function renderInlineMarkdown(s) {
  return escapeHtml(s)
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/\*([^*]+)\*/g, '<em>$1</em>')
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>')
}

function renderMarkdown(markdown) {
  const lines = String(markdown || '').replace(/\r\n/g, '\n').split('\n')
  const out = []
  let listOpen = false

  function closeList() {
    if (listOpen) {
      out.push('</ul>')
      listOpen = false
    }
  }

  for (const raw of lines) {
    const line = raw.trim()
    if (!line) {
      closeList()
      continue
    }
    const heading = /^(#{1,3})\s+(.+)$/.exec(line)
    if (heading) {
      closeList()
      const level = heading[1].length
      out.push(`<h${level}>${renderInlineMarkdown(heading[2])}</h${level}>`)
      continue
    }
    const li = /^[-*]\s+(.+)$/.exec(line)
    if (li) {
      if (!listOpen) {
        out.push('<ul>')
        listOpen = true
      }
      out.push(`<li>${renderInlineMarkdown(li[1])}</li>`)
      continue
    }
    closeList()
    out.push(`<p>${renderInlineMarkdown(line)}</p>`)
  }
  closeList()
  return out.join('')
}

function pickRepresentativeMod(group) {
  const on = group.find((x) => !x.disabled)
  if (on) return on
  const sorted = [...group].sort((a, b) =>
    compareModVersionStrings(b.manifest?.version, a.manifest?.version)
  )
  return sorted[0] || group[0]
}

/** 已规范：同一 slug 多版本合并为一行，仅展示当前启用版本的信息；未规范仍一行一个目录 */
const collapsedTableRows = computed(() => {
  const mods = overview.value?.mods ?? []
  const out = []
  let i = 0
  while (i < mods.length) {
    const m = mods[i]
    if (!m.layoutNormalized) {
      out.push({ mod: m, groupMods: [m], normalizedGroup: false })
      i++
      continue
    }
    const key = folderGroupKey(m)
    let j = i + 1
    while (j < mods.length && folderGroupKey(mods[j]) === key) {
      j++
    }
    const group = mods.slice(i, j)
    out.push({
      mod: pickRepresentativeMod(group),
      groupMods: group,
      normalizedGroup: true,
    })
    i = j
  }
  return out
})

function collapsedRowKey(item) {
  return item.normalizedGroup ? folderGroupKey(item.mod) : rowKey(item.mod)
}

function versionOptionsForGroup(anchorMod) {
  const mods = overview.value?.mods ?? []
  const key = folderGroupKey(anchorMod)
  const g = mods.filter((x) => folderGroupKey(x) === key)
  g.sort((a, b) => {
    const c = compareModVersionStrings(a.manifest?.version, b.manifest?.version)
    if (c !== 0) return -c
    return rowKey(a).localeCompare(rowKey(b))
  })
  return g.map((mod) => ({
    k: rowKey(mod),
    label: mod.manifest?.version || versionFolderFallback(mod.folderName),
    folderName: mod.folderName,
    manifestFile: mod.manifestFile,
    disabled: mod.disabled,
  }))
}

function versionFolderFallback(folderName) {
  const p = String(folderName).replace(/\\/g, '/').split('/')
  return p.length >= 2 ? p[p.length - 1] : folderName
}

function selectedVersionKeyForGroup(anchorMod) {
  const opts = versionOptionsForGroup(anchorMod)
  const on = opts.find((o) => !o.disabled)
  return (on || opts[0])?.k ?? ''
}

async function onVersionSelectFromGroup(ev, anchorMod) {
  const sel = ev.target.value
  if (!sel || saving.value) return
  const cur = selectedVersionKeyForGroup(anchorMod)
  if (sel === cur) return
  const z = sel.indexOf('\u0000')
  if (z < 0) return
  const folderName = sel.slice(0, z)
  const manifestFile = sel.slice(z + 1)
  actionError.value = ''
  saving.value = true
  try {
    await ActivateModVersion(folderName, manifestFile)
    await refreshMods()
  } catch (e) {
    actionError.value = String(e)
    ev.target.value = cur
  } finally {
    saving.value = false
  }
}

function manifestJSON(m) {
  if (!m?.manifest) return ''
  return JSON.stringify(m.manifest, null, 2)
}

function rowKey(m) {
  return `${m.folderName}\u0000${m.manifestFile}`
}

/** 折叠且正文超出 3 行时为 true（由 DOM 测量） */
const descOverflowWhenCollapsed = reactive({})
const descElRefs = new Map()

function isDescExpanded(m) {
  return !!descExpanded[rowKey(m)]
}

function bindDescEl(el, m) {
  const k = rowKey(m)
  if (!el) {
    descElRefs.delete(k)
    return
  }
  descElRefs.set(k, el)
  measureDescOverflow(k)
}

function measureDescOverflow(k) {
  nextTick(() => {
    requestAnimationFrame(() => {
      const el = descElRefs.get(k)
      if (!el) return
      if (descExpanded[k]) return
      descOverflowWhenCollapsed[k] = el.scrollHeight > el.clientHeight + 1
    })
  })
}

/** 已展开时始终显示「收起」；折叠时仅在实际超出 3 行时显示「展开」 */
function showDescToggleButton(m) {
  const k = rowKey(m)
  if (descExpanded[k]) return true
  return !!descOverflowWhenCollapsed[k]
}

function toggleDescExpand(m) {
  const k = rowKey(m)
  descExpanded[k] = !descExpanded[k]
  measureDescOverflow(k)
}

async function onToggleMod(row) {
  if (saving.value) return
  actionError.value = ''
  saving.value = true
  try {
    if (row.disabled) {
      await ActivateModVersion(row.folderName, row.manifestFile)
    } else {
      await DisableMod(row.folderName, row.manifestFile)
    }
    await refreshMods()
  } catch (e) {
    actionError.value = String(e)
  } finally {
    saving.value = false
  }
}

async function refreshState() {
  loadError.value = ''
  try {
    const st = await GetUIState()
    gameExePath.value = st.gameExePath || ''
    modsRoot.value = st.modsRoot || ''
    if (st.configOK) {
      await refreshMods()
    } else {
      overview.value = null
    }
  } catch (e) {
    loadError.value = String(e)
  }
}

async function refreshMods() {
  loadError.value = ''
  for (const k of Object.keys(descOverflowWhenCollapsed)) {
    delete descOverflowWhenCollapsed[k]
  }
  try {
    overview.value = await ListMods()
  } catch (e) {
    overview.value = {mods: [], duplicateIDs: [], modsDir: ''}
    loadError.value = String(e)
  }
}

async function loadCurrentVersion() {
  try {
    currentVersion.value = await CurrentVersion()
  } catch {
    currentVersion.value = ''
  }
}

let updatePollTimer = null

function stopUpdatePolling() {
  if (updatePollTimer) {
    clearInterval(updatePollTimer)
    updatePollTimer = null
  }
}

function applyUpdateDownloadState(st, { showNoUpdate = false } = {}) {
  updateDownloadState.value = st
  checkingUpdate.value = !!st?.checking
  if (st?.info) {
    updateInfo.value = st.info
  }
  if (st?.error) {
    updateError.value = st.error
    return
  }
  if (st?.downloading) {
    updateStatus.value = `发现新版本 v${st.info?.latestVersion}，正在后台下载更新包…`
    return
  }
  if (st?.ready) {
    updateStatus.value = `更新包 v${st.info?.latestVersion} 已下载完成`
    if (!updateReadyPromptShown.value) {
      updateReadyPromptShown.value = true
      updatePromptOpen.value = true
    }
    return
  }
  if (st?.hasUpdate) {
    updateStatus.value = `发现新版本 v${st.info?.latestVersion}`
    return
  }
  if (showNoUpdate) {
    updateStatus.value = '当前已是最新版本'
  }
}

async function pollUpdateDownloadState() {
  try {
    const st = await GetUpdateDownloadState()
    applyUpdateDownloadState(st)
    if (!st.checking && !st.downloading) {
      stopUpdatePolling()
    }
  } catch (e) {
    updateError.value = String(e)
    stopUpdatePolling()
  }
}

function startUpdatePolling() {
  if (updatePollTimer) return
  updatePollTimer = setInterval(pollUpdateDownloadState, 2000)
}

async function startBackgroundUpdate({ manual = false } = {}) {
  if (manual) {
    updateError.value = ''
    updateStatus.value = ''
  }
  try {
    const st = await StartBackgroundUpdate()
    applyUpdateDownloadState(st, { showNoUpdate: manual })
    if (st.checking || st.downloading) {
      startUpdatePolling()
    }
  } catch (e) {
    updateError.value = String(e)
  }
}

async function installUpdate() {
  updateError.value = ''
  installingUpdate.value = true
  updateStatus.value = '正在安装更新，完成后会自动重启…'
  try {
    await InstallUpdate()
  } catch (e) {
    updateError.value = String(e)
    installingUpdate.value = false
  }
}

function openSettings() {
  settingsOpen.value = true
}

function closeSettings() {
  settingsOpen.value = false
}

async function openAbout() {
  updateError.value = ''
  try {
    aboutMarkdown.value = await AboutMarkdown()
    aboutOpen.value = true
  } catch (e) {
    updateError.value = String(e)
  }
}

function closeAbout() {
  aboutOpen.value = false
}

async function onDetectSteam() {
  actionError.value = ''
  try {
    const p = await DetectSteamGameExe()
    if (p) gameExePath.value = p
    else actionError.value = '未在 Steam 库中找到游戏（可手动选择主程序）'
  } catch (e) {
    actionError.value = String(e)
  }
}

async function onBrowseExe() {
  actionError.value = ''
  try {
    const p = await PickGameExe()
    if (p) gameExePath.value = p
  } catch (e) {
    actionError.value = String(e)
  }
}

async function applyGamePath() {
  actionError.value = ''
  saving.value = true
  try {
    await SetGameExe(gameExePath.value.trim())
    await refreshState()
  } catch (e) {
    actionError.value = String(e)
  } finally {
    saving.value = false
  }
}

async function openModsDirectory() {
  actionError.value = ''
  try {
    await OpenModsFolder()
  } catch (e) {
    actionError.value = String(e)
  }
}

async function onOpenModFolder(folderName) {
  actionError.value = ''
  try {
    await OpenModFolder(folderName)
  } catch (e) {
    actionError.value = String(e)
  }
}

async function onExportModFolderZip(folderName) {
  actionError.value = ''
  try {
    await ExportModFolderZip(folderName)
  } catch (e) {
    actionError.value = String(e)
  }
}

function openDetail(row) {
  detail.value = row
}

function closeDetail() {
  detail.value = null
}

function folderOuterForEdit(m) {
  if (m.layoutNormalized && m.folderName) {
    const parts = String(m.folderName).replace(/\\/g, '/').split('/')
    return parts[0] || m.folderName
  }
  return m.folderName
}

function innerSegmentForFolder(folderName, layoutNormalized) {
  if (!layoutNormalized || !folderName) return ''
  const p = String(folderName).replace(/\\/g, '/').split('/')
  return p.length >= 2 ? p.slice(1).join('/') : ''
}

function openEdit(row) {
  actionError.value = ''
  editing.value = {
    folderName: row.folderName,
    newFolderName: folderOuterForEdit(row),
    manifestFile: row.manifestFile,
    layoutNormalized: !!row.layoutNormalized,
    id: row.manifest.id,
    name: row.manifest.name,
    description: row.manifest.description,
    affects_gameplay: row.manifest.affects_gameplay,
  }
}

function closeEdit() {
  editing.value = null
}

async function submitEdit() {
  if (!editing.value) return
  actionError.value = ''
  saving.value = true
  try {
    const inner = innerSegmentForFolder(editing.value.folderName, editing.value.layoutNormalized)
    const newOuter = editing.value.newFolderName.trim()
    const newFull = inner ? `${newOuter}/${inner}` : newOuter
    await SaveModEdit({
      folderName: editing.value.folderName,
      manifestFile: editing.value.manifestFile,
      newFolderName: newFull,
      layoutNormalized: editing.value.layoutNormalized,
      id: editing.value.id,
      name: editing.value.name,
      description: editing.value.description,
      affects_gameplay: editing.value.affects_gameplay,
    })
    closeEdit()
    await refreshMods()
  } catch (e) {
    actionError.value = String(e)
  } finally {
    saving.value = false
  }
}

function openDeleteConfirmFromEdit() {
  if (!editing.value) return
  deleteConfirm.value = {
    folderName: editing.value.folderName,
    manifestFile: editing.value.manifestFile,
    modName: (editing.value.name || '').trim() || editing.value.id,
  }
}

function closeDeleteConfirm() {
  deleteConfirm.value = null
}

async function runDelete(deleteEntireSlug) {
  if (!deleteConfirm.value || saving.value) return
  const { folderName, manifestFile } = deleteConfirm.value
  actionError.value = ''
  saving.value = true
  try {
    await DeleteModEntry(folderName, manifestFile, deleteEntireSlug)
    closeDeleteConfirm()
    closeEdit()
    if (detail.value?.folderName === folderName && detail.value?.manifestFile === manifestFile) {
      closeDetail()
    }
    await refreshMods()
  } catch (e) {
    actionError.value = String(e)
  } finally {
    saving.value = false
  }
}

async function onBrowseImport() {
  importError.value = ''
  try {
    const p = await PickImportArchive()
    if (p) importPath.value = p
  } catch (e) {
    importError.value = String(e)
  }
}

async function doImport() {
  importError.value = ''
  if (!importPath.value.trim()) {
    importError.value = '请先选择压缩包'
    return
  }
  saving.value = true
  try {
    await ImportModArchive(importPath.value.trim())
    importPath.value = ''
    await refreshMods()
  } catch (e) {
    importError.value = String(e)
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await loadCurrentVersion()
  await refreshState()
  startBackgroundUpdate()
})
</script>

<template>
  <div class="layout">
    <header class="header">
      <div class="header-top">
        <div class="header-brand">
          <h1>杀戮尖塔 2 Mod 管理器</h1>
          <span class="header-badge">v{{ currentVersion || 'dev' }}</span>
        </div>
        <button type="button" class="settings-btn" title="设置" aria-label="设置" @click="openSettings">
          ⚙
        </button>
      </div>
      <p class="muted header-desc">
        配置游戏路径后，将扫描游戏目录下的 <code>mods</code> 文件夹并列出符合格式的 mod。
      </p>
    </header>

    <section class="card">
      <h2>游戏路径</h2>
      <div class="row">
        <input v-model="gameExePath" class="grow" type="text" placeholder="SlayTheSpire2.exe 完整路径" />
        <button type="button" @click="onDetectSteam">从 Steam 检测</button>
        <button type="button" @click="onBrowseExe">浏览…</button>
        <button type="button" class="primary" :disabled="saving" @click="applyGamePath">保存并初始化 mods</button>
      </div>
      <div v-if="modsRoot" class="meta-row">
        <span class="meta-path"
          >mods 目录：<code>{{ modsRoot }}</code></span
        >
        <button type="button" @click="openModsDirectory">打开文件夹</button>
      </div>
      <div v-if="loadError" class="msg err">{{ loadError }}</div>
      <div v-if="actionError" class="msg err">{{ actionError }}</div>
    </section>

    <section v-if="duplicateBanner" class="msg warn">{{ duplicateBanner }}</section>

    <section class="card">
      <div class="row spread">
        <h2>已安装的 Mod</h2>
        <button type="button" :disabled="saving" @click="refreshMods">刷新列表</button>
      </div>
      <div v-if="!overview && !loadError" class="muted">请先保存有效的游戏路径。</div>
      <div v-else-if="overview && overview.mods.length === 0" class="muted">
        未找到符合 manifest 格式的 mod（需包含 id、has_pck、has_dll、affects_gameplay 等字段的 JSON；已关闭的 mod 仅保留 *.json.bak 也会被识别）。
      </div>
      <div v-else-if="overview && overview.mods.length" class="table-wrap">
        <table class="grid">
          <colgroup>
            <col class="cw-folder" />
            <col class="cw-id" />
            <col class="cw-name" />
            <col class="cw-author" />
            <col class="cw-ver" />
            <col class="cw-desc" />
            <col class="cw-switch" />
            <col class="cw-avail" />
            <col class="cw-act" />
          </colgroup>
          <thead>
            <tr>
              <th>文件夹</th>
              <th>id</th>
              <th>name</th>
              <th>author</th>
              <th>version</th>
              <th class="col-desc">description</th>
              <th>启用</th>
              <th class="col-avail">可用性</th>
              <th class="col-actions">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in collapsedTableRows"
              :key="collapsedRowKey(item)"
              :class="{ dim: item.mod.disabled }"
            >
              <td class="folder-cell cell-wrap">
                <div class="folder-name mono">
                  {{ displayFolderLabel(item.mod) }}
                </div>
                <div class="folder-actions">
                  <button type="button" @click.stop="onOpenModFolder(openFolderTargetPath(item.mod))">
                    打开文件夹
                  </button>
                  <button type="button" @click.stop="onExportModFolderZip(openFolderTargetPath(item.mod))">
                    导出 Zip…
                  </button>
                </div>
              </td>
              <td class="cell-wrap">
                <span class="mono">{{ item.mod.manifest.id }}</span>
              </td>
              <td class="cell-wrap">{{ item.mod.manifest.name }}</td>
              <td class="cell-wrap">{{ item.mod.manifest.author }}</td>
              <td v-if="item.normalizedGroup" class="cell-wrap ver-td">
                <select
                  class="ver-select"
                  :value="selectedVersionKeyForGroup(item.mod)"
                  :disabled="saving"
                  @change="onVersionSelectFromGroup($event, item.mod)"
                >
                  <option
                    v-for="opt in versionOptionsForGroup(item.mod)"
                    :key="opt.k"
                    :value="opt.k"
                  >
                    {{ opt.label }}
                  </option>
                </select>
              </td>
              <td v-else class="cell-wrap">{{ item.mod.manifest.version }}</td>
              <td class="desc-cell col-desc">
                <div class="desc-row">
                  <div
                    :ref="(el) => bindDescEl(el, item.mod)"
                    class="desc-body"
                    :class="{ 'desc-clamped': !isDescExpanded(item.mod) }"
                  >
                    {{ item.mod.manifest.description }}
                  </div>
                  <button
                    v-if="showDescToggleButton(item.mod)"
                    type="button"
                    class="btn-link btn-desc-inline"
                    @click.stop="toggleDescExpand(item.mod)"
                  >
                    {{ isDescExpanded(item.mod) ? '收起' : '展开' }}
                  </button>
                </div>
              </td>
              <td class="col-switch">
                <label
                  class="switch"
                  :title="item.mod.disabled ? '已关闭，点击启用' : '运行中，点击关闭'"
                >
                  <input
                    type="checkbox"
                    role="switch"
                    :checked="!item.mod.disabled"
                    :disabled="saving"
                    @click.prevent="onToggleMod(item.mod)"
                  />
                  <span class="slider" aria-hidden="true" />
                </label>
              </td>
              <td class="avail-cell col-avail cell-wrap">
                <span v-if="item.mod.available" class="dot" title="id 唯一且依赖已满足" />
                <div v-else class="avail-bad">
                  <span v-if="!item.mod.idUnique" class="avail-line"
                    >id 与 {{ item.mod.conflictWith.join('、') }} 重复</span
                  >
                  <span
                    v-if="item.mod.missingDependencies && item.mod.missingDependencies.length"
                    class="avail-line"
                    >缺少依赖：{{ item.mod.missingDependencies.join('、') }}</span
                  >
                </div>
              </td>
              <td class="actions">
                <button type="button" @click="openDetail(item.mod)">详情</button>
                <button type="button" @click="openEdit(item.mod)">编辑</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="card">
      <h2>导入 Mod</h2>
      <p class="muted">
        支持 .zip / .rar。ZIP 会按 UTF-8 标记自动识别中文文件名；国内常见 GBK 编码的 zip 也会尝试修正。.rar 若中文乱码可改用 zip。
        导入位置由 manifest 的 <strong>id</strong> 决定：若该 id 已存在则并入其目录下新版本子文件夹；否则在 mods 下新建以 id 命名的目录。根目录可直接放 manifest、dll、pck 等文件；若只有<strong>一个顶层文件夹</strong>，会在其下<strong>递归查找</strong>第一个含有 manifest 的子目录作为 mod 根（支持多一层或数层嵌套）。若根下有多于一个<strong>并列子文件夹</strong>，则每个顶层文件夹内各找一份 mod 分别导入。
      </p>
      <div class="row">
        <input v-model="importPath" class="grow" type="text" placeholder="压缩包路径" readonly />
        <button type="button" @click="onBrowseImport">选择压缩包…</button>
      </div>
      <div class="row">
        <button type="button" class="primary" :disabled="saving" @click="doImport">导入到 mods</button>
      </div>
      <div v-if="importError" class="msg err import-err">{{ importError }}</div>
    </section>

    <div v-if="settingsOpen" class="overlay" @click.self="closeSettings">
      <div class="dialog dialog-settings">
        <h3>设置</h3>
        <div class="settings-list">
          <button
            type="button"
            :disabled="checkingUpdate || updateDownloading || installingUpdate"
            @click="startBackgroundUpdate({ manual: true })"
          >
            {{ checkingUpdate ? '检查中…' : updateDownloading ? '下载中…' : '检查更新' }}
          </button>
          <button type="button" @click="openAbout">关于</button>
        </div>
        <div v-if="updateStatus" class="msg ok">{{ updateStatus }}</div>
        <div v-if="updateError" class="msg err">{{ updateError }}</div>
        <div v-if="updateInfo?.hasUpdate" class="update-box">
          <p class="muted small">
            当前版本 v{{ updateInfo.currentVersion }}，最新版本 v{{ updateInfo.latestVersion }}。
          </p>
          <div class="row end tight">
            <button type="button" class="primary" :disabled="!updateReady || installingUpdate" @click="installUpdate">
              {{
                installingUpdate
                  ? '安装中…'
                  : updateReady
                    ? '安装并重启'
                    : updateDownloading
                      ? '后台下载中…'
                      : '等待下载完成'
              }}
            </button>
          </div>
        </div>
        <div class="row end">
          <button type="button" @click="closeSettings">关闭</button>
        </div>
      </div>
    </div>

    <div v-if="aboutOpen" class="overlay overlay-front" @click.self="closeAbout">
      <div class="dialog dialog-wide">
        <h3>关于</h3>
        <div class="about-markdown" v-html="renderedAboutHtml"></div>
        <div class="row end">
          <button type="button" @click="closeAbout">关闭</button>
        </div>
      </div>
    </div>

    <div v-if="updatePromptOpen" class="overlay overlay-front" @click.self="updatePromptOpen = false">
      <div class="dialog dialog-update">
        <h3>更新包已下载完成</h3>
        <p class="muted">
          当前版本 v{{ updateInfo?.currentVersion }}，最新版本 v{{ updateInfo?.latestVersion }}。是否立即安装并重启？
        </p>
        <div v-if="updateError" class="msg err">{{ updateError }}</div>
        <div class="row end">
          <button type="button" :disabled="installingUpdate" @click="updatePromptOpen = false">稍后再说</button>
          <button type="button" class="primary" :disabled="installingUpdate" @click="installUpdate">
            {{ installingUpdate ? '安装中…' : '安装并重启' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="detail" class="overlay" @click.self="closeDetail">
      <div class="dialog dialog-wide">
        <h3>Mod 详情</h3>
        <dl class="detail-dl">
          <dt>文件夹名</dt>
          <dd class="mono">{{ displayFolderLabel(detail) }}（{{ detail.folderName }}）</dd>
          <dt>Manifest 文件名</dt>
          <dd class="mono">{{ detail.manifestFile }}</dd>
          <dt>状态</dt>
          <dd>{{ detail.disabled ? '已关闭（json/pck/dll 已加 .bak）' : '运行中' }}</dd>
          <template v-if="detail.layoutNormalized && detail.alternateVersions?.length">
            <dt>其它版本</dt>
            <dd class="muted small-hint">在列表「version」列的下拉框中切换启用版本。</dd>
          </template>
        </dl>
        <p class="detail-hint">以下为 manifest JSON 中的全部字段：</p>
        <pre class="json-block">{{ manifestJSON(detail) }}</pre>
        <div class="row end">
          <button type="button" @click="closeDetail">关闭</button>
        </div>
      </div>
    </div>

    <div v-if="editing" class="overlay" @click.self="closeEdit">
      <div class="dialog dialog-wide">
        <h3>编辑 Mod</h3>
        <div class="field-row">
          <span class="field-label">id（唯一）</span>
          <input v-model="editing.id" class="field-input" type="text" />
        </div>
        <div class="field-row">
          <span class="field-label">name</span>
          <input v-model="editing.name" class="field-input" type="text" />
        </div>
        <div class="field-row">
          <span class="field-label">文件夹名</span>
          <input
            v-model="editing.newFolderName"
            class="field-input"
            type="text"
            placeholder="mods 下最外层目录名（两段式布局时不含版本子文件夹）"
          />
        </div>
        <div class="field-row field-row-top">
          <span class="field-label">description</span>
          <textarea v-model="editing.description" class="field-input" rows="5"></textarea>
        </div>
        <div class="field-row">
          <span class="field-label">affects_gameplay</span>
          <label class="checkbox-inline">
            <input v-model="editing.affects_gameplay" type="checkbox" />
            <span>启用</span>
          </label>
        </div>
        <div class="row between">
          <button type="button" class="danger" :disabled="saving" @click="openDeleteConfirmFromEdit">
            删除此 Mod
          </button>
          <div class="row end tight">
            <button type="button" @click="closeEdit">取消</button>
            <button type="button" class="primary" :disabled="saving" @click="submitEdit">保存</button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="deleteConfirm" class="overlay overlay-front" @click.self="closeDeleteConfirm">
      <div class="dialog dialog-delete" role="dialog" aria-modal="true" aria-labelledby="delete-dialog-title">
        <h3 id="delete-dialog-title">删除 Mod</h3>
        <p class="delete-mod-name">{{ deleteConfirm.modName }}</p>
        <p class="muted delete-intro">请选择要删除的范围（此操作不可撤销）：</p>
        <ul class="delete-options">
          <li>
            <strong>仅删除当前版本</strong>：移除当前 manifest 所在的<strong>整个版本目录</strong>（含 json、pck、dll 等）；同一 mod
            的其它版本会保留。
          </li>
          <li>
            <strong>删除全部版本</strong>：删除该 mod 在磁盘上的<strong>整个顶层目录</strong>（slug 下所有已安装版本一并移除）。
          </li>
        </ul>
        <div class="delete-actions">
          <button type="button" :disabled="saving" @click="closeDeleteConfirm">取消</button>
          <button type="button" class="danger-muted" :disabled="saving" @click="runDelete(false)">
            仅删除当前版本
          </button>
          <button type="button" class="danger" :disabled="saving" @click="runDelete(true)">删除全部版本</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.layout {
  --surface: rgba(30, 41, 59, 0.55);
  --surface-border: rgba(148, 163, 184, 0.14);
  --surface-hover: rgba(51, 65, 85, 0.45);
  --ring: rgba(56, 189, 248, 0.55);
  --primary: #38bdf8;
  --primary-deep: #0284c7;
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.35);
  --shadow-md: 0 8px 24px rgba(0, 0, 0, 0.35);
  --shadow-lg: 0 22px 50px rgba(0, 0, 0, 0.45);
  max-width: min(1580px, 98vw);
  margin: 0 auto;
  padding: 28px 28px 72px;
  text-align: left;
}

.header {
  margin-bottom: 28px;
  padding-bottom: 22px;
  border-bottom: 1px solid var(--surface-border);
}

.header-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 10px;
}

.header-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  min-width: 0;
}

.header h1 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
  letter-spacing: -0.03em;
  background: linear-gradient(120deg, #f8fafc 0%, #bae6fd 45%, #7dd3fc 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.header-badge {
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 4px 8px;
  border-radius: 6px;
  border: 1px solid rgba(56, 189, 248, 0.35);
  background: rgba(56, 189, 248, 0.12);
  color: #7dd3fc;
}

.header-desc {
  margin: 0;
  max-width: 52rem;
  line-height: 1.55;
}

.settings-btn {
  width: 40px;
  height: 40px;
  padding: 0;
  border-radius: 999px;
  font-size: 1.1rem;
  line-height: 1;
  flex: 0 0 auto;
  background: rgba(15, 23, 42, 0.55);
}

.muted {
  color: rgba(226, 232, 240, 0.72);
  font-size: 0.92rem;
  margin: 0 0 12px;
}

.card {
  position: relative;
  background: var(--surface);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid var(--surface-border);
  border-radius: 14px;
  padding: 20px 22px;
  margin-bottom: 18px;
  box-shadow: var(--shadow-sm);
  transition:
    border-color 0.25s ease,
    box-shadow 0.25s ease,
    background 0.25s ease;
}

.card:hover {
  border-color: rgba(148, 163, 184, 0.22);
  box-shadow: var(--shadow-md);
}

.card h2 {
  margin: 0 0 16px;
  font-size: 1.08rem;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: #f1f5f9;
  display: flex;
  align-items: center;
  gap: 10px;
}

.card h2::before {
  content: '';
  width: 4px;
  height: 1.1em;
  border-radius: 3px;
  background: linear-gradient(180deg, #38bdf8, #6366f1);
  flex-shrink: 0;
}
.row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 10px;
}
.row.spread {
  justify-content: space-between;
  align-items: center;
}
.row.end {
  justify-content: flex-end;
  margin-top: 12px;
}
.row.between {
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 14px;
}
.row.tight {
  margin-top: 0;
  gap: 8px;
}
.grow {
  flex: 1;
  min-width: 200px;
}
label.grow {
  display: block;
}

button {
  position: relative;
  padding: 9px 16px;
  border-radius: 9px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  background: rgba(255, 255, 255, 0.07);
  color: #f1f5f9;
  cursor: pointer;
  white-space: nowrap;
  font-size: 0.9rem;
  font-weight: 600;
  letter-spacing: 0.01em;
  user-select: none;
  transition:
    transform 0.14s cubic-bezier(0.33, 1, 0.68, 1),
    box-shadow 0.2s ease,
    background 0.2s ease,
    border-color 0.2s ease,
    color 0.2s ease,
    opacity 0.2s ease;
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.06) inset;
}

button:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.11);
  border-color: rgba(186, 230, 253, 0.28);
  transform: translateY(-1px);
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.08) inset,
    0 6px 16px rgba(0, 0, 0, 0.28);
}

button:active:not(:disabled) {
  transform: translateY(0) scale(0.97);
  box-shadow: 0 1px 0 rgba(0, 0, 0, 0.2) inset;
}

button:focus-visible {
  outline: 2px solid var(--ring);
  outline-offset: 2px;
}

button.primary {
  border-color: transparent;
  background: linear-gradient(135deg, #0ea5e9 0%, #2563eb 100%);
  color: #fff;
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.2) inset,
    0 4px 14px rgba(14, 165, 233, 0.35);
}

button.primary:hover:not(:disabled) {
  background: linear-gradient(135deg, #38bdf8 0%, #3b82f6 100%);
  border-color: transparent;
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.25) inset,
    0 8px 22px rgba(56, 189, 248, 0.4);
}

button.primary:active:not(:disabled) {
  background: linear-gradient(135deg, #0284c7 0%, #1d4ed8 100%);
}

button.danger {
  border-color: rgba(248, 113, 113, 0.45);
  color: #fecaca;
  background: rgba(239, 68, 68, 0.1);
}

button.danger:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.2);
  border-color: rgba(252, 165, 165, 0.55);
  color: #fff;
}

button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}
.meta-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 14px;
  margin-top: 10px;
  font-size: 0.86rem;
  color: rgba(226, 232, 240, 0.78);
}
.meta-path {
  flex: 1;
  min-width: 220px;
}
.meta-path code {
  font-size: 0.88em;
  word-break: break-all;
}
.table-wrap {
  overflow-x: auto;
  margin: 4px -4px 0;
  padding: 4px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.15);
}
.grid {
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
  font-size: 0.92rem;
}
.folder-cell {
  vertical-align: top;
}
.folder-name {
  margin-bottom: 10px;
  line-height: 1.4;
  word-break: break-word;
}
.tag-norm {
  display: inline-block;
  margin-left: 6px;
  padding: 1px 6px;
  font-size: 0.72rem;
  font-weight: 600;
  border-radius: 4px;
  background: rgba(80, 200, 120, 0.2);
  color: rgba(180, 255, 200, 0.95);
  vertical-align: middle;
}
.alt-versions {
  margin: 0;
  padding-left: 1.1rem;
}
.alt-ver-li {
  margin-bottom: 8px;
  list-style: disc;
}
.small-hint {
  margin: 8px 0 0;
  font-size: 0.82rem;
}
.folder-actions {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}
.folder-actions button {
  width: 100%;
  padding: 7px 10px;
  font-size: 0.82rem;
  font-weight: 600;
}
.ver-select {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  background: rgba(15, 23, 42, 0.65);
  color: inherit;
  font-size: 0.86rem;
  cursor: pointer;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    background 0.2s ease;
}
.ver-select:hover:not(:disabled) {
  border-color: rgba(56, 189, 248, 0.35);
}
.ver-select:focus {
  outline: none;
  border-color: rgba(56, 189, 248, 0.55);
  box-shadow: 0 0 0 3px rgba(56, 189, 248, 0.18);
}
.ver-select:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.ver-td {
  vertical-align: top;
}
.cw-folder {
  width: 14%;
}
.cw-id {
  width: 11%;
}
.cw-name {
  width: 11%;
}
.cw-author {
  width: 11%;
}
.cw-ver {
  width: 10%;
}
/* description 列相对更早版本约缩短 1/4 */
.cw-desc {
  width: 24%;
}
.cw-switch {
  width: 5%;
}
.cw-avail {
  width: 6%;
}
.cw-act {
  width: 12%;
}
.cell-wrap {
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-word;
}
.grid th,
.grid td {
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
  padding: 11px 12px;
  text-align: left;
  vertical-align: top;
}

.grid thead th {
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: rgba(148, 163, 184, 0.95);
  background: rgba(15, 23, 42, 0.5);
  border-bottom: 1px solid rgba(148, 163, 184, 0.18);
}

.grid tbody tr {
  transition: background 0.16s ease;
}

.grid tbody tr:hover td {
  background: rgba(56, 189, 248, 0.06);
}

.grid tr.dim td {
  opacity: 0.72;
}
.col-actions {
  min-width: 200px;
}
.actions {
  white-space: nowrap;
}
.actions button {
  margin-right: 6px;
  margin-bottom: 4px;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.85em;
}
.desc-cell {
  vertical-align: top;
  min-width: 0;
  overflow-wrap: anywhere;
}
.desc-row {
  display: flex;
  flex-direction: row;
  align-items: flex-end;
  gap: 4px;
  min-width: 0;
}
.desc-body {
  flex: 1;
  min-width: 0;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}
.desc-body.desc-clamped {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  overflow: hidden;
}
.btn-link {
  display: inline-block;
  margin-top: 6px;
  padding: 5px 10px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: #7dd3fc;
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
  text-decoration: none;
  transition:
    background 0.18s ease,
    color 0.18s ease,
    transform 0.12s ease;
}

.btn-desc-inline {
  flex-shrink: 0;
  align-self: flex-end;
  margin-top: 0;
  white-space: nowrap;
}

.btn-link:hover {
  color: #e0f2fe;
  background: rgba(56, 189, 248, 0.12);
}

.btn-link:active {
  transform: scale(0.96);
}

.btn-link:focus-visible {
  outline: 2px solid var(--ring);
  outline-offset: 2px;
}
.col-switch {
  vertical-align: middle;
  white-space: nowrap;
}
.switch {
  position: relative;
  display: inline-block;
  width: 46px;
  height: 28px;
  cursor: pointer;
}

.switch:active input:not(:disabled) ~ .slider {
  filter: brightness(0.92);
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}
.switch .slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: linear-gradient(180deg, #64748b, #475569);
  border-radius: 28px;
  transition:
    background 0.22s ease,
    box-shadow 0.22s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.35) inset;
}
.switch .slider:before {
  position: absolute;
  content: '';
  height: 22px;
  width: 22px;
  left: 3px;
  bottom: 3px;
  background: linear-gradient(180deg, #fff, #e2e8f0);
  border-radius: 50%;
  transition: transform 0.22s cubic-bezier(0.33, 1, 0.68, 1);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.35);
}
.switch input:checked + .slider {
  background: linear-gradient(180deg, #22c55e, #16a34a);
  box-shadow: 0 0 12px rgba(34, 197, 94, 0.35);
}
.switch input:checked + .slider:before {
  transform: translateX(18px);
}
.switch input:focus-visible + .slider {
  outline: 2px solid var(--ring);
  outline-offset: 3px;
}
.switch input:disabled + .slider {
  opacity: 0.55;
  cursor: not-allowed;
}
.avail-cell {
  font-size: 0.78rem;
  line-height: 1.35;
  vertical-align: middle;
  word-break: break-word;
}
.avail-bad {
  color: #fecaca;
}
.avail-line {
  display: block;
}
.avail-line + .avail-line {
  margin-top: 4px;
}
.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #22c55e;
  box-shadow:
    0 0 0 2px rgba(34, 197, 94, 0.25),
    0 0 10px rgba(34, 197, 94, 0.35);
  vertical-align: middle;
}
input[type='text'],
textarea {
  width: 100%;
  padding: 10px 12px;
  border-radius: 9px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: rgba(15, 23, 42, 0.55);
  color: #f1f5f9;
  margin-top: 6px;
  box-sizing: border-box;
  font-size: 0.92rem;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    background 0.2s ease;
}

input[type='text']:hover,
textarea:hover {
  border-color: rgba(148, 163, 184, 0.32);
}

input[type='text']:read-only {
  cursor: default;
  opacity: 0.92;
}

input[type='text']:focus,
textarea:focus {
  outline: none;
  border-color: rgba(56, 189, 248, 0.55);
  box-shadow: 0 0 0 3px rgba(56, 189, 248, 0.15);
  background: rgba(15, 23, 42, 0.75);
}

.meta {
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.75);
}
code {
  font-size: 0.85em;
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(15, 23, 42, 0.85);
  border: 1px solid rgba(148, 163, 184, 0.15);
  color: #bae6fd;
}
.msg {
  margin-top: 12px;
  padding: 12px 14px;
  border-radius: 10px;
  font-size: 0.9rem;
  line-height: 1.45;
}
.msg.err {
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(248, 113, 113, 0.35);
  color: #fecaca;
}
.msg.ok {
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(74, 222, 128, 0.32);
  color: #bbf7d0;
}
.small {
  font-size: 0.86rem;
}
.import-err {
  margin-top: 12px;
}
.msg.warn {
  background: rgba(234, 179, 8, 0.1);
  border: 1px solid rgba(250, 204, 21, 0.35);
  margin-bottom: 18px;
  color: #fef3c7;
}
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(2, 6, 23, 0.72);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
  padding: 20px;
  animation: overlayIn 0.22s ease;
}

@keyframes overlayIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.dialog {
  width: min(520px, 100%);
  background: linear-gradient(165deg, rgba(30, 41, 59, 0.98) 0%, rgba(15, 23, 42, 0.99) 100%);
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 16px;
  padding: 22px 24px;
  box-shadow: var(--shadow-lg);
  animation: dialogIn 0.28s cubic-bezier(0.33, 1, 0.68, 1);
}

@keyframes dialogIn {
  from {
    opacity: 0;
    transform: translateY(12px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}
.dialog-wide {
  width: min(640px, 100%);
  max-height: min(90vh, 900px);
  overflow: auto;
}
.dialog-settings {
  width: min(520px, 100%);
}
.dialog-update {
  width: min(500px, 100%);
}
.dialog h3 {
  margin: 0 0 16px;
  font-size: 1.15rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #f8fafc;
}
.field-row {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 12px;
}
.field-row-top {
  align-items: flex-start;
}
.field-label {
  flex: 0 0 140px;
  text-align: right;
  font-size: 0.9rem;
  color: rgba(255, 255, 255, 0.85);
}
.field-input {
  flex: 1;
  min-width: 0;
}
.field-row .field-input {
  margin-top: 0;
}
.checkbox-inline {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}
.detail-dl {
  display: grid;
  grid-template-columns: 120px 1fr;
  gap: 8px 12px;
  margin: 0 0 12px;
  font-size: 0.9rem;
}
.detail-dl dt {
  margin: 0;
  color: rgba(255, 255, 255, 0.55);
}
.detail-dl dd {
  margin: 0;
}
.detail-hint {
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.55);
  margin: 0 0 6px;
}
.json-block {
  margin: 0;
  padding: 14px 16px;
  border-radius: 10px;
  background: rgba(2, 6, 23, 0.55);
  border: 1px solid rgba(148, 163, 184, 0.12);
  font-size: 0.78rem;
  line-height: 1.45;
  overflow-x: auto;
  white-space: pre;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
.settings-list {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 10px;
}
.settings-list button {
  min-height: 44px;
  width: 100%;
}
.update-box {
  margin-top: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid rgba(56, 189, 248, 0.2);
  background: rgba(14, 165, 233, 0.08);
}
.about-markdown {
  margin: 0;
  padding: 16px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  background: rgba(2, 6, 23, 0.48);
  color: rgba(241, 245, 249, 0.92);
  word-break: break-word;
  font-size: 0.92rem;
  line-height: 1.65;
}
.about-markdown :deep(h1),
.about-markdown :deep(h2),
.about-markdown :deep(h3) {
  margin: 0 0 12px;
  color: #f8fafc;
  line-height: 1.25;
}
.about-markdown :deep(h1) {
  font-size: 1.3rem;
}
.about-markdown :deep(h2) {
  font-size: 1.12rem;
}
.about-markdown :deep(h3) {
  font-size: 1rem;
}
.about-markdown :deep(p) {
  margin: 0 0 10px;
}
.about-markdown :deep(ul) {
  margin: 0 0 12px;
  padding-left: 1.25rem;
}
.about-markdown :deep(li + li) {
  margin-top: 6px;
}
.about-markdown :deep(code) {
  font-size: 0.86em;
}
.about-markdown :deep(a) {
  color: #7dd3fc;
}
.overlay-front {
  z-index: 60;
}
.dialog-delete {
  width: min(500px, 100%);
}
.dialog-delete h3 {
  margin-bottom: 10px;
}
.delete-mod-name {
  margin: 0 0 6px;
  font-size: 1.02rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.95);
}
.delete-intro {
  margin: 0 0 12px;
}
.delete-options {
  margin: 0 0 18px;
  padding-left: 1.15rem;
  line-height: 1.55;
  font-size: 0.9rem;
  color: rgba(255, 255, 255, 0.88);
}
.delete-options li + li {
  margin-top: 8px;
}
.delete-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: flex-end;
  align-items: center;
  margin-top: 4px;
}
button.danger-muted {
  border-color: rgba(248, 113, 113, 0.4);
  color: #fecaca;
  background: rgba(248, 113, 113, 0.1);
}

button.danger-muted:hover:not(:disabled) {
  background: rgba(248, 113, 113, 0.2);
  border-color: rgba(252, 165, 165, 0.55);
}

button.danger-muted:active:not(:disabled) {
  transform: scale(0.97);
}
</style>
