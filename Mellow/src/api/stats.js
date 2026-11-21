import { call } from './request'

/**
 * 数据统计接口封装（后台）
 *
 * 后端见 app/api/controller/stats.go：只有 CPU / 内存 / 磁盘的**告警阈值**读写
 * （存 config 表 SYSTEM_ALERT_THRESHOLD，判定在服务端定时采样里，见 app/timer/stats.go）。
 *
 * 注意：页面上的实时数据（资源曲线、在线人数）不走 HTTP，而是 WebSocket 推送，
 * 见 @/utils/socket 与 docs/socket.md。
 */

// 读取告警阈值
export const getStatsAlert = () => call('stats', 'alert', { method: 'GET', config: { dedupe: false } })

// 保存告警阈值（仅超级管理员）：{ enabled, cpu, mem, disk, cooldown }
export const saveStatsAlert = (data) => call('stats', 'alert', { method: 'POST', data })
