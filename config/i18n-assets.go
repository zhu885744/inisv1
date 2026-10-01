package config

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// 内置多语言文件：仓库里已提交 config/i18n/*.json，这里再嵌一份进二进制，
// 这样把二进制单独拷到服务器（连 config 目录都没有）时也能正常使用多语言。
//
//go:embed i18n/*.json
var i18nAssets embed.FS

// releaseI18n - 把内置的多语言文件释放到 config/i18n/（缺失时才写）
//
// 已存在的文件一律不覆盖：管理员可能自己改过翻译，重新部署不该把改动冲掉。
// app.toml 等配置文件由 app/facade/bootstrap.go 负责生成，这里只补语言包。
//
// 这里用 fmt 输出：本函数可能在日志组件完全就绪之前执行（见 facade/log.go 的说明）。
func releaseI18n() {
	entries, err := fs.ReadDir(i18nAssets, "i18n")
	if err != nil {
		return
	}

	target := filepath.Join("config", "i18n")
	if err := os.MkdirAll(target, 0755); err != nil {
		fmt.Printf("[inis] 创建语言包目录失败：%s（%v）\n", target, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		file := filepath.Join(target, entry.Name())
		if _, err := os.Stat(file); err == nil {
			continue
		}

		body, err := i18nAssets.ReadFile("i18n/" + entry.Name())
		if err != nil {
			continue
		}

		if err := os.WriteFile(file, body, 0644); err != nil {
			fmt.Printf("[inis] 释放语言文件失败：%s（%v）\n", file, err)
			continue
		}

		fmt.Printf("[inis] 已释放内置语言文件：%s\n", file)
	}
}
