/**
 * 敏感操作的身份验证（组合式，登录态下证明「操作者是本人」）
 *
 * 适用场景：修改密码 / 修改账号 / 修改邮箱 / 修改手机号。
 * 后端提供两种凭证（见 app/api/controller/users-identity.go 的 verifyIdentity）：
 *   - 当前密码                → 参数 password
 *   - 已绑定联系方式的验证码  → 参数 email_code / phone_code（发送见 PUT /api/users/verify-code）
 *
 * 把「模式切换 + 发码倒计时 + 参数组装」收在一处，各页面只绑定表单字段，
 * 避免每个页面各写一份（账号安全页与联系方式页共用）。
 *
 * 用法：
 *   const { mode, sending, countdown, hasBoundContact, targetLabel, toggle, send, payload } = useIdentityVerify()
 *   // 提交时：{ ...form, ...payload(form) }
 * 未设置密码、也未绑定任何联系方式的账号（纯第三方登录注册）没有可验证的凭证：
 * 后端允许留空通过（见 verifyIdentityFor），页面把提示写清楚即可。
 *
 * options.prefer：优先使用的联系方式通道（'email' | 'sms'），默认 'email'（与后端
 * verify-code 不传 mode 时的「邮箱优先」保持一致）。
 * 邮箱设置卡片传 'email'、手机号设置卡片传 'sms'：验证码发给「与该卡片同类」的已绑定联系方式，
 * 避免在手机号卡片里出现「改用邮箱验证码」；prefer 的通道未绑定时自动回退另一种联系方式。
 */
import { ref, computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { sendIdentityCode } from '@/api/users'
import { toast } from '@/utils/toast'

export function useIdentityVerify(options = {}) {
  const userStore = useUserStore()

  const prefer = options.prefer === 'sms' ? 'sms' : 'email'

  // password：当前密码；code：发到已绑定联系方式的验证码
  const mode = ref('password')
  const sending = ref(false)
  const countdown = ref(0)
  let timer = null

  const boundEmail = computed(() => Boolean((userStore.user || {}).email))
  const boundPhone = computed(() => Boolean((userStore.user || {}).phone))

  const hasBoundContact = computed(() => boundEmail.value || boundPhone.value)

  // 实际使用的通道：优先 prefer，未绑定则回退另一种；都没有则为空（不可发码）
  const channel = computed(() => {
    if (prefer === 'sms') {
      if (boundPhone.value) return 'sms'
      return boundEmail.value ? 'email' : ''
    }
    if (boundEmail.value) return 'email'
    return boundPhone.value ? 'sms' : ''
  })

  const targetLabel = computed(() => (channel.value === 'sms' ? '手机号' : '邮箱'))

  function toggle() {
    mode.value = mode.value === 'password' ? 'code' : 'password'
  }

  async function send() {
    if (sending.value || countdown.value > 0) return false
    if (!hasBoundContact.value) {
      toast.error('当前账号未绑定邮箱或手机号')
      return false
    }

    sending.value = true
    try {
      await sendIdentityCode(channel.value === 'sms' ? 'sms' : 'email')
      toast.success(`验证码已发送至你绑定的${targetLabel.value}`)
      countdown.value = 60
      if (timer) clearInterval(timer)
      timer = setInterval(() => {
        if (countdown.value > 0) countdown.value--
        else { clearInterval(timer); timer = null }
      }, 1000)
      return true
    } catch {
      // 拦截器已提示（服务未开启 / 发送过于频繁等）
      return false
    } finally {
      sending.value = false
    }
  }

  /**
   * 组装身份验证参数
   * @param {{password?: string, identityCode?: string}} form 表单对象
   * @returns {{password: string} | {email_code: string} | {phone_code: string}}
   */
  function payload(form = {}) {
    if (mode.value === 'code') {
      return channel.value === 'sms' ? { phone_code: form.identityCode } : { email_code: form.identityCode }
    }
    return { password: form.password }
  }

  // 组件卸载时清理倒计时
  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  return { mode, sending, countdown, hasBoundContact, targetLabel, channel, toggle, send, payload, stop }
}
