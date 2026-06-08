package validator

type IpWhite struct{}

var IpWhiteMessage = map[string]string{}

func (this IpWhite) Message() map[string]string {
	return IpWhiteMessage
}

func (this IpWhite) Struct() any {
	return this
}
