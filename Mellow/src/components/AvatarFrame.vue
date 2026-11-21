<template>
  <span class="avatar-frame" :style="{ width: size, height: size }">
    <!-- 底层圆形用户头像：没有头像地址时不渲染 <img>，改用昵称首字占位 -->
    <img
      v-if="currentSrc"
      :src="currentSrc"
      :alt="alt"
      class="avatar-frame-img"
      loading="lazy"
      decoding="async"
      @error="onError"
    />
    <span v-else class="avatar-frame-text" :style="textStyle" aria-hidden="true">{{ initial }}</span>
    <!-- 顶层头像框：按「洞」定位，装饰可自由向外延伸 -->
    <img
      v-if="frame && !frameFailed"
      :src="frameSrc"
      class="avatar-frame-overlay"
      :class="{ 'is-animated': animated }"
      :style="frameStyle"
      alt=""
      loading="lazy"
      decoding="async"
      @error="frameFailed = true"
    />
    <!--
      右下角角标：目前用于认证标识（VerifiedBadge）

      z-index 必须大于头像框的 2，否则会被放大的头像框素材盖住；
      容器刻意不设 overflow:hidden（见 .avatar-frame），角标才能探出头像边界。
      也支持 #badge 插槽自定义内容（比如以后要挂角标式勋章）。
    -->
    <span
      v-if="verified?.title || $slots.badge"
      class="avatar-frame-badge"
      :style="badgeStyle"
    >
      <slot name="badge">
        <!-- dot 形态：圆形实心 + 白色对勾，贴在头像右下角 -->
        <VerifiedBadge v-if="verified?.title" :cert="verified" size="sm" variant="dot" />
      </slot>
    </span>
  </span>
</template>

<script setup>
import { ref, computed } from 'vue'
import VerifiedBadge from './VerifiedBadge.vue'
import { thumbUrl } from '@/utils/image'

// 与后端 app/model/decoration.go 的 frameHoleMin / frameHoleMax / frameOffsetLimit 保持一致
const HOLE_MIN = 0.2
const HOLE_MAX = 1.5
const OFFSET_LIMIT = 0.5

const props = defineProps({
  src: { type: String, default: '' },
  frame: { type: String, default: '' },
  alt: { type: String, default: '头像' },
  /**
   * 占位文字来源（昵称）：头像为空时取其首字，比如「云溪」→「云」
   *
   * 与 alt 分开是因为 alt 常常是「用户头像」「头像预览」这类描述性文案，
   * 直接取首字会变成「用」「头」。不传时回退 alt，并过滤掉含「头像」的文案。
   */
  name: { type: String, default: '' },
  size: { type: String, default: '80px' },
  /**
   * 头像框「洞」的大小：洞直径 ÷ 素材宽度
   *
   * 1   = 洞正好是素材的内切圆，素材按头像大小 1:1 渲染
   * 0.6 = 素材要放大到 1.67 倍，洞才等于头像（装饰画得大的框通常是这个量级）
   *
   * 数据来自 users.json.frame_hole（佩戴时由服务端回写），
   * 没配过的老数据取默认 1，等同于「素材就是贴合头像的」。
   */
  frameHole: { type: Number, default: 1 },
  /** 洞心相对素材中心的横向偏移，单位=素材宽度的比例（修正画布留白不对称） */
  frameOffsetX: { type: Number, default: 0 },
  /** 洞心相对素材中心的纵向偏移，同上 */
  frameOffsetY: { type: Number, default: 0 },
  // 头像加载失败时的回退图
  fallback: { type: String, default: '' },
  /**
   * 是否播动效（后台装扮的「开启动效」）
   * 数据来自 users.json.frame_animated（佩戴头像框时由服务端回写）
   */
  animated: { type: Boolean, default: false },
  /**
   * 认证摘要（后端 CertSummary：{ title, cert_type, type_text, color, ... }）
   *
   * 有值时在头像右下角渲染 VerifiedBadge；没有 title 视为无认证，不渲染。
   * 列表场景的 cert 由后端 SanitizeAuthor 注入到内嵌作者对象里，
   * 个人主页则来自 user-certification/one。
   */
  verified: { type: Object, default: null }
})

const failed = ref(false)
const frameFailed = ref(false)

function clampOffset(value) {
  const num = Number(value)
  if (!Number.isFinite(num)) return 0
  return Math.min(OFFSET_LIMIT, Math.max(-OFFSET_LIMIT, num))
}

// 洞大小（素材宽度 ÷ 头像直径的倒数），范围与后端一致
const hole = computed(() => {
  const raw = Number(props.frameHole)
  return Number.isFinite(raw) && raw > 0 ? Math.min(HOLE_MAX, Math.max(HOLE_MIN, raw)) : 1
})

// 用 CSS 变量传参，避免在模板里拼一长串 transform
const frameStyle = computed(() => ({
  '--frame-hole': hole.value,
  '--frame-offset-x': clampOffset(props.frameOffsetX),
  '--frame-offset-y': clampOffset(props.frameOffsetY)
}))

// 头像的实际显示宽度（size 形如 "44px"）：取 2 倍图，高 DPI 屏也清晰
const displayWidth = computed(() => {
  const px = parseFloat(String(props.size))
  return Number.isFinite(px) && px > 0 ? px : 80
})

/**
 * 实际渲染的头像地址：
 *   加载失败 → fallback（灰底圆形占位图）
 *   没有地址 → 空字符串，交给模板渲染「昵称首字」
 *
 * 为什么区分这两种空：公开页面里作者头像是空的，并不一定是图挂了 ——
 * 由 QQ 邮箱推导出来的 qlogo 头像会在脱敏时被置空（见 app/model/privacy.go
 * 的 hideDerivedAvatar）。这种情况显示首字比显示灰圆更像「有头像」，也不泄露 QQ 号。
 *
 * 统一按显示尺寸取缩略图：头像位只有 20~90px，原来拉的是 1500~2000px 的原图
 * （单张 1.2~1.5MB），是首屏流量与解码耗时的大头，见 utils/image.js。
 */
const currentSrc = computed(() => thumbUrl(failed.value ? props.fallback : props.src, displayWidth.value * 2))

// 头像框素材会被放大（洞越小放大越多），按放大后的尺寸取图，上限 800px
const frameSrc = computed(() =>
  thumbUrl(props.frame, Math.min(800, Math.ceil((displayWidth.value * 2) / hole.value)))
)

// 占位首字：优先昵称，其次 alt；alt 是「用户头像」「头像预览」这类描述文案时不用
const initial = computed(() => {
  const nickname = String(props.name || '').trim()
  const altText = String(props.alt || '').trim()
  const source = nickname || (/头像/.test(altText) ? '' : altText)
  return source ? [...source][0].toUpperCase() : '用'
})

// 占位字号跟随头像尺寸（size 形如 "40px"）：40px → 17px，75px → 32px
const textStyle = computed(() => {
  const px = parseFloat(String(props.size))
  const fontSize = Number.isFinite(px) && px > 0 ? Math.max(12, Math.round(px * 0.42)) : 18
  return { fontSize: `${fontSize}px` }
})

// 角标字号跟随头像尺寸：75px → 21px，44px → 13px（下限 11px，再小就认不出是盾牌）
const badgeStyle = computed(() => {
  const px = parseFloat(String(props.size))
  const fontSize = Number.isFinite(px) && px > 0 ? Math.max(11, Math.round(px * 0.28)) : 18
  return { '--avatar-badge-font': `${fontSize}px` }
})

function onError() {
  if (props.fallback) failed.value = true
}
</script>

<style scoped>
.avatar-frame {
  position: relative;
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  /* 不要加 overflow:hidden，否则放大后的头像框装饰会被截断 */
}

/* 用户头像：占满容器，圆形裁剪 */
.avatar-frame-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 50%;
  display: block;
  z-index: 1;
}

/* 昵称首字占位：与旧灰圆占位同底色，但带上可辨识的文字 */
.avatar-frame-text {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #e8e6dd;
  color: #8a8a82;
  font-weight: 600;
  line-height: 1;
  user-select: none;
  z-index: 1;
}

/*
 * 右下角角标（认证标识）
 *
 * 定位：right/bottom 贴 0、不做 translate。
 * 圆形容器里，这样徽章圆心到头像圆心的距离 ≈ 半径，正好落在头像右下边缘上
 * （和主流 App 的认证角标一致）；再加 translate 就会整块飘到头像外面。
 * z-index 必须高于头像框的 2。font-size 由 badgeStyle 按头像尺寸注入。
 */
.avatar-frame-badge {
  position: absolute;
  right: 0;
  bottom: 0;
  z-index: 3;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: var(--avatar-badge-font, 15px);
  line-height: 1;
  cursor: help;
}

/*
 * VerifiedBadge 自带的 sm 字号（12px）会盖掉父级 font-size，
 * 这里强制继承，角标才能真正跟随头像大小缩放。
 * 双圈白色 drop-shadow 当作描边：浅色头像上也看得清盾牌轮廓。
 */
.avatar-frame-badge :deep(.verified-badge) {
  font-size: inherit;
  filter: drop-shadow(0 0 1.5px rgba(255, 255, 255, 0.95))
    drop-shadow(0 1px 2px rgba(0, 0, 0, 0.18));
}

/*
 * 头像框：以「洞」为基准定位
 *
 * --frame-hole      洞直径 ÷ 素材宽度 → 素材宽度取「头像直径 / hole」
 * --frame-offset-x  洞心横向偏移（素材宽度的比例）
 * --frame-offset-y  洞心纵向偏移
 *
 * height:auto 让 <img> 保持素材自身比例：非正方形素材也不会被拉伸，
 * 同时 translate(-50%,-50%) 正好以图片实际尺寸居中。
 */
.avatar-frame-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  width: calc(100% / var(--frame-hole, 1));
  /*
   * 必须解除全局 `img { max-width: 100% }` 的限制：
   * 洞模型靠「宽度 > 头像」把素材放大，而 max-width 会把它卡在 100%，
   * 表现就是「洞大小调了没反应、但偏移有效」（偏移走 transform，不受 max-width 影响）。
   */
  max-width: none;
  height: auto;
  pointer-events: none;
  z-index: 2;
  transform-origin: center center;
  /* translate 的百分比以元素自身宽度为基准：先居中，再按偏移量平移 */
  transform: translate(
    calc(-50% - var(--frame-offset-x, 0) * 100%),
    calc(-50% - var(--frame-offset-y, 0) * 100%)
  );
}

/* 开启动效：亮度呼吸 + 泛光（与头衔的呼吸动效风格保持一致） */
.avatar-frame-overlay.is-animated {
  animation: avatar-frame-breath 2.4s ease-in-out infinite;
}

@keyframes avatar-frame-breath {
  0%,
  100% {
    filter: brightness(1) drop-shadow(0 0 0 rgba(255, 255, 255, 0));
  }
  50% {
    filter: brightness(1.22) drop-shadow(0 0 6px rgba(255, 255, 255, 0.55));
  }
}
</style>
