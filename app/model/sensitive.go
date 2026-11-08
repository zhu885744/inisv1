package model

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"

	"inis/app/facade"

	"github.com/spf13/cast"
)

/**
 * 全站敏感词库（独立 config/sensitive.toml，不入库）
 *
 * 与旧版「一个词表 + 命中即拒绝」的区别：
 *   - 每条规则带 action / scenes / level / match，按场景决定处置；
 *   - 处置动作：block 拒绝 / limit 限流拒绝 / review 进审核 / warn 提示 / log 仅记录；
 *   - 白名单优先于规则，命中即放行（官方公告、认证机构、业务模板）；
 *   - 匹配前做基础归一化（全半角、大小写、繁简、空格/符号插入）；
 *   - contains 规则用 Aho-Corasick 多模式匹配，规则量大时不退化成逐词 Contains。
 *
 * 配置由 facade.SensitiveToml 提供，文件变更通过 viper watch 推进「配置代次」，
 * 这里按代次缓存编译结果（规则 + 自动机），无需重启，也不会每条消息重建。
 */

const (
	// 业务场景
	SceneComment = "comment"
	SceneArticle = "article"
	SceneMoment  = "moment"
	SceneDM      = "dm"
	SceneChat    = "chat"

	// 处置动作
	ActionAllow  = "allow"  // 白名单 / 未命中
	ActionBlock  = "block"  // 直接拒绝
	ActionLimit  = "limit"  // 限流拒绝（提示稍后再试）
	ActionReview = "review" // 放行但进入审核队列
	ActionWarn   = "warn"   // 放行并提示
	ActionLog    = "log"    // 仅记录，不打扰用户

	// 风险等级
	LevelHigh   = "high"
	LevelMedium = "medium"
	LevelLow    = "low"

	// SensitiveWordMaxRunes - 单条词条长度上限
	//
	// 超长词条会被运行时直接丢弃；保存校验必须用同一口径拦下来，
	// 否则管理员看到「保存成功」却发现规则不生效（静默失败）。
	SensitiveWordMaxRunes = 64
)

// SensitiveRule - 一条词条规则（白名单与普通规则同构）
type SensitiveRule struct {
	Word   string
	Action string
	Scenes []string
	Level  string
	Match  string
}

// SensitiveDecision - 一次判定结果
type SensitiveDecision struct {
	Action  string `json:"action"`
	Word    string `json:"word"`
	Level   string `json:"level"`
	Match   string `json:"match"`
	Scene   string `json:"scene"`
	Version string `json:"version"`
}

// Blocked - 是否需要拒绝本次内容
//
// limit 也按「拒绝」处理：真正的动态限流需要独立的计数窗口，
// 这里先保证命中限流规则时不会把内容发出去（调用方用 UserMessage 提示稍后再试）。
func (this SensitiveDecision) Blocked() bool {
	return this.Action == ActionBlock || this.Action == ActionLimit
}

// UserMessage - 给终端用户的提示（不暴露内部规则细节）
func (this SensitiveDecision) UserMessage() string {
	if this.Action == ActionLimit {
		return "操作过于频繁，请稍后再试！"
	}
	if this.Word != "" {
		return fmt.Sprintf("内容包含敏感词「%s」，请修改后重试！", this.Word)
	}
	return "内容不符合社区规范，请修改后重试！"
}

// NoticeMessage - 命中 warn 动作时的提示文案
//
// warn 的语义是「放行并提示」：不把文案透给用户，warn 就和 log 没有区别。
// 其它动作返回空串（block / limit 走 UserMessage，review 走审核提示）。
func (this SensitiveDecision) NoticeMessage() string {
	if this.Action != ActionWarn {
		return ""
	}
	if this.Word != "" {
		return fmt.Sprintf("内容包含敏感词「%s」，已放行，请注意用词！", this.Word)
	}
	return "内容可能不符合社区规范，已放行，请注意用词！"
}

// Sensitive - 敏感词公共类（无状态，编译结果按配置代次缓存）
type Sensitive struct{}

// 编译后的词库
type sensitiveCompiled struct {
	enabled       bool
	defaultAction string
	version       string
	rules         []SensitiveRule
	whitelist     []SensitiveRule
	exact         map[string][]int
	contains      *acMatcher
}

// 编译缓存：viper 配置变化会推进代次，代次不一致才重建
var sensitiveCache struct {
	sync.RWMutex
	generation int64
	ready      bool
	compiled   *sensitiveCompiled
}

// Decide - 按场景判定文本，并记录命中审计
//
// uid <= 0 时不记审计（后台「测试」按钮用 0，避免测试流量污染统计）。
func (this *Sensitive) Decide(scene string, uid int, text string) SensitiveDecision {

	compiled := this.compiled()
	decision := compiled.decide(scene, text)

	if decision.Action != ActionAllow && decision.Action != "" {
		recordSensitiveHit(scene, uid, text, decision)
	}

	return decision
}

// Config - 当前词库结构（后台展示 / 调试用）
func (this *Sensitive) Config() map[string]any {

	compiled := this.compiled()

	return map[string]any{
		"enabled":        boolToInt(compiled.enabled),
		"default_action": compiled.defaultAction,
		"version":        compiled.version,
		"rules":          rulesToMap(compiled.rules),
		"whitelist":      rulesToMap(compiled.whitelist),
		"scenes": []string{
			SceneComment, SceneArticle, SceneMoment, SceneDM, SceneChat,
		},
		"actions": []string{
			ActionBlock, ActionLimit, ActionReview, ActionWarn, ActionLog,
		},
	}
}

// compiled - 取编译结果（代次一致直接复用）
func (this *Sensitive) compiled() *sensitiveCompiled {

	generation := facade.SensitiveGeneration()

	sensitiveCache.RLock()
	if sensitiveCache.ready && sensitiveCache.generation == generation {
		compiled := sensitiveCache.compiled
		sensitiveCache.RUnlock()
		return compiled
	}
	sensitiveCache.RUnlock()

	compiled := buildSensitive()

	sensitiveCache.Lock()
	sensitiveCache.generation = generation
	sensitiveCache.compiled = compiled
	sensitiveCache.ready = true
	sensitiveCache.Unlock()

	return compiled
}

// buildSensitive - 解析 TOML 配置并编译匹配器
func buildSensitive() *sensitiveCompiled {

	compiled := &sensitiveCompiled{
		enabled:       true,
		defaultAction: ActionReview,
		version:       "1",
		exact:         make(map[string][]int),
	}

	if facade.SensitiveToml == nil {
		compiled.enabled = false
		compiled.contains = buildAC(nil)
		return compiled
	}

	compiled.enabled = cast.ToBool(facade.SensitiveToml.Get("enabled", true))
	compiled.defaultAction = normalizeAction(cast.ToString(facade.SensitiveToml.Get("default_action", ActionReview)), ActionReview)
	if version := strings.TrimSpace(cast.ToString(facade.SensitiveToml.Get("version", "1"))); version != "" {
		compiled.version = version
	}

	compiled.rules = parseSensitiveRules(toSensitiveRows(facade.SensitiveToml.Get("rules")), compiled.defaultAction, false)
	compiled.whitelist = parseSensitiveRules(toSensitiveRows(facade.SensitiveToml.Get("whitelist")), ActionAllow, true)

	for index, rule := range compiled.rules {
		if rule.Match != "exact" {
			continue
		}
		key := sensitiveCompact(rule.Word)
		if key == "" {
			continue
		}
		compiled.exact[key] = append(compiled.exact[key], index)
	}
	compiled.contains = buildAC(compiled.rules)

	return compiled
}

// toSensitiveRows - 把 viper 返回的数组统一成 []any
//
// TOML 的 [[rules]] 在 viper 里常被解码为 []map[string]any，
// spf13/cast 的 ToSlice 对这类具体类型的切片并不总能处理，这里用反射兜底。
func toSensitiveRows(value any) []any {

	if value == nil {
		return nil
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice {
		return nil
	}

	rows := make([]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		rows = append(rows, rv.Index(i).Interface())
	}

	return rows
}

// parseSensitiveRules - 把 TOML 里的 [[rules]] / [[whitelist]] 转成内部结构
func parseSensitiveRules(items []any, defaultAction string, forceAllow bool) []SensitiveRule {

	rules := make([]SensitiveRule, 0, len(items))

	for _, item := range items {
		row := cast.ToStringMap(item)

		word := strings.TrimSpace(cast.ToString(row["word"]))
		if word == "" {
			continue
		}
		// 保存接口已拦超长词条；手改配置文件时仍会走到这里，静默丢弃不可接受
		if len([]rune(word)) > SensitiveWordMaxRunes {
			facade.Log.Warn(map[string]any{
				"word":  word,
				"limit": SensitiveWordMaxRunes,
			}, "敏感词条超长，已忽略")
			continue
		}

		action := ActionAllow
		if !forceAllow {
			action = normalizeAction(cast.ToString(row["action"]), defaultAction)
		}

		match := strings.ToLower(strings.TrimSpace(cast.ToString(row["match"])))
		if match != "exact" {
			match = "contains"
		}

		scenes := make([]string, 0, 5)
		for _, scene := range cast.ToStringSlice(row["scenes"]) {
			scene = strings.ToLower(strings.TrimSpace(scene))
			if scene == "" {
				continue
			}
			// 场景拼错时规则永远不命中；这里保留原值而不是删掉 ——
			// scenes 删空会被当成「全部场景生效」，拼错反而扩大打击面
			if !isKnownScene(scene) {
				facade.Log.Warn(map[string]any{
					"word":  word,
					"scene": scene,
				}, "敏感词规则的场景无法识别，该规则不会在这些场景生效")
			}
			scenes = append(scenes, scene)
		}

		rules = append(rules, SensitiveRule{
			Word:   word,
			Action: action,
			Scenes: scenes,
			Level:  normalizeLevel(cast.ToString(row["level"])),
			Match:  match,
		})
	}

	return rules
}

// decide - 核心判定：白名单 → 规则匹配 → 选取优先级最高的处置
func (this *sensitiveCompiled) decide(scene, text string) SensitiveDecision {

	decision := SensitiveDecision{Action: ActionAllow, Scene: scene, Version: this.version}

	if !this.enabled || strings.TrimSpace(text) == "" {
		return decision
	}

	// 富文本先剥标签再归一化：<b> 会把词条切断（广<b>告</b>），
	// 不剥离等于给 HTML 绕过留口子
	compact := sensitiveCompact(sensitivePlainText(text))
	if compact == "" {
		return decision
	}

	// 白名单优先：官方公告 / 认证机构 / 业务模板等
	for _, rule := range this.whitelist {
		if rule.matchScene(scene) && rule.matches(compact) {
			return decision
		}
	}

	best := -1
	for _, index := range this.exact[compact] {
		if !this.rules[index].matchScene(scene) {
			continue
		}
		best = betterRule(this.rules, best, index)
	}
	for _, index := range this.contains.findAll(compact) {
		if !this.rules[index].matchScene(scene) {
			continue
		}
		best = betterRule(this.rules, best, index)
	}

	if best < 0 {
		return decision
	}

	rule := this.rules[best]
	return SensitiveDecision{
		Action:  rule.Action,
		Word:    rule.Word,
		Level:   rule.Level,
		Match:   rule.Match,
		Scene:   scene,
		Version: this.version,
	}
}

// isKnownScene - 是否是已接入的业务场景（配置里拼错场景时用于告警）
func isKnownScene(scene string) bool {
	switch scene {
	case SceneComment, SceneArticle, SceneMoment, SceneDM, SceneChat:
		return true
	}
	return false
}

// matchScene - 规则是否作用于当前场景（scenes 为空 = 全部场景）
func (this SensitiveRule) matchScene(scene string) bool {
	if len(this.Scenes) == 0 {
		return true
	}
	for _, item := range this.Scenes {
		if item == scene {
			return true
		}
	}
	return false
}

// matches - 白名单匹配（exact 走整段相等，contains 走包含）
func (this SensitiveRule) matches(compact string) bool {
	word := sensitiveCompact(this.Word)
	if word == "" {
		return false
	}
	if this.Match == "exact" {
		return compact == word
	}
	return strings.Contains(compact, word)
}

// betterRule - 选处置更重 / 风险更高 / 词更长的规则
func betterRule(rules []SensitiveRule, current, candidate int) int {

	if current < 0 {
		return candidate
	}

	currentRule := rules[current]
	candidateRule := rules[candidate]

	if actionRank(candidateRule.Action) != actionRank(currentRule.Action) {
		if actionRank(candidateRule.Action) > actionRank(currentRule.Action) {
			return candidate
		}
		return current
	}
	if levelRank(candidateRule.Level) != levelRank(currentRule.Level) {
		if levelRank(candidateRule.Level) > levelRank(currentRule.Level) {
			return candidate
		}
		return current
	}
	if len([]rune(candidateRule.Word)) > len([]rune(currentRule.Word)) {
		return candidate
	}

	return current
}

func actionRank(action string) int {
	switch action {
	case ActionBlock:
		return 5
	case ActionLimit:
		return 4
	case ActionReview:
		return 3
	case ActionWarn:
		return 2
	case ActionLog:
		return 1
	default:
		return 0
	}
}

func levelRank(level string) int {
	switch level {
	case LevelHigh:
		return 3
	case LevelMedium:
		return 2
	case LevelLow:
		return 1
	default:
		return 0
	}
}

func normalizeAction(action, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case ActionBlock:
		return ActionBlock
	case ActionLimit:
		return ActionLimit
	case ActionReview:
		return ActionReview
	case ActionWarn:
		return ActionWarn
	case ActionLog:
		return ActionLog
	case ActionAllow:
		return ActionAllow
	default:
		return fallback
	}
}

func normalizeLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case LevelHigh:
		return LevelHigh
	case LevelLow:
		return LevelLow
	default:
		return LevelMedium
	}
}

func rulesToMap(rules []SensitiveRule) []map[string]any {
	result := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		result = append(result, map[string]any{
			"word":   rule.Word,
			"action": rule.Action,
			"scenes": rule.Scenes,
			"level":  rule.Level,
			"match":  rule.Match,
		})
	}
	return result
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// ---------------------------------------------------------------------------
// 归一化：全半角 / 大小写 / 零宽字符 / 常见繁简 / 空格与符号插入
// ---------------------------------------------------------------------------

// sensitiveTradToSimp - 常见繁体 → 简体（只覆盖高风险词里的高频字，不追求完整映射）
//
// 完整繁简表太大且会随词库变化失真；这里只做「基础归一化」，
// 剩下靠规则里补充变体或人工复核闭环，避免词表无限膨胀。
var sensitiveTradToSimp = map[rune]rune{
	'廣': '广', '開': '开', '戶': '户', '貸': '贷', '賭': '赌',
	'詐': '诈', '騙': '骗', '黃': '黄', '發': '发', '證': '证',
	'職': '职', '單': '单', '帶': '带', '賺': '赚', '錢': '钱',
	'銀': '银', '額': '额', '匯': '汇', '兌': '兑', '換': '换',
	'幣': '币', '虛': '虚', '擬': '拟', '盤': '盘', '場': '场',
	'網': '网', '絡': '络', '遊': '游', '戲': '戏', '娛': '娱',
	'樂': '乐', '視': '视', '頻': '频', '電': '电', '話': '话',
	'聯': '联', '繫': '系', '資': '资', '訊': '讯', '軟': '软',
	'體': '体', '應': '应', '註': '注', '冊': '册', '號': '号',
}

// sensitiveNormalize - 宽度折叠 + 大小写 + 零宽字符 + 常见繁简
func sensitiveNormalize(text string) string {

	var builder strings.Builder
	builder.Grow(len(text))

	for _, char := range text {
		// 零宽字符：常被用来切断关键词
		switch char {
		case '\u200b', '\u200c', '\u200d', '\ufeff':
			continue
		}

		// 全角 ASCII → 半角；全角空格 → 普通空格
		if char == '\u3000' {
			char = ' '
		} else if char >= '\uff01' && char <= '\uff5e' {
			char -= 0xfee0
		}

		if mapped, ok := sensitiveTradToSimp[char]; ok {
			char = mapped
		}

		builder.WriteRune(unicode.ToLower(char))
	}

	return builder.String()
}

// sensitivePlainText - 去掉 HTML 标签，只保留可读文本
//
// 编辑器提交的正文是 HTML，标签本身会切断词条（广<b>告</b>），
// 直接对原文判定等于留了一个绕过口子；链接 / 样式里的属性整段丢弃，
// 但标签之间的文字保留，所以 <a>广告</a> 依然能被命中。
func sensitivePlainText(text string) string {

	if !strings.Contains(text, "<") {
		return text
	}

	runes := []rune(text)

	var builder strings.Builder
	builder.Grow(len(text))

	for index := 0; index < len(runes); index++ {
		char := runes[index]

		// 只有「<字母」或「</」才当标签开头，避免把 "1<2" 这类正常文本吞掉
		if char == '<' && index+1 < len(runes) && (runes[index+1] == '/' || unicode.IsLetter(runes[index+1])) {
			closed := false
			for index++; index < len(runes); index++ {
				if runes[index] == '>' {
					closed = true
					break
				}
			}
			if !closed {
				break
			}
			continue
		}

		builder.WriteRune(char)
	}

	return builder.String()
}

// sensitiveCompact - 在归一化基础上移除空白 / 标点 / 符号
//
// 主要对抗「广 告」「广-告」「广*告」这类插入分隔符的绕过方式。
func sensitiveCompact(text string) string {

	normalized := sensitiveNormalize(text)

	var builder strings.Builder
	builder.Grow(len(normalized))

	for _, char := range normalized {
		if unicode.IsSpace(char) || unicode.IsPunct(char) || unicode.IsSymbol(char) {
			continue
		}
		builder.WriteRune(char)
	}

	return builder.String()
}

// ---------------------------------------------------------------------------
// Aho-Corasick：contains 规则的多模式匹配
// ---------------------------------------------------------------------------

type acNode struct {
	next map[rune]int
	fail int
	out  []int // 命中规则在 compiled.rules 里的下标
}

type acMatcher struct {
	nodes []acNode
	rules []SensitiveRule
}

// buildAC - 根据 contains 规则构建自动机
func buildAC(rules []SensitiveRule) *acMatcher {

	matcher := &acMatcher{
		nodes: []acNode{{next: make(map[rune]int)}},
		// 保存完整规则表：out 里的下标必须能直接索引 compiled.rules，
		// 否则 skipped 的 exact 规则会让下标整体错位。
		rules: rules,
	}

	for index, rule := range rules {
		if rule.Match == "exact" {
			continue
		}

		word := sensitiveCompact(rule.Word)
		if word == "" {
			continue
		}

		node := 0
		for _, char := range word {
			next, ok := matcher.nodes[node].next[char]
			if !ok {
				matcher.nodes = append(matcher.nodes, acNode{next: make(map[rune]int)})
				next = len(matcher.nodes) - 1
				matcher.nodes[node].next[char] = next
			}
			node = next
		}
		matcher.nodes[node].out = append(matcher.nodes[node].out, index)
	}

	matcher.buildFail()

	return matcher
}

// buildFail - BFS 构建失败指针，并把 fail 节点的输出合并到当前节点
func (this *acMatcher) buildFail() {

	queue := make([]int, 0, len(this.nodes))

	for _, child := range this.nodes[0].next {
		this.nodes[child].fail = 0
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for char, child := range this.nodes[current].next {
			fail := this.nodes[current].fail

			for fail != 0 {
				if next, ok := this.nodes[fail].next[char]; ok {
					fail = next
					break
				}
				fail = this.nodes[fail].fail
			}
			if next, ok := this.nodes[fail].next[char]; ok && next != child {
				fail = next
			}

			this.nodes[child].fail = fail
			this.nodes[child].out = append(this.nodes[child].out, this.nodes[fail].out...)
			queue = append(queue, child)
		}
	}
}

// findAll - 返回命中规则在 matcher.rules 里的下标（已去重）
func (this *acMatcher) findAll(text string) []int {

	if len(this.nodes) == 0 || len(this.rules) == 0 {
		return nil
	}

	result := make([]int, 0, 4)
	seen := make([]bool, len(this.rules))
	node := 0

	for _, char := range text {
		for node != 0 {
			if _, ok := this.nodes[node].next[char]; ok {
				break
			}
			node = this.nodes[node].fail
		}
		if next, ok := this.nodes[node].next[char]; ok {
			node = next
		} else {
			node = 0
		}

		for _, index := range this.nodes[node].out {
			if !seen[index] {
				seen[index] = true
				result = append(result, index)
			}
		}
	}

	return result
}
