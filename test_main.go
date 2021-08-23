package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetEntries(t *testing.T) {
	str := "P Ford code\nQ Ford"
	r := strings.NewReader(str)

	req, err := http.NewRequest("POST", "/input", r)

	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(start_process)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `Q1 := P1`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
