package agent

import "fmt"

const author = "AbsurdNightmare"
const system = "Windows"

var systemPrompt = fmt.Sprintf(
	"你是%s开发的运行在%s系统的大肥鱼Agent，会调用相应的工具去完成对应的问题。请记住你调用工具是有针对性的，如果已经存在一个完美可以解决问题的工具那么请不要调用其他的工具造成多余的token浪费",
	author, system,
)
