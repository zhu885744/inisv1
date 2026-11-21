<template>
  <!--
    滑块验证用的「图片」素材：内联 SVG 实景（天空 / 太阳 / 云 / 远近山峦）
    - 不依赖任何外部图片：加载零延迟、无跨域与断网风险，也不会被 CDN 缓存问题影响
    - 尺寸由外部给定并 preserveAspectRatio="none" 拉伸，保证「轨道」与「拼图块」
      用同一张图的两个实例，切出来的纹理能严丝合缝地接上
  -->
  <svg
    class="captcha-scene"
    :width="width"
    :height="height"
    viewBox="0 0 300 52"
    preserveAspectRatio="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
  >
    <defs>
      <linearGradient id="cs-sky" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#8ab4e8" />
        <stop offset="58%" stop-color="#c6daf2" />
        <stop offset="100%" stop-color="#eaf1fa" />
      </linearGradient>
      <linearGradient id="cs-far" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stop-color="#5b8dc4" />
        <stop offset="100%" stop-color="#83abda" />
      </linearGradient>
      <linearGradient id="cs-near" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#3d739f" />
        <stop offset="100%" stop-color="#26496a" />
      </linearGradient>
    </defs>

    <!-- 天空 -->
    <rect x="0" y="0" width="300" height="52" fill="url(#cs-sky)" />

    <!-- 太阳（含光晕） -->
    <circle cx="238" cy="15" r="16" fill="#ffd98a" opacity="0.25" />
    <circle cx="238" cy="15" r="9" fill="#ffe1a3" />

    <!-- 云 -->
    <ellipse cx="58" cy="13" rx="22" ry="6.5" fill="#ffffff" opacity="0.8" />
    <ellipse cx="76" cy="16" rx="13" ry="4.5" fill="#ffffff" opacity="0.65" />
    <ellipse cx="188" cy="10" rx="16" ry="4.5" fill="#ffffff" opacity="0.5" />

    <!-- 远山 -->
    <path
      d="M0 34 L44 18 L86 34 L124 14 L166 36 L206 22 L248 38 L300 24 L300 52 L0 52 Z"
      fill="url(#cs-far)"
    />
    <!-- 近景山脊 -->
    <path
      d="M0 43 L56 33 L112 45 L176 35 L240 47 L300 39 L300 52 L0 52 Z"
      fill="url(#cs-near)"
    />
  </svg>
</template>

<script setup>
defineProps({
  width: { type: Number, default: 300 },
  height: { type: Number, default: 52 }
})
</script>

<style scoped>
.captcha-scene {
  display: block;
  pointer-events: none;
}
</style>
