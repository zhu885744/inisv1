// 配置
globalThis.inis = {
    api: {
        // 请填写 API 服务地址
        uri: '',
        // 请求密钥 - 开启了 API KEY 时需要填写，否则留空
        key: ''
    },
    socket: {
        // 服务端地址 - 留空则自动从 api.uri 推导
        uri: '',
        // 重连间隔（毫秒）
        reconnect_interval: 5000,
        // 心跳间隔（毫秒）
        heartbeat_interval: 10000,
    },
    // 延迟 - 毫秒
    lazy_time: 500,
}