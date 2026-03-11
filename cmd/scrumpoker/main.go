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

var flagBind = &cli.IntFlag{
	Name:    "port",
	Value:   1323,
	Usage:   "Port to bind the server",
	Aliases: []string{"p"},
	Sources: cli.EnvVars("LISTEN_PORT"),
}

func main() {
	cmd := &cli.Command{
		Name:  "scrumpoker",
		Usage: "A simple scrum poker app",
		Flags: []cli.Flag{
			flagBind,
			&cli.StringFlag{
				Name:    "persist-file",
				Usage:   "File to persist rooms data",
				Aliases: []string{"pf"},
				Sources: cli.EnvVars("PERSIST_FILE"),
			},
			&cli.IntFlag{
				Name:    "max-rooms",
				Usage:   "Maximum number of rooms that can be created (0 for unlimited)",
				Aliases: []string{"mr"},
				Sources: cli.EnvVars("MAX_ROOMS"),
			},
		},
		Commands: []*cli.Command{{
			Name:  "healthcheck",
			Usage: "Run a health check against the server",
			Action: func(ctx context.Context, c *cli.Command) error {
				ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v1/health", c.Int(flagBind.Name)), nil)
				if err != nil {
					return err
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				}
				fmt.Println("OK")
				return nil
			},
		}},
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
			e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
				Filesystem: ui.StaticAssets(),
				HTML5:      true,
			}))

			base := e.Group("/api/v1")

			healthHandler := health.NewHandler()
			healthHandler.Register(base.Group("/health"))

			roomsHandler, err := rooms.NewHandler(ctx, c.Int("max-rooms"))
			if err != nil {
				return err
			}
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
				Address:         fmt.Sprintf(":%d", c.Int(flagBind.Name)),
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
