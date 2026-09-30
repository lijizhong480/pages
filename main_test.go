package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testApp(t *testing.T) *App {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &App{store: s, token: "secret"}
}
func request(method, path string, body io.Reader, token string) *http.Request {
	r := httptest.NewRequest(method, path, body)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}
func jsonRequest(method, path, body, token string) *http.Request {
	r := request(method, path, strings.NewReader(body), token)
	r.Header.Set("Content-Type", "application/json")
	return r
}
func serve(t *testing.T, h http.Handler, r *http.Request, want int) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("%s %s: want %d got %d: %s", r.Method, r.URL.Path, want, w.Code, w.Body.String())
	}
	return w
}
func createProject(t *testing.T, h http.Handler, workspace, project string) {
	t.Helper()
	serve(t, h, jsonRequest("POST", "/api/workspaces/"+workspace+"/projects", `{"slug":"`+project+`","name":"Test project"}`, "secret"), 201)
}

func TestWorkspaceProjectPageLifecycle(t *testing.T) {
	a := testApp(t)
	h := a.routes()
	serve(t, h, jsonRequest("POST", "/api/workspaces", `{"slug":"team","name":"Team"}`, "secret"), 201)
	createProject(t, h, "team", "site")
	w := serve(t, h, jsonRequest("POST", "/api/workspaces/team/projects/site/pages", `{"slug":"hello","title":"Hello","html":"<!doctype html><h1>Hello</h1>"}`, "secret"), 201)
	if !strings.Contains(w.Body.String(), `"url":"/p/team/site/hello"`) {
		t.Fatal(w.Body.String())
	}
	var first Page
	if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	w = serve(t, h, request("GET", "/api/workspaces/team/projects/site/pages", nil, "secret"), 200)
	if !strings.Contains(w.Body.String(), `"slug":"hello"`) {
		t.Fatal(w.Body.String())
	}
	serve(t, h, request("GET", "/p/team/site/hello", nil, ""), 404)
	w = serve(t, h, request("GET", first.LatestPreviewURL, nil, ""), 200)
	if !strings.Contains(w.Body.String(), "<h1>Hello</h1>") {
		t.Fatal(w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Fatal("missing sandbox CSP")
	}
	serve(t, h, jsonRequest("POST", "/api/workspaces/team/projects/site/pages/hello/publish", `{"version":"`+first.LatestVersion+`"}`, "secret"), 200)
	w = serve(t, h, request("GET", "/p/team/site/hello", nil, ""), 200)
	if !strings.Contains(w.Body.String(), "<h1>Hello</h1>") {
		t.Fatal(w.Body.String())
	}
	secondResponse := serve(t, h, jsonRequest("POST", "/api/workspaces/team/projects/site/pages", `{"slug":"hello","title":"Hello 2","html":"<h1>Version 2</h1>"}`, "secret"), 201)
	var second Page
	_ = json.Unmarshal(secondResponse.Body.Bytes(), &second)
	w = serve(t, h, request("GET", "/p/team/site/hello", nil, ""), 200)
	if strings.Contains(w.Body.String(), "Version 2") {
		t.Fatal("draft leaked to production")
	}
	serve(t, h, jsonRequest("POST", "/api/workspaces/team/projects/site/pages/hello/publish", `{"version":"`+second.LatestVersion+`"}`, "secret"), 200)
	w = serve(t, h, request("GET", "/p/team/site/hello", nil, ""), 200)
	if !strings.Contains(w.Body.String(), "Version 2") {
		t.Fatal(w.Body.String())
	}
	serve(t, h, jsonRequest("POST", "/api/workspaces/team/projects/site/pages/hello/publish", `{"version":"`+first.LatestVersion+`"}`, "secret"), 200)
	w = serve(t, h, request("GET", "/p/team/site/hello", nil, ""), 200)
	if !strings.Contains(w.Body.String(), "<h1>Hello</h1>") {
		t.Fatal("version switch failed")
	}
	serve(t, h, request("DELETE", "/api/workspaces/team/projects/site/pages/hello", nil, "secret"), 204)
	w = serve(t, h, jsonRequest("PATCH", "/api/workspaces/team", `{"name":"Renamed Team"}`, "secret"), 200)
	if !strings.Contains(w.Body.String(), `"name":"Renamed Team"`) {
		t.Fatal(w.Body.String())
	}
	serve(t, h, request("DELETE", "/api/workspaces/team", nil, "secret"), 204)
	serve(t, h, request("GET", "/api/workspaces/team/projects", nil, "secret"), 404)
}

func TestWorkspaceTokenScopeAndIsolation(t *testing.T) {
	a := testApp(t)
	h := a.routes()
	for _, workspace := range []string{"team-a", "team-b"} {
		serve(t, h, jsonRequest("POST", "/api/workspaces", `{"slug":"`+workspace+`","name":"Team"}`, "secret"), 201)
		createProject(t, h, workspace, "site")
	}
	w := serve(t, h, jsonRequest("POST", "/api/workspaces/team-a/tokens", `{"name":"AI agent","scopes":["read","write"]}`, "secret"), 201)
	var result struct {
		Secret string `json:"secret"`
		Token  Token  `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	serve(t, h, jsonRequest("POST", "/api/workspaces/team-a/projects/site/pages", `{"slug":"agent-page","html":"<p>AI</p>"}`, result.Secret), 201)
	serve(t, h, request("GET", "/api/workspaces/team-b/projects", nil, result.Secret), 403)
	serve(t, h, jsonRequest("POST", "/api/workspaces", `{"slug":"forbidden","name":"No"}`, result.Secret), 403)
	reset := serve(t, h, request("POST", "/api/workspaces/team-a/tokens/"+result.Token.ID+"/reset", nil, "secret"), 200)
	var rotated struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(reset.Body.Bytes(), &rotated); err != nil {
		t.Fatal(err)
	}
	serve(t, h, request("GET", "/api/workspaces/team-a/projects", nil, result.Secret), 401)
	serve(t, h, request("GET", "/api/workspaces/team-a/projects", nil, rotated.Secret), 200)
	stateBytes, err := json.Marshal(a.store.st)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stateBytes, []byte(result.Secret)) {
		t.Fatal("plaintext token persisted")
	}
	if bytes.Contains(stateBytes, []byte(rotated.Secret)) {
		t.Fatal("rotated plaintext token persisted")
	}
}

func TestMultipartRawAndValidation(t *testing.T) {
	a := testApp(t)
	h := a.routes()
	createProject(t, h, "default", "site")
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("slug", "multipart-page")
	file, _ := mw.CreateFormFile("file", "demo.html")
	_, _ = file.Write([]byte("<p>multipart</p>"))
	_ = mw.Close()
	r := request("POST", "/api/workspaces/default/projects/site/pages", &body, "secret")
	r.Header.Set("Content-Type", mw.FormDataContentType())
	serve(t, h, r, 201)
	r = request("POST", "/api/workspaces/default/projects/site/pages?slug=raw-page&title=Raw", strings.NewReader("<p>raw</p>"), "secret")
	r.Header.Set("Content-Type", "text/html")
	serve(t, h, r, 201)
	serve(t, h, request("GET", "/api/workspaces", nil, ""), 401)
	serve(t, h, jsonRequest("POST", "/api/workspaces/default/projects/site/pages", `{"slug":"../bad","html":"x"}`, "secret"), 400)
}

func TestLegacyMigrationAndRoute(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "pages"), 0750); err != nil {
		t.Fatal(err)
	}
	legacy := `{"old":{"slug":"old","title":"Old","size":12,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","url":"/p/old"}}`
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(legacy), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pages", "old.html"), []byte("<h1>old</h1>"), 0640); err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := (&App{store: s, token: "secret"}).routes()
	serve(t, h, request("GET", "/p/old", nil, ""), 200)
	serve(t, h, request("GET", "/p/default/legacy/old", nil, ""), 200)
}

func TestWorkspaceAndProjectOrdering(t *testing.T) {
	a := testApp(t)
	h := a.routes()
	serve(t, h, jsonRequest("POST", "/api/workspaces", `{"slug":"alpha","name":"Alpha"}`, "secret"), 201)
	serve(t, h, jsonRequest("POST", "/api/workspaces", `{"slug":"beta","name":"Beta"}`, "secret"), 201)
	serve(t, h, jsonRequest("PUT", "/api/workspaces/order", `{"slugs":["beta","default","alpha"]}`, "secret"), 204)
	w := serve(t, h, request("GET", "/api/workspaces", nil, "secret"), 200)
	body := w.Body.String()
	if !(strings.Index(body, `"slug":"beta"`) < strings.Index(body, `"slug":"default"`) && strings.Index(body, `"slug":"default"`) < strings.Index(body, `"slug":"alpha"`)) {
		t.Fatal(body)
	}
	createProject(t, h, "beta", "one")
	createProject(t, h, "beta", "two")
	serve(t, h, jsonRequest("PUT", "/api/workspaces/beta/projects/order", `{"slugs":["two","one"]}`, "secret"), 204)
	w = serve(t, h, request("GET", "/api/workspaces/beta/projects", nil, "secret"), 200)
	body = w.Body.String()
	if strings.Index(body, `"slug":"two"`) > strings.Index(body, `"slug":"one"`) {
		t.Fatal(body)
	}
}

func TestUserLoginRolesAndWorkspaceAuthorization(t *testing.T) {
	a := testApp(t)
	h := a.routes()
	setup := serve(t, h, jsonRequest("POST", "/api/auth/setup", `{"username":"admin","name":"管理员","password":"strong-pass-123"}`, ""), 201)
	var auth struct {
		CSRF string `json:"csrfToken"`
	}
	if err := json.Unmarshal(setup.Body.Bytes(), &auth); err != nil {
		t.Fatal(err)
	}
	cookies := setup.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("missing session cookie")
	}
	adminCookie := cookies[0]
	sessionJSON := func(method, path, body string, cookie *http.Cookie, csrf string) *http.Request {
		r := jsonRequest(method, path, body, "")
		r.AddCookie(cookie)
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		return r
	}
	serve(t, h, sessionJSON("POST", "/api/workspaces", `{"slug":"team","name":"Team"}`, adminCookie, auth.CSRF), 201)
	serve(t, h, sessionJSON("POST", "/api/workspaces/team/projects", `{"slug":"site","name":"Site"}`, adminCookie, auth.CSRF), 201)
	created := serve(t, h, sessionJSON("POST", "/api/users", `{"username":"writer","name":"Writer","role":"user","password":"writer-pass-123"}`, adminCookie, auth.CSRF), 201)
	var user UserView
	if err := json.Unmarshal(created.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	serve(t, h, sessionJSON("PUT", "/api/workspaces/team/members/"+user.ID, `{"role":"viewer"}`, adminCookie, auth.CSRF), 204)
	login := serve(t, h, jsonRequest("POST", "/api/auth/login", `{"username":"writer","password":"writer-pass-123"}`, ""), 200)
	var memberAuth struct {
		CSRF string `json:"csrfToken"`
	}
	_ = json.Unmarshal(login.Body.Bytes(), &memberAuth)
	memberCookie := login.Result().Cookies()[0]
	serve(t, h, sessionJSON("POST", "/api/workspaces/team/projects/site/pages", `{"slug":"no","html":"x"}`, memberCookie, memberAuth.CSRF), 403)
	serve(t, h, sessionJSON("DELETE", "/api/workspaces/team", ``, memberCookie, memberAuth.CSRF), 403)
	serve(t, h, sessionJSON("POST", "/api/workspaces", `{"slug":"writer-space","name":"Writer Space"}`, memberCookie, memberAuth.CSRF), 201)
	owned := serve(t, h, sessionJSON("GET", "/api/workspaces", ``, memberCookie, memberAuth.CSRF), 200)
	if !strings.Contains(owned.Body.String(), `"slug":"writer-space"`) || !strings.Contains(owned.Body.String(), `"currentRole":"owner"`) {
		t.Fatal(owned.Body.String())
	}
	serve(t, h, sessionJSON("POST", "/api/workspaces/writer-space/projects", `{"slug":"personal-site","name":"Personal Site"}`, memberCookie, memberAuth.CSRF), 201)
	serve(t, h, sessionJSON("POST", "/api/workspaces/writer-space/projects/personal-site/pages", `{"slug":"home","html":"<p>mine</p>"}`, memberCookie, memberAuth.CSRF), 201)
	serve(t, h, sessionJSON("PATCH", "/api/workspaces/writer-space", `{"name":"My Workspace"}`, memberCookie, memberAuth.CSRF), 200)
	serve(t, h, sessionJSON("DELETE", "/api/workspaces/writer-space", ``, memberCookie, memberAuth.CSRF), 204)
	serve(t, h, sessionJSON("PUT", "/api/workspaces/team/members/"+user.ID, `{"role":"editor"}`, adminCookie, auth.CSRF), 204)
	serve(t, h, sessionJSON("POST", "/api/workspaces/team/projects/site/pages", `{"slug":"yes","html":"<p>yes</p>"}`, memberCookie, memberAuth.CSRF), 201)
	serve(t, h, sessionJSON("POST", "/api/workspaces/team/projects/site/pages", `{"slug":"csrf","html":"x"}`, memberCookie, ""), 403)
}
