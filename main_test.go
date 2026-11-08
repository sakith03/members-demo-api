package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func request(t *testing.T, handler http.Handler, method, path, body string, status int) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != status {
		t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, status, w.Body.String())
	}
	return w
}

func memberFrom(t *testing.T, w *httptest.ResponseRecorder) Member {
	t.Helper()
	var body struct {
		Data Member `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Data
}

func TestMemberLifecycle(t *testing.T) {
	handler := newAPI()
	request(t, handler, "GET", "/healthz", "", 200)
	initial := request(t, handler, "GET", "/members", "", 200)
	if !strings.Contains(initial.Body.String(), `"count":2`) {
		t.Fatal("expected two seed members")
	}
	created := memberFrom(t, request(t, handler, "POST", "/members", `{"name":"  Charlie Fernando  ","email":"charlie@example.com"}`, 201))
	if created.ID != 3 || created.Name != "Charlie Fernando" || created.Role != "member" {
		t.Fatalf("unexpected created member: %+v", created)
	}
	path := fmt.Sprintf("/members/%d", created.ID)
	if got := memberFrom(t, request(t, handler, "GET", path, "", 200)); got != created {
		t.Fatalf("read differs: %+v", got)
	}
	updated := memberFrom(t, request(t, handler, "PUT", path, `{"name":"Charlie Updated","email":"charlie.updated@example.com","role":"admin"}`, 200))
	if updated.ID != created.ID || updated.Name != "Charlie Updated" || updated.Role != "admin" {
		t.Fatalf("unexpected update: %+v", updated)
	}
	if got := memberFrom(t, request(t, handler, "GET", path, "", 200)); got != updated {
		t.Fatalf("update was not stored: %+v", got)
	}
	listed := request(t, handler, "GET", "/members", "", 200)
	var body struct {
		Data  []Member `json:"data"`
		Count int      `json:"count"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Count != 3 || len(body.Data) != 3 || body.Data[2] != updated {
		t.Fatalf("unexpected list: %+v", body)
	}
	deleted := request(t, handler, "DELETE", path, "", 204)
	if deleted.Body.Len() != 0 {
		t.Fatal("delete must return an empty body")
	}
	request(t, handler, "GET", path, "", 404)
	request(t, handler, "PUT", path, `{"name":"Missing","email":"missing@example.com"}`, 404)
	request(t, handler, "DELETE", path, "", 404)
	// IDs never get reused within a process, including after deletion.
	next := memberFrom(t, request(t, handler, "POST", "/members", `{"name":"Next","email":"next@example.com"}`, 201))
	if next.ID <= created.ID {
		t.Fatal("deleted ID was reused")
	}
}

func TestRejectsInvalidRequestsWithoutChangingMembers(t *testing.T) {
	handler := newAPI()
	for _, body := range []string{
		`{`, `null`, `[]`, `{}`, `{"name":"  ","email":"a@example.com"}`,
		`{"name":"Test","email":"invalid"}`,
		`{"name":"Test","email":"Test <test@example.com>"}`,
		`{"name":"Test","email":"a@example.com","unexpected":true}`,
		`{"name":"Test","email":"a@example.com"} {}`,
	} {
		request(t, handler, "POST", "/members", body, 400)
	}
	request(t, handler, "POST", "/members", `{"name":"`+strings.Repeat("x", 65536)+`","email":"a@example.com"}`, 413)
	request(t, handler, "GET", "/members/abc", "", 400)
	request(t, handler, "GET", "/members/0", "", 400)
	request(t, handler, "PATCH", "/members/1", `{}`, 405)
	request(t, handler, "GET", "/unknown", "", 404)
	r := httptest.NewRequest("POST", "/members", strings.NewReader(`{"name":"Test","email":"a@example.com"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatalf("missing content type: got %d", w.Code)
	}
	listed := request(t, handler, "GET", "/members", "", 200)
	if !strings.Contains(listed.Body.String(), `"count":2`) {
		t.Fatal("invalid requests changed the member list")
	}
}

func TestConcurrentCreatesHaveUniqueIDs(t *testing.T) {
	handler := newAPI()
	var group sync.WaitGroup
	for i := 0; i < 30; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			request(t, handler, "POST", "/members", `{"name":"Concurrent","email":"demo@example.com"}`, 201)
			request(t, handler, "GET", "/members", "", 200)
		}()
	}
	group.Wait()
	w := request(t, handler, "GET", "/members", "", 200)
	var body struct {
		Data  []Member `json:"data"`
		Count int      `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Count != 32 || len(body.Data) != 32 {
		t.Fatalf("lost members: %+v", body)
	}
	for i, member := range body.Data {
		if member.ID != i+1 {
			t.Fatalf("unexpected or duplicate ID: %d", member.ID)
		}
	}
}
