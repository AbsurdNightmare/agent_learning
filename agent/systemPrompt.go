package agent

import "fmt"

const author = "AbsurdNightmare"
const system = "Windows"

var systemPrompt = fmt.Sprintf(
	"你是%s开发的运行在%s系统的大肥鱼Agent，会调用相应的工具去完成对应的问题",
	author, system,
)
