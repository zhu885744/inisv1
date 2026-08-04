<template>
    <div class="container-box">
        <el-row :gutter="20" style="display: flex;">
            <el-col :span="12" style="display: flex;">
                <el-button v-on:click="method.refresh()">刷新</el-button>
            </el-col>
            <el-col :span="12" style="display: flex; justify-content: flex-end; z-index: -1">
                <el-button disabled>
                    {{ state.item.title }}
                </el-button>
            </el-col>
        </el-row>
        <el-row :gutter="20" style="margin-top: 12px">
            <el-col :span="24">
                <el-tabs v-model="state.item.tabs" id="tabs-area">

                    <el-tab-pane name="security">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">安全</span>
                        </template>
                        <el-row :gutter="20">
                            <el-col :span="8">
                                <atom-api-key ref="api-key"></atom-api-key>
                            </el-col>
                            <el-col :span="8">
                                <atom-qps ref="qps"></atom-qps>
                            </el-col>
                            <el-col :span="8">
                                <atom-qps-black ref="qps-black"></atom-qps-black>
                            </el-col>
                            <el-col :span="8">
                                <atom-page-limit ref="page-limit"></atom-page-limit>
                            </el-col>
                            <el-col :span="8">
                                <atom-jwt ref="jwt"></atom-jwt>
                            </el-col>
                            <el-col :span="8">
                                <atom-allow-register ref="allow-register"></atom-allow-register>
                            </el-col>
                        </el-row>
                    </el-tab-pane>

                    <el-tab-pane name="optimize">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">缓存</span>
                        </template>
                        <el-row :gutter="20">
                            <el-col :span="8">
                                <atom-cache-redis ref="cache-redis" v-on:refresh="method.refresh"></atom-cache-redis>
                            </el-col>
                            <el-col :span="8">
                                <atom-cache-file ref="cache-file" v-on:refresh="method.refresh"></atom-cache-file>
                            </el-col>
                            <el-col :span="8">
                                <atom-cache-ram ref="cache-ram" v-on:refresh="method.refresh"></atom-cache-ram>
                            </el-col>
                        </el-row>
                    </el-tab-pane>

                    <el-tab-pane name="storage">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">存储</span>
                        </template>
                        <el-row :gutter="20">
                            <el-col :span="8">
                                <atom-storage-local ref="storage-local" v-on:refresh="method.refresh"></atom-storage-local>
                            </el-col>
                            <el-col :span="8">
                                <atom-storage-oss ref="storage-oss" v-on:refresh="method.refresh"></atom-storage-oss>
                            </el-col>
                            <el-col :span="8">
                                <atom-storage-cos ref="storage-cos" v-on:refresh="method.refresh"></atom-storage-cos>
                            </el-col>
                            <el-col :span="8">
                                <atom-storage-kodo ref="storage-kodo" v-on:refresh="method.refresh"></atom-storage-kodo>
                            </el-col>
                        </el-row>
                    </el-tab-pane>

                    <el-tab-pane name="sms">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">短信</span>
                        </template>
                        <el-row :gutter="20">
                            <el-col :span="8">
                                <atom-sms-email ref="sms-email" v-on:refresh="method.refresh"></atom-sms-email>
                            </el-col>
                            <el-col :span="8">
                                <atom-sms-aliyun ref="sms-aliyun" v-on:refresh="method.refresh"></atom-sms-aliyun>
                            </el-col>
                            <el-col :span="8">
                                <atom-sms-aliyun-verify ref="sms-aliyun-verify" v-on:refresh="method.refresh"></atom-sms-aliyun-verify>
                            </el-col>
                            <el-col :span="8">
                                <atom-sms-tencent ref="sms-tencent" v-on:refresh="method.refresh"></atom-sms-tencent>
                            </el-col>
                        </el-row>
                    </el-tab-pane>

                    <el-tab-pane name="other">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">配置</span>
                        </template>
                        <el-row :gutter="20">
                            <el-col :span="8">
                                <atom-page ref="page" v-on:refresh="method.refresh"></atom-page>
                            </el-col>
                            <el-col :span="8">
                                <atom-article ref="article" v-on:refresh="method.refresh"></atom-article>
                            </el-col>
                            <el-col :span="8">
                                <atom-comment ref="comment" v-on:refresh="method.refresh"></atom-comment>
                            </el-col>
                        </el-row>
                        <el-row :gutter="20" style="margin-top: 12px">
                            <el-col :span="8">
                                <atom-exp-rules ref="exp-rules" v-on:refresh="method.refresh"></atom-exp-rules>
                            </el-col>
                        </el-row>
                    </el-tab-pane>

                    <el-tab-pane name="inis">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">更新</span>
                        </template>
                        <el-row :gutter="20">
                            <el-col :span="8">
                                <atom-upgrade ref="upgrade" v-on:refresh="method.refresh"></atom-upgrade>
                            </el-col>
                        </el-row>
                    </el-tab-pane>
                </el-tabs>
            </el-col>
        </el-row>
    </div>
</template>

<script setup>
import { useCommStore } from '{src}/store/comm'
import AtomSmsEmail from '{src}/comps/admin/atom/sms-email.vue'
import AtomSmsAliyun from '{src}/comps/admin/atom/sms-aliyun.vue'
import AtomSmsAliyunVerify from '{src}/comps/admin/atom/sms-aliyun-verify.vue'
import AtomSmsTencent from '{src}/comps/admin/atom/sms-tencent.vue'
import AtomApiKey from '{src}/comps/admin/atom/api-key.vue'
import AtomAllowRegister from '{src}/comps/admin/atom/allow-register.vue'
import AtomQps from '{src}/comps/admin/atom/qps.vue'
import AtomQpsBlack from '{src}/comps/admin/atom/qps-black.vue'
import AtomPageLimit from '{src}/comps/admin/atom/page-limit.vue'
import AtomJwt from '{src}/comps/admin/atom/jwt.vue'
import AtomCacheRedis from '{src}/comps/admin/atom/cache-redis.vue'
import AtomCacheFile from '{src}/comps/admin/atom/cache-file.vue'
import AtomCacheRam from '{src}/comps/admin/atom/cache-ram.vue'
import AtomStorageLocal from '{src}/comps/admin/atom/storage-local.vue'
import AtomStorageOss from '{src}/comps/admin/atom/storage-oss.vue'
import AtomStorageCos from '{src}/comps/admin/atom/storage-cos.vue'
import AtomStorageKodo from '{src}/comps/admin/atom/storage-kodo.vue'
import AtomPage from '{src}/comps/admin/atom/page.vue'
import AtomArticle from '{src}/comps/admin/atom/article.vue'
import AtomComment from '{src}/comps/admin/atom/comment.vue'
import AtomExpRules from '{src}/comps/admin/atom/exp-rules.vue'
import AtomUpgrade from '{src}/comps/admin/atom/upgrade.vue'

const { ctx, proxy } = getCurrentInstance()
const store  = {
    comm: useCommStore(),
}
const state  = reactive({
    item: {
        title : '系统配置',
        tabs  : 'security',
    },
    refresh: {
        inis    : ['device-bind','upgrade'],
        other   : ['page','article','comment','exp-rules'],
        optimize: ['cache-redis','cache-file','cache-ram'],
        sms     : ['sms-email','sms-aliyun','sms-aliyun-verify','sms-tencent'],
        storage : ['storage-local','storage-oss','storage-cos','storage-kodo'],
        security: ['api-key','qps','page-limit','jwt','allow-register','qps-black'],
    },
})

// 方法
const method = {
  // 刷新
  refresh(...args) {
    // 允许刷新的参数
    const allow = [...state.refresh[state.item.tabs]]
    // 如果没有传参则刷新所有
    let targetItems = args.length === 0 ? allow : args.filter(item => allow.includes(item))
    
    // 批量刷新（增加安全校验）
    targetItems.forEach(item => {
      // 1. 校验ref是否存在
      const compRef = proxy.$refs[item]
      if (!compRef) {
        console.warn(`组件ref不存在：${item}`)
        return
      }
      // 2. 校验init是否为函数
      if (typeof compRef.init !== 'function') {
        console.warn(`组件${item}未暴露init方法`)
        return
      }
      // 3. 安全调用init
      compRef.init()
    })
  },
}
</script>