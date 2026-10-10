package controller

import (
	"bufio"
	"errors"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"

	"inis/app/facade"
)

// ============================== 日志解析（后台运维） ==============================
//
// 站点日志分两类，落盘位置与格式都不同，这里统一解析成同一种结构供后台展示：
//
//  1. 系统日志 runtime/logs/<日期>/{info,warn,error,debug}.log
//     —— zap 的 JSON 编码，一行一条：
//     {"level":"info","time":"2026-10-10 19:49:47","caller":"log.go:42","msg":"...",<业务字段>}
//     按天分目录；lumberjack 按大小/天数轮转，备份文件形如 info-2026-10-10T12-00-00.000.log
//  2. 通知日志 runtime/sms/{sms,email}.log
//     —— 纯文本，一行一条：[2026-10-10 19:49:47] [成功] email | k: v | k: v
//
// 解析结果统一为：{time, level, msg, caller, fields, raw}
//   - fields：zap 里的业务附加值（或通知日志的 "k: v" 明细）
//   - JSON 解析失败 / 文本行不匹配时退化为 {raw: 原行}，保证任何一行都能看到原文
//
// 安全：只允许读取 runtime/ 下白名单目录里的 .log；
// 日期与文件名都用正则白名单校验（不含路径分隔符，天然防目录穿越）。
type Log struct {
	base
}

const (
	logScopeSystem = "system" // runtime/logs
	logScopeNotify = "notify" // runtime/sms

	// logMaxLines - 单文件最多扫描行数（防御异常大的文件把内存吃满）
	logMaxLines = 200000
	// logMaxPage - 单页最大条数
	logMaxPage = 200
	// logMaxLineSize - 单行最大字节数（日志字段可能很长）
	logMaxLineSize = 2 * 1024 * 1024
)

// logDateAllow - 允许的日期目录名
var logDateAllow = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// logFileAllow - 允许的系统日志文件名（含轮转备份 info-2026-10-10T12-00-00.000.log）
var logFileAllow = regexp.MustCompile(`^[a-z]+(-[0-9A-Za-z.\-]+)?\.log$`)

// notifyFileAllow - 允许的通知日志文件名
var notifyFileAllow = regexp.MustCompile(`^(sms|email)\.log$`)

// logTextLine - 通知日志行：[时间] [状态] 类型 | k: v | k: v
var logTextLine = regexp.MustCompile(`^\[([^\]]+)\]\s*\[([^\]]+)\]\s*(\S+)(.*)$`)

// IGET - 日志查询（只读）
func (this *Log) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"dates": this.dates,
		"files": this.files,
		"read":  this.read,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

// IPOST / IPUT / IDEL - 日志是只读数据，不提供写入入口
func (this *Log) IPOST(ctx *gin.Context) { this.readonly(ctx) }
func (this *Log) IPUT(ctx *gin.Context)  { this.readonly(ctx) }
func (this *Log) IDEL(ctx *gin.Context)  { this.readonly(ctx) }

func (this *Log) readonly(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "日志为只读数据，不支持该操作！"), 405)
}

func (this *Log) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// dates - 可用的日志日期（系统日志按天分目录）+ 通知日志
func (this *Log) dates(ctx *gin.Context) {

	names := make([]string, 0)

	entries, err := os.ReadDir(filepath.Join("runtime", "logs"))
	if err == nil {
		for _, item := range entries {
			if !item.IsDir() || !logDateAllow.MatchString(item.Name()) {
				continue
			}
			names = append(names, item.Name())
		}
	}
	// 最近的日期排在最前
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	list := make([]facade.H, 0, len(names))
	for _, date := range names {

		files, err := os.ReadDir(filepath.Join("runtime", "logs", date))
		if err != nil {
			continue
		}

		size, count := int64(0), 0
		for _, item := range files {
			if item.IsDir() || !logFileAllow.MatchString(item.Name()) {
				continue
			}
			if info, err := item.Info(); err == nil {
				size += info.Size()
			}
			count++
		}

		list = append(list, facade.H{
			"date":  date,
			"files": count,
			"size":  size,
		})
	}

	this.json(ctx, gin.H{
		"dates":  list,
		"notify": this.notifyFiles(),
		"today":  time.Now().Format("2006-01-02"),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// files - 某一天的文件列表（含轮转备份）；scope=notify 时返回通知日志
func (this *Log) files(ctx *gin.Context) {

	params := this.params(ctx, map[string]any{"scope": logScopeSystem})

	if cast.ToString(params["scope"]) == logScopeNotify {
		this.json(ctx, gin.H{
			"scope": logScopeNotify,
			"list":  this.notifyFiles(),
		}, facade.Lang(ctx, "查询成功！"), 200)
		return
	}

	date := strings.TrimSpace(cast.ToString(params["date"]))
	if utils.Is.Empty(date) {
		date = time.Now().Format("2006-01-02")
	}
	if !logDateAllow.MatchString(date) {
		this.json(ctx, nil, facade.Lang(ctx, "日期格式应为 YYYY-MM-DD！"), 400)
		return
	}

	list := make([]facade.H, 0)

	entries, err := os.ReadDir(filepath.Join("runtime", "logs", date))
	if err == nil {
		for _, item := range entries {
			if item.IsDir() || !logFileAllow.MatchString(item.Name()) {
				continue
			}
			size, updateTime := int64(0), int64(0)
			if info, e := item.Info(); e == nil {
				size = info.Size()
				updateTime = info.ModTime().Unix()
			}
			list = append(list, facade.H{
				"name":        item.Name(),
				"size":        size,
				"update_time": updateTime,
				// 轮转备份：lumberjack 在文件名里插入时间戳（info-2026-10-10T12-00-00.000.log）
				"rotated": strings.Contains(strings.TrimSuffix(item.Name(), ".log"), "-"),
			})
		}
	}

	// 排序：级别文件按 info → warn → error → debug，轮转备份排在同类之后
	levelOrder := map[string]int{"debug": 3, "info": 0, "warn": 1, "error": 2}
	weight := func(name string) (int, int) {
		base := strings.SplitN(strings.TrimSuffix(name, ".log"), "-", 2)
		order, ok := levelOrder[base[0]]
		if !ok {
			order = 9
		}
		rotated := 0
		if len(base) > 1 {
			rotated = 1
		}
		return order, rotated
	}
	sort.Slice(list, func(i, j int) bool {
		oi, ri := weight(cast.ToString(list[i]["name"]))
		oj, rj := weight(cast.ToString(list[j]["name"]))
		if oi != oj {
			return oi < oj
		}
		return ri < rj
	})

	this.json(ctx, gin.H{
		"scope": logScopeSystem,
		"date":  date,
		"list":  list,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// read - 读取并解析日志（级别 / 关键字过滤 + 分页，默认最新在前）
func (this *Log) read(ctx *gin.Context) {

	params := this.params(ctx, map[string]any{
		"scope": logScopeSystem,
		"page":  1,
		"limit": 50,
		"order": "desc",
	})

	scope := cast.ToString(params["scope"])
	level := strings.ToLower(strings.TrimSpace(cast.ToString(params["level"])))
	keyword := strings.TrimSpace(cast.ToString(params["keyword"]))

	path, err := this.resolvePath(params)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		// 文件不存在（还没写过日志 / 已被轮转清理）不算错误，回空列表让前端走空态
		if os.IsNotExist(err) {
			this.json(ctx, gin.H{
				"scope": scope, "date": cast.ToString(params["date"]), "file": cast.ToString(params["file"]),
				"list": []any{}, "total": 0, "lines": 0, "pages": 0, "page": 1, "limit": 50,
				"levels": map[string]int{"debug": 0, "info": 0, "warn": 0, "error": 0, "other": 0},
				"truncated": false, "empty": true,
			}, facade.Lang(ctx, "该日志文件不存在或已被清理！"), 200)
			return
		}
		this.json(ctx, nil, facade.Lang(ctx, "读取日志文件失败：%v", err.Error()), 400)
		return
	}
	defer file.Close()

	items := make([]facade.H, 0, 256)
	levels := map[string]int{"debug": 0, "info": 0, "warn": 0, "error": 0, "other": 0}
	total, lines, truncated := 0, 0, false

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), logMaxLineSize)

	for scanner.Scan() {

		lines++
		if lines > logMaxLines {
			truncated = true
			break
		}

		raw := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(raw) == "" {
			continue
		}

		item := this.parseLine(scope, raw)

		// 级别统计针对整个文件（不受过滤影响），便于前端一眼看出各级别数量
		if index, ok := levels[cast.ToString(item["level"])]; ok {
			levels[cast.ToString(item["level"])] = index + 1
		} else {
			levels["other"]++
		}

		if !utils.Is.Empty(level) && cast.ToString(item["level"]) != level {
			continue
		}
		if !utils.Is.Empty(keyword) && !strings.Contains(strings.ToLower(raw), strings.ToLower(keyword)) {
			continue
		}

		total++
		items = append(items, item)
	}

	// 日志文件本身是时间正序，默认倒过来「最新在前」
	if cast.ToString(params["order"]) != "asc" {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}

	page := cast.ToInt(params["page"])
	if page < 1 {
		page = 1
	}
	limit := cast.ToInt(params["limit"])
	if limit <= 0 {
		limit = 50
	}
	if limit > logMaxPage {
		limit = logMaxPage
	}

	start, end := (page-1)*limit, page*limit
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}

	pages := 0
	if total > 0 {
		pages = int(math.Ceil(float64(total) / float64(limit)))
	}

	this.json(ctx, gin.H{
		"scope":     scope,
		"date":      cast.ToString(params["date"]),
		"file":      cast.ToString(params["file"]),
		"list":      items[start:end],
		"total":     total,
		"lines":     lines,
		"pages":     pages,
		"page":      page,
		"limit":     limit,
		"levels":    levels,
		"truncated": truncated,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// resolvePath - 把 (scope, date, file) 解析成可读的日志路径（白名单校验，防目录穿越）
func (this *Log) resolvePath(params map[string]any) (string, error) {

	scope := cast.ToString(params["scope"])
	file := strings.TrimSpace(cast.ToString(params["file"]))

	if scope == logScopeNotify {
		if !notifyFileAllow.MatchString(file) {
			return "", errors.New("不支持的通知日志文件！")
		}
		return filepath.Join("runtime", "sms", file), nil
	}

	date := strings.TrimSpace(cast.ToString(params["date"]))
	if utils.Is.Empty(date) {
		date = time.Now().Format("2006-01-02")
	}

	if !logDateAllow.MatchString(date) || !logFileAllow.MatchString(file) {
		return "", errors.New("日志路径不合法！")
	}

	return filepath.Join("runtime", "logs", date, file), nil
}

// notifyFiles - 通知日志（runtime/sms）文件列表
func (this *Log) notifyFiles() []facade.H {

	entries, err := os.ReadDir(filepath.Join("runtime", "sms"))
	if err != nil {
		return []facade.H{}
	}

	list := make([]facade.H, 0, len(entries))
	for _, item := range entries {
		if item.IsDir() || !notifyFileAllow.MatchString(item.Name()) {
			continue
		}
		size, updateTime := int64(0), int64(0)
		if info, e := item.Info(); e == nil {
			size = info.Size()
			updateTime = info.ModTime().Unix()
		}
		list = append(list, facade.H{
			"name":        item.Name(),
			"size":        size,
			"update_time": updateTime,
		})
	}

	return list
}

// parseLine - 解析一行日志
//
// 系统日志（zap JSON）取 time/level/msg/caller，其余键进 fields；
// 通知日志（文本）解析 [时间] [状态] 类型 | k: v；
// 都不匹配时只保留 raw，保证「任何一行都能看到原文」。
func (this *Log) parseLine(scope string, raw string) facade.H {

	if scope == logScopeSystem {

		if decoded := utils.Json.Decode(raw); decoded != nil {
			if values, ok := decoded.(map[string]any); ok {

				fields := map[string]any{}
				for key, val := range values {
					switch key {
					case "time", "level", "msg", "caller":
						continue
					}
					fields[key] = val
				}

				return facade.H{
					"time":   cast.ToString(values["time"]),
					"level":  strings.ToLower(cast.ToString(values["level"])),
					"msg":    cast.ToString(values["msg"]),
					"caller": cast.ToString(values["caller"]),
					"fields": fields,
					"raw":    raw,
				}
			}
		}

		return facade.H{"level": "", "fields": map[string]any{}, "raw": raw}
	}

	// 通知日志：[2026-10-10 19:49:47] [成功] email | k: v | k: v
	if match := logTextLine.FindStringSubmatch(raw); match != nil {

		level := "info"
		if !strings.Contains(match[2], "成功") {
			level = "error"
		}

		fields := map[string]any{}
		for _, part := range strings.Split(match[4], "|") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if index := strings.Index(part, ":"); index > 0 {
				fields[strings.TrimSpace(part[:index])] = strings.TrimSpace(part[index+1:])
			} else {
				fields["detail"] = part
			}
		}

		return facade.H{
			"time":   match[1],
			"level":  level,
			"msg":    match[3],
			"status": match[2],
			"fields": fields,
			"raw":    raw,
		}
	}

	return facade.H{"level": "", "fields": map[string]any{}, "raw": raw}
}
