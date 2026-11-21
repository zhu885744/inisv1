/**
 * 工具
 */
export const debounce = (fn, wait = 300) => {
  let t
  return function (...args) {
    clearTimeout(t)
    t = setTimeout(() => fn.apply(this, args), wait)
  }
}

/**
 * 复制文本到剪贴板（优先 Clipboard API，非安全上下文降级 execCommand）
 * @returns {Promise<boolean>} 是否复制成功
 */
export async function copyText(text) {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    /* 继续走降级方案 */
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.left = '-9999px'
  ta.style.top = '0'
  document.body.appendChild(ta)
  ta.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    /* ignore */
  }
  document.body.removeChild(ta)
  return ok
}

/**
 * 解析 tags 字段（格式 "1|2|3|" 或 字符串）
 */
export const parseTagsField = (field) => {
  if (!field) return []
  if (Array.isArray(field)) return field
  return String(field).split('|').filter(Boolean)
}

/**
 * 解析 group / tags 的 id 字段（"|1|2|"、"1,2"、数组 均兼容）
 * @returns {number[]} id 数组
 */
export const parseIdField = (field) => {
  if (field === null || field === undefined || field === '') return []
  if (Array.isArray(field)) {
    return field.map((i) => parseInt(i, 10)).filter((i) => !Number.isNaN(i))
  }
  return String(field)
    .split(/[|,]/)
    .map((i) => parseInt(i.trim(), 10))
    .filter((i) => !Number.isNaN(i))
}

/**
 * 把 id 数组转成后端存储格式 "|1|2|"
 */
export const toIdField = (ids) => {
  const list = (Array.isArray(ids) ? ids : [ids])
    .map((i) => parseInt(i, 10))
    .filter((i) => !Number.isNaN(i) && i > 0)
  if (!list.length) return ''
  return `|${list.join('|')}|`
}

/**
 * 字符串截断
 *
 * 以 [emoji:url] 表情标记为整体单位截断，避免把标记切成半截导致
 * 列表/摘要里显示 [emoji:https://cs... 这样的残缺文本
 */
export const truncate = (str, len = 80) => {
  if (!str) return ''
  const text = String(str).replace(/<[^>]+>/g, '')
  if (text.length <= len) return text

  const parts = text.split(/(\[emoji:[^\]]+\])/g)
  let out = ''
  for (const part of parts) {
    if (!part) continue
    if (out.length + part.length > len) {
      // 表情标记整体放不下时直接丢弃，不显示半截图片/标记
      if (part.startsWith('[emoji:')) break
      out += part.slice(0, Math.max(0, len - out.length))
      break
    }
    out += part
  }
  return out + '...'
}

/**
 * 深拷贝
 */
export const clone = (obj) => JSON.parse(JSON.stringify(obj))

// 媒体短代码：@[live](视频 "封面" "说明") / @[video](url "标题") / @[audio] / @[file]
const MEDIA_SHORTCODE_RE = /@\[(live|video|audio|file)\]\s*\(([^)]*)\)/g
// 未闭合的短代码：摘要按字数截断时会把它切成半截（@[live](https://… 没有右括号）
const MEDIA_SHORTCODE_BROKEN_RE = /@\[(?:live|video|audio|file)\]\s*\([^)]*$/g
// 连括号都没有的残留标记（渲染器判参数不足时原样输出）
const MEDIA_SHORTCODE_TAG_RE = /@\[(?:live|video|audio|file)\](?!\()/g

const isUrlLike = (value) => /^(?:https?:)?\/\//i.test(value) || value.startsWith('/')

/**
 * 正文 / 摘要转纯文本（列表摘要、摘要展示用）
 *
 * 只做轻量字符串清洗，**不走 markdown-it**：列表一页十几张卡，为了几百字摘要
 * 去加载 markdown-it + highlight.js（gzip 约 88KB）并逐篇解析并不划算。
 *
 * 重点是媒体短代码 —— `@[live](视频 "封面" "说明")` 这类标记是给渲染器看的，
 * 直接截断展示会露出 `@live`、封面图地址之类的字样：
 * 能取到标题 / 说明就换成那段文字（说明 → live 第三参数、file 的文件名、
 * video / audio 的标题），取不到就整体丢掉；被截断的残缺短代码也一并清掉。
 *
 * 图片、链接、HTML 标签、行首记号与强调符号都清掉；`[emoji:url]` 保留，
 * 交给 EmojiText 渲染成表情图。
 */
export const stripArticleMarkup = (text) => {
  return String(text || '')
    .replace(/```[\s\S]*?```/g, ' ') // 代码块整体丢掉
    .replace(MEDIA_SHORTCODE_RE, (_match, _kind, args) => {
      // 引号里的参数都是文字；live 的封面图是 URL，过滤掉不当作摘要
      const labels = [...String(args).matchAll(/"([^"]*)"/g)]
        .map((item) => item[1].trim())
        .filter((label) => label && !isUrlLike(label))
      return labels.length ? ` ${labels[0]} ` : ' '
    })
    .replace(MEDIA_SHORTCODE_BROKEN_RE, ' ')
    .replace(MEDIA_SHORTCODE_TAG_RE, ' ')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ') // 图片
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1') // 链接 → 只留文字
    .replace(/<[^>]*>/g, ' ') // 富文本残留标签
    .replace(/^[>#*+\-\s]{1,6}/gm, '') // 行首标题 / 引用 / 列表记号
    .replace(/[*_~`]+/g, '') // 强调符号
    .replace(/\s+/g, ' ')
    .trim()
}

/**
 * 获取 cookie（统一复用 utils/cookie，避免实现分叉）
 */
export { getCookie } from './cookie'

/**
 * 提取 result.author.user 之类的关联数据
 */
export const pickAuthor = (item) => {
  return item?.result?.user?.nickname
    ? item.result.user
    : item?.result?.author || item?.user || { id: item?.uid, nickname: '匿名', avatar: '' }
}

/**
 * 解析评论/动态的作者信息（兼容后端多种关联结构）
 * comment/flat 实际把作者放在 result.author
 */
export const pickCommentAuthor = (item) => {
  const author =
    item?.result?.author ||
    item?.author ||
    item?.user ||
    (item?.result && item.result.user ? item.result.user : null) ||
    {}
  return {
    id: author?.id || item?.uid || null,
    nickname: author?.nickname || item?.nickname || '匿名',
    avatar: author?.avatar || item?.avatar || '',
    title: pickUserTitle(author),
    level: pickUserLevel(author),
    // json 里带佩戴中的勋章（wearing_badge），评论昵称旁要渲染它
    json: author?.json || item?.json || {},
    // 认证摘要：后端 SanitizeAuthor 注入到内嵌作者对象，头像右下角的盾牌读它
    cert: author?.cert || item?.cert || null
  }
}

/**
 * 提取用户等级（兼容 result.level.current / users_rating.users_grade / users_rating.grade / grade）
 * 参考作者主页 Author.vue 的取值逻辑
 */
export const pickUserLevel = (user) => {
  if (!user || typeof user !== 'object') return 0
  const lv = user?.result?.level?.current || user?.users_rating?.users_grade
  return lv?.value ?? user?.users_rating?.grade ?? user?.grade ?? 0
}

/**
 * 提取用户头衔
 */
export const pickUserTitle = (user) => {
  return user?.title || ''
}

/**
 * 头衔配色映射（与 Profile 页 10 个头衔一致）
 */
const titleColorMap = {
  '掌门': 'title-zhangmen',
  '长老': 'title-zhanglao',
  '护法': 'title-hufa',
  '内门弟子': 'title-neimen',
  '外门弟子': 'title-waimen',
  '炼气修士': 'title-lianqi',
  '筑基修士': 'title-zhuji',
  '结丹修士': 'title-jiedan',
  '元婴老祖': 'title-yuanying',
  '化神大能': 'title-huashen',
  // 兼容旧头衔
  '侠客': 'title-xiake',
  '学徒': 'title-xuetu'
}
export const getTitleColorClass = (title) => {
  return titleColorMap[title] || 'title-default'
}

/**
 * 提取用户佩戴的头衔样式（装扮系统写入 users.json.title_decoration.payload）
 *
 * 有值时应当用 <TitleBadge :payload="..." :text="user.title" /> 渲染 ——
 * 后台配的文字颜色 / 背景 / 边框 / 发光 / 图标 / 动效都在 payload 里，
 * 只按文字查色板（getTitleColorClass）会把这些样式全部丢掉
 * （表现为「后台设了颜色，前台一直显示默认色」）。
 *
 * 没有样式（空 payload / 历史纯文本头衔）时返回 null，调用方回退到旧色板渲染。
 */
export const pickUserTitlePayload = (user) => {
  const deco = user?.json?.title_decoration
  if (!deco) return null

  // 兼容两种历史写法：{id, name, payload} 与「直接就是 payload」
  let payload = deco.payload ?? deco
  if (!payload) return null
  if (typeof payload === 'string') {
    try {
      payload = JSON.parse(payload)
    } catch {
      return null
    }
  }
  if (typeof payload !== 'object') return null

  // 一个样式字段都没有时视为「无样式」，交给色板兜底（避免套上 TitleBadge 的默认皮肤）
  // image 也算样式字段：图片头衔（没有文字、只有徽章图）同样必须走 TitleBadge 渲染
  const styleKeys = ['image', 'color', 'bg', 'border', 'glow', 'icon', 'animated']
  return styleKeys.some((key) => payload[key]) ? payload : null
}

/** 头像框是否开启动效（后端佩戴时回写 users.json.frame_animated） */
export const pickUserFrameAnimated = (user) => {
  const flag = user?.json?.frame_animated
  return flag === true || Number(flag) === 1
}

/**
 * 头像框的定位参数（后端佩戴时按装扮 payload 回写到 users.json）
 *
 * 返回 { hole, offsetX, offsetY }，直接透传给 AvatarFrame：
 *   - hole      洞直径 ÷ 素材宽度，1 表示素材按头像大小 1:1 渲染
 *   - offsetX/Y 洞心偏移，单位是素材宽度的比例
 *
 * 老数据没有这几个字段时返回默认值（1 / 0 / 0），等同于「素材就是贴合头像的」。
 */
export const pickUserFrameGeometry = (user) => {
  const json = user?.json || {}
  return {
    hole: Number(json.frame_hole) > 0 ? Number(json.frame_hole) : 1,
    offsetX: Number(json.frame_offset_x) || 0,
    offsetY: Number(json.frame_offset_y) || 0
  }
}

/**
 * 从装扮 payload 解析头像框几何参数（后台对齐预览 / 前台佩戴预览共用）
 *
 * 与后端 app/model/decoration.go 的 frameGeometry 同口径：
 *   - 优先取 hole；
 *   - 没有 hole 时按旧的 scale 换算（hole = 1/scale），老装扮保持原观感；
 *   - 都没有则 hole = 1（素材按头像大小 1:1 渲染）。
 */
export const frameGeometryFromPayload = (payload = {}) => {
  const rawHole = Number(payload?.hole)
  const rawScale = Number(payload?.scale)

  let hole = 1
  if (rawHole > 0) hole = rawHole
  else if (rawScale > 0) hole = 1 / rawScale

  return {
    hole,
    offsetX: Number(payload?.offset_x) || 0,
    offsetY: Number(payload?.offset_y) || 0
  }
}

/**
 * 判断用户是否为管理员
 * 兼容两种用户结构：user.auth（顶层）和 user.result.auth（嵌套）
 * 参考主题实现：auth.all 或 auth.group.list 中存在 key === 'admin'
 */
export const isAdmin = (user) => {
  if (!user || typeof user !== 'object') return false
  // 系统管理员（id=1）恒为管理员：与后端口径一致（adminGroupId=1），
  // 避免个别响应没带 result.auth 时把管理员当普通用户
  if (Number(user?.id) === 1) return true
  // 兼容 result.auth 和顶层 auth
  const auth = user?.result?.auth || user?.auth
  if (!auth) return false
  // 拥有 all 权限
  if (auth.all === true || auth.all === 1) return true
  // 权限组中存在 admin
  const groups = auth?.group?.list || auth?.group || []
  if (Array.isArray(groups)) {
    return groups.some((g) => g?.key === 'admin')
  }
  return false
}

/**
 * 用户是否为后台页面权限集合（供菜单 / 路由过滤，见 views/admin/Layout.vue）
 *
 * @returns {null|Set<string>} null 表示拥有全部页面（系统管理员 / pages=all），
 *                             否则为允许访问的后台页面 path 集合（空集合 = 没有任何后台页面）
 *
 * ⚠ 空 pages 必须解释为「没有任何后台页面」：
 * 后端 auth-group 里普通用户分组的 pages 就是空串（见 model/auth-group.go 的
 * builtinAuthGroups），若这里把空当成「全部」，普通用户会看到整个后台菜单。
 */
export const adminAllowedPaths = (user, flat) => {
  const list = Array.isArray(flat) ? flat : []

  // 系统管理员：接口权限 all / id=1 / admin 组 —— 全部页面
  if (Number(user?.id) === 1) return null

  const auth = user?.result?.auth || user?.auth
  if (!auth) return new Set()

  if (auth.all === true || auth.all === 1) return null

  const groups = auth?.group?.list || auth?.group || []
  if (Array.isArray(groups) && groups.some((g) => g?.key === 'admin')) return null

  const pagesHash = auth?.pages?.hash
  // 空 = 未授予任何后台页面（普通用户组）；含 all = 全部页面
  if (!Array.isArray(pagesHash) || !pagesHash.length) return new Set()
  if (pagesHash.includes('all')) return null

  const uniq = [...new Set(pagesHash)].filter(Boolean)
  return new Set(list.filter((p) => uniq.includes(p.hash)).map((p) => p.path))
}

/**
 * 判断用户是否可以进入管理后台
 *
 * 与 isAdmin 的区别：
 * - isAdmin 是「超级管理员」口径（拥有全部接口权限 all，或权限组 key === 'admin'），
 *   用于「能管理一切」的场景（动态置顶 / 删除等）；
 * - 这里判断的是「有没有后台页面权限」：权限组只要给了任意后台页面（auth-group.pages），
 *   就允许进入后台（菜单与路由会再按这些页面过滤），这样非 admin 的运营组也能进入
 *   自己那部分页面。
 *
 * ⚠ pages 为空 = 没有后台页面权限（普通用户组的默认状态），不能当成「全部」：
 * 否则每个注册用户都能看到后台入口和整套菜单。
 */
export const canEnterAdmin = (user) => {
  if (!user || typeof user !== 'object') return false
  // 系统管理员（id=1）始终显示后台入口：即使某个响应没带 result.auth，
  // 也不该把管理员挡在后台外面（后端同样把 id=1 视作系统管理员）
  if (Number(user?.id) === 1) return true
  const auth = user?.result?.auth || user?.auth
  if (!auth) return false
  if (auth.all === true || auth.all === 1) return true

  const groups = auth?.group?.list || auth?.group || []
  if (Array.isArray(groups) && groups.some((g) => g?.key === 'admin')) return true

  const pages = auth?.pages?.hash
  if (!Array.isArray(pages) || !pages.length) return false
  return pages.includes('all') || pages.some(Boolean)
}