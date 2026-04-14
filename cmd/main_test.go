package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func swapInstallHelm(t *testing.T, fn func(context.Context, []string, []string, []string, map[string]interface{}) error) {
	t.Helper()
	orig := installHelm
	installHelm = fn
	t.Cleanup(func() { installHelm = orig })
}

// --- route and simple handler tests ---

func TestSetupRouter_Routes(t *testing.T) {
	t.Parallel()
	router := setupRouter()
	routes := router.Routes()

	expected := map[string]string{
		"GET /ok":         "/ok",
		"GET /health":     "/health",
		"GET /version":    "/version",
		"POST /v1/create": "/v1/create",
	}

	found := make(map[string]bool)
	for _, r := range routes {
		key := r.Method + " " + r.Path
		found[key] = true
	}

	for key := range expected {
		if !found[key] {
			t.Errorf("route %s not registered", key)
		}
	}
}

func TestHealthCheckHandler(t *testing.T) {
	t.Parallel()
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ok", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %q", body["status"])
	}
	if body["message"] != "Service is running" {
		t.Errorf("expected message 'Service is running', got %q", body["message"])
	}
}

func TestVersionHandler(t *testing.T) {
	t.Parallel()
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/version", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["version"] != Version {
		t.Errorf("expected version %q, got %q", Version, body["version"])
	}
	if body["gitCommit"] != GitCommit {
		t.Errorf("expected gitCommit %q, got %q", GitCommit, body["gitCommit"])
	}
	if body["buildTime"] != BuildTime {
		t.Errorf("expected buildTime %q, got %q", BuildTime, body["buildTime"])
	}
}

// --- parameter validation tests ---

func TestCreateHelmReleaseHandler_MissingChart(t *testing.T) {
	t.Parallel()
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/create?namespace=ns&release=rel", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["error"] != "Missing required parameter: chart" {
		t.Errorf("unexpected error: %q", body["error"])
	}
}

func TestCreateHelmReleaseHandler_MissingNamespace(t *testing.T) {
	t.Parallel()
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/create?chart=c&release=rel", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["error"] != "Missing required parameter: namespace" {
		t.Errorf("unexpected error: %q", body["error"])
	}
}

func TestCreateHelmReleaseHandler_MissingRelease(t *testing.T) {
	t.Parallel()
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/create?chart=c&namespace=ns", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["error"] != "Missing required parameter: release" {
		t.Errorf("unexpected error: %q", body["error"])
	}
}

// --- error dispatch tests (mock installHelm) ---

func TestCreateHelmReleaseHandler_Success(t *testing.T) {
	swapInstallHelm(t, func(_ context.Context, _ []string, _ []string, _ []string, _ map[string]interface{}) error {
		return nil
	})

	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/create?chart=c&namespace=ns&release=rel", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["status"] != "success" {
		t.Errorf("expected status 'success', got %q", body["status"])
	}
	if body["release"] != "rel" {
		t.Errorf("expected release 'rel', got %q", body["release"])
	}
}

func TestCreateHelmReleaseHandler_DeadlineExceeded(t *testing.T) {
	swapInstallHelm(t, func(_ context.Context, _ []string, _ []string, _ []string, _ map[string]interface{}) error {
		return context.DeadlineExceeded
	})

	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/create?chart=c&namespace=ns&release=rel", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status 504, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["error"] != "Installation timed out" {
		t.Errorf("unexpected error: %q", body["error"])
	}
}

func TestCreateHelmReleaseHandler_Canceled(t *testing.T) {
	swapInstallHelm(t, func(_ context.Context, _ []string, _ []string, _ []string, _ map[string]interface{}) error {
		return context.Canceled
	})

	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/create?chart=c&namespace=ns&release=rel", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["error"] != "Installation canceled" {
		t.Errorf("unexpected error: %q", body["error"])
	}
}

func TestCreateHelmReleaseHandler_GenericError(t *testing.T) {
	swapInstallHelm(t, func(_ context.Context, _ []string, _ []string, _ []string, _ map[string]interface{}) error {
		return errors.New("something broke")
	})

	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/create?chart=c&namespace=ns&release=rel", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["error"] != "Failed to install Helm chart" {
		t.Errorf("unexpected error: %q", body["error"])
	}
}
