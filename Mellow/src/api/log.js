import { call } from './request'

/**
 * 日志解析（后台运维）接口封装 —— 只读
 *
 * 后端见 app/api/controller/log.go，三类数据：
 *   - dates：可查看的日期（runtime/logs/<日期>）+ 通知日志文件（runtime/sms）
 *   - files：某一天的日志文件列表（含 lumberjack 轮转备份 info-2026-10-10T12-00-00.000.log）
 *   - read ：读取并解析日志条目（级别 / 关键字过滤 + 分页，默认最新在前）
 *
 * 解析结构（每条）：{ time, level, msg, caller, fields, raw }
 *   - 系统日志是 zap 的 JSON 行 → 业务字段进 fields
 *   - 通知日志是文本行 → "k: v" 明细进 fields
 *   - 解析不出来时只有 raw（原样展示，保证任何一行都能看到）
 */

// 可查看的日期列表 + 通知日志文件（data: { dates, notify, today }）
export const listLogDates = () =>
  call('log', 'dates', { method: 'GET', config: { dedupe: false } })

// 某一天的日志文件列表；params.scope = 'notify' 时返回通知日志文件
export const listLogFiles = (params = {}) =>
  call('log', 'files', { method: 'GET', params, config: { dedupe: false } })

// 读取并解析日志：scope(system/notify) / date / file / level / keyword / page / limit / order
export const readLog = (params = {}) =>
  call('log', 'read', {
    method: 'GET',
    params: { limit: 50, ...params },
    config: { dedupe: false, silent: true }
  })
