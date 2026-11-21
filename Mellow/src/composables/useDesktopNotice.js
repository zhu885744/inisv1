import { computed, ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { toast } from '@/utils/toast'
import {
  desktopNoticePermission,
  desktopNoticeSupported,
  requestDesktopNoticePermission,
  showDesktopNotice
} from '@/utils/desktopNotify'

/**
 * 系统级桌面通知的开关逻辑（设置页与消息页面板共用）
 *
 * 权限申请必须发生在用户点击的调用栈里，所以这里把「点击 → 申请 → 落偏好」
 * 收成一个 toggle()，两个入口只管调用，避免各写一套出现行为不一致。
 * 偏好本身仍存在 chat store（localStorage），这里只负责与浏览器权限打交道。
 */
export function useDesktopNotice() {
  const chatStore = useChatStore()

  const supported = desktopNoticeSupported()
  const permission = ref(desktopNoticePermission())

  // 真正的开关状态：用户开了，且浏览器确实授权了（浏览器设置里撤销后要能自愈）
  const enabled = computed(() => !!chatStore.alert.desktop && permission.value === 'granted')
  // 浏览器层面被拒绝：再点也不会弹授权窗，只能引导用户去站点设置里改
  const blocked = computed(() => permission.value === 'denied')

  async function toggle() {
    // 已经真正开启：直接关
    if (enabled.value) {
      chatStore.setAlert('desktop', false)
      return
    }

    // 偏好还开着但浏览器权限已被撤销：先把失效的偏好归位，再走一次申请
    if (chatStore.alert.desktop) chatStore.setAlert('desktop', false)

    if (!supported) {
      toast.warning('当前环境不支持桌面通知（浏览器要求 HTTPS）')
      return
    }

    // 关键：必须在点击的同步调用栈里申请，晚了会被浏览器判定为「已阻止」
    const result = await requestDesktopNoticePermission()
    permission.value = result

    if (result === 'granted') {
      chatStore.setAlert('desktop', true)

      // 立刻弹一条示例通知：用户当场就能确认系统层面确实收到了
      showDesktopNotice({
        title: '桌面通知已开启',
        body: '切到其他标签页或最小化窗口，也能收到新私信提醒',
        tag: 'mellow-chat-demo'
      })
      return
    }

    if (result === 'denied') {
      toast.warning('浏览器已拒绝通知权限，请在地址栏左侧的站点设置里允许通知')
      return
    }

    // default：用户把授权弹窗直接关掉了
    toast.info('未获得通知权限，可稍后再试')
  }

  return { supported, permission, enabled, blocked, toggle }
}
