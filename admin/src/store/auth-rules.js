import { defineStore } from 'pinia'
import cache from '{src}/utils/cache'
import utils from '{src}/utils/utils'
import axios from '{src}/utils/request'

// 规则名称解析：把「【分组】名称」或「分组-名称」拆分为 { title, label }
// 名称缺失或格式不匹配时使用兜底值，避免 rule 为 null 时 rule[2] 解析报错
const parseRuleItem = (item = {}) => {

    const name = utils.is.empty(item?.name) ? '' : String(item.name).trim()

    // 匹配特殊字符串隔开的
    const match = name.match(/^[【|\[](.+?)[】|\]](.+)/) || name.match(/(.+)[^\w\u4e00-\u9fa5\s](.+)/)

    if (match) {
        return { title: String(match[1]).trim(), label: String(match[2]).trim() }
    }

    // 兜底：无名称/格式异常时，使用路由或 ID 展示，并统一归入“未分组”
    const label = name || item?.route || (utils.is.empty(item?.id) ? '未命名规则' : `规则 #${item.id}`)

    return { title: '未分组', label }
}

// 规则树
const tree = (state = {}) => {

    const cacheName = 'auth-rule-tree'

    if (cache.has(cacheName)) return (state.tree = cache.get(cacheName))

    axios.get('/api/auth-rules/column', {
        field: 'id,name,method,route,hash'
    }).then(({ code, data }) => {

        if (code !== 200) return

        state.flat = JSON.parse(JSON.stringify(data))
        // 缓存数据
        cache.set('auth-rule-flat', JSON.parse(JSON.stringify(data)), inis.cache)

        // 分组数据
        let group  = []
        let titles = []

        for (let item of data) {

            const { title, label } = parseRuleItem(item)
            const son = { id: item.id, value: parseInt(item.hash), label }

            // 判断标题是否存在数组中
            if (utils.in.array(title, titles)) {

                group.find(groupItem => groupItem?.value === title)?.children.push(son)

            } else {

                titles.push(title)
                group.push({ value: title, label: title, children: [son] })
            }
        }

        // 缓存数据
        cache.set(cacheName, group, inis.cache)

        state.tree = group
    })

    return state.tree
}

// 规则扁平化
const flat = (state = {}) => {

    const cacheName = 'auth-rule-flat'

    if (cache.has(cacheName)) return (state.flat = cache.get(cacheName))

    axios.get('/api/auth-rules/column', {
        field: 'id,name,method,route,hash'
    }).then(({ code, data }) => {

        if (code !== 200) return

        state.flat = data

        // 缓存数据
        cache.set(cacheName, data, inis.cache)
    })
}

export const useAuthRulesStore = defineStore('auth-rules', {
    state: () => ({
        tree: [],       // 规则树
        flat: [],       // 规则扁平化
    }),
    getters: {
        // 获取规则树
        getTree(state = {}) {
            return tree(state)
        },
        // 获取规则扁平化
        getFlat(state = {}) {
            return flat(state)
        }
    },
    actions: {
        async setFlat() {
            const cacheName = 'auth-rule-flat'

            if (cache.has(cacheName)) {
                this.flat = cache.get(cacheName)
                return { code: 200, msg: 'cache', data: this.flat }
            }

            const { code, msg, data } = await axios.get('/api/auth-rules/column', {
                field: 'id,name,method,route,hash'
            })

            // 保持返回结构完整，避免调用方解构 undefined 报错
            if (code !== 200) return { code, msg, data }

            this.flat = data

            // 缓存数据
            cache.set(cacheName, data, inis.cache)

            return { code, msg, data }
        },
        async setTree() {

            const cacheName = 'auth-rule-tree'

            const { code, msg, data } = (await this.setFlat()) || {}

            if (code !== 200) return { code, msg, data }
            // 分组数据
            let group  = []
            let titles = []

            for (let item of (data || [])) {

                const { title, label } = parseRuleItem(item)
                const son = { id: item.id, value: parseInt(item.hash), label }

                // 判断标题是否存在数组中
                if (utils.in.array(title, titles)) {

                    group.find(groupItem => groupItem?.value === title)?.children.push(son)

                } else {

                    titles.push(title)
                    group.push({ value: title, label: title, children: [son] })
                }
            }

            // 缓存数据
            cache.set(cacheName, group, inis.cache)

            this.tree = group

            return { code, msg, data: group }
        }
    }
})