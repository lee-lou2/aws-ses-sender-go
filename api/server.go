package api

import (
	"aws-ses-sender-go/config"
	"fmt"
	"github.com/gofiber/fiber/v3/middleware/pprof"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	jsoniter "github.com/json-iterator/go"
)

func Run() {
	app := fiber.New(
		fiber.Config{
			AppName:     "aws-ses-sender-go",
			JSONEncoder: jsoniter.Marshal,
			JSONDecoder: jsoniter.Unmarshal,
		},
	)

	// Middleware
	app.Use(pprof.New())
	app.Use(recoverer.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} ${pid} ${status} - ${method} ${path} ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	}))

	// Routes
	setV1Routes(app)

	log.Fatal(app.Listen(fmt.Sprintf(":%s", config.GetEnv("SERVER_PORT", "3000"))))
}
