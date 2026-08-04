import { defineStore } from 'pinia'
import { cache } from '@/utils/network'
import { request as axios } from '@/utils/network'

export const useBannerStore = defineStore('banner', {
    state: () => ({
        current: [],
    }),
    getters: {
        getCurrent(state) {
            return state.current
        },
    },
    actions: {
        async setCurrent() {

            const cacheName = 'banner-current'

            if (cache.has(cacheName)) {
                this.current = cache.get(cacheName)
                return { code: 200, msg: 'cache', data: this.current }
            }

            const { code, msg, data } = await axios.get('/api/banner/all', {
                limit: 5,
                order: 'create_time desc',
            })

            if (code !== 200) return { code, msg, data: [] }

            this.current = data.data || []

            cache.set(cacheName, this.current, globalThis?.inis?.cache || 3600)

            return { code, msg, data: this.current }
        },
    }
})
