import cache from '{src}/utils/cache'
import utils from '{src}/utils/utils'
import { push } from '{src}/utils/route'
import { useAuthPagesStore } from '{src}/store/auth-pages'

const config = {
    hasIcon: true,
    iconType: 'svg-icon',
    menuWidth: 140,
    customClass: 'dark-howdy-menu scale-up-top-left',
    useLongPressInMobile: true,
    menuWrapperCss: {
        background: '#0b0b0bcc',
        borderRadius: '8px',
        padding: '5px 4px',
        boxShadow: '#00000080 0 10px 30px',
        lineColor: 'rgba(255,255,255,.1)',
        lineMargin: '5px 10px',
        backdropFilter: 'blur(10px)',
    },
    menuItemCss: {
        arrowSize: '10px',
        iconFontSize: '18px',
        labelColor: '#FFF',
        hoverLabelColor: '#FFF',
        iconColor: '#ffffff00',
        arrowColor: '#ffffff00'
    },
    menuList: [],
}

const list = async () => {

    let authPagesFlat = useAuthPagesStore().getFlat

    // 如果 authPagesFlat 为空，堵塞，等待获取权限列表
    if (utils.is.empty(authPagesFlat)) {
        await new Promise((resolve) => {
            let timer = setInterval(() => {
                authPagesFlat = useAuthPagesStore().getFlat
                if (!utils.is.empty(authPagesFlat)) {
                    clearInterval(timer)
                    resolve()
                }
            }, 100)
        })
    }

    // 用户的页面权限
    let pagesHash = cache.get('user-info')?.['result']?.['auth']?.['pages']?.['hash'] || ''
    // 用户权限列表
    let list = [{
        label: '创作',
        name : 'create',
        icon: `<svg class="icon" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" width="16" height="16">
            <path fill="rgb(var(--menu-icon-color))" d="M666.7 927.4H289.2c-24.7 0-44.7-20-44.7-44.7 0-24.7 20-44.7 44.7-44.7h377.4c59.9 0 81.3-52.6 81.3-101.9V361.3c0-24.7 20-44.7 44.7-44.7 24.7 0 44.7 20 44.7 44.7v374.8c0 112.7-70.2 191.3-170.6 191.3z"></path>
            <path fill="rgb(var(--menu-icon-color))" d="M235.4 778.8c-24.7 0-44.7-20-44.7-44.7V289.3c0-105.3 85.7-191 191-191h364.6c117.2 0 212.5 95.3 212.5 212.5v50.6c0 24.7-20 44.7-44.7 44.7H441.9c-24.7 0-44.7-20-44.7-44.7s20-44.7 44.7-44.7h427.6v-5.9c0-67.9-55.2-123.1-123.1-123.1H381.7c-56.1 0-101.7 45.6-101.7 101.7v444.8c0.1 24.6-19.9 44.6-44.6 44.6z"></path>
            <path fill="rgb(var(--menu-icon-color))" d="M441.6 406c-24.7 0-44.7-20-44.7-44.7v-50.6c0-43.8-23.6-84.7-61.6-106.7-21.4-12.3-28.7-39.7-16.3-61.1 12.3-21.4 39.7-28.7 61.1-16.3 65.5 37.9 106.2 108.4 106.2 184v50.6c0 24.8-20 44.8-44.7 44.8zM289.2 928.2c-123.6 0-224.1-100.5-224.1-224.1v-37.7c0-24.7 20-44.7 44.7-44.7h105.8c24.7 0 44.7 20 44.7 44.7s-20 44.7-44.7 44.7h-61c3.6 71.1 62.6 127.7 134.5 127.7 24.7 0 44.7 20 44.7 44.7 0.1 24.7-19.9 44.7-44.6 44.7zM643.5 564.2H380.8c-24.7 0-44.7-20-44.7-44.7 0-24.7 20-44.7 44.7-44.7h262.7c24.7 0 44.7 20 44.7 44.7 0 24.7-20 44.7-44.7 44.7zM493.4 711.1H380.8c-24.7 0-44.7-20-44.7-44.7s20-44.7 44.7-44.7h112.5c24.7 0 44.7 20 44.7 44.7s-20 44.7-44.6 44.7z"></path>
        </svg>`,
        items: ['/admin/article', '/admin/article/group', '/admin/article/write', '/admin/pages','/admin/pages/write','/admin/moments'],
        children: [],
    },{
        label: '管理',
        name : 'manage',
        icon: `<svg class="icon" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" width="20" height="20">
            <path fill="rgb(var(--menu-icon-color))" d="M743.2704 264.32l91.8784 53.0176v130.56l-156.5184 90.368-0.384-126.848-97.28-56.32zM404.9664 826.3936v-173.9008l103.7568 59.5456 102.912-59.3152-0.7936 173.0048-103.7056 59.5968-102.0928-58.88z m42.3168-472.704l-108.1088 61.9008v121.7024L179.2 444.0832v-126.6176l102.5792-59.264 165.6576 95.5136z m66.8928 38.7072l98.4576 56.8064-0.6144 127.4112-103.3728 59.6992-103.808-60.5952v-122.2656l109.3888-61.056z"></path>
            <path fill="rgb(var(--menu-icon-color))" opacity=".5" d="M673.5872 224.128l-160.4096 91.8272L347.52 220.16 507.1616 128zM338.7136 788.1472L179.2 695.9872v-172.9536l159.744 91.8272v173.44z m496.4352-264.576v172.4928l-155.8272 89.856-0.4608-172.2368 156.16-90.2144z"></path>
        </svg>`,
        items: [
            '/admin/users', '/admin/bill', '/admin/order', '/admin/comment', '/admin/placard', '/admin/banner', '/admin/tags', '/admin/badge', '/admin/level', '/admin/links',
            '/admin/links/group', '/admin/system/version', '/admin/exp', '/admin/goods', '/admin/integral', '/admin/message', '/admin/notification-preferences', '/admin/attachment', '/admin/system'
        ],
        children: [],
    },{
        label: '安全',
        name : 'security',
        icon: `<svg class="icon" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" width="18" height="18">
            <path fill="rgb(var(--menu-icon-color))" d="M665.6 512H358.4a34.133333 34.133333 0 0 1-34.133333-34.133333v-34.133334h375.466666v34.133334a34.133333 34.133333 0 0 1-34.133333 34.133333z"></path>
            <path fill="rgb(var(--menu-icon-color))" d="M484.522667 77.192533l-307.2 134.997334A68.266667 68.266667 0 0 0 136.533333 274.653867V486.229333c0 129.5872 57.9584 245.384533 160.085334 325.188267l176.0256 124.2112a68.266667 68.266667 0 0 0 78.711466 0l174.6944-123.221333C829.508267 731.5968 887.466667 615.799467 887.466667 486.229333V274.670933a68.266667 68.266667 0 0 0-40.789334-62.498133l-307.2-134.997333a68.266667 68.266667 0 0 0-54.954666 0zM204.8 274.670933l307.2-134.997333 307.2 134.997333V486.229333c0 108.305067-48.0768 204.373333-133.853867 271.394134L512 879.837867l-174.6944-123.221334C252.8768 690.551467 204.8 594.500267 204.8 486.1952V274.670933z"></path>
            <path fill="rgb(var(--menu-icon-color))" d="M512 290.133333a34.133333 34.133333 0 0 1 34.048 31.573334L546.133333 324.266667v307.2a34.133333 34.133333 0 0 1-68.181333 2.56L477.866667 631.466667V324.266667a34.133333 34.133333 0 0 1 34.133333-34.133334z"></path>
        </svg>`,
        items: ['/admin/auth/rules', '/admin/auth/group', '/admin/api/keys', '/admin/auth/pages', '/admin/ip/black', '/admin/ip/white', '/admin/qps/warn'],
        children: [],
    }]

    // 用户拥有的页面权限
    let pages = []
    // 用户拥有的接口权限
    if (utils.in.array('all', pagesHash)) {
        pages = authPagesFlat.map(item => item.path)
    } else {
        // 去重 去空
        pagesHash = [...new Set(pagesHash)].filter(item => item)
        // 从 all 中找到 hash = pagesHash 的项
        pages = authPagesFlat.filter(item => pagesHash.indexOf(item.hash) !== -1).map(item => item.path)
    }

    for (let index in list) {
        // 取 pages 和 list 的交集
        let cross = list[index].items.filter(item => pages.indexOf(item) !== -1)
        // 把交集的到的结果加上 all 二维数组继续进行补全，根据 all.map(item => item.path) 的结果
        let children = cross.map(item => authPagesFlat.find(i => i.path === item))
        // 修正 children item 的字段 label => name，fn: () => push(item.path)
        children = children.map(item => {
            item.label = item.name
            item.fn = () => push(item.path)
            // 优先使用 svg 字段
            if (item.svg) {
                item.icon = item.svg
                item.isSvg = true
            }
            // 如果 icon 是 SVG 字符串，标记为 SVG
            if (item.icon && typeof item.icon === 'string' && item.icon.includes('<svg')) {
                item.isSvg = true
            }
            return item
        })
        // 把 children 赋值给 list[index].children
        list[index].children = children
    }

    // 如果 list[index].children 为空，就删除 list[index]
    list = list.filter(item => item.children.length)

    return list
}

export { list, config }