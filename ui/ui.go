package ui

import (
	"embed"
	"io/fs"

	"github.com/labstack/echo/v5"
)

//go:embed dist/*
var staticAssets embed.FS

func StaticAssets() fs.FS {
	return echo.MustSubFS(staticAssets, "dist")
}
