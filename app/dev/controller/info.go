package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/mem"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"os"
)

type Info struct {
	base
}

// IGET - GET请求本体
func (this *Info) IGET(ctx *gin.Context) {
	allow := map[string]any{
		"time":    this.time,
		"system":  this.system,
		"device":  this.device,
		"version": this.version,
		"renew":   this.renew,
		"kill":    this.kill,
	}
	this.handleHTTPMethod(ctx, allow)
}

// IPOST - POST请求本体
func (this *Info) IPOST(ctx *gin.Context) {
	this.handleHTTPMethod(ctx, map[string]any{})
}

// IPUT - PUT请求本体
func (this *Info) IPUT(ctx *gin.Context) {
	this.handleHTTPMethod(ctx, map[string]any{})
}

// IDEL - DELETE请求本体
func (this *Info) IDEL(ctx *gin.Context) {
	this.handleHTTPMethod(ctx, map[string]any{})
}

// INDEX - GET请求本体
func (this *Info) INDEX(ctx *gin.Context) {
	this.json(ctx, map[string]any{
		"system": this.getSystemInfo(ctx),
	}, facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// system - 系统信息
func (this *Info) system(ctx *gin.Context) {
	var memory map[string]any
	if vm, err := mem.VirtualMemory(); err == nil {
		memory = map[string]any{
			"free":  vm.Free,
			"used":  vm.Used,
			"total": vm.Total,
		}
	}

	info := map[string]any{
		"path":         utils.Get.Pwd(),
		"pid":          utils.Get.Pid(),
		"port": map[string]any{
			"run":  this.get(ctx, "port"),
			"real": facade.AppToml.Get("app.port"),
		},
		"memory":       memory,
		"domain":       this.get(ctx, "domain"),
		"GOOS":         runtimeInfo.GOOS,
		"GOARCH":       runtimeInfo.GOARCH,
		"GOROOT":       runtimeInfo.GOROOT,
		"NumCPU":       runtimeInfo.NumCPU,
		"NumGoroutine": runtimeInfo.NumGoroutine,
	}

	this.json(ctx, info, facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// version - 版本信息
func (this *Info) version(ctx *gin.Context) {
	this.json(ctx, map[string]any{
		"go":   utils.Version.Go(),
		"inis": facade.Version,
		"text": "这是一个奇迹！",
	}, facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// device - 设备信息
func (this *Info) device(ctx *gin.Context) {
	item := facade.Comm.Device()
	if item.Error != nil {
		this.json(ctx, nil, item.Error.Error(), DefaultErrorCode)
		return
	}
	this.json(ctx, item.Json["data"], facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// time - 时间信息
func (this *Info) time(ctx *gin.Context) {
	this.json(ctx, this.getCurrentTime(), facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// renew - 更新
func (this *Info) renew(ctx *gin.Context) {
	path, err := os.Executable()
	if err != nil {
		this.json(ctx, nil, err.Error(), DefaultErrorCode)
		return
	}

	args := os.Args[1:]
	if _, err := os.StartProcess(path, append([]string{path}, args...), &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}); err != nil {
		this.json(ctx, nil, err.Error(), DefaultErrorCode)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}

// kill 杀死进程
func (this *Info) kill(ctx *gin.Context) {
	go func() {
		// 延迟执行，确保响应先返回
		os.Exit(0)
	}()

	this.json(ctx, nil, facade.Lang(ctx, defaultResponseMsg), DefaultSuccessCode)
}
