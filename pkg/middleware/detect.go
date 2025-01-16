package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/itorix/go-fiber/pkg/config"
	"github.com/itorix/go-fiber/pkg/models"
)

type DetectMiddleware struct {
	config *config.Config
}

func NewDetectMiddleware(cfg *config.Config) *DetectMiddleware {
	return &DetectMiddleware{
		config: cfg,
	}
}

type CustomRoundTripper struct {
	ctx    context.Context
	base   http.RoundTripper
	config *config.Config
}

func (t *CustomRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a new request with the stored context
	newReq := req.Clone(t.ctx)

	tracingCtxRaw := t.ctx.Value(TracingContextKey)

	// Extract tracing context
	tracingCtx, ok := tracingCtxRaw.(*TracingContext)

	if !ok {
		return t.base.RoundTrip(newReq)
	}

	// Set tracing headers
	newReq.Header.Set(t.config.TraceIDHeader, tracingCtx.TraceID)
	newReq.Header.Set(t.config.ParentSpanIDHeader, tracingCtx.SpanID)
	newSpanID := generateID()
	newReq.Header.Set(t.config.SpanIDHeader, newSpanID)
	newReq.Header.Set(t.config.RequestTimestampHeader, strconv.FormatInt(time.Now().UnixMilli(), 10))
	return t.base.RoundTrip(newReq)
}

type RequestData struct {
	Method       string
	Path         string
	Body         string
	Hostname     string
	Protocol     string
	Request      *fiber.Request
	Response     *fiber.Response
	Referer      string
	LocalIP      string
	ResponseBody string
	StatusCode   int
	Host         string
	IP           string
	Headers      map[string]string
}

func ExtractRequestData(c *fiber.Ctx) (*RequestData, error) {
	data := &RequestData{
		Method:       c.Method(),
		Path:         c.Path(),
		Body:         string(c.Body()),
		Hostname:     c.Hostname(),
		Protocol:     c.Protocol(),
		Request:      c.Request(),
		Response:     c.Response(),
		Referer:      c.Get("Referer"),
		LocalIP:      c.Context().LocalIP().String(),
		ResponseBody: string(c.Response().Body()),
		StatusCode:   c.Response().StatusCode(),
		Host:         string(c.Request().Host()),
		IP:           c.IP(),
		Headers:      make(map[string]string),
	}

	c.Request().Header.VisitAll(func(key, val []byte) {
		data.Headers[string(key)] = string(val)
	})

	return data, nil
}

func generateID() string {
	return uuid.New().String()
}

type TracingContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
}

const (
	TracingContextKey contextKey = "tracing-context"
)

type contextKey string

func (m *DetectMiddleware) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		originalBody := c.Body()
		SetRequestContext(c)
		defer cleanupRequestContext()
		traceID := c.Get(m.config.TraceIDHeader)
		if traceID == "" {
			traceID = generateID()
		}

		tracingCtx := &TracingContext{
			TraceID:      traceID,
			SpanID:       generateID(),
			ParentSpanID: c.Get(m.config.ParentSpanIDHeader),
		}

		baseCtx := context.Background()
		ctx := context.WithValue(baseCtx, TracingContextKey, tracingCtx)

		client := &http.Client{
			Transport: &CustomRoundTripper{
				base:   http.DefaultTransport,
				ctx:    ctx,
				config: m.config,
			},
		}
		c.Locals("tracing-context", tracingCtx)
		c.Locals("http-client", client)

		// Set response headers
		c.Set(m.config.TraceIDHeader, tracingCtx.TraceID)
		c.Set(m.config.SpanIDHeader, tracingCtx.SpanID)
		c.Set(m.config.ParentSpanIDHeader, tracingCtx.ParentSpanID)

		c.Request().Header.Set(m.config.TraceIDHeader, tracingCtx.TraceID)
		c.Request().Header.Set(m.config.SpanIDHeader, tracingCtx.SpanID)
		c.Request().Header.Set(m.config.ParentSpanIDHeader, tracingCtx.ParentSpanID)
		c.Request().Header.Set(m.config.RequestTimestampHeader, strconv.FormatInt(time.Now().UnixMilli(), 10))
		err := c.Next()
		data, dataErr := ExtractRequestData(c)
		if dataErr != nil {
			return dataErr
		}

		go func() {
			m.handleComplianceCheck(data, originalBody)
		}()
		return err
	}
}

func (m *DetectMiddleware) handleComplianceCheck(c *RequestData, originalBody []byte) {

	if c == nil {
		return
	}

	checkDTO := m.buildComplianceDTO(c, originalBody)
	if checkDTO == nil {
		return
	}
	m.sendComplianceCheck(checkDTO)
}

// getPort extracts port from the host header or returns default based on scheme
func getPort(host, scheme string) int {
	if host == "" {
		return 80
	}

	// Check if port is explicitly specified
	if strings.Contains(host, ":") {
		portStr := strings.Split(host, ":")[1]
		err := host
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
		log.Printf("Error parsing port: %v\n", err)
	}

	// Default ports based on scheme
	if strings.ToLower(scheme) == "https" {
		return 443
	}
	return 80
}

func (m *DetectMiddleware) buildComplianceDTO(c *RequestData, originalBody []byte) *models.ComplianceCheckDTO {
	if c == nil {
		return nil
	}

	host := c.Host

	port := getPort(host, c.Protocol)

	checkDTO := &models.ComplianceCheckDTO{
		Request: models.Request{
			HeaderParams: make(map[string]interface{}),
			QueryParams:  make(map[string]interface{}),
			Verb:         c.Method,
			Path:         c.Path,
			Hostname:     c.Hostname,
			RequestBody:  string(originalBody),
			Scheme:       c.Protocol,
			Port:         port,
		},
		Response: models.Response{
			HeaderParams: make(map[string]interface{}),
			ResponseBody: string(c.ResponseBody),
			StatusCode:   fmt.Sprintf("%d", c.StatusCode),
		},
		ClientIP:   c.IP,
		ClientHost: c.Referer,
		ServerIP:   c.LocalIP,
		ServerHost: c.Hostname,
	}

	// Log header addition
	c.Request.Header.VisitAll(func(key, value []byte) {
		checkDTO.Request.HeaderParams[strings.ToLower(string(key))] = string(value)
	})

	headers := c.Headers
	for key, value := range headers {
		checkDTO.Request.HeaderParams[strings.ToLower(key)] = value
	}

	if m.config.EnableTracing {
		checkDTO.Request.HeaderParams[m.config.ResponseTimestampHeader] = time.Now().UnixMilli()
		checkDTO.Request.HeaderParams[m.config.GatewayTypeHeader] = "MICROSERVICES"
	}

	return checkDTO
}

// sendComplianceCheck sends the compliance check data to the configured API
func (config *DetectMiddleware) sendComplianceCheck(checkDTO *models.ComplianceCheckDTO) {
	jsonData, err := json.Marshal(checkDTO)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", config.config.DetectAPI, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client-secret", config.config.APIKey)
	req.Header.Set("x-client-id", config.config.WorkspaceID)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

}

var (
	// requestContexts stores the Fiber context for each request
	requestContexts = sync.Map{}
)

// Helper to set the context for the current goroutine
func SetRequestContext(c *fiber.Ctx) {
	goid := getGoroutineID()
	requestContexts.Store(goid, c)
}

// Helper to get the context for the current goroutine
func getRequestContext() *fiber.Ctx {
	goid := getGoroutineID()
	if ctx, ok := requestContexts.Load(goid); ok {
		return ctx.(*fiber.Ctx)
	}
	return nil
}

// Helper to clean up the context
func cleanupRequestContext() {
	goid := getGoroutineID()
	requestContexts.Delete(goid)
}

func GetClient() *http.Client {
	c := getRequestContext()
	if c == nil {
		panic("no request context found")
	}

	clientRaw := c.Locals("http-client")
	if client, ok := clientRaw.(*http.Client); ok {
		return client
	}

	// If no client found, create new one with tracing context
	tracingCtx := c.Locals("tracing-context").(*TracingContext)
	baseCtx := context.Background()
	ctx := context.WithValue(baseCtx, TracingContextKey, tracingCtx)

	return &http.Client{
		Transport: &CustomRoundTripper{
			base: http.DefaultTransport,
			ctx:  ctx,
		},
		Timeout: 10 * time.Second,
	}
}

func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
