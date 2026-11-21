/**
 * 滑块验证网关（全局单例）
 *
 * 为什么要有这一层：发送验证码被后端判定为「5 分钟内重复请求」时会返回
 * code=428 + { need_captcha: true, captcha: {...} }。这个信号可能出现在任何一个
 * 发送验证码的入口（验证码登录 / 注册 / 找回密码 / 改绑邮箱手机号 / 注销账户），
 * 如果每个调用方都自己弹一次滑块，就会散落一堆重复逻辑。
 *
 * 于是：request.js 的响应拦截器统一识别 need_captcha → 调用这里的 requestSliderCaptcha
 * → 由 App.vue 挂载的 <SliderCaptcha> 弹窗完成交互 → 拦截器拿到票据后自动重放原请求。
 * 业务代码完全不需要感知滑块的存在。
 *
 * 注意：这里不 import 任何业务模块，避免与 request.js 形成循环依赖。
 */

let sliderHandler = null

/** 注册滑块弹窗的实现（由 SliderCaptcha 组件在挂载时调用） */
export function registerSliderHandler(handler) {
  sliderHandler = handler
}

/** 注销（组件卸载时调用，避免持有已销毁组件的引用） */
export function unregisterSliderHandler() {
  sliderHandler = null
}

/**
 * 请求用户完成一次滑块验证
 * @param {object} challenge 后端下发的题目 { token, target, width, tolerance, expire }
 * @returns {Promise<{token: string, value: number}|null>} 用户取消或组件未挂载时返回 null
 */
export function requestSliderCaptcha(challenge) {
  if (typeof sliderHandler !== 'function') {
    return Promise.resolve(null)
  }
  return Promise.resolve(sliderHandler(challenge)).catch(() => null)
}
