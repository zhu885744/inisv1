import { devService } from './request'

/**
 * 安装向导接口（后端 app/dev/controller/install.go）
 *
 * 走 devService（不带 /api 前缀）：未安装时 /api 会被 Install 中间件整体拦住（412），
 * 而这些接口正是用来完成安装的，所以必须绕开；装好之后它们自己也会返回 412。
 *
 * 请求体统一用 JSON 对象：devService 实例的默认头就是 application/json，
 * 若改用 URLSearchParams，axios 会带出「头是 JSON、体是表单」的错配
 * （它对 URLSearchParams 用的是 setContentType(..., false)，不会覆盖实例默认头），
 * 后端按 JSON 解析失败 → 全部参数静默丢失，表现为「username 不能为空！」。
 */

const BASE = '/dev/install'

/** 安装状态：data 为 true 表示已完成安装（见 facade.Installed） */
export const installCheck = () => devService.get(`${BASE}/check`)

/** 运行环境信息（系统 / 架构 / Go 版本 / 程序版本 / CPU 核数），用于安装向导第一步展示 */
export const installInfo = () => devService.get(BASE)

/** 写入数据库配置（生成 config/database.toml） */
export const installConnectDB = (data) => devService.post(`${BASE}/connect-db`, { ...data })

/** 初始化数据表并创建默认管理员账号 */
export const installInitDB = () => devService.post(`${BASE}/init-db`, {})

/** 完成安装：解除安装锁（删除 install.lock） */
export const installLock = () => devService.post(`${BASE}/lock`, {})
