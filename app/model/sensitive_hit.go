package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"inis/app/facade"
)

/**
 * 敏感词命中审计
 *
 * 记录「哪个场景、哪条规则、什么处置、哪一版词库」，用于统计命中率 / 拦截率，
 * 也为后续申诉与误伤分析留数据基础。
 *
 * 隐私取舍：
 *   - 私信 / 聊天室不落原文摘要，只存内容哈希，避免人工看到用户全文；
 *   - 其它场景只在 block / review 时保留 120 字摘要，便于复核误伤；
 *   - uid 原样存储（后台需要定位用户），但不在日志里输出全文。
 */

type SensitiveHits struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid         int    `gorm:"type:int(32); index:idx_sensitive_hits_uid; comment:命中用户;" json:"uid"`
	Scene       string `gorm:"size:16; index:idx_sensitive_hits_scene; comment:业务场景;" json:"scene"`
	Action      string `gorm:"size:16; comment:处置动作;" json:"action"`
	Word        string `gorm:"size:64; comment:命中的词;" json:"word"`
	Level       string `gorm:"size:16; comment:风险等级;" json:"level"`
	Match       string `gorm:"size:16; comment:匹配方式;" json:"match"`
	Version     string `gorm:"size:32; comment:词库版本;" json:"version"`
	ContentHash string `gorm:"size:64; comment:内容哈希（不存全文）;" json:"content_hash"`
	Excerpt     string `gorm:"size:255; comment:内容摘要（私信/聊天室不存）;" json:"excerpt"`
	// 机审维度：本地词库命中为 0 / 空，云端审核记为百度的 type / subType
	// （30 涉政、33 涉黄、36 广告…），后台据此按维度统计与筛选
	AuditType    int    `gorm:"type:int(12); default:0; index:idx_sensitive_hits_audit; comment:机审主维度;" json:"audit_type"`
	AuditSubType int    `gorm:"type:int(32); default:0; comment:机审子维度;" json:"audit_sub_type"`
	AuditMsg     string `gorm:"size:255; default:''; comment:机审判定说明;" json:"audit_msg"`
	CreateTime   int64  `gorm:"autoCreateTime; index:idx_sensitive_hits_time; comment:创建时间;" json:"create_time"`
}

func InitSensitiveHits() {
	if err := facade.DB.Drive().AutoMigrate(&SensitiveHits{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "SensitiveHits表迁移失败")
	}
}

// recordSensitiveHit - 异步落审计，失败只记日志，绝不影响业务主流程
func recordSensitiveHit(scene string, uid int, text string, decision SensitiveDecision) {

	if facade.DB == nil || uid <= 0 {
		return
	}

	// 私信 / 聊天室：实时且隐私敏感，不保存原文摘要
	excerpt := ""
	if scene != SceneDM && scene != SceneChat &&
		(decision.Action == ActionBlock || decision.Action == ActionReview) {
		// 摘要存纯文本：后台复核时看到的应该是用户读到的内容，而不是一堆标签
		excerpt = truncateRunes(strings.TrimSpace(sensitivePlainText(text)), 120)
	}

	sum := sha256.Sum256([]byte(sensitiveCompact(sensitivePlainText(text))))

	hit := SensitiveHits{
		Uid:          uid,
		Scene:        scene,
		Action:       decision.Action,
		Word:         decision.Word,
		Level:        decision.Level,
		Match:        decision.Match,
		Version:      decision.Version,
		ContentHash:  hex.EncodeToString(sum[:]),
		Excerpt:      excerpt,
		AuditType:    decision.AuditType,
		AuditSubType: decision.AuditSubType,
		AuditMsg:     truncateRunes(strings.TrimSpace(decision.AuditMsg), 240),
	}

	go func(item SensitiveHits) {
		defer func() {
			if recover() != nil {
				facade.Log.Error(map[string]any{"scene": item.Scene, "uid": item.Uid}, "写入敏感词命中审计失败")
			}
		}()
		if _, err := facade.DB.Model(&item).Create(&item); err != nil {
			facade.Log.Error(map[string]any{"error": err, "scene": item.Scene}, "写入敏感词命中审计失败")
		}
	}(hit)
}

// SensitiveHitStats - 最近 N 天的命中统计（按场景 + 动作聚合）
//
// auditType > 0 时只统计该机审维度（百度的 type），0 表示全部。
func SensitiveHitStats(days int, auditType int) []map[string]any {

	result := make([]map[string]any, 0)

	if facade.DB == nil {
		return result
	}
	if days <= 0 {
		days = 7
	}

	since := time.Now().AddDate(0, 0, -days).Unix()

	var rows []struct {
		Scene  string
		Action string
		Total  int64
	}

	query := facade.DB.Drive().Model(&SensitiveHits{}).
		Select("scene, action, COUNT(*) AS total").
		Where("create_time >= ?", since)
	if auditType > 0 {
		query = query.Where("audit_type = ?", auditType)
	}

	err := query.Group("scene, action").Order("scene asc").Scan(&rows).Error
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "查询敏感词命中统计失败")
		return result
	}

	for _, row := range rows {
		result = append(result, map[string]any{
			"scene":  row.Scene,
			"action": row.Action,
			"total":  row.Total,
		})
	}

	return result
}

// SensitiveHitDimensions - 最近 N 天按机审维度的命中统计（降序）
//
// 只统计 audit_type > 0 的记录（即云端审核命中的），本地词库命中没有维度概念。
func SensitiveHitDimensions(days int) []map[string]any {

	result := make([]map[string]any, 0)

	if facade.DB == nil {
		return result
	}
	if days <= 0 {
		days = 7
	}

	since := time.Now().AddDate(0, 0, -days).Unix()

	var rows []struct {
		AuditType int
		Total     int64
	}

	err := facade.DB.Drive().Model(&SensitiveHits{}).
		Select("audit_type, COUNT(*) AS total").
		Where("create_time >= ? AND audit_type > 0", since).
		Group("audit_type").
		Order("total desc").
		Scan(&rows).Error
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "查询机审维度统计失败")
		return result
	}

	for _, row := range rows {
		result = append(result, map[string]any{
			"type":  row.AuditType,
			"label": AuditTypeLabel(row.AuditType),
			"total": row.Total,
		})
	}

	return result
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "…"
}
