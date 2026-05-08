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
  EnableMod,
  OpenModsFolder,
  OpenModFolder,
  ExportModFolderZip,
} from '../wailsjs/go/app/App'
const gameExePath = ref('')
const modsRoot = ref('')
const overview = ref(null)
const loadError = ref('')
const actionError = ref('')
/** 仅导入 Mod 相关错误，显示在「导入 Mod」卡片内 */
const importError = ref('')
const saving = ref(false)
const importPath = ref('')
const importFolderName = ref('')
const editing = ref(null)
const detail = ref(null)
/** 长 description 展开状态 key = folderName + manifestFile */
const descExpanded = reactive({})

const duplicateBanner = computed(() => {
  if (!overview.value?.duplicateIDs?.length) return ''
  return `以下 mod id 重复出现：${overview.value.duplicateIDs.join('、')}（请手动调整相关 JSON 中的 id）`
})

/** 列表按文件夹名聚合：连续相同 folderName 合并第一列，并带 rowspan */
const rowsWithFolderSpan = computed(() => {
  const mods = overview.value?.mods ?? []
  const out = []
  let i = 0
  while (i < mods.length) {
    const fn = mods[i].folderName
    let j = i + 1
    while (j < mods.length && mods[j].folderName === fn) {
      j++
    }
    const span = j - i
    for (let k = i; k < j; k++) {
      out.push({
        mod: mods[k],
        folderFirst: k === i,
        folderSpan: span,
      })
    }
    i = j
  }
  return out
})

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
      await EnableMod(row.folderName, row.manifestFile)
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

function openEdit(row) {
  actionError.value = ''
  editing.value = {
    folderName: row.folderName,
    newFolderName: row.folderName,
    manifestFile: row.manifestFile,
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
    await SaveModEdit({
      folderName: editing.value.folderName,
      manifestFile: editing.value.manifestFile,
      newFolderName: editing.value.newFolderName.trim(),
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

async function deleteModFromEdit() {
  if (!editing.value) return
  const ok = confirm(
    `确定删除此 Mod？\n将删除 manifest「${editing.value.manifestFile}」及对应的 {id}.pck / .dll（含 .bak）。\n同目录下的其它 mod 不会被删除。`
  )
  if (!ok) return
  actionError.value = ''
  saving.value = true
  try {
    await DeleteModEntry(editing.value.folderName, editing.value.manifestFile)
    const fn = editing.value.folderName
    const mf = editing.value.manifestFile
    closeEdit()
    if (detail.value?.folderName === fn && detail.value?.manifestFile === mf) closeDetail()
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
    await ImportModArchive(importPath.value.trim(), importFolderName.value.trim())
    importPath.value = ''
    importFolderName.value = ''
    await refreshMods()
  } catch (e) {
    importError.value = String(e)
  } finally {
    saving.value = false
  }
}

onMounted(refreshState)
</script>

<template>
  <div class="layout">
    <header class="header">
      <h1>杀戮尖塔 2 Mod 管理器</h1>
      <p class="muted">配置游戏路径后，将扫描游戏目录下的 <code>mods</code> 文件夹并列出符合格式的 mod。</p>
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
              v-for="item in rowsWithFolderSpan"
              :key="rowKey(item.mod)"
              :class="{ dim: item.mod.disabled }"
            >
              <td
                v-if="item.folderFirst"
                class="folder-cell cell-wrap"
                :rowspan="item.folderSpan"
              >
                <div class="folder-name mono">{{ item.mod.folderName }}</div>
                <div class="folder-actions">
                  <button type="button" @click.stop="onOpenModFolder(item.mod.folderName)">
                    打开文件夹
                  </button>
                  <button type="button" @click.stop="onExportModFolderZip(item.mod.folderName)">
                    导出 Zip…
                  </button>
                </div>
              </td>
              <td class="cell-wrap">
                <span class="mono">{{ item.mod.manifest.id }}</span>
              </td>
              <td class="cell-wrap">{{ item.mod.manifest.name }}</td>
              <td class="cell-wrap">{{ item.mod.manifest.author }}</td>
              <td class="cell-wrap">{{ item.mod.manifest.version }}</td>
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
        若根目录只有一个文件夹，将把该文件夹<strong>内容</strong>放进目标目录；自定义文件夹名将始终作为 mods 下的文件夹名；散落文件则装入以压缩包名或自定义名命名的文件夹。
      </p>
      <div class="row">
        <input v-model="importPath" class="grow" type="text" placeholder="压缩包路径" readonly />
        <button type="button" @click="onBrowseImport">选择压缩包…</button>
      </div>
      <div class="row">
        <label class="grow">
          自定义文件夹名（可选，任意情况均可填写；留空则按压缩包结构自动命名）
          <input v-model="importFolderName" type="text" placeholder="例如 MyMod；留空则自动" />
        </label>
      </div>
      <div class="row">
        <button type="button" class="primary" :disabled="saving" @click="doImport">导入到 mods</button>
      </div>
      <div v-if="importError" class="msg err import-err">{{ importError }}</div>
    </section>

    <div v-if="detail" class="overlay" @click.self="closeDetail">
      <div class="dialog dialog-wide">
        <h3>Mod 详情</h3>
        <dl class="detail-dl">
          <dt>文件夹名</dt>
          <dd class="mono">{{ detail.folderName }}</dd>
          <dt>Manifest 文件名</dt>
          <dd class="mono">{{ detail.manifestFile }}</dd>
          <dt>状态</dt>
          <dd>{{ detail.disabled ? '已关闭（json/pck/dll 已加 .bak）' : '运行中' }}</dd>
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
            placeholder="mods 目录下的文件夹名称"
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
          <button type="button" class="danger" :disabled="saving" @click="deleteModFromEdit">删除此 Mod</button>
          <div class="row end tight">
            <button type="button" @click="closeEdit">取消</button>
            <button type="button" class="primary" :disabled="saving" @click="submitEdit">保存</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.layout {
  max-width: min(1580px, 98vw);
  margin: 0 auto;
  padding: 26px 32px 60px;
  text-align: left;
}
.header h1 {
  margin: 0 0 8px;
  font-size: 1.35rem;
}
.muted {
  color: rgba(255, 255, 255, 0.65);
  font-size: 0.9rem;
  margin: 0 0 12px;
}
.card {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 16px 18px;
  margin-bottom: 16px;
}
.card h2 {
  margin: 0 0 12px;
  font-size: 1.05rem;
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
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: rgba(255, 255, 255, 0.06);
  color: inherit;
  cursor: pointer;
  white-space: nowrap;
}
button.primary {
  background: #3b82f6;
  border-color: #3b82f6;
}
button.danger {
  border-color: rgba(248, 113, 113, 0.45);
  color: #fecaca;
}
button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.meta-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 14px;
  margin-top: 8px;
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.75);
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
  margin-bottom: 8px;
  line-height: 1.35;
  word-break: break-word;
}
.folder-actions {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 6px;
}
.folder-actions button {
  width: 100%;
  padding: 6px 8px;
  font-size: 0.85rem;
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
  width: 6%;
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
  width: 10%;
}
.cell-wrap {
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-word;
}
.grid th,
.grid td {
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  padding: 8px 10px;
  text-align: left;
  vertical-align: top;
}
.grid tr.dim td {
  opacity: 0.72;
}
.col-actions {
  min-width: 160px;
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
  padding: 0;
  border: none;
  background: transparent;
  color: #93c5fd;
  font-size: 0.82rem;
  cursor: pointer;
  text-decoration: underline;
}
.btn-desc-inline {
  flex-shrink: 0;
  align-self: flex-end;
  margin-top: 0;
  white-space: nowrap;
}
.btn-link:hover {
  color: #bfdbfe;
}
.col-switch {
  vertical-align: middle;
  white-space: nowrap;
}
.switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 26px;
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
  background-color: #64748b;
  border-radius: 26px;
  transition: background-color 0.2s;
}
.switch .slider:before {
  position: absolute;
  content: '';
  height: 20px;
  width: 20px;
  left: 3px;
  bottom: 3px;
  background: #fff;
  border-radius: 50%;
  transition: transform 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.35);
}
.switch input:checked + .slider {
  background-color: #22c55e;
}
.switch input:checked + .slider:before {
  transform: translateX(18px);
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
  box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.25);
  vertical-align: middle;
}
input[type='text'],
textarea {
  width: 100%;
  padding: 8px 10px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(0, 0, 0, 0.25);
  color: inherit;
  margin-top: 6px;
  box-sizing: border-box;
}
.meta {
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.75);
}
code {
  font-size: 0.85em;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.35);
}
.msg {
  margin-top: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 0.9rem;
}
.msg.err {
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.35);
}
.import-err {
  margin-top: 12px;
}
.msg.warn {
  background: rgba(234, 179, 8, 0.12);
  border: 1px solid rgba(234, 179, 8, 0.35);
  margin-bottom: 16px;
}
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
  padding: 20px;
}
.dialog {
  width: min(520px, 100%);
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 18px 20px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.45);
}
.dialog-wide {
  width: min(640px, 100%);
  max-height: min(90vh, 900px);
  overflow: auto;
}
.dialog h3 {
  margin: 0 0 14px;
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
  padding: 12px 14px;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.35);
  font-size: 0.78rem;
  line-height: 1.45;
  overflow-x: auto;
  white-space: pre;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
</style>
