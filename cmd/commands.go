package main

type Command int

const (
	Archon Command = iota
	Patcher
	GenerateServerCertificates
	RunPacketAnalyzer
	RunAccountTool
)

func (c Command) String() string {
	switch c {
	case Archon:
		return "server"
	case Patcher:
		return "patcher"
	case GenerateServerCertificates:
		return "certgen"
	case RunPacketAnalyzer:
		return "analyzer"
	case RunAccountTool:
		return "account"
	default:
		return ""
	}
}
