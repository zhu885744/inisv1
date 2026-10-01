package facade

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

func Lang(ctx *gin.Context, key string, args ...any) (result string) {

	var lang string
	// 获取语言
	lang, _ = ctx.Cookie("inis_lang")
	lang = ctx.DefaultQuery("inis_lang", lang)
	lang = strings.ToLower(lang)

	model := utils.LangModel{
		Directory: "config/i18n/",
	}

	// 设置语言
	if !utils.Is.Empty(lang) {
		model.Lang = lang
	}

	result = cast.ToString(utils.Lang(model).Value(key, args...))

	// 兜底：语言包文件缺失时（二进制部署只带 config 目录，config/i18n 下没有语言文件），
	// go-utils 对**任何** key 都返回空串 —— 于是所有提示都变成空消息，
	// 前端只能显示自己的兜底文案（"登录状态异常…"），排查时根本看不到原因。
	// 这里退回原文：中文源码本身就是兜底文案，与「key 未收录 → 返回原文」的既有行为一致。
	if utils.Is.Empty(result) {
		return formatFallback(key, args...)
	}

	return result
}

// formatFallback - 语言包缺失时的兜底文案：按占位符顺序把参数填回原文
//
// 刻意不用 fmt.Sprintf：那会让 go vet 把 Lang 判定成「printf 包装函数」，
// 进而把项目里大量「把变量当文案传进来」的既有调用点（Lang(ctx, msg)）报成
// non-constant format string。这里只做 %s/%v/%d 的顺序替换，够用且不影响 vet。
func formatFallback(text string, args ...any) string {
	if len(args) == 0 {
		return text
	}

	placeholders := []string{"%s", "%v", "%d"}
	for _, arg := range args {
		for _, placeholder := range placeholders {
			if strings.Contains(text, placeholder) {
				text = strings.Replace(text, placeholder, cast.ToString(arg), 1)
				break
			}
		}
	}

	return text
}
