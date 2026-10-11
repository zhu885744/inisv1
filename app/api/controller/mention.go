package controller

import (
	"strings"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// ============================== 用户 @ 提及 ==============================
//
// 本控制器只提供「候选列表」一个接口：@ 的解析、规范化（@昵称 → @[昵称](/author/uid)）
// 与站内通知都在保存内容时完成，调用点共四处：
//   - 评论：app/api/controller/comment.go
//   - 动态：app/api/controller/moments.go
//   - 文章：app/api/controller/article.go
//   - 独立页：app/api/controller/pages.go
// 统一实现在 app/model/mention.go（model.Mention.Parse / Notify）。
//
// 候选范围**仅限当前登录用户已关注的人**：前端不展示、后端也不解析未关注的人。
type Mention struct {
	base
}

// IGET - 查询（候选列表）
func (this *Mention) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"candidates": this.candidates,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

// IPOST / IPUT / IDEL - 无对应场景（@ 只在保存内容时处理，没有独立的写接口）
func (this *Mention) IPOST(ctx *gin.Context) { this.deny(ctx) }
func (this *Mention) IPUT(ctx *gin.Context)  { this.deny(ctx) }
func (this *Mention) IDEL(ctx *gin.Context)  { this.deny(ctx) }

func (this *Mention) deny(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "该操作不支持！"), 405)
}

func (this *Mention) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// candidates - 我关注的人（@ 候选列表）
//
// limit 一次给足（最多 500）：前端打开选择器只请求一次，后续输入过滤在本地做，
// 避免每敲一个字符就打一次接口。
func (this *Mention) candidates(ctx *gin.Context) {
	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx, map[string]any{"keyword": "", "limit": 500})

	list := (&model.Mention{}).Following(user.Id, cast.ToString(params["keyword"]), cast.ToInt(params["limit"]))

	this.json(ctx, list, facade.Lang(ctx, "查询成功！"), 200)
}
