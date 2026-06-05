package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/satyabodharao/pack-calculator/internal/repository"
	"github.com/satyabodharao/pack-calculator/internal/service"
)

func newTestHandler() *Handler {
	svc := service.New(repository.NewMemoryRepository(), nil)
	return NewHandler(svc, nil)
}

func TestHandler_GetPackSizes(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
	rec := httptest.NewRecorder()

	h.PackSizes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp packSizesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.PackSizes) != 5 {
		t.Errorf("expected 5 default pack sizes, got %v", resp.PackSizes)
	}
}

func TestHandler_PutPackSizes(t *testing.T) {
	h := newTestHandler()
	body := `{"pack_sizes":[100,200,300]}`
	req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.PackSizes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp packSizesResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.PackSizes) != 3 || resp.PackSizes[0] != 100 {
		t.Errorf("unexpected pack sizes: %v", resp.PackSizes)
	}
}

func TestHandler_PutPackSizes_Invalid(t *testing.T) {
	h := newTestHandler()
	body := `{"pack_sizes":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.PackSizes(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandler_PackSizes_MethodNotAllowed(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodDelete, "/api/pack-sizes", nil)
	rec := httptest.NewRecorder()

	h.PackSizes(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHandler_Calculate(t *testing.T) {
	h := newTestHandler()
	payload, _ := json.Marshal(calculateRequest{Items: 12001})
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	h.Calculate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp calculateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.TotalItems != 12250 || resp.TotalPacks != 4 {
		t.Errorf("got items=%d packs=%d, want 12250/4", resp.TotalItems, resp.TotalPacks)
	}
}

func TestHandler_Calculate_InvalidOrder(t *testing.T) {
	h := newTestHandler()
	body := `{"items":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Calculate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandler_Calculate_BadJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	h.Calculate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandler_Calculate_MethodNotAllowed(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/calculate", nil)
	rec := httptest.NewRecorder()

	h.Calculate(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHandler_Health(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
