/**
 * Mellow 运行时配置
 *
 * 字段说明：
 * - apiUri    后端根地址。
 *             留空 '' 表示「与当前站点同源」——主题由后端自身托管时，
 *             请求会自动打到 /api，即后端自己的地址，换域名/换端口都不用改；
 *             前后端不同源时再填绝对地址，如 'https://api.example.com'。
 * - socketUri socket 地址；留空表示同源（用 wss://当前域名/socket）。
 *             接受这些写法，路径一律固定为 /socket（写全也会被覆盖）：
 *               ''                        → wss://当前域名/socket
 *               'wss://api.example.com'   → wss://api.example.com/socket
 *               'https://api.example.com' → wss://api.example.com/socket
 *               '//api.example.com'       → wss://api.example.com/socket
 *               'api.example.com'         → wss://api.example.com/socket
 *             页面为 https 时会强制使用 wss（否则被浏览器按混合内容拦截）。
 *             socketUri 为空时回落到 apiUri，都为空则同源。
 *             优先级高于打包变量 VITE_SOCKET，改这个文件不需要重新打包。
 */
window.__INIS_CONFIG__ = {
  apiUri: '',
  socketUri: ''
}