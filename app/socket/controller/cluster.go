package controller

import (
	"context"
	"encoding/json"
	"inis/app/facade"
	"time"

	"github.com/spf13/cast"
)

// 跨实例消息转发（Redis Pub/Sub）
//
// 背景：hub 里的 clients / 消息分发全部在本进程内存中，
// 多实例部署时 A 实例上的广播、单播、私聊、踢下线都只对 A 上的连接生效，
// 连在 B 上的用户永远收不到 —— 这是「没适配 redis」里最影响功能的一处。
//
// 做法：
//   - 本实例处理完一条消息后，把它投递到 Redis 频道，其它实例订阅到后只做「本地分发」；
//   - 订阅侧的分发**不再二次发布**，否则会形成 A→B→A 的死循环；
//   - 系统状态（type=status）是本实例的 CPU / 内存 / 表计数快照，**不转发**，
//     否则管理员会看到别的机器上的资源数据。
//
// 注意：只在 cache.toml 的 default = "redis" 时启用。file / ram 驱动下保持原来的单实例行为。
const clusterChannel = "inis:socket:cluster"

// clusterEnvelope 跨实例信封
//
// data 用 string 装原始消息（JSON 文本）：避免多一层 map 编解码导致的字段丢失，
// 也让「原样转发」这件事没有任何歧义。
type clusterEnvelope struct {
	From string `json:"from"` // 发布者实例 id（用于日志与自回环兜底）
	Kind string `json:"kind"` // msg：普通消息；kick：踢下线
	Data string `json:"data"` // msg 为原始消息 JSON；kick 为目标 clientId
}

// clusterInstanceID 本实例标识（每次启动随机，仅用于日志）
var clusterInstanceID = guid()

// clusterEnabled 是否启用跨实例转发
//
// 必须同时满足「缓存驱动是 redis」与「Redis 客户端已创建」：
// 只配了 redis 但没启用（default=file）时，会话本身都跨不了实例，转发也没有意义。
func clusterEnabled() bool {
	return facade.Redis != nil &&
		facade.Redis.Client != nil &&
		cast.ToString(facade.CacheToml.Get("default")) == facade.CacheModeRedis
}

func init() {
	if !clusterEnabled() {
		socketLog("未启用 redis 缓存驱动，socket 按单实例模式运行（广播不会跨实例）")
		return
	}

	go clusterSubscribe()
}

// clusterPublish 把一条本实例产生的消息投递给其它实例（失败只记日志，不影响本地投递）
func clusterPublish(kind string, data []byte) {
	if !clusterEnabled() {
		return
	}

	payload, err := json.Marshal(clusterEnvelope{
		From: clusterInstanceID,
		Kind: kind,
		Data: string(data),
	})
	if err != nil {
		socketLog("打包跨实例消息失败: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := facade.Redis.Client.Publish(ctx, clusterChannel, payload).Err(); err != nil {
		socketLog("跨实例消息发布失败: %v", err)
	}
}

// clusterSubscribe 订阅频道并转发到 hub（断线自动重连）
//
// go-redis 的 PubSub 在连接断开后 Channel() 会被关闭，
// 所以这里用无限循环重建订阅：只要进程还在，就能在 Redis 恢复后自动接上。
func clusterSubscribe() {
	for {
		pubsub := facade.Redis.Client.Subscribe(context.Background(), clusterChannel)

		if _, err := pubsub.Receive(context.Background()); err != nil {
			socketLog("订阅 socket 集群频道失败: %v，5 秒后重试", err)
			_ = pubsub.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		socketLog("已订阅 socket 集群频道: %s（实例 %s）", clusterChannel, clusterInstanceID)

		for message := range pubsub.Channel() {
			handleClusterPayload(message.Payload)
		}

		socketLog("socket 集群频道订阅中断，5 秒后重连")
		_ = pubsub.Close()
		time.Sleep(5 * time.Second)
	}
}

// handleClusterPayload 处理其它实例转发来的消息
func handleClusterPayload(payload string) {
	var envelope clusterEnvelope
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		socketLog("跨实例消息解析失败: %v", err)
		return
	}

	if envelope.From == clusterInstanceID {
		// Redis 不会把消息回投给发布者，这里只是兜底
		return
	}

	switch envelope.Kind {
	case "msg":
		// 非阻塞投递：hub goroutine 忙的时候宁可丢一条，也不能卡住订阅协程导致整个实例收不到消息
		select {
		case Hub.cluster <- []byte(envelope.Data):
		default:
			socketLog("跨实例消息队列已满，丢弃一条消息")
		}
	case "kick":
		select {
		case Hub.clusterKick <- envelope.Data:
		default:
			socketLog("跨实例踢下线队列已满，忽略目标: %s", envelope.Data)
		}
	default:
		socketLog("未知的跨实例消息类型: %s", envelope.Kind)
	}
}
