package model

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 话题（Hashtag）：**标签即话题**
//
// 复用 tags 表（管理后台的「标签管理」就是话题管理），额外用意：
//   - 动态 / 文章正文里写 #话题# → 写库前规范化为 #[话题名](/topic/12)，前端渲染成可点击链接；
//   - 话题不存在时自动创建，正文里的话题会写进 tags 字段（动态 moments.tags、文章 article.tags，
//     格式与文章原有口径一致：|1|2|3|），话题聚合页据此 LIKE 检索；
//   - 与 @提及（mention.go）同一套思路：规范化入库 + 幂等认领 + 前端只负责渲染。
type Topic struct {
	Id   int
	Name string
}

// 话题名长度上限（tags.name 是 size:32）、单条内容最多关联的话题数（防刷话题）
const (
	topicNameMaxLen     = 32
	topicMaxPerContent  = 5
	topicCandidateLimit = 500
)

// 热度权重：内容量比关注数更能说明话题活跃度，长文比动态更重
const (
	topicWeightArticle = 3
	topicWeightMoment  = 2
	topicWeightFollow  = 2
)

var (
	// 已规范化：#\[名称\](/topic/12)
	topicLinkPattern = regexp.MustCompile(`#\[([^\[\]]*)\]\(/topic/(\d+)\)`)
	// 待规范化：#话题#（内部不允许出现 # 与空白，因此「# 标题」「## 标题」这类 markdown 标题不会命中）
	topicTextPattern = regexp.MustCompile(`#([^#\s]{1,32})#`)
	// 话题名里会破坏 markdown 链接语法的字符
	topicNameCleaner = regexp.MustCompile(`[\[\]()|]`)
)

// Parse - 规范化正文里的 #话题#（自动创建话题），并返回关联到的话题 id（去重、保序）
//
// 与 @提及 一样是幂等的：已规范化的 #[名称](/topic/12) 原样保留并计入关联列表，
// 所以「草稿 → 发布」这一步不会丢话题。
func (this *Topic) Parse(content string) (string, []int) {

	if !strings.Contains(content, "#") {
		return content, nil
	}

	ids := []int{}
	seen := map[int]bool{}

	// 1. 先把已规范化的链接挖成占位符，避免下面的 #话题# 规则把它二次拆解
	protected := []string{}
	body := topicLinkPattern.ReplaceAllStringFunc(content, func(match string) string {
		group := topicLinkPattern.FindStringSubmatch(match)
		if id := cast.ToInt(group[2]); id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
		protected = append(protected, match)
		return fmt.Sprintf("\x00%d\x00", len(protected)-1)
	})

	// 2. 处理正文里新写的 #话题#
	body = topicTextPattern.ReplaceAllStringFunc(body, func(match string) string {
		if len(ids) >= topicMaxPerContent {
			return match
		}

		name := cleanTopicName(match[1 : len(match)-1])
		if name == "" {
			return match
		}

		tag, err := this.FindOrCreate(name)
		if err != nil || tag.Id <= 0 {
			// 建话题失败就当普通文本，不影响内容发布
			return match
		}

		if !seen[tag.Id] {
			seen[tag.Id] = true
			ids = append(ids, tag.Id)
		}

		return fmt.Sprintf("#[%v](/topic/%v)", name, tag.Id)
	})

	// 3. 还原被保护的链接
	for i, item := range protected {
		body = strings.Replace(body, fmt.Sprintf("\x00%d\x00", i), item, 1)
	}

	return body, ids
}

// FindOrCreate - 按名称找话题，没有就创建
//
// 历史数据可能重名（tags 表没有唯一索引），此时取 id 最小的那条，保证多次解析结果一致
// （与 @提及 的「同名取 id 小者」策略一致）。
//
// 注意：Parse 在「发布动态 / 保存文章」的关键链路上，建话题失败**不能**影响内容发布，
// 所以这里兜住 panic，失败就当作者没写话题。
func (this *Topic) FindOrCreate(name string) (tag Tags, err error) {

	defer func() {
		if r := recover(); r != nil {
			facade.Log.Error(map[string]any{"error": r, "name": name}, "话题创建异常")
			tag, err = Tags{}, errors.New("话题创建失败")
		}
	}()

	name = cleanTopicName(name)
	if name == "" {
		return tag, errors.New("话题名不能为空！")
	}

	item, _ := facade.DB.Model(&Tags{}).Where("name", name).Order("id asc").Find()
	if !utils.Is.Empty(item) {
		tag.Id = cast.ToInt(item["id"])
		tag.Name = cast.ToString(item["name"])
		return tag, nil
	}

	now := time.Now().Unix()
	table := Tags{Name: name, CreateTime: now, UpdateTime: now}
	if _, err = facade.DB.Model(&table).Create(&table); err != nil {
		return tag, err
	}

	tag.Id, tag.Name = table.Id, table.Name
	return tag, nil
}

// BuildTagsField - 话题 id → tags 字段（|1|2|3|，与文章原有口径一致，便于 LIKE 检索）
func (this *Topic) BuildTagsField(ids []int) string {

	if len(ids) == 0 {
		return ""
	}

	parts := []string{}
	for _, id := range uniqueIds(ids) {
		if id <= 0 {
			continue
		}
		parts = append(parts, cast.ToString(id))
	}

	if len(parts) == 0 {
		return ""
	}

	return "|" + strings.Join(parts, "|") + "|"
}

// MergeTagsField - 把话题 id 合并进已有 tags 字段（文章作者手选的标签 + 正文话题，取并集）
func (this *Topic) MergeTagsField(field string, ids []int) string {

	merged := this.TagsFieldIds(field)
	merged = append(merged, ids...)

	return this.BuildTagsField(merged)
}

// TagsFieldIds - 解析 tags 字段（|1|2|3|）为 id 列表
func (this *Topic) TagsFieldIds(field string) []int {

	if utils.Is.Empty(field) {
		return []int{}
	}

	result := []int{}
	for _, item := range strings.Split(cast.ToString(field), "|") {
		if id := cast.ToInt(strings.TrimSpace(item)); id > 0 {
			result = append(result, id)
		}
	}

	return uniqueIds(result)
}

// uniqueIds - []int 去重保序并剔除非法 id
//
// 不用 utils.ArrayUnique：它返回 []any，转不回 []int。
func uniqueIds(ids []int) []int {

	seen := make(map[int]bool, len(ids))
	result := make([]int, 0, len(ids))

	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}

	return result
}

// Counts - 批量统计话题的内容量
//
// 口径与后台「标签管理」的文章数一致（articleCountByTagIds）：只算审核通过的内容，
// 动态额外要求 status=1（发布态，不含草稿）。
func (this *Topic) Counts() (articles map[string]int, moments map[string]int) {

	return this.countByModel(&[]Article{}), this.countByModel(&[]Moments{})
}

// countByModel - 把某个内容表的 tags 列拉出来在内存里统计（一次查询，避免 N+1）
func (this *Topic) countByModel(table any) map[string]int {

	result := make(map[string]int)

	query := facade.DB.Model(table).Where("audit", 1)
	if _, ok := table.(*[]Moments); ok {
		query = query.Where("status", 1)
	}

	column, err := query.Column("tags")
	if err != nil || utils.Is.Empty(column) {
		return result
	}

	for _, item := range cast.ToStringSlice(column) {
		for _, id := range this.TagsFieldIds(item) {
			result[cast.ToString(id)]++
		}
	}

	return result
}

// Heat - 话题热度分（内容量 + 关注数加权）
func (this *Topic) Heat(article, moment, follow int) int {
	return article*topicWeightArticle + moment*topicWeightMoment + follow*topicWeightFollow
}

// Board - 话题榜（按热度排序）
//
// keyword 非空时按话题名模糊过滤（话题联想候选用同一份逻辑）；
// onlyRecommend 为真只返回精选话题。
func (this *Topic) Board(keyword string, onlyRecommend bool, limit int) []map[string]any {

	if limit <= 0 || limit > topicCandidateLimit {
		limit = 20
	}

	query := facade.DB.Model(&[]Tags{})
	if onlyRecommend {
		query = query.Where("recommend", 1)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		query = query.Like("name", "%"+kw+"%")
	}

	// 候选池：榜单排序要在内存里按热度算，先取一个有上限的候选集
	items, _ := query.Order("id desc").Limit(topicCandidateLimit).Select()
	if len(items) == 0 {
		return []map[string]any{}
	}

	articles, moments := this.Counts()
	follows := (&TagFollows{}).CountByTagIds()

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		id := cast.ToString(item["id"])
		row := map[string]any{
			"id":             item["id"],
			"name":           item["name"],
			"description":    item["description"],
			"avatar":         item["avatar"],
			"moderator_uid":  item["moderator_uid"],
			"recommend":      item["recommend"],
			"create_time":    item["create_time"],
			"article_count":  articles[id],
			"moment_count":   moments[id],
			"follow_count":   follows[id],
		}
		row["heat"] = this.Heat(articles[id], moments[id], follows[id])
		result = append(result, row)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if cast.ToInt(result[i]["heat"]) == cast.ToInt(result[j]["heat"]) {
			return cast.ToInt(result[i]["id"]) > cast.ToInt(result[j]["id"])
		}
		return cast.ToInt(result[i]["heat"]) > cast.ToInt(result[j]["heat"])
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result
}

// cleanTopicName - 清洗话题名：去掉会破坏链接语法的字符，按长度截断
func cleanTopicName(name string) string {

	name = topicNameCleaner.ReplaceAllString(strings.TrimSpace(name), "")
	name = strings.TrimSpace(name)

	if name == "" {
		return ""
	}

	runes := []rune(name)
	if len(runes) > topicNameMaxLen {
		name = string(runes[:topicNameMaxLen])
	}

	return name
}
