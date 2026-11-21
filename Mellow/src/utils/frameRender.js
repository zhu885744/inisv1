import { reactive } from 'vue'
import { getDecorationOne } from '@/api/decoration'
import { frameGeometryFromPayload, pickUserFrameGeometry, pickUserFrameAnimated } from './helper'

/**
 * 头像框渲染参数解析
 *
 * 为什么不直接用 users.json 上的 frame_hole：
 * 那几个字段是「佩戴那一刻」抄进 users.json 的副本。后台调了洞大小 / 偏移之后，
 * 副本只有在「重新佩戴」「保存装扮」「后端重启回填」这些时机才会更新，
 * 任何一个环节没跑到，前台就一直显示旧尺寸（表现就是「后台改了，前台没反应」）。
 *
 * 装扮的 payload 才是唯一真相。这里按 decorations.avatar_frame 的 id 取一次详情现算，
 * 结果按 id 缓存在内存里 —— 一个页面里侧栏 / 资料卡 / 作者页共用，只会请求一次。
 *
 * 取不到（未登录、网络失败、装扮已删）时退回 users.json 上的回写值，保证老数据也能渲染。
 */

// id -> payload（null 表示请求进行中或失败，不再重复请求）
const payloadCache = reactive({})
const pendingIds = new Set()

function ensurePayload(id) {
  if (!id || pendingIds.has(id) || id in payloadCache) return

  pendingIds.add(id)
  getDecorationOne(id)
    .then((res) => {
      payloadCache[id] = res?.data?.payload || null
    })
    .catch(() => {
      payloadCache[id] = null
    })
    .finally(() => {
      pendingIds.delete(id)
    })
}

function isAnimated(flag) {
  return flag === true || Number(flag) === 1
}

/**
 * 解析某个用户当前头像框的渲染参数
 *
 * @param {object} json 用户的 json 字段（users.json），需要里面的 decorations / frame_hole 等
 * @returns {{hole:number, offsetX:number, offsetY:number, animated:boolean}}
 */
export function resolveFrameRender(json = {}) {
  const wearing = json?.decorations || {}
  const id = Number(wearing.avatar_frame) || 0

  ensurePayload(id)

  const payload = id ? payloadCache[id] : null
  if (payload) {
    return {
      ...frameGeometryFromPayload(payload),
      animated: isAnimated(payload.animated)
    }
  }

  // 装扮详情还没回来 / 没有 id：先用 users.json 上的回写值顶上
  return {
    ...pickUserFrameGeometry({ json }),
    animated: pickUserFrameAnimated({ json })
  }
}

/** 装扮被编辑后清掉缓存（后台保存装扮时调用，避免同一会话里还拿到旧 payload） */
export function invalidateFrameRenderCache(id = 0) {
  const key = Number(id)
  if (key) {
    delete payloadCache[key]
    pendingIds.delete(key)
    return
  }
  Object.keys(payloadCache).forEach((key) => delete payloadCache[key])
  pendingIds.clear()
}
