package controller

import (
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type Rss struct {
	base
}

func (this *Rss) IGET(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "不支持此请求路径，请使用 /api/rss"), 405)
}

func (this *Rss) IPOST(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "不支持POST请求！"), 405)
}

func (this *Rss) IPUT(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "不支持PUT请求！"), 405)
}

func (this *Rss) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "不支持DELETE请求！"), 405)
}

func (this *Rss) INDEX(ctx *gin.Context) {
	this.index(ctx)
}

func (this *Rss) index(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"limit": 20,
		"order": "publish_time desc",
		"full":  false,
	})

	limit := cast.ToInt(params["limit"])
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	showFull := cast.ToBool(params["full"])

	var articles []model.Article
	facade.DB.Model(&articles).
		Where("audit", 1).
		Where("publish_time", "<=", time.Now().Unix()).
		Where("publish_time", "!=", 0).
		Order(params["order"]).
		Limit(limit).
		Select()

	siteName := cast.ToString(facade.AppToml.Get("app.name", "inis"))

	var siteURL string
	if url := cast.ToString(params["url"]); url != "" {
		siteURL = url
	} else if url := cast.ToString(facade.AppToml.Get("app.url")); url != "" {
		siteURL = url
	} else {
		host := cast.ToString(facade.AppToml.Get("app.host", "localhost"))
		port := cast.ToInt(facade.AppToml.Get("app.port", 8080))
		siteURL = fmt.Sprintf("http://%s:%d", host, port)
	}

	siteDescription := cast.ToString(facade.AppToml.Get("app.description", ""))

	xml := this.generateRSS(siteName, siteURL, siteDescription, articles, showFull)

	// 设置标准的 RSS 响应头
	ctx.Header("Content-Type", "application/xml; charset=utf-8")
	ctx.Header("Cache-Control", "public, max-age=3600") // 缓存1小时
	ctx.Header("Last-Modified", time.Now().UTC().Format(time.RFC1123))
	ctx.Header("ETag", fmt.Sprintf("%x", utils.Hash.Sum32(xml)))
	ctx.String(200, xml)
}

func (this *Rss) generateRSS(siteName, siteURL, siteDescription string, articles []model.Article, showFull bool) string {
	var sb strings.Builder

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">` + "\n")
	sb.WriteString(`  <channel>` + "\n")
	sb.WriteString(fmt.Sprintf(`    <title>%s</title>`+"\n", this.escapeXML(siteName)))
	sb.WriteString(fmt.Sprintf(`    <link>%s</link>`+"\n", siteURL)) // link 不使用 CDATA
	sb.WriteString(fmt.Sprintf(`    <description>%s</description>`+"\n", this.escapeXML(siteDescription)))
	sb.WriteString(`    <language>zh-cn</language>` + "\n")
	sb.WriteString(`    <generator>inis RSS Generator</generator>` + "\n")
	sb.WriteString(`    <ttl>60</ttl>` + "\n") // 建议抓取器60分钟刷新一次
	sb.WriteString(fmt.Sprintf(`    <lastBuildDate>%s</lastBuildDate>`+"\n", this.formatRSSDate(time.Now())))
	sb.WriteString(fmt.Sprintf(`    <atom:link href="%s/rss" rel="self" type="application/rss+xml"/>`+"\n", siteURL))

	for _, article := range articles {
		itemURL := fmt.Sprintf("%s/archives/%d", siteURL, article.Id)

		sb.WriteString(`    <item>` + "\n")
		sb.WriteString(fmt.Sprintf(`      <title>%s</title>`+"\n", this.escapeXML(article.Title)))
		sb.WriteString(fmt.Sprintf(`      <link>%s</link>`+"\n", itemURL))                    // link 不使用 CDATA
		sb.WriteString(fmt.Sprintf(`      <guid isPermaLink="true">%s</guid>`+"\n", itemURL)) // guid 不使用 CDATA

		// description 必须有内容，否则某些抓取器认为 item 无效
		var content string
		if showFull && article.Content != "" {
			content = article.Content
		} else if article.Abstract != "" {
			content = article.Abstract
		} else {
			content = article.Title // 如果没有摘要，使用标题作为描述
		}
		sb.WriteString(fmt.Sprintf(`      <description>%s</description>`+"\n", this.escapeXML(content)))

		// 使用标准 RSS 日期格式
		if article.PublishTime > 0 {
			sb.WriteString(fmt.Sprintf(`      <pubDate>%s</pubDate>`+"\n", this.formatRSSDate(time.Unix(article.PublishTime, 0))))
		}

		sb.WriteString(`    </item>` + "\n")
	}

	sb.WriteString(`  </channel>` + "\n")
	sb.WriteString(`</rss>`)

	return sb.String()
}

// escapeXML - XML 转义，避免 CDATA 问题
func (this *Rss) escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// formatRSSDate - 格式化 RSS 标准日期（RFC 822）
func (this *Rss) formatRSSDate(t time.Time) string {
	// RFC 822 格式：Wed, 02 Oct 2002 13:00:00 GMT
	return t.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
}
