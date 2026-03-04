package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/floj/scrumpoker/pkg/handler/health"
	"github.com/floj/scrumpoker/pkg/handler/rooms"
	"github.com/floj/scrumpoker/ui"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "scrumpoker",
		Usage: "A simple scrum poker app",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "bind",
				Value:   ":1323",
				Usage:   "Address to bind the server",
				Aliases: []string{"b"},
				Sources: cli.EnvVars("BIND"),
			},
			&cli.StringFlag{
				Name:    "persist-file",
				Usage:   "File to persist rooms data",
				Aliases: []string{"pf"},
				Sources: cli.EnvVars("PERSIST_FILE"),
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {

			e := echo.New()
			logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
			e.Use(middleware.Recover())
			e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
				UnsafeAllowOriginFunc: func(c *echo.Context, origin string) (allowedOrigin string, allowed bool, err error) {
					return origin, true, nil
				},
				AllowMethods: []string{http.MethodGet, http.MethodOptions, http.MethodPost, http.MethodDelete},
			}))
			e.Use(loggerMiddleware(logger))
			e.StaticFS("/", ui.StaticAssets())

			base := e.Group("/api/v1")

			healthHandler := health.NewHandler()
			healthHandler.Register(base.Group("/health"))

			roomsHandler, stop, err := rooms.NewHandler()
			if err != nil {
				return err
			}
			defer stop()
			if persistFile := c.String("persist-file"); persistFile != "" {
				if err := roomsHandler.LoadRooms(persistFile); err != nil {
					return err
				}
				defer func() {
					if err := roomsHandler.SaveRooms(persistFile); err != nil {
						slog.Error("Failed to save rooms", slog.String("file", persistFile), slog.Any("error", err))
					}
				}()
			}

			roomsHandler.Register(base.Group("/rooms"))

			sc := echo.StartConfig{
				Address:         c.String("bind"),
				GracefulTimeout: 5 * time.Second,
			}
			if err := sc.Start(ctx, e); err != nil {
				return err
			}
			return nil

		},
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
