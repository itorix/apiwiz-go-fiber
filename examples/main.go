package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"log"
	"net/http"
	"wizsource.apiwiz.io/backend/apiwiz-enable-detect-go-fiber/pkg/config"
	"wizsource.apiwiz.io/backend/apiwiz-enable-detect-go-fiber/pkg/middleware"
)

func main() {
	app := fiber.New()

	// Configure APIWiz Detect with logging
	cfg := &config.Config{
		APIKey:                  "XeX0u28Ya1a3CBm5tihMQFteeA6fZTy8avUIsJ0WnNOaEAM90Hcv9G2xo5z5hI4WMyHffxTAbP2LcXK4u6n5Pw==",
		WorkspaceID:             "stage-data",
		DetectAPI:               "https://dev-api.apiwiz.io/v1/apiwiz-runtime-agent/compliance/detect",
		EnableTracing:           true,
		TraceIDHeader:           "traceid",
		SpanIDHeader:            "spanid",
		ParentSpanIDHeader:      "parentspanid",
		RequestTimestampHeader:  "request-timestamp",
		ResponseTimestampHeader: "response-timestamp",
		GatewayTypeHeader:       "gateway-type",
	}

	detect := middleware.NewDetectMiddleware(cfg)
	app.Use(detect.Middleware())

	app.Get("/1", func(c *fiber.Ctx) error {
		fmt.Printf("1 Headers\n")
		c.Request().Header.VisitAll(func(key, val []byte) {
			fmt.Printf("Header - %s: %s\n", string(key), string(val))
		})

		req, err := http.NewRequest("GET", "http://localhost:3000/2", nil)
		if err != nil {
			log.Printf("Error creating request: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating request to /2")
		}

		client := middleware.GetClient()
		resp, err := client.Do(req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error calling /2")
		}
		defer resp.Body.Close()

		newreq, newerr := http.NewRequest("GET", "http://localhost:3000/3", nil)
		if newerr != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating request to /3")
		}

		newclient := middleware.GetClient()
		newresp, newerr := newclient.Do(newreq)
		if newerr != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error calling /3")
		}
		defer newresp.Body.Close()

		return c.SendString("Endpoint 1")
	})

	app.Get("/2", func(c *fiber.Ctx) error {
		fmt.Printf("2 Headers\n")
		c.Request().Header.VisitAll(func(key, val []byte) {
			fmt.Printf("Header - %s: %s\n", string(key), string(val))
		})
		newreq, newerr := http.NewRequest("GET", "http://localhost:3000/3", nil)
		if newerr != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating request to /3")
		}

		newclient := middleware.GetClient()
		newresp, newerr := newclient.Do(newreq)
		if newerr != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error calling /3")
		}
		defer newresp.Body.Close()

		return c.SendString("Endpoint 2")
	})

	app.Get("/3", func(c *fiber.Ctx) error {
		fmt.Printf("3 Headers\n")
		c.Request().Header.VisitAll(func(key, val []byte) {
			fmt.Printf("Header - %s: %s\n", string(key), string(val))
		})
		newreq, newerr := http.NewRequest("GET", "http://localhost:3000/4", nil)
		if newerr != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating request to /4")
		}

		newclient := middleware.GetClient()
		newresp, newerr := newclient.Do(newreq)
		if newerr != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error calling /4")
		}
		defer newresp.Body.Close()

		return c.SendString("Endpoint 3")
	})

	app.Get("/4", func(c *fiber.Ctx) error {
		fmt.Printf("4 Headers\n")
		c.Request().Header.VisitAll(func(key, val []byte) {
			fmt.Printf("Header - %s: %s\n", string(key), string(val))
		})

		return c.SendString("Endpoint 4")
	})

	// Define /endpoint-b route which calls /endpoint-a internally

	// Start the server
	err := app.Listen(":3000")
	if err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}
