package migration

import (
	"embed"
)

//go:embed assets/*
var baseMigrationFS embed.FS
