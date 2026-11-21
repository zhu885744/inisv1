<template>
  <div class="layout-form">
    <!-- ============ 工具条 ============ -->
    <section class="card card-pad toolbar">
      <div class="toolbar-main">
        <span class="toolbar-badge"><i class="bi bi-layout-three-columns" /></span>
        <div>
          <h3 class="toolbar-title">网站布局</h3>
          <p class="toolbar-desc">
            左栏自定义菜单、右栏个性侧边栏；下面按前台的栏宽比例预览，点选元素即可在下方属性面板里改，
            拖拽或上下箭头调整顺序。中栏内容区只作布局参考，不可编辑
          </p>
        </div>
      </div>

      <div class="toolbar-side">
        <span class="pill is-quiet">菜单 {{ menus.length }} 项</span>
        <span class="pill is-quiet">侧栏 {{ sidebar.modules.length }} 个模块</span>
        <span class="pill" :class="sidebar.enabled ? 'is-on' : 'is-off'">
          <i class="bi bi-dot" /> 右侧栏{{ sidebar.enabled ? '已启用' : '已关闭' }}
        </span>
        <button class="btn btn-sm" :disabled="saving" @click="resetSidebar">
          <i class="bi bi-arrow-counterclockwise" /> 恢复默认侧栏
        </button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">
          <i class="bi bi-check2" /> {{ saving ? '保存中…' : '保存布局' }}
        </button>
      </div>
    </section>

    <!-- ============ 预览画布 ============ -->
    <section class="card canvas-card">
      <header class="canvas-head">
        <span class="canvas-tip">
          <i class="bi bi-hand-index-thumb" /> 点选元素编辑 · 拖拽调整顺序
        </span>
        <span class="canvas-legend">
          <span><i class="bi bi-pencil-square" /> 可编辑</span>
          <span class="is-muted"><i class="bi bi-lock" /> 内容区仅示意</span>
        </span>
      </header>

      <div v-if="loading" class="tip-line"><span class="spinner" /> 正在读取站点配置…</div>

      <div v-else class="canvas">
        <!-- ---------- 左栏：站点信息 + 自定义菜单 ---------- -->
        <div class="col col-left">
          <div class="site-head">
            <div class="site-title">{{ siteTitle }}</div>
            <div class="site-desc">{{ siteDesc }}</div>
            <div class="fake-search"><i class="bi bi-search" /> 搜索 <span class="kbd">ctrl k</span></div>
          </div>

          <div class="col-label">
            <span>自定义菜单</span>
            <button type="button" class="mini-btn" @click="addMenu">
              <i class="bi bi-plus-lg" /> 添加
            </button>
          </div>

          <p v-if="!menus.length" class="col-empty">未配置时前台显示默认导航</p>

          <div class="nav-list">
            <div
              v-for="(menu, index) in menus"
              :key="menu.id"
              class="nav-item"
              :class="{
                'is-active': isSelected('menu', menu.id),
                'is-dragging': drag.kind === 'menu' && drag.index === index
              }"
              draggable="true"
              @click="openEditor('menu', menu.id)"
              @dragstart="onDragStart('menu', index, $event)"
              @dragover.prevent="onDragOver('menu')"
              @drop.prevent="onDrop('menu', index)"
              @dragend="resetDrag"
            >
              <i class="nav-icon" :class="menu.icon || 'bi bi-link-45deg'" />
              <span class="nav-name">{{ menu.name || '未命名菜单' }}</span>
              <span class="nav-actions">
                <button type="button" title="上移" :disabled="index === 0" @click.stop="moveMenu(index, -1)">
                  <i class="bi bi-arrow-up" />
                </button>
                <button
                  type="button"
                  title="下移"
                  :disabled="index === menus.length - 1"
                  @click.stop="moveMenu(index, 1)"
                >
                  <i class="bi bi-arrow-down" />
                </button>
                <button type="button" title="删除" @click.stop="removeMenu(index)">
                  <i class="bi bi-trash3" />
                </button>
              </span>
            </div>

            <!-- 「管理」入口前台始终保留，这里只作示意 -->
            <div class="nav-item is-locked">
              <i class="nav-icon bi bi-gear" />
              <span class="nav-name">管理</span>
              <span class="lock-tag">固定</span>
            </div>
          </div>
        </div>

        <!-- ---------- 中栏：内容区示意 ---------- -->
        <div class="col col-main">
          <div class="col-label">
            <span>内容区</span>
            <span class="col-sub">仅示意</span>
          </div>

          <div class="mock-tabs">
            <span class="is-active">全部</span>
            <span>文章</span>
            <span>动态</span>
          </div>

          <div v-for="i in 2" :key="`mock-${i}`" class="mock-post">
            <div class="mock-head">
              <span class="mock-avatar" />
              <span class="mock-lines">
                <span class="mock-line is-name" />
                <span class="mock-line is-meta" />
              </span>
            </div>
            <span class="mock-line" />
            <span class="mock-line is-short" />
            <span class="mock-media" />
          </div>
        </div>

        <!-- ---------- 右栏：个性侧边栏 ---------- -->
        <div class="col col-right" :class="{ 'is-disabled': !sidebar.enabled }">
          <div class="col-label">
            <span>个性侧边栏</span>
            <label class="switch is-mini" title="前台是否渲染右侧整栏">
              <FormCheckbox v-model="sidebar.enabled" />
              <span class="switch-slider" />
            </label>
          </div>

          <p v-if="!sidebar.enabled" class="col-empty">
            已关闭：前台整栏不渲染，中栏会占满这块宽度；模块配置仍保留，随时可重新打开
          </p>

          <div class="side-list">
            <div
              v-for="(module, index) in sidebar.modules"
              :key="module.id"
              class="side-card"
              :class="{
                'is-active': isSelected('module', module.id),
                'is-hidden': module.visible === false,
                'is-dragging': drag.kind === 'module' && drag.index === index
              }"
              draggable="true"
              @click="openEditor('module', module.id)"
              @dragstart="onDragStart('module', index, $event)"
              @dragover.prevent="onDragOver('module')"
              @drop.prevent="onDrop('module', index)"
              @dragend="resetDrag"
            >
              <div class="side-card-head">
                <i :class="moduleIcon(module)" />
                <span class="side-title">{{ moduleTitle(module) }}</span>
                <span v-if="module.visible === false" class="side-tag">已隐藏</span>
              </div>

              <div class="side-body">
                <!-- 用户卡片 -->
                <template v-if="module.type === 'user'">
                  <div class="sk-user">
                    <span class="sk-avatar" />
                    <span class="sk-lines">
                      <span class="mock-line is-name" />
                      <span class="mock-line is-meta" />
                    </span>
                  </div>
                </template>

                <!-- 自定义 HTML -->
                <template v-else-if="module.type === 'html'">
                  <div class="sk-html">{{ module.content ? '自定义 HTML 内容' : '（空）点下方「自定义内容」填写' }}</div>
                </template>

                <!-- 自定义链接 -->
                <template v-else-if="module.type === 'links'">
                  <div v-if="!module.items.length" class="sk-html">（还没有链接）</div>
                  <div v-else class="sk-chips">
                    <span v-for="(link, i) in module.items.slice(0, 4)" :key="i" class="sk-chip">
                      <i :class="link.icon || 'bi bi-link-45deg'" /> {{ link.name || '未命名' }}
                    </span>
                  </div>
                </template>

                <!-- 列表类：按条数画骨架行 -->
                <template v-else>
                  <span v-for="n in skeletonRows(module)" :key="n" class="sk-row">
                    <span class="sk-no">{{ n }}</span>
                    <span class="mock-line" />
                  </span>
                </template>
              </div>

              <div class="side-actions">
                <button type="button" :title="module.visible === false ? '显示' : '隐藏'" @click.stop="toggleVisible(module)">
                  <i :class="module.visible === false ? 'bi bi-eye-slash' : 'bi bi-eye'" />
                </button>
                <button type="button" title="上移" :disabled="index === 0" @click.stop="moveModule(index, -1)">
                  <i class="bi bi-arrow-up" />
                </button>
                <button
                  type="button"
                  title="下移"
                  :disabled="index === sidebar.modules.length - 1"
                  @click.stop="moveModule(index, 1)"
                >
                  <i class="bi bi-arrow-down" />
                </button>
                <button type="button" title="删除" @click.stop="removeModule(index)">
                  <i class="bi bi-trash3" />
                </button>
              </div>
            </div>
          </div>

          <div class="side-add">
            <FormSelect v-model="newModuleType" :options="SIDEBAR_TYPE_OPTIONS" />
            <button type="button" class="mini-btn is-primary" @click="addModule">
              <i class="bi bi-plus-lg" /> 添加模块
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- ============ 属性弹窗（点选元素后弹出，不占页面高度） ============ -->
    <AdminFormDialog
      v-model:visible="editorVisible"
      :title="editorTitle"
      :icon="editorIcon"
      width="660px"
      @cancel="closeEditor"
    >
      <!-- 菜单项属性 -->
      <template v-if="selectedMenu">
        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">名称</label>
            <FormInput v-model="selectedMenu.name" placeholder="例如：关于" />
          </div>
          <div class="form-item">
            <label class="form-label">图标（bootstrap-icons）</label>
            <!-- 预览直接嵌在输入框左侧，右侧是「图标库」外链按钮，整行不再换行 -->
            <div class="icon-field">
              <i class="icon-field-preview" :class="selectedMenu.icon || 'bi bi-question-circle'" />
              <FormInput v-model="selectedMenu.icon" placeholder="bi bi-info-circle" />
              <a
                class="icon-field-link"
                href="https://icons.getbootstrap.com/"
                target="_blank"
                rel="noopener noreferrer"
                title="去 Bootstrap Icons 官网查找并复制类名"
              >
                <i class="bi bi-box-arrow-up-right" />
              </a>
            </div>
            <p class="form-hint">搜索后复制类名（如 <code>bi bi-info-circle</code>）粘进来即可</p>
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">跳转类型</label>
            <FormSelect v-model="selectedMenu.target" :options="MENU_TARGET_OPTIONS" />
          </div>
          <div class="form-item">
            <label class="form-label">链接</label>
            <FormInput
              v-model="selectedMenu.url"
              :placeholder="selectedMenu.target === 'external' ? 'https://example.com' : '/about'" />
            <p class="form-hint">
              {{ selectedMenu.target === 'external' ? '站外链接，会在新窗口打开' : '站内路径，例如 /about、/topics' }}
            </p>
          </div>
        </div>
      </template>

      <!-- 侧栏模块属性 -->
      <template v-else-if="selectedModule">
        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">模块类型</label>
            <FormSelect
              :model-value="selectedModule.type"
              :options="SIDEBAR_TYPE_OPTIONS"
              @update:modelValue="(value) => changeModuleType(selectedModule, value)" />
            <p class="form-hint">{{ moduleTypeDesc(selectedModule.type) }}；切换类型会按新类型重置标题与内容</p>
          </div>
          <div class="form-item">
            <label class="form-label">标题</label>
            <FormInput
              v-model="selectedModule.title"
              :placeholder="moduleTypeDefaults(selectedModule.type).title || '使用类型默认标题'" />
            <p class="form-hint">留空则使用类型默认标题</p>
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">图标（bootstrap-icons）</label>
            <div class="icon-field">
              <i class="icon-field-preview" :class="selectedModule.icon || 'bi bi-question-circle'" />
              <FormInput v-model="selectedModule.icon" placeholder="bi bi-shuffle" />
              <a
                class="icon-field-link"
                href="https://icons.getbootstrap.com/"
                target="_blank"
                rel="noopener noreferrer"
                title="去 Bootstrap Icons 官网查找并复制类名"
              >
                <i class="bi bi-box-arrow-up-right" />
              </a>
            </div>
            <p class="form-hint">留空则用该类型的默认图标</p>
          </div>
          <div v-if="supportsField(selectedModule.type, 'limit')" class="form-item">
            <label class="form-label">显示条数</label>
            <FormInput v-model.number="selectedModule.limit" type="number" min="1" max="100" />
            <p class="form-hint">该模块列表最多展示多少条</p>
          </div>
        </div>

        <div v-if="supportsField(selectedModule.type, 'moreUrl')" class="form-grid">
          <div class="form-item">
            <label class="form-label">「更多」文案</label>
            <FormInput v-model="selectedModule.moreText" placeholder="留空则显示「更多」" />
          </div>
          <div class="form-item">
            <label class="form-label">「更多」链接</label>
            <FormInput v-model="selectedModule.moreUrl" placeholder="站内路径如 /topics，或完整网址" />
            <p class="form-hint">留空则不显示右上角入口；站外链接自动新窗口打开</p>
          </div>
        </div>

        <div v-if="supportsField(selectedModule.type, 'content')" class="form-item">
          <label class="form-label">自定义内容（HTML）</label>
          <FormTextarea
            v-model="selectedModule.content"
            class="code"
            rows="6"
            placeholder="<p>这里可以写公告、广告位、友链或任意 HTML</p>" />
          <p class="form-hint">原样渲染到前台，请只粘贴可信内容</p>
        </div>

        <div v-if="supportsField(selectedModule.type, 'items')" class="links-box">
          <div class="links-head">
            <span class="form-label">链接列表</span>
            <button type="button" class="mini-btn" @click="addLink(selectedModule)">
              <i class="bi bi-plus-lg" /> 添加链接
            </button>
          </div>

          <p v-if="!selectedModule.items.length" class="col-empty">还没有链接</p>

          <div v-for="(link, index) in selectedModule.items" :key="index" class="link-row">
            <FormInput v-model="link.name" placeholder="名称" />
            <FormInput v-model="link.icon" placeholder="bi bi-link" />
            <FormInput v-model="link.url" placeholder="/about 或 https://…" />
            <FormSelect v-model="link.target" :options="MENU_TARGET_OPTIONS" />
            <button type="button" class="icon-btn" title="删除" @click="removeLink(selectedModule, index)">
              <i class="bi bi-trash3" />
            </button>
          </div>
        </div>

        <div class="flags">
          <label class="flag">
            <FormCheckbox v-model="selectedModule.visible" />
            <span>前台显示该模块</span>
          </label>
          <label class="flag">
            <FormCheckbox v-model="selectedModule.collapsible" />
            <span>允许访客折叠</span>
          </label>
          <label v-if="selectedModule.collapsible" class="flag">
            <FormCheckbox v-model="selectedModule.collapsed" />
            <span>默认折叠</span>
          </label>
        </div>
      </template>

      <template #footer>
        <button
          v-if="selectedMenu"
          type="button"
          class="btn btn-danger"
          @click="removeSelectedMenu"
        >
          <i class="bi bi-trash3" /> 删除该菜单
        </button>
        <button
          v-else-if="selectedModule"
          type="button"
          class="btn btn-danger"
          @click="removeSelectedModule"
        >
          <i class="bi bi-trash3" /> 删除该模块
        </button>
        <span class="foot-spacer" />
        <button type="button" class="btn" @click="closeEditor">关闭</button>
        <button type="button" class="btn btn-primary" :disabled="saving" @click="save">
          <i class="bi bi-check2" /> {{ saving ? '保存中…' : '保存布局' }}
        </button>
      </template>
    </AdminFormDialog>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 网站布局（系统配置 → 网站表单设置）
 *
 * 把原来「网站设置」里的两块表单（自定义菜单 / 个性侧边栏）抽出来，改成
 * **所见即所得** 的画布：左栏菜单、中栏内容示意、右栏侧边模块，栏宽比例与前台一致，
 * 点选元素弹出属性弹窗编辑（不占页面高度），拖拽或上下箭头调整顺序。
 *
 * 数据仍然是 config 表的 Mellow_functions 一条记录：
 * - custom_menus：[{ id, name, icon, url, target: 'internal' | 'external' }]
 *   设置后前台只显示这些导航项（「管理」入口始终保留）；为空则显示默认导航。
 * - sidebar：见 utils/sidebar.js（enabled + modules[]，模块类型/字段都在那一个文件里注册）。
 *
 * 保存策略与「网站设置」一致：先取最新完整配置再合并，只覆盖本页负责的两个键，
 * 避免把其它后台页面（默认头像、上传模式等）写入的字段抹掉。
 */
import { computed, onMounted, reactive, ref } from 'vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import { saveConfig } from '@/api/config'
import { useSiteStore } from '@/stores/site'
import { toast } from '@/utils/toast'
import {
  SIDEBAR_TYPES,
  createSidebarModule,
  defaultSidebarConfig,
  normalizeSidebarConfig,
  sidebarModuleIcon,
  sidebarModuleTitle,
  sidebarTypeMeta
} from '@/utils/sidebar'

const siteStore = useSiteStore()

// 站点设置存储在 config 表的 Mellow_functions 记录（配置本体在 json）
const CONFIG_KEY = 'Mellow_functions'

const loading = ref(false)
const saving = ref(false)

// 站点标题 / 描述只用于画布顶部示意（真身仍在「网站设置」里改）
const siteMeta = reactive({ title: '', description: '' })

const menus = ref([])
const sidebar = reactive(defaultSidebarConfig())

// 当前选中项：kind = 'menu' | 'module'，id 对应各自的 id
const selection = reactive({ kind: '', id: 0 })

// 拖拽排序的临时状态（只允许同列表内拖动）
const drag = reactive({ kind: '', index: -1 })

const MENU_TARGET_OPTIONS = [
  { value: 'internal', label: '站内跳转' },
  { value: 'external', label: '站外跳转' }
]

// 模块类型下拉：与 utils/sidebar.js 的注册表同源，新增类型只需改那一个文件
const SIDEBAR_TYPE_OPTIONS = SIDEBAR_TYPES.map((item) => ({ value: item.value, label: item.name }))
const newModuleType = ref('articleRank')

const siteTitle = computed(() => siteMeta.title || '您的站点标题')
const siteDesc = computed(() => siteMeta.description || '在「网站设置」里填写站点描述')

// 菜单与侧栏模块的 id 各自从 1 开始，必然重号：
// 必须同时按 kind 过滤，否则点「模块 3」会被选中成「菜单 3」，弹窗就显示错对象
const selectedMenu = computed(() =>
  selection.kind === 'menu' ? menus.value.find((item) => item.id === selection.id) || null : null
)
const selectedModule = computed(() =>
  selection.kind === 'module'
    ? sidebar.modules.find((item) => item.id === selection.id) || null
    : null
)

function isSelected(kind, id) {
  return selection.kind === kind && selection.id === id
}

// 属性弹窗的显隐（点选元素后打开，不占页面高度）
const editorVisible = ref(false)

/** 拖拽过程中不弹窗：拖完松手会顺带触发一次 click */
let dragging = false

const editorTitle = computed(() => {
  if (selectedMenu.value) return `菜单项 · ${selectedMenu.value.name || '未命名菜单'}`
  if (selectedModule.value) return `侧栏模块 · ${moduleTitle(selectedModule.value)}`
  return '属性'
})

const editorIcon = computed(() =>
  selectedMenu.value ? 'bi bi-signpost-split' : 'bi bi-window-stack'
)

/** 只改选中态（画布高亮），不弹窗 */
function select(kind, id) {
  selection.kind = kind
  selection.id = id
}

/** 选中并打开属性弹窗 */
function openEditor(kind, id) {
  if (dragging) return
  select(kind, id)
  editorVisible.value = true
}

/** 关闭弹窗（保留选中态，画布上仍高亮，方便接着点相邻元素） */
function closeEditor() {
  editorVisible.value = false
}

// ---------- 自定义菜单 ----------

/**
 * 归一化菜单数据：优先 custom_menus（数组），没有则把旧版 custom_nav_links
 * （每行「名称 || 链接」）迁移成数组，保证老站点打开就能看到自己的导航
 */
function normalizeMenus(list, legacyText) {
  if (Array.isArray(list) && list.length) {
    const used = new Set()
    return list.map((item, index) => {
      // 老配置里可能有两项 id 相同，会让 v-for key 冲突，这里顺手去重
      let id = Number(item?.id) || index + 1
      while (used.has(id)) id += 1
      used.add(id)

      return {
        id,
        name: item?.name || '',
        icon: item?.icon || '',
        url: item?.url || '',
        target: item?.target === 'external' ? 'external' : 'internal'
      }
    })
  }

  return String(legacyText || '')
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line, index) => {
      const [name, url] = line.split('||').map((s) => s.trim())
      if (!name || !url) return null
      return {
        id: index + 1,
        name,
        icon: '',
        url,
        target: /^https?:\/\//i.test(url) ? 'external' : 'internal'
      }
    })
    .filter(Boolean)
}

function addMenu() {
  const list = menus.value
  const id = list.length ? Math.max(...list.map((item) => Number(item.id) || 0)) + 1 : 1
  list.push({ id, name: '新菜单', icon: 'bi bi-link-45deg', url: '', target: 'internal' })
  // 新建即打开属性弹窗，直接填名称与链接
  openEditor('menu', id)
}

function removeMenu(index) {
  const removed = menus.value[index]
  menus.value.splice(index, 1)
  if (removed && isSelected('menu', removed.id)) {
    selection.kind = ''
    closeEditor()
  }
}

function removeSelectedMenu() {
  if (selection.kind !== 'menu') return
  const index = menus.value.findIndex((item) => item.id === selection.id)
  if (index >= 0) removeMenu(index)
}

// ---------- 侧边栏模块 ----------

function moduleTitle(module) {
  return sidebarModuleTitle(module)
}

function moduleIcon(module) {
  return sidebarModuleIcon(module)
}

function moduleTypeDesc(type) {
  return sidebarTypeMeta(type).desc || ''
}

function moduleTypeDefaults(type) {
  return sidebarTypeMeta(type).defaults || {}
}

/** 某类型是否支持某个字段（limit / moreUrl / content / items） */
function supportsField(type, key) {
  return sidebarTypeMeta(type).fields.includes(key)
}

/** 列表类模块在预览里画几行骨架（最多 4 行，免得画布被拉得太长） */
function skeletonRows(module) {
  const limit = Number(module.limit) || 6
  return Math.max(1, Math.min(limit, 4))
}

function changeModuleType(module, type) {
  const fresh = createSidebarModule(type, [])
  // 换类型只重置类型相关字段，保留 id / 显隐 / 折叠这些通用设置
  Object.assign(module, fresh, {
    id: module.id,
    visible: module.visible,
    collapsible: module.collapsible,
    collapsed: module.collapsed
  })
}

function addModule() {
  const created = createSidebarModule(newModuleType.value, sidebar.modules)
  sidebar.modules.push(created)
  // 新建即打开属性弹窗，直接改标题 / 条数 / 自定义内容
  openEditor('module', created.id)
}

function removeModule(index) {
  const removed = sidebar.modules[index]
  sidebar.modules.splice(index, 1)
  if (removed && isSelected('module', removed.id)) {
    selection.kind = ''
    closeEditor()
  }
}

function removeSelectedModule() {
  if (selection.kind !== 'module') return
  const index = sidebar.modules.findIndex((item) => item.id === selection.id)
  if (index >= 0) removeModule(index)
}

function toggleVisible(module) {
  module.visible = module.visible === false
}

/** 侧栏恢复默认：回到改造前那 8 个模块的顺序（默认值，未保存前可反悔） */
function resetSidebar() {
  if (!window.confirm('恢复默认侧栏？当前模块顺序与自定义内容会被默认的 8 个模块替换（保存后才生效）。')) return
  const normalized = defaultSidebarConfig()
  sidebar.enabled = normalized.enabled
  sidebar.modules = normalized.modules
  selection.kind = ''
  selection.id = 0
  closeEditor()
}

function addLink(module) {
  const id = module.items.reduce((max, item) => Math.max(max, Number(item?.id) || 0), 0) + 1
  module.items.push({ id, name: '新链接', icon: 'bi bi-link-45deg', url: '', target: 'internal' })
}

function removeLink(module, index) {
  module.items.splice(index, 1)
}

// ---------- 排序（↑↓ 按钮 + 拖拽） ----------

function moveItem(list, index, target) {
  if (target < 0 || target >= list.length || target === index) return
  const [moved] = list.splice(index, 1)
  list.splice(target, 0, moved)
}

function moveMenu(index, offset) {
  moveItem(menus.value, index, index + offset)
}

function moveModule(index, offset) {
  moveItem(sidebar.modules, index, index + offset)
}

function onDragStart(kind, index, event) {
  dragging = true
  drag.kind = kind
  drag.index = index
  if (event?.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    // Firefox 需要 setData 才会触发后续 drag 事件
    event.dataTransfer.setData('text/plain', String(index))
  }
}

function onDragOver(kind) {
  // 只允许同一列表内排序，跨列表拖动没有意义
  if (drag.kind && drag.kind !== kind) return
}

function onDrop(kind, index) {
  if (drag.kind !== kind || drag.index < 0) return resetDrag()
  const list = kind === 'menu' ? menus.value : sidebar.modules
  moveItem(list, drag.index, index)
  resetDrag()
}

function resetDrag() {
  drag.kind = ''
  drag.index = -1
  // 拖完松手可能紧跟一次 click，等这一拍过去再放开选中/弹窗
  setTimeout(() => {
    dragging = false
  }, 120)
}

// ---------- 读取 / 保存 ----------

async function load() {
  loading.value = true
  try {
    const config = (await siteStore.load(true)) || {}
    siteMeta.title = config.title || ''
    siteMeta.description = config.description || ''
    menus.value = normalizeMenus(config.custom_menus, config.custom_nav_links)

    const normalized = normalizeSidebarConfig(config.sidebar)
    sidebar.enabled = normalized.enabled
    sidebar.modules = normalized.modules

    // 默认选中第一个侧栏模块，打开就有内容可看
    if (sidebar.modules.length) select('module', sidebar.modules[0].id)
  } catch {
    toast.error('读取站点配置失败')
  } finally {
    loading.value = false
  }
}

async function save() {
  if (menus.value.some((item) => !String(item.name || '').trim())) {
    toast.warning('有菜单项还没填名称')
    return
  }

  saving.value = true
  try {
    // 先取最新完整配置再合并：本页只认识 custom_menus / sidebar，
    // 直接整体覆盖会把其它后台页面写入的键抹掉
    const latest = (await siteStore.load(true)) || {}
    await saveConfig(CONFIG_KEY, {
      ...latest,
      custom_menus: menus.value.map((item) => ({
        id: item.id,
        name: item.name,
        icon: item.icon,
        url: item.url,
        target: item.target === 'external' ? 'external' : 'internal'
      })),
      sidebar: {
        enabled: sidebar.enabled,
        modules: sidebar.modules
      }
    })
    toast.success('布局已保存，前台刷新即可看到')
    // 同步失效 site store 缓存，保证前台立即读取到最新配置
    await siteStore.invalidate()
  } catch {
    toast.error('保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.layout-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 工具条 ---------- */
.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  background:
    radial-gradient(120% 140% at 0% 0%, var(--accent-wash), transparent 60%),
    var(--bg-card);
}
.toolbar-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.toolbar-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: var(--radius);
}
.toolbar-badge .bi {
  font-size: 19px;
}
.toolbar-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.toolbar-desc {
  margin: 4px 0 0;
  max-width: 74ch;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}
.toolbar-side {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.pill {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 3px 10px;
  font-size: 12px;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--text-soft);
  background: var(--bg-muted);
}
.pill .bi {
  font-size: 16px;
  line-height: 1;
}
.pill.is-quiet {
  color: var(--text-muted);
}
.pill.is-on {
  color: var(--success);
  border-color: color-mix(in srgb, var(--success) 30%, transparent);
  background: color-mix(in srgb, var(--success) 8%, transparent);
}
.pill.is-off {
  color: var(--text-muted);
}

/* ---------- 画布 ---------- */
.canvas-card {
  padding: 14px;
  background: var(--bg-muted);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}
.canvas-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 4px 12px;
  font-size: 12px;
  color: var(--text-muted);
}
.canvas-tip .bi {
  color: var(--primary);
}
.canvas-legend {
  display: flex;
  gap: 12px;
}
.canvas-legend .is-muted {
  color: var(--text-light, var(--text-muted));
}

/* 三栏：栏宽比例按前台（左 200 / 右 240，中栏自适应） */
.canvas {
  display: grid;
  grid-template-columns: 200px minmax(0, 1fr) 232px;
  gap: 14px;
  align-items: start;
}
.col {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.col-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-soft);
}
.col-sub {
  font-weight: 400;
  color: var(--text-light, var(--text-muted));
}
.col-empty {
  margin: 0;
  padding: 12px 10px;
  font-size: 11px;
  line-height: 1.6;
  text-align: center;
  color: var(--text-light, var(--text-muted));
  background: var(--bg-card);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
}

/* 左栏：站点信息 + 导航 */
.col-left .site-head {
  padding: 10px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
}
.site-title {
  font-family: var(--font-serif, serif);
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.site-desc {
  margin-top: 4px;
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fake-search {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding: 5px 9px;
  font-size: 11px;
  color: var(--text-light, var(--text-muted));
  background: var(--bg-muted);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.kbd {
  margin-left: auto;
  padding: 0 4px;
  font-size: 10px;
  border: 1px solid var(--border);
  border-radius: 4px;
}
.nav-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  font-size: 12.5px;
  color: var(--text-soft);
  background: var(--bg-card);
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color var(--dur-fast, 0.15s), border-color var(--dur-fast, 0.15s);
}
.nav-item:hover {
  background: var(--bg-muted);
}
.nav-item.is-active {
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-color: var(--primary-soft, var(--primary));
  font-weight: 600;
}
.nav-item.is-dragging {
  opacity: 0.45;
}
.nav-item.is-locked {
  margin-top: 4px;
  color: var(--text-light, var(--text-muted));
  background: transparent;
  border: 1px dashed var(--border-soft);
  cursor: default;
}
.nav-icon {
  font-size: 13px;
  color: var(--primary);
}
.nav-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lock-tag {
  font-size: 10px;
  color: var(--text-light, var(--text-muted));
}
.nav-actions {
  display: none;
  align-items: center;
  gap: 2px;
}
.nav-item:hover .nav-actions {
  display: flex;
}
/* 侧栏模块的操作按钮浮在卡片右上角，避免 hover 时把卡片撑高、预览跳动 */
.side-actions {
  position: absolute;
  top: 5px;
  right: 5px;
  display: none;
  align-items: center;
  gap: 2px;
  padding: 1px;
  background: var(--bg-card);
  border-radius: 5px;
}
.side-card:hover .side-actions {
  display: flex;
}
.nav-actions button,
.side-actions button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  color: var(--text-muted);
  background: transparent;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
.nav-actions button:hover,
.side-actions button:hover {
  color: var(--primary);
  background: var(--bg-muted);
}
.nav-actions button:disabled,
.side-actions button:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

/* 中栏：内容示意 */
.mock-tabs {
  display: flex;
  gap: 4px;
  padding: 4px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
}
.mock-tabs span {
  padding: 3px 12px;
  font-size: 11px;
  color: var(--text-muted);
  border-radius: 999px;
}
.mock-tabs .is-active {
  color: var(--primary-deep);
  background: var(--accent-soft);
  font-weight: 600;
}
.mock-post {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
}
.mock-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.mock-avatar,
.sk-avatar {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  background: var(--bg-muted);
  border: 1px solid var(--border);
  border-radius: 50%;
}
.mock-lines,
.sk-lines {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}
.mock-line {
  display: block;
  height: 7px;
  background: var(--bg-muted);
  border-radius: 999px;
}
.mock-line.is-name {
  width: 42%;
}
.mock-line.is-meta {
  width: 24%;
  height: 5px;
}
.mock-line.is-short {
  width: 62%;
}
.mock-media {
  display: block;
  height: 78px;
  background: linear-gradient(135deg, var(--bg-muted), var(--accent-wash));
  border-radius: var(--radius-sm);
}

/* 右栏：侧边模块 */
.col-right.is-disabled {
  opacity: 0.55;
}
.side-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.side-card {
  position: relative;
  padding: 9px 10px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  cursor: pointer;
  transition: border-color var(--dur-fast, 0.15s), box-shadow var(--dur-fast, 0.15s);
}
.side-card:hover {
  border-color: var(--accent-ring, var(--border));
}
.side-card.is-active {
  border-color: var(--primary-soft, var(--primary));
  box-shadow: 0 0 0 2px var(--accent-ring, transparent);
}
.side-card.is-hidden {
  opacity: 0.5;
}
.side-card.is-dragging {
  opacity: 0.4;
}
.side-card-head {
  display: flex;
  align-items: center;
  gap: 6px;
  /* 给右上角的悬浮操作按钮留位，免得标题被盖住 */
  padding: 0 58px 6px 0;
  margin-bottom: 8px;
  border-bottom: 1px dashed var(--border-soft, var(--border));
}
.side-card-head .bi {
  font-size: 12px;
  color: var(--primary);
}
.side-title {
  flex: 1;
  font-size: 12px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.side-tag {
  font-size: 10px;
  color: var(--warning);
}
.side-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.sk-user {
  display: flex;
  align-items: center;
  gap: 8px;
}
.sk-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
/* 骨架行里的横线要占满剩余宽度（.mock-line 是 block，放进 flex 行会塌成 0 宽） */
.sk-row .mock-line {
  flex: 1;
}
.sk-no {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
  font-size: 9px;
  line-height: 13px;
  text-align: center;
  color: #fff;
  background: var(--primary);
  border-radius: 3px;
}
.sk-html {
  padding: 10px 8px;
  font-size: 11px;
  text-align: center;
  color: var(--text-light, var(--text-muted));
  background: var(--bg-muted);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
}
.sk-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.sk-chip {
  padding: 2px 8px;
  font-size: 11px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 999px;
}
.side-add {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-top: 4px;
}
/* 下拉默认 width:100%，在窄栏里会把「添加」按钮挤出去 */
.side-add .select-menu {
  flex: 1 1 auto;
  width: auto;
  min-width: 0;
}
.side-add .mini-btn {
  flex-shrink: 0;
}

/* 小按钮 / 开关 */
.mini-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  font-size: 11px;
  color: var(--text-soft);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 999px;
  cursor: pointer;
  transition: all var(--dur-fast, 0.15s);
}
.mini-btn:hover {
  color: var(--primary);
  border-color: var(--primary);
}
.mini-btn.is-primary {
  color: #fff;
  background: var(--primary);
  border-color: var(--primary);
}
.switch {
  position: relative;
  display: inline-block;
  width: 42px;
  height: 24px;
  flex-shrink: 0;
}
.switch.is-mini {
  width: 33px;
  height: 19px;
}
.switch input {
  width: 0;
  height: 0;
  opacity: 0;
}
.switch-slider {
  position: absolute;
  inset: 0;
  cursor: pointer;
  background: var(--border);
  border-radius: 999px;
  transition: background-color var(--dur-fast, 0.15s);
}
.switch-slider::before {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 18px;
  height: 18px;
  background: #fff;
  border-radius: 50%;
  box-shadow: var(--shadow-sm);
  transition: transform var(--dur-fast, 0.15s);
}
.switch.is-mini .switch-slider::before {
  width: 13px;
  height: 13px;
  top: 3px;
  left: 3px;
}
.switch input:checked + .switch-slider {
  background: var(--primary);
}
.switch input:checked + .switch-slider::before {
  transform: translateX(18px);
}
.switch.is-mini input:checked + .switch-slider::before {
  transform: translateX(14px);
}

/* ---------- 属性弹窗内容 ---------- */
/* 弹窗底部：删除按钮靠左，其余按钮靠右 */
.foot-spacer {
  flex: 1;
}
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 14px;
}
/* 图标输入框：左侧嵌图标预览、右侧嵌「图标库」外链，整行不换行 */
.icon-field {
  position: relative;
  display: flex;
  align-items: center;
}
.icon-field .input {
  /* 两端留出内嵌元素的位置，文字不会与图标重叠 */
  padding-left: 34px;
  padding-right: 34px;
  font-family: var(--font-mono, monospace);
  font-size: 12.5px;
}
.icon-field-preview {
  position: absolute;
  left: 11px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 15px;
  line-height: 1;
  color: var(--primary);
  pointer-events: none;
}
.icon-field-link {
  position: absolute;
  right: 6px;
  top: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  transform: translateY(-50%);
  font-size: 12px;
  color: var(--text-muted);
  text-decoration: none;
  border-radius: var(--radius-sm);
  transition: color var(--dur-fast, 0.15s), background-color var(--dur-fast, 0.15s);
}
.icon-field-link:hover {
  color: var(--primary);
  background: var(--bg-muted);
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.form-hint code {
  padding: 0 4px;
  font-size: 11px;
  background: var(--bg-muted);
  border-radius: 3px;
}
.code {
  font-family: var(--font-mono, monospace);
  font-size: 12px;
}
.links-box {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.links-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.link-row {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1.6fr 0.9fr 34px;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.icon-btn:hover {
  color: var(--danger);
  border-color: var(--danger);
}
.flags {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding-top: 12px;
  border-top: 1px dashed var(--border-soft, var(--border));
}
.flag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-soft);
  cursor: pointer;
}
.tip-line {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 24px 4px;
  font-size: 12px;
  color: var(--text-muted);
}
.spinner {
  width: 13px;
  height: 13px;
  border: 2px solid var(--border);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: layout-spin 0.7s linear infinite;
}
@keyframes layout-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1100px) {
  .canvas {
    grid-template-columns: 180px minmax(0, 1fr) 200px;
    gap: 10px;
  }
}
@media (max-width: 900px) {
  /* 手机上画布折成一列：仍然按「左栏 → 侧栏」的顺序，方便逐项调 */
  .canvas {
    grid-template-columns: minmax(0, 1fr);
  }
  .col-main {
    display: none;
  }
  .form-grid,
  .link-row {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
