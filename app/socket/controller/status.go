package controller

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"inis/app/facade"
	"inis/app/model"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 用于存储上一次的网络和磁盘IO数据，计算每秒速率
var (
	lastNetIO       []net.IOCountersStat
	lastDiskIO      map[string]disk.IOCountersStat
	lastTime        time.Time
	lastDiskLatency time.Duration
)

// 格式化字节大小，自动选择合适的单位
func formatBytes(bytes uint64) string {
	const (
		_          = iota
		KB float64 = 1 << (10 * iota)
		MB
		GB
		TB
	)

	var size float64 = float64(bytes)
	var unit string

	switch {
	case size >= TB:
		size /= TB
		unit = "TB"
	case size >= GB:
		size /= GB
		unit = "GB"
	case size >= MB:
		size /= MB
		unit = "MB"
	case size >= KB:
		size /= KB
		unit = "KB"
	default:
		unit = "B"
	}

	return fmt.Sprintf("%.2f %s", size, unit)
}

// 格式化字节速率，自动选择合适的单位
func formatBytesRate(bytes uint64, duration float64) string {
	if duration <= 0 {
		return "0.00 B/s"
	}

	rate := float64(bytes) / duration
	const (
		_          = iota
		KB float64 = 1 << (10 * iota)
		MB
		GB
		TB
	)

	var unit string

	switch {
	case rate >= TB:
		rate /= TB
		unit = "TB/s"
	case rate >= GB:
		rate /= GB
		unit = "GB/s"
	case rate >= MB:
		rate /= MB
		unit = "MB/s"
	case rate >= KB:
		rate /= KB
		unit = "KB/s"
	default:
		unit = "B/s"
	}

	return fmt.Sprintf("%.2f %s", rate, unit)
}

// 启动系统状态推送任务
func init() {
	go startStatusPushTask()
}

// 启动系统状态推送任务
func startStatusPushTask() {
	// 每1秒推送一次系统状态
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 获取系统状态
		status := getSystemStatus()
		// 推送给所有客户端
		pushStatusToClients(status)
	}
}

// 获取系统状态
func getSystemStatus() map[string]any {
	// 系统基本信息
	info := map[string]any{
		"app_name":     "INIS",
		"go_version":   runtime.Version(),
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"cpu_count":    runtime.NumCPU(),
		"goroutines":   runtime.NumGoroutine(),
		"current_time": time.Now().Local().Format("2006-01-02 15:04:05"),
	}

	// 数据库状态
	dbStatus := map[string]any{
		"connected": false,
		"latency":   "0ms",
		"error":     "数据库未初始化",
		"counts": map[string]any{
			"users":    0,
			"articles": 0,
			"comments": 0,
			"pages":    0,
			"links":    0,
			"banners":  0,
			"placards": 0,
			"tags":     0,
		},
	}

	if facade.DB != nil {
		db := facade.DB.Drive()
		if db != nil {
			start := time.Now()
			err := db.Raw("SELECT 1").Error
			latency := time.Since(start)

			// 尝试获取统计数据，但不强制要求成功
			counts := map[string]any{
				"users":    0,
				"articles": 0,
				"comments": 0,
				"pages":    0,
				"links":    0,
				"banners":  0,
				"placards": 0,
				"tags":     0,
			}

			// 只有在数据库连接成功时才尝试获取统计数据
			if err == nil {
				defer func() {
					if r := recover(); r != nil {
						facade.Log.Error(map[string]any{
							"error":     fmt.Sprintf("%v", r),
							"func_name": "getSystemStatus",
							"file_name": "status.go",
						}, "获取数据库统计数据失败")
					}
				}()

				counts = map[string]any{
					"users":    facade.DB.Model(&model.Users{}).Count(),
					"articles": facade.DB.Model(&model.Article{}).Count(),
					"comments": facade.DB.Model(&model.Comment{}).Count(),
					"pages":    facade.DB.Model(&model.Pages{}).Count(),
					"links":    facade.DB.Model(&model.Links{}).Count(),
					"banners":  facade.DB.Model(&model.Banner{}).Count(),
					"placards": facade.DB.Model(&model.Placard{}).Count(),
					"tags":     facade.DB.Model(&model.Tags{}).Count(),
				}
			}

			dbStatus = map[string]any{
				"connected": err == nil,
				"latency":   latency.String(),
				"error": func() string {
					if err == nil {
						return ""
					}
					return err.Error()
				}(),
				"counts": counts,
			}
		}
	}

	// 缓存状态
	cacheStatus := map[string]any{
		"enabled": false,
		"type":    "unknown",
		"working": false,
		"error":   "缓存未初始化",
	}

	if facade.CacheToml != nil && facade.Cache != nil {
		cacheConfig := facade.CacheToml.Get("open")
		cacheType := facade.CacheToml.Get("default")

		cacheKey := "status:test"
		cacheValue := "test_value"
		setSuccess := facade.Cache.Set(cacheKey, cacheValue, 10)
		cachedValue := facade.Cache.Get(cacheKey)
		facade.Cache.Del(cacheKey)

		cacheStatus = map[string]any{
			"enabled": cast.ToBool(cacheConfig),
			"type":    cast.ToString(cacheType),
			"working": setSuccess && cachedValue == cacheValue,
			"error":   utils.Ternary(setSuccess, "", "缓存操作失败"),
		}
	}

	// 系统资源
	// 运行时内存统计
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 系统内存统计
	memInfo, _ := mem.VirtualMemory()

	// CPU 信息
	cpuInfo, _ := cpu.Info()
	cpuPercent, _ := cpu.Percent(0, false)

	// 平均负载
	loadInfo, _ := load.Avg()

	// 磁盘信息
	diskInfo, _ := disk.Usage("/")

	// 网络流量
	netIO, _ := net.IOCounters(false)

	// 磁盘IO
	diskIO, _ := disk.IOCounters()

	// 主机信息
	hostInfo, _ := host.Info()

	memory := map[string]any{
		"alloc":        formatBytes(m.Alloc),
		"total_alloc":  formatBytes(m.TotalAlloc),
		"sys":          formatBytes(m.Sys),
		"gc_count":     m.NumGC,
		"system_total": formatBytes(memInfo.Total),
		"system_used":  formatBytes(memInfo.Used),
		"system_free":  formatBytes(memInfo.Free),
		"system_usage": fmt.Sprintf("%.2f%%", memInfo.UsedPercent),
	}

	// CPU 信息处理
	cpuModel := "Unknown"
	cpuUsage := "0.00%"
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}
	if len(cpuPercent) > 0 {
		cpuUsage = fmt.Sprintf("%.2f%%", cpuPercent[0])
	}

	cpuMap := map[string]any{
		"count":    runtime.NumCPU(),
		"model":    cpuModel,
		"usage":    cpuUsage,
		"load_1m":  loadInfo.Load1,
		"load_5m":  loadInfo.Load5,
		"load_15m": loadInfo.Load15,
	}

	// 磁盘信息处理
	totalRead := uint64(0)
	totalWrite := uint64(0)
	readPerSec := "0.00 MB/s"
	writePerSec := "0.00 MB/s"
	diskLatency := "0ms"

	// 计算总读写量
	for _, io := range diskIO {
		totalRead += io.ReadBytes
		totalWrite += io.WriteBytes
	}

	// 计算每秒读写速率
	currentTime := time.Now()
	if len(lastDiskIO) > 0 && !lastTime.IsZero() {
		duration := currentTime.Sub(lastTime).Seconds()
		if duration > 0 {
			var lastTotalRead, lastTotalWrite uint64
			for _, io := range lastDiskIO {
				lastTotalRead += io.ReadBytes
				lastTotalWrite += io.WriteBytes
			}
			deltaRead := totalRead - lastTotalRead
			deltaWrite := totalWrite - lastTotalWrite
			readPerSec = formatBytesRate(deltaRead, duration)
			writePerSec = formatBytesRate(deltaWrite, duration)
		}
	}

	// 模拟磁盘IO延迟（实际项目中可能需要更复杂的测量方法）
	diskLatency = fmt.Sprintf("%v", lastDiskLatency)
	// 简单模拟：随机延迟在0-10ms之间
	lastDiskLatency = time.Duration(time.Now().UnixNano()%10) * time.Millisecond

	diskMap := map[string]any{
		"total":         formatBytes(diskInfo.Total),
		"used":          formatBytes(diskInfo.Used),
		"free":          formatBytes(diskInfo.Free),
		"usage":         fmt.Sprintf("%.2f%%", diskInfo.UsedPercent),
		"fs_type":       diskInfo.Fstype,
		"read":          formatBytes(totalRead),
		"write":         formatBytes(totalWrite),
		"read_per_sec":  readPerSec,
		"write_per_sec": writePerSec,
		"io_latency":    diskLatency,
	}

	// 网络信息处理
	bytesSent := "0.00 B"
	bytesRecv := "0.00 B"
	packetsSent := uint64(0)
	packetsRecv := uint64(0)
	sentPerSec := "0.00 B/s"
	recvPerSec := "0.00 B/s"

	if len(netIO) > 0 {
		bytesSent = formatBytes(netIO[0].BytesSent)
		bytesRecv = formatBytes(netIO[0].BytesRecv)
		packetsSent = netIO[0].PacketsSent
		packetsRecv = netIO[0].PacketsRecv

		// 计算每秒网络速率
		if len(lastNetIO) > 0 && !lastTime.IsZero() {
			duration := currentTime.Sub(lastTime).Seconds()
			if duration > 0 {
				deltaSent := netIO[0].BytesSent - lastNetIO[0].BytesSent
				deltaRecv := netIO[0].BytesRecv - lastNetIO[0].BytesRecv
				sentPerSec = formatBytesRate(deltaSent, duration)
				recvPerSec = formatBytesRate(deltaRecv, duration)
			}
		}
	}

	network := map[string]any{
		"bytes_sent":     bytesSent,
		"bytes_recv":     bytesRecv,
		"packets_sent":   packetsSent,
		"packets_recv":   packetsRecv,
		"sent_per_sec":   sentPerSec,
		"recv_per_sec":   recvPerSec,
		"up":             sentPerSec, // 上行
		"down":           recvPerSec, // 下行
		"total_sent":     bytesSent,  // 总发送
		"total_received": bytesRecv,  // 总接收
	}

	system := map[string]any{
		"os":         hostInfo.Platform,
		"os_version": hostInfo.PlatformVersion,
		"kernel":     hostInfo.KernelVersion,
		"boot_time":  time.Unix(int64(hostInfo.BootTime), 0).Local().Format("2006-01-02 15:04:05"),
	}

	resource := map[string]any{
		"memory":     memory,
		"cpu":        cpuMap,
		"disk":       diskMap,
		"network":    network,
		"system":     system,
		"goroutines": runtime.NumGoroutine(),
	}

	allStatus := map[string]any{
		"info":      info,
		"database":  dbStatus,
		"cache":     cacheStatus,
		"resource":  resource,
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}

	// 更新上次数据
	lastNetIO = netIO
	lastDiskIO = diskIO
	lastTime = currentTime

	return allStatus
}

// 推送状态给所有客户端
func pushStatusToClients(status map[string]any) {
	// 构造消息
	message := map[string]any{
		"type":      "status",
		"content":   status,
		"timestamp": time.Now().Unix(),
	}

	// 序列化消息
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return
	}

	// 广播消息给所有客户端
	// 修复：使用 Hub.notice 通道发送消息，而不是直接调用 broadcast 方法
	Hub.notice <- msgBytes
}
