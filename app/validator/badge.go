package validator

type Badge struct{}

var BadgeMessage = map[string]string{}

func (this Badge) Message() map[string]string {
	return BadgeMessage
}

func (this Badge) Struct() any {
	return this
}
