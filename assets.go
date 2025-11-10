package pika

import (
	"embed"
	"io/fs"
)

//go:embed bin/agents/*
var agentsFS embed.FS

func AgentFS() fs.FS {
	sub, _ := fs.Sub(agentsFS, "bin/agents")
	return sub
}
