package controller

import (
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

type Rss struct {
	base
}

func (this *Rss) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"index": this.index,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
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

	ctx.Header("Content-Type", "application/xml; charset=utf-8")
	ctx.String(200, xml)
}

func (this *Rss) generateRSS(siteName, siteURL, siteDescription string, articles []model.Article, showFull bool) string {
	var sb strings.Builder

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">` + "\n")
	sb.WriteString(`  <channel>` + "\n")
	sb.WriteString(fmt.Sprintf(`    <title><![CDATA[%s]]></title>`+"\n", siteName))
	sb.WriteString(fmt.Sprintf(`    <link><![CDATA[%s]]></link>`+"\n", siteURL))
	sb.WriteString(fmt.Sprintf(`    <description><![CDATA[%s]]></description>`+"\n", siteDescription))
	sb.WriteString(fmt.Sprintf(`    <language>zh-cn</language>` + "\n"))
	sb.WriteString(fmt.Sprintf(`    <lastBuildDate>%s</lastBuildDate>`+"\n", time.Now().Format(time.RFC1123Z)))
	sb.WriteString(fmt.Sprintf(`    <atom:link href="%s/rss" rel="self" type="application/rss+xml"/>`+"\n", siteURL))

	for _, article := range articles {
		itemURL := fmt.Sprintf("%s/archives/%d", siteURL, article.Id)

		sb.WriteString(`    <item>` + "\n")
		sb.WriteString(fmt.Sprintf(`      <title><![CDATA[%s]]></title>`+"\n", article.Title))
		sb.WriteString(fmt.Sprintf(`      <link><![CDATA[%s]]></link>`+"\n", itemURL))
		sb.WriteString(fmt.Sprintf(`      <guid isPermaLink="true"><![CDATA[%s]]></guid>`+"\n", itemURL))

		var content string
		if showFull && article.Content != "" {
			content = article.Content
		} else if article.Abstract != "" {
			content = article.Abstract
		}

		if content != "" {
			sb.WriteString(fmt.Sprintf(`      <description><![CDATA[%s]]></description>`+"\n", content))
		}

		if article.PublishTime > 0 {
			sb.WriteString(fmt.Sprintf(`      <pubDate>%s</pubDate>`+"\n", time.Unix(article.PublishTime, 0).Format(time.RFC1123Z)))
		}

		sb.WriteString(`    </item>` + "\n")
	}

	sb.WriteString(`  </channel>` + "\n")
	sb.WriteString(`</rss>`)

	return sb.String()
}
