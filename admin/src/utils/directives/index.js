// 在src文件夹下新建directives文件夹，新建index.js文件
import load from './load.js'

const directives = {
    load
}

export default {
    install(Vue) {
        // 通过遍历directives对象完成全局注册
        Object.keys(directives).forEach(item => {
            Vue.directive(item, directives[item])
        })
    }
}
