package timer

import "inis/app/facade"

type LogStruct struct {}

var Log *LogStruct

func (this *LogStruct) Run() {

	// 每天凌晨 00:00:00 执行
	_ = Timer.Every(1).Day().At("00:00:00").Do(facade.InitLog)
}