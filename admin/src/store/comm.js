import { defineStore } from 'pinia'
import cache from '{src}/utils/cache'
import utils from '{src}/utils/utils'
import axios from '{src}/utils/request'
import { push } from '{src}/utils/route'

// 校验token
const checkToken = (state = {}) => {

    const cacheName = 'user-info'
    // 缓存中存在用户信息
    if (cache.has(cacheName)) return state.login.finish = true

    axios.post('/api/comm/check-token').then(({ code, msg, data })=> {

        if (code === 412) return
        // token 已失效，仅清除本地登录状态，不要再次发起 logout 请求（避免失效 token 触发 401/405）
        if (code === 401) return clearLogin(state)
        if (code !== 200) return notyf.error(msg)

        state.login.user   = data.user
        state.login.finish = true
        // 登录会话有效期（秒），后端返回，回退到 15 天
        const validSeconds = Number(data.valid_time) > 0 ? Number(data.valid_time) : 15 * 24 * 60 * 60
        cache.set(cacheName, data.user, Math.ceil(validSeconds / 60))
    })
}

// 仅清除本地登录状态（不发起网络请求）
const clearLogin = (state = {}) => {
    state.login.user   = {}
    state.login.finish = false
    cache.del('user-info')
    utils.clear.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN')
}

// 登出
const logout = (state = {}, path = null) => {

    clearLogin(state)

    // 登出请求失败不影响本地状态清除，忽略错误
    axios.post('/api/comm/logout').catch(() => {})

    // 返回首页
    if (!utils.is.empty(path)) setTimeout(() => {
        push(path)
    }, 300)
}

export const useCommStore = defineStore('comm', {
    state: () => ({
        login: {
            // 登录状态 - 是否登录完成
            finish: false,
            // 当前登录的用户信息
            user  : cache.get('user-info'),
        },
        progress: false,
        nav: {
            title: ''
        }
    }),
    // methods
    actions: {
        // 登出
        logout(path = null) {
            logout(this, path)
        },
    },
    // computed
    getters: {
        // 获取登录信息
        getLogin: (state = {}) => {
            // 校验token
            checkToken(state)
            // 返回登录信息
            return state.login
        }
    },
})