package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed web/*
var webFS embed.FS

const maxHTMLSize = 5 << 20

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,40}$`)

type Page struct {
	Slug             string        `json:"slug"`
	Title            string        `json:"title"`
	Size             int64         `json:"size"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	PublishedAt      *time.Time    `json:"publishedAt,omitempty"`
	URL              string        `json:"url"`
	LatestVersion    string        `json:"latestVersion,omitempty"`
	PublishedVersion string        `json:"publishedVersion,omitempty"`
	LatestPreviewURL string        `json:"latestPreviewUrl,omitempty"`
	VersionCount     int           `json:"versionCount"`
	Versions         []PageVersion `json:"versions,omitempty"`
}

type PageVersion struct {
	ID         string    `json:"id"`
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"createdAt"`
	PreviewURL string    `json:"previewUrl"`
}

type Project struct {
	Slug        string          `json:"slug"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	Pages       map[string]Page `json:"pages"`
	Order       int             `json:"order"`
}

type Workspace struct {
	Slug         string                `json:"slug"`
	Name         string                `json:"name"`
	CreatedAt    time.Time             `json:"createdAt"`
	UpdatedAt    time.Time             `json:"updatedAt"`
	Projects     map[string]*Project   `json:"projects"`
	Members      map[string]Membership `json:"members,omitempty"`
	CurrentRole  string                `json:"currentRole,omitempty"`
	ProjectCount int                   `json:"projectCount,omitempty"`
	Order        int                   `json:"order"`
}

type Membership struct {
	UserID    string    `json:"userId"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Name         string     `json:"name"`
	Email        string     `json:"email,omitempty"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	PasswordSalt string     `json:"passwordSalt"`
	PasswordHash string     `json:"passwordHash"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
}

type Session struct {
	UserID    string    `json:"userId"`
	CSRF      string    `json:"csrf"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type UserView struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Name        string     `json:"name"`
	Email       string     `json:"email,omitempty"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
}

type Token struct {
	ID         string     `json:"id"`
	Workspace  string     `json:"workspace"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Hash       string     `json:"hash,omitempty"`
	Scopes     []string   `json:"scopes"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

type state struct {
	Version    int                   `json:"version"`
	Workspaces map[string]*Workspace `json:"workspaces"`
	Tokens     map[string]*Token     `json:"tokens"`
	Users      map[string]*User      `json:"users"`
	Sessions   map[string]*Session   `json:"sessions"`
}

type Store struct {
	dir string
	mu  sync.RWMutex
	st  state
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "workspaces"), 0750); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, st: state{Version: 4, Workspaces: map[string]*Workspace{}, Tokens: map[string]*Token{}, Users: map[string]*User{}, Sessions: map[string]*Session{}}}
	b, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err == nil && len(b) > 0 {
		if err := json.Unmarshal(b, &s.st); err != nil {
			return nil, fmt.Errorf("read state: %w", err)
		}
		if s.st.Workspaces == nil {
			s.st.Workspaces = map[string]*Workspace{}
		}
		if s.st.Tokens == nil {
			s.st.Tokens = map[string]*Token{}
		}
		if s.st.Users == nil {
			s.st.Users = map[string]*User{}
		}
		if s.st.Sessions == nil {
			s.st.Sessions = map[string]*Session{}
		}
		for _, workspace := range s.st.Workspaces {
			if workspace.Members == nil {
				workspace.Members = map[string]Membership{}
			}
		}
		if err := s.migrateVersions(); err != nil {
			return nil, err
		}
		if err := s.normalizeOrdering(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := s.migrateLegacy(); err != nil {
		return nil, err
	}
	if err := s.migrateVersions(); err != nil {
		return nil, err
	}
	if err := s.normalizeOrdering(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) normalizeOrdering() error {
	changed := s.st.Version < 4
	workspaces := make([]*Workspace, 0, len(s.st.Workspaces))
	for _, w := range s.st.Workspaces {
		workspaces = append(workspaces, w)
	}
	sort.Slice(workspaces, func(i, j int) bool {
		if workspaces[i].CreatedAt.Equal(workspaces[j].CreatedAt) {
			return workspaces[i].Name < workspaces[j].Name
		}
		return workspaces[i].CreatedAt.Before(workspaces[j].CreatedAt)
	})
	for i, w := range workspaces {
		if w.Order == 0 {
			w.Order = i + 1
			changed = true
		}
		projects := make([]*Project, 0, len(w.Projects))
		for _, p := range w.Projects {
			projects = append(projects, p)
		}
		sort.Slice(projects, func(i, j int) bool {
			if projects[i].CreatedAt.Equal(projects[j].CreatedAt) {
				return projects[i].Name < projects[j].Name
			}
			return projects[i].CreatedAt.Before(projects[j].CreatedAt)
		})
		for j, p := range projects {
			if p.Order == 0 {
				p.Order = j + 1
				changed = true
			}
		}
	}
	if changed {
		s.st.Version = 4
		return s.saveLocked()
	}
	return nil
}

func (s *Store) migrateVersions() error {
	changed := s.st.Version < 3
	for workspaceSlug, workspace := range s.st.Workspaces {
		for projectSlug, project := range workspace.Projects {
			for slug, page := range project.Pages {
				if len(page.Versions) > 0 {
					continue
				}
				id := "ver_" + randomHex(8)
				created := page.UpdatedAt
				if created.IsZero() {
					created = time.Now().UTC()
				}
				version := PageVersion{ID: id, Number: 1, Title: page.Title, Size: page.Size, CreatedAt: created, PreviewURL: previewURL(workspaceSlug, projectSlug, slug, id)}
				src := s.pagePath(workspaceSlug, projectSlug, slug)
				dst := s.versionPath(workspaceSlug, projectSlug, slug, id)
				if body, err := os.ReadFile(src); err == nil {
					if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
						return err
					}
					if err := os.WriteFile(dst, body, 0640); err != nil {
						return err
					}
				}
				page.Versions = []PageVersion{version}
				page.LatestVersion = id
				page.PublishedVersion = id
				page.LatestPreviewURL = version.PreviewURL
				page.VersionCount = 1
				published := created
				page.PublishedAt = &published
				project.Pages[slug] = page
				changed = true
			}
		}
	}
	if changed {
		s.st.Version = 3
		return s.saveLocked()
	}
	return nil
}

func (s *Store) migrateLegacy() error {
	now := time.Now().UTC()
	w := &Workspace{Slug: "default", Name: "默认工作空间", CreatedAt: now, UpdatedAt: now, Projects: map[string]*Project{}, Members: map[string]Membership{}}
	p := &Project{Slug: "legacy", Name: "历史页面", Description: "从旧版 Page Service 自动迁移", CreatedAt: now, UpdatedAt: now, Pages: map[string]Page{}}
	legacy := map[string]Page{}
	b, err := os.ReadFile(filepath.Join(s.dir, "index.json"))
	if err == nil && len(b) > 0 {
		if err := json.Unmarshal(b, &legacy); err != nil {
			return fmt.Errorf("read legacy index: %w", err)
		}
	}
	for slug, page := range legacy {
		page.URL = pageURL("default", "legacy", slug)
		p.Pages[slug] = page
		src := filepath.Join(s.dir, "pages", slug+".html")
		dst := s.pagePath("default", "legacy", slug)
		if body, readErr := os.ReadFile(src); readErr == nil {
			if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
				return err
			}
			if err := os.WriteFile(dst, body, 0640); err != nil {
				return err
			}
		}
	}
	if len(p.Pages) > 0 {
		w.Projects[p.Slug] = p
	}
	s.st.Workspaces[w.Slug] = w
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(s.st, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, "state.json.tmp")
	if err := os.WriteFile(tmp, b, 0640); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, "state.json"))
}

func (s *Store) pagePath(workspace, project, slug string) string {
	return filepath.Join(s.dir, "workspaces", workspace, "projects", project, "pages", slug+".html")
}

func (s *Store) versionPath(workspace, project, slug, version string) string {
	return filepath.Join(s.dir, "workspaces", workspace, "projects", project, "pages", slug, "versions", version+".html")
}

func (s *Store) ListWorkspaces(only string) []Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Workspace{}
	for slug, w := range s.st.Workspaces {
		if only != "" && slug != only {
			continue
		}
		copy := *w
		copy.ProjectCount = len(w.Projects)
		copy.Projects = nil
		copy.Members = nil
		out = append(out, copy)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].Name < out[j].Name
		}
		return out[i].Order < out[j].Order
	})
	return out
}

func (s *Store) ListWorkspacesForUser(userID string, platform bool) []Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Workspace{}
	for _, w := range s.st.Workspaces {
		if !platform {
			member, ok := w.Members[userID]
			if !ok {
				continue
			}
			copy := *w
			copy.ProjectCount = len(w.Projects)
			copy.Projects = nil
			copy.Members = nil
			copy.CurrentRole = member.Role
			out = append(out, copy)
			continue
		}
		copy := *w
		copy.ProjectCount = len(w.Projects)
		copy.Projects = nil
		copy.Members = nil
		copy.CurrentRole = "platform_admin"
		out = append(out, copy)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].Name < out[j].Name
		}
		return out[i].Order < out[j].Order
	})
	return out
}

func (s *Store) CreateWorkspace(slug, name string) (Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.st.Workspaces[slug]; ok {
		return Workspace{}, errors.New("workspace already exists")
	}
	now := time.Now().UTC()
	maxOrder := 0
	for _, existing := range s.st.Workspaces {
		if existing.Order > maxOrder {
			maxOrder = existing.Order
		}
	}
	w := &Workspace{Slug: slug, Name: name, CreatedAt: now, UpdatedAt: now, Projects: map[string]*Project{}, Members: map[string]Membership{}, Order: maxOrder + 1}
	s.st.Workspaces[slug] = w
	if err := s.saveLocked(); err != nil {
		delete(s.st.Workspaces, slug)
		return Workspace{}, err
	}
	return *w, nil
}

func (s *Store) DeleteWorkspace(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.st.Workspaces[slug]; !ok {
		return os.ErrNotExist
	}
	delete(s.st.Workspaces, slug)
	for id, token := range s.st.Tokens {
		if token.Workspace == slug {
			delete(s.st.Tokens, id)
		}
	}
	if err := s.saveLocked(); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.dir, "workspaces", slug))
}

func (s *Store) UpdateWorkspace(slug, name string) (Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.st.Workspaces[slug]
	if !ok {
		return Workspace{}, os.ErrNotExist
	}
	w.Name = name
	w.UpdatedAt = time.Now().UTC()
	if err := s.saveLocked(); err != nil {
		return Workspace{}, err
	}
	copy := *w
	copy.ProjectCount = len(w.Projects)
	copy.Projects = nil
	copy.Members = nil
	return copy, nil
}

func (s *Store) ReorderWorkspaces(slugs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(slugs) != len(s.st.Workspaces) {
		return errors.New("workspace order must include every workspace")
	}
	seen := map[string]bool{}
	for index, slug := range slugs {
		w, ok := s.st.Workspaces[slug]
		if !ok || seen[slug] {
			return errors.New("invalid workspace order")
		}
		seen[slug] = true
		w.Order = index + 1
	}
	return s.saveLocked()
}

func (s *Store) ListProjects(workspace string) ([]Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return nil, os.ErrNotExist
	}
	out := make([]Project, 0, len(w.Projects))
	for _, p := range w.Projects {
		copy := *p
		copy.Pages = nil
		out = append(out, copy)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].Name < out[j].Name
		}
		return out[i].Order < out[j].Order
	})
	return out, nil
}

func (s *Store) CreateProject(workspace, slug, name, description string) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return Project{}, os.ErrNotExist
	}
	if _, ok := w.Projects[slug]; ok {
		return Project{}, errors.New("project already exists")
	}
	now := time.Now().UTC()
	maxOrder := 0
	for _, existing := range w.Projects {
		if existing.Order > maxOrder {
			maxOrder = existing.Order
		}
	}
	p := &Project{Slug: slug, Name: name, Description: description, CreatedAt: now, UpdatedAt: now, Pages: map[string]Page{}, Order: maxOrder + 1}
	w.Projects[slug], w.UpdatedAt = p, now
	if err := s.saveLocked(); err != nil {
		delete(w.Projects, slug)
		return Project{}, err
	}
	return *p, nil
}

func (s *Store) DeleteProject(workspace, project string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return os.ErrNotExist
	}
	if _, ok := w.Projects[project]; !ok {
		return os.ErrNotExist
	}
	delete(w.Projects, project)
	w.UpdatedAt = time.Now().UTC()
	if err := s.saveLocked(); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.dir, "workspaces", workspace, "projects", project))
}

func (s *Store) ReorderProjects(workspace string, slugs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return os.ErrNotExist
	}
	if len(slugs) != len(w.Projects) {
		return errors.New("project order must include every project")
	}
	seen := map[string]bool{}
	for index, slug := range slugs {
		p, ok := w.Projects[slug]
		if !ok || seen[slug] {
			return errors.New("invalid project order")
		}
		seen[slug] = true
		p.Order = index + 1
	}
	w.UpdatedAt = time.Now().UTC()
	return s.saveLocked()
}

func (s *Store) Put(workspace, project, slug, title string, html []byte) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return Page{}, os.ErrNotExist
	}
	p, ok := w.Projects[project]
	if !ok {
		return Page{}, os.ErrNotExist
	}
	now := time.Now().UTC()
	page, exists := p.Pages[slug]
	if !exists {
		page = Page{Slug: slug, CreatedAt: now, URL: pageURL(workspace, project, slug), Versions: []PageVersion{}}
	}
	versionID := "ver_" + randomHex(8)
	version := PageVersion{ID: versionID, Number: len(page.Versions) + 1, Title: title, Size: int64(len(html)), CreatedAt: now, PreviewURL: previewURL(workspace, project, slug, versionID)}
	page.Title, page.Size, page.UpdatedAt, page.LatestVersion, page.LatestPreviewURL = title, int64(len(html)), now, versionID, version.PreviewURL
	page.Versions = append(page.Versions, version)
	page.VersionCount = len(page.Versions)
	path := s.versionPath(workspace, project, slug, versionID)
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return Page{}, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, html, 0640); err != nil {
		return Page{}, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return Page{}, err
	}
	p.Pages[slug], p.UpdatedAt, w.UpdatedAt = page, now, now
	if err := s.saveLocked(); err != nil {
		return Page{}, err
	}
	return page, nil
}

func (s *Store) ListPages(workspace, project string) ([]Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return nil, os.ErrNotExist
	}
	p, ok := w.Projects[project]
	if !ok {
		return nil, os.ErrNotExist
	}
	out := make([]Page, 0, len(p.Pages))
	for _, page := range p.Pages {
		page.VersionCount = len(page.Versions)
		page.Versions = nil
		out = append(out, page)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (s *Store) GetPage(workspace, project, slug string) (Page, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return Page{}, "", false
	}
	p, ok := w.Projects[project]
	if !ok {
		return Page{}, "", false
	}
	page, ok := p.Pages[slug]
	if !ok || page.PublishedVersion == "" {
		return Page{}, "", false
	}
	return page, s.versionPath(workspace, project, slug, page.PublishedVersion), true
}

func (s *Store) GetVersion(workspace, project, slug, version string) (PageVersion, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return PageVersion{}, "", false
	}
	p, ok := w.Projects[project]
	if !ok {
		return PageVersion{}, "", false
	}
	page, ok := p.Pages[slug]
	if !ok {
		return PageVersion{}, "", false
	}
	for _, candidate := range page.Versions {
		if candidate.ID == version {
			return candidate, s.versionPath(workspace, project, slug, version), true
		}
	}
	return PageVersion{}, "", false
}

func (s *Store) ListVersions(workspace, project, slug string) (Page, []PageVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return Page{}, nil, os.ErrNotExist
	}
	p, ok := w.Projects[project]
	if !ok {
		return Page{}, nil, os.ErrNotExist
	}
	page, ok := p.Pages[slug]
	if !ok {
		return Page{}, nil, os.ErrNotExist
	}
	versions := append([]PageVersion(nil), page.Versions...)
	sort.Slice(versions, func(i, j int) bool { return versions[i].Number > versions[j].Number })
	page.Versions = nil
	return page, versions, nil
}

func (s *Store) PublishVersion(workspace, project, slug, versionID string) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return Page{}, os.ErrNotExist
	}
	p, ok := w.Projects[project]
	if !ok {
		return Page{}, os.ErrNotExist
	}
	page, ok := p.Pages[slug]
	if !ok {
		return Page{}, os.ErrNotExist
	}
	var selected *PageVersion
	for i := range page.Versions {
		if page.Versions[i].ID == versionID {
			selected = &page.Versions[i]
			break
		}
	}
	if selected == nil {
		return Page{}, os.ErrNotExist
	}
	if _, err := os.Stat(s.versionPath(workspace, project, slug, versionID)); err != nil {
		return Page{}, err
	}
	now := time.Now().UTC()
	page.PublishedVersion = versionID
	page.PublishedAt = &now
	page.Title = selected.Title
	page.Size = selected.Size
	page.UpdatedAt = now
	p.Pages[slug] = page
	p.UpdatedAt = now
	w.UpdatedAt = now
	if err := s.saveLocked(); err != nil {
		return Page{}, err
	}
	page.Versions = nil
	return page, nil
}

func (s *Store) DeletePage(workspace, project, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return os.ErrNotExist
	}
	p, ok := w.Projects[project]
	if !ok {
		return os.ErrNotExist
	}
	if _, ok := p.Pages[slug]; !ok {
		return os.ErrNotExist
	}
	if err := os.RemoveAll(filepath.Join(s.dir, "workspaces", workspace, "projects", project, "pages", slug)); err != nil {
		return err
	}
	_ = os.Remove(s.pagePath(workspace, project, slug))
	delete(p.Pages, slug)
	now := time.Now().UTC()
	p.UpdatedAt, w.UpdatedAt = now, now
	return s.saveLocked()
}

func (s *Store) CreateToken(workspace, name string, scopes []string, expiresAt *time.Time) (Token, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.st.Workspaces[workspace]; !ok {
		return Token{}, "", os.ErrNotExist
	}
	secret := "pgs_" + randomHex(24)
	id := "tok_" + randomHex(8)
	now := time.Now().UTC()
	t := &Token{ID: id, Workspace: workspace, Name: name, Prefix: secret[:12], Hash: tokenHash(secret), Scopes: scopes, CreatedAt: now, ExpiresAt: expiresAt}
	s.st.Tokens[id] = t
	if err := s.saveLocked(); err != nil {
		delete(s.st.Tokens, id)
		return Token{}, "", err
	}
	copy := *t
	copy.Hash = ""
	return copy, secret, nil
}

func (s *Store) ListTokens(workspace string) []Token {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Token{}
	for _, token := range s.st.Tokens {
		if token.Workspace == workspace {
			copy := *token
			copy.Hash = ""
			out = append(out, copy)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) RevokeToken(workspace, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.st.Tokens[id]
	if !ok || token.Workspace != workspace {
		return os.ErrNotExist
	}
	now := time.Now().UTC()
	token.RevokedAt = &now
	return s.saveLocked()
}

func (s *Store) ResetToken(workspace, id string) (Token, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.st.Tokens[id]
	if !ok || token.Workspace != workspace {
		return Token{}, "", os.ErrNotExist
	}
	if token.RevokedAt != nil {
		return Token{}, "", errors.New("token revoked")
	}
	secret := "pgs_" + randomHex(24)
	token.Hash = tokenHash(secret)
	token.Prefix = secret[:12]
	token.LastUsedAt = nil
	if err := s.saveLocked(); err != nil {
		return Token{}, "", err
	}
	copy := *token
	copy.Hash = ""
	return copy, secret, nil
}

func (s *Store) Authenticate(secret string) (*Token, bool) {
	hash := tokenHash(secret)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, token := range s.st.Tokens {
		if subtle.ConstantTimeCompare([]byte(hash), []byte(token.Hash)) != 1 {
			continue
		}
		if token.RevokedAt != nil || (token.ExpiresAt != nil && token.ExpiresAt.Before(now)) {
			return nil, false
		}
		persistUsage := token.LastUsedAt == nil || now.Sub(*token.LastUsedAt) >= 5*time.Minute
		token.LastUsedAt = &now
		if persistUsage {
			_ = s.saveLocked()
		}
		copy := *token
		copy.Hash = ""
		return &copy, true
	}
	return nil, false
}

func userView(user *User) UserView {
	return UserView{ID: user.ID, Username: user.Username, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status, CreatedAt: user.CreatedAt, LastLoginAt: user.LastLoginAt}
}

func (s *Store) NeedsSetup() bool { s.mu.RLock(); defer s.mu.RUnlock(); return len(s.st.Users) == 0 }

func (s *Store) CreateFirstAdmin(username, name, email, password string) (UserView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.st.Users) > 0 {
		return UserView{}, errors.New("setup already completed")
	}
	user := newUser(username, name, email, "platform_admin", password)
	s.st.Users[user.ID] = user
	for _, workspace := range s.st.Workspaces {
		if workspace.Members == nil {
			workspace.Members = map[string]Membership{}
		}
		workspace.Members[user.ID] = Membership{UserID: user.ID, Role: "owner", CreatedAt: time.Now().UTC()}
	}
	if err := s.saveLocked(); err != nil {
		delete(s.st.Users, user.ID)
		return UserView{}, err
	}
	return userView(user), nil
}

func (s *Store) CreateUser(username, name, email, role, password string) (UserView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.st.Users {
		if strings.EqualFold(existing.Username, username) {
			return UserView{}, errors.New("username already exists")
		}
	}
	user := newUser(username, name, email, role, password)
	s.st.Users[user.ID] = user
	if err := s.saveLocked(); err != nil {
		delete(s.st.Users, user.ID)
		return UserView{}, err
	}
	return userView(user), nil
}

func newUser(username, name, email, role, password string) *User {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	now := time.Now().UTC()
	return &User{ID: "usr_" + randomHex(8), Username: username, Name: name, Email: email, Role: role, Status: "active", PasswordSalt: hex.EncodeToString(salt), PasswordHash: passwordHash(password, salt), CreatedAt: now}
}

func (s *Store) ListUsers() []UserView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]UserView, 0, len(s.st.Users))
	for _, user := range s.st.Users {
		out = append(out, userView(user))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *Store) UpdateUser(id, name, email, role, status, password string) (UserView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.st.Users[id]
	if !ok {
		return UserView{}, os.ErrNotExist
	}
	user.Name, user.Email, user.Role, user.Status = name, email, role, status
	if password != "" {
		salt := make([]byte, 16)
		_, _ = rand.Read(salt)
		user.PasswordSalt = hex.EncodeToString(salt)
		user.PasswordHash = passwordHash(password, salt)
		for key, session := range s.st.Sessions {
			if session.UserID == id {
				delete(s.st.Sessions, key)
			}
		}
	}
	if status == "disabled" {
		for key, session := range s.st.Sessions {
			if session.UserID == id {
				delete(s.st.Sessions, key)
			}
		}
	}
	if err := s.saveLocked(); err != nil {
		return UserView{}, err
	}
	return userView(user), nil
}

func (s *Store) Login(username, password string) (UserView, string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var user *User
	for _, candidate := range s.st.Users {
		if strings.EqualFold(candidate.Username, username) {
			user = candidate
			break
		}
	}
	if user == nil || user.Status != "active" {
		return UserView{}, "", "", errors.New("invalid credentials")
	}
	salt, err := hex.DecodeString(user.PasswordSalt)
	if err != nil || subtle.ConstantTimeCompare([]byte(passwordHash(password, salt)), []byte(user.PasswordHash)) != 1 {
		return UserView{}, "", "", errors.New("invalid credentials")
	}
	secret := "ses_" + randomHex(32)
	csrf := randomHex(24)
	now := time.Now().UTC()
	s.st.Sessions[tokenHash(secret)] = &Session{UserID: user.ID, CSRF: csrf, CreatedAt: now, ExpiresAt: now.Add(7 * 24 * time.Hour)}
	user.LastLoginAt = &now
	if err := s.saveLocked(); err != nil {
		return UserView{}, "", "", err
	}
	return userView(user), secret, csrf, nil
}

func (s *Store) AuthenticateSession(secret string) (UserView, string, bool) {
	if secret == "" {
		return UserView{}, "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.st.Sessions[tokenHash(secret)]
	if !ok || session.ExpiresAt.Before(time.Now()) {
		return UserView{}, "", false
	}
	user, ok := s.st.Users[session.UserID]
	if !ok || user.Status != "active" {
		return UserView{}, "", false
	}
	return userView(user), session.CSRF, true
}

func (s *Store) Logout(secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.st.Sessions, tokenHash(secret))
	return s.saveLocked()
}

type MemberView struct {
	UserView
	WorkspaceRole string    `json:"workspaceRole"`
	JoinedAt      time.Time `json:"joinedAt"`
}

func (s *Store) ListMembers(workspace string) ([]MemberView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return nil, os.ErrNotExist
	}
	out := []MemberView{}
	for id, m := range w.Members {
		if user, ok := s.st.Users[id]; ok {
			out = append(out, MemberView{UserView: userView(user), WorkspaceRole: m.Role, JoinedAt: m.CreatedAt})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].JoinedAt.Before(out[j].JoinedAt) })
	return out, nil
}
func (s *Store) SetMember(workspace, userID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return os.ErrNotExist
	}
	if _, ok := s.st.Users[userID]; !ok {
		return os.ErrNotExist
	}
	if w.Members == nil {
		w.Members = map[string]Membership{}
	}
	existing, exists := w.Members[userID]
	if !exists {
		existing = Membership{UserID: userID, CreatedAt: time.Now().UTC()}
	}
	existing.Role = role
	w.Members[userID] = existing
	w.UpdatedAt = time.Now().UTC()
	return s.saveLocked()
}
func (s *Store) RemoveMember(workspace, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return os.ErrNotExist
	}
	if _, ok := w.Members[userID]; !ok {
		return os.ErrNotExist
	}
	delete(w.Members, userID)
	return s.saveLocked()
}
func (s *Store) MemberRole(workspace, userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.st.Workspaces[workspace]
	if !ok {
		return ""
	}
	return w.Members[userID].Role
}

func passwordHash(password string, salt []byte) string {
	return hex.EncodeToString(pbkdf2([]byte(password), salt, 120000, 32, sha256.New))
}
func pbkdf2(password, salt []byte, iterations, keyLen int, h func() hash.Hash) []byte {
	size := h().Size()
	blocks := (keyLen + size - 1) / size
	out := make([]byte, 0, blocks*size)
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(h, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(h, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

type authInfo struct {
	platform  bool
	workspace string
	scopes    map[string]bool
	user      *UserView
	csrf      string
}
type App struct {
	store *Store
	token string
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.home)
	mux.HandleFunc("GET /app.js", asset("web/app.js", "application/javascript; charset=utf-8"))
	mux.HandleFunc("GET /style.css", asset("web/style.css", "text/css; charset=utf-8"))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { jsonOut(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/auth/setup-status", a.setupStatus)
	mux.HandleFunc("POST /api/auth/setup", a.setupAdmin)
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.HandleFunc("POST /api/auth/logout", a.sessionRequired(a.logout))
	mux.HandleFunc("GET /api/auth/me", a.sessionRequired(a.me))
	mux.HandleFunc("GET /api/users", a.platformAdmin(a.listUsers))
	mux.HandleFunc("POST /api/users", a.platformAdmin(a.createUser))
	mux.HandleFunc("PATCH /api/users/{user}", a.platformAdmin(a.updateUser))
	mux.HandleFunc("GET /api/workspaces", a.authorize("read", "", a.listWorkspaces))
	mux.HandleFunc("POST /api/workspaces", a.authorize("admin", "", a.createWorkspace))
	mux.HandleFunc("PUT /api/workspaces/order", a.authorize("admin", "", a.reorderWorkspaces))
	mux.HandleFunc("PATCH /api/workspaces/{workspace}", a.authorize("admin", "workspace", a.updateWorkspace))
	mux.HandleFunc("DELETE /api/workspaces/{workspace}", a.authorize("admin", "workspace", a.deleteWorkspace))
	mux.HandleFunc("GET /api/workspaces/{workspace}/projects", a.authorize("read", "workspace", a.listProjects))
	mux.HandleFunc("POST /api/workspaces/{workspace}/projects", a.authorize("admin", "workspace", a.createProject))
	mux.HandleFunc("PUT /api/workspaces/{workspace}/projects/order", a.authorize("admin", "workspace", a.reorderProjects))
	mux.HandleFunc("DELETE /api/workspaces/{workspace}/projects/{project}", a.authorize("admin", "workspace", a.deleteProject))
	mux.HandleFunc("GET /api/workspaces/{workspace}/tokens", a.authorize("admin", "workspace", a.listTokens))
	mux.HandleFunc("POST /api/workspaces/{workspace}/tokens", a.authorize("admin", "workspace", a.createToken))
	mux.HandleFunc("DELETE /api/workspaces/{workspace}/tokens/{token}", a.authorize("admin", "workspace", a.revokeToken))
	mux.HandleFunc("POST /api/workspaces/{workspace}/tokens/{token}/reset", a.authorize("admin", "workspace", a.resetToken))
	mux.HandleFunc("GET /api/workspaces/{workspace}/members", a.authorize("admin", "workspace", a.listMembers))
	mux.HandleFunc("GET /api/workspaces/{workspace}/directory", a.authorize("admin", "workspace", a.directoryUsers))
	mux.HandleFunc("PUT /api/workspaces/{workspace}/members/{user}", a.authorize("admin", "workspace", a.setMember))
	mux.HandleFunc("DELETE /api/workspaces/{workspace}/members/{user}", a.authorize("admin", "workspace", a.removeMember))
	mux.HandleFunc("GET /api/workspaces/{workspace}/projects/{project}/pages", a.authorize("read", "workspace", a.listPages))
	mux.HandleFunc("POST /api/workspaces/{workspace}/projects/{project}/pages", a.authorize("write", "workspace", a.uploadPage))
	mux.HandleFunc("GET /api/workspaces/{workspace}/projects/{project}/pages/{slug}/versions", a.authorize("read", "workspace", a.listVersions))
	mux.HandleFunc("POST /api/workspaces/{workspace}/projects/{project}/pages/{slug}/publish", a.authorize("write", "workspace", a.publishVersion))
	mux.HandleFunc("DELETE /api/workspaces/{workspace}/projects/{project}/pages/{slug}", a.authorize("write", "workspace", a.deletePage))
	mux.HandleFunc("GET /preview/{workspace}/{project}/{slug}/{version}", a.showPreview)
	mux.HandleFunc("GET /p/{workspace}/{project}/{slug}", a.showPage)
	mux.HandleFunc("GET /p/{slug}", a.showLegacyPage)
	return securityHeaders(requestLog(mux))
}

func (a *App) authorize(scope, workspaceParam string, next func(http.ResponseWriter, *http.Request, authInfo)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("pages_session"); err == nil {
			user, csrf, ok := a.store.AuthenticateSession(cookie.Value)
			if ok {
				if r.Method != http.MethodGet && r.Method != http.MethodHead && !secureEqual(r.Header.Get("X-CSRF-Token"), csrf) {
					jsonError(w, 403, "csrf_failed", "安全校验失败，请刷新后重试")
					return
				}
				auth := authInfo{platform: user.Role == "platform_admin", user: &user, csrf: csrf, scopes: map[string]bool{}}
				if workspaceParam != "" && !auth.platform {
					auth.workspace = r.PathValue(workspaceParam)
					role := a.store.MemberRole(auth.workspace, user.ID)
					auth.scopes = roleScopes(role)
					if role == "" {
						jsonError(w, 403, "workspace_forbidden", "你不是该工作空间的成员")
						return
					}
				}
				if !auth.platform && workspaceParam != "" && !auth.scopes[scope] {
					jsonError(w, 403, "insufficient_role", "当前空间角色没有此操作权限")
					return
				}
				next(w, r, auth)
				return
			}
		}
		secret := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if secret == "" {
			secret = r.Header.Get("X-API-Key")
		}
		if secret != "" && secureEqual(secret, a.token) {
			next(w, r, authInfo{platform: true})
			return
		}
		token, ok := a.store.Authenticate(secret)
		if !ok {
			jsonError(w, 401, "unauthorized", "请提供有效的 API Token")
			return
		}
		if workspaceParam != "" && token.Workspace != r.PathValue(workspaceParam) {
			jsonError(w, 403, "workspace_forbidden", "Token 无权访问该工作空间")
			return
		}
		scopes := map[string]bool{}
		for _, v := range token.Scopes {
			scopes[v] = true
		}
		if !scopes[scope] && !scopes["admin"] {
			jsonError(w, 403, "insufficient_scope", "Token 缺少 "+scope+" 权限")
			return
		}
		next(w, r, authInfo{workspace: token.Workspace, scopes: scopes})
	}
}

func roleScopes(role string) map[string]bool {
	switch role {
	case "owner", "admin":
		return map[string]bool{"read": true, "write": true, "admin": true}
	case "editor":
		return map[string]bool{"read": true, "write": true}
	case "viewer":
		return map[string]bool{"read": true}
	default:
		return map[string]bool{}
	}
}

func (a *App) sessionRequired(next func(http.ResponseWriter, *http.Request, UserView, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("pages_session")
		if err != nil {
			jsonError(w, 401, "login_required", "请先登录")
			return
		}
		user, csrf, ok := a.store.AuthenticateSession(cookie.Value)
		if !ok {
			clearSessionCookie(w)
			jsonError(w, 401, "login_required", "登录已过期")
			return
		}
		if r.Method != http.MethodGet && !secureEqual(r.Header.Get("X-CSRF-Token"), csrf) {
			jsonError(w, 403, "csrf_failed", "安全校验失败")
			return
		}
		next(w, r, user, csrf)
	}
}
func (a *App) platformAdmin(next func(http.ResponseWriter, *http.Request, UserView)) http.HandlerFunc {
	return a.sessionRequired(func(w http.ResponseWriter, r *http.Request, user UserView, _ string) {
		if user.Role != "platform_admin" {
			jsonError(w, 403, "platform_admin_required", "需要平台管理员权限")
			return
		}
		next(w, r, user)
	})
}
func setSessionCookie(w http.ResponseWriter, r *http.Request, secret string) {
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{Name: "pages_session", Value: secret, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: secure, MaxAge: 7 * 24 * 60 * 60})
}
func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "pages_session", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}

func asset(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := webFS.ReadFile(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(b)
	}
}
func (a *App) home(w http.ResponseWriter, _ *http.Request) {
	b, _ := webFS.ReadFile("web/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func (a *App) setupStatus(w http.ResponseWriter, _ *http.Request) {
	jsonOut(w, 200, map[string]bool{"needsSetup": a.store.NeedsSetup()})
}
func (a *App) setupAdmin(w http.ResponseWriter, r *http.Request) {
	var in struct{ Username, Name, Email, Password string }
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	in.Name = strings.TrimSpace(in.Name)
	if !validUserInput(w, in.Username, in.Name, in.Password) {
		return
	}
	user, err := a.store.CreateFirstAdmin(in.Username, in.Name, strings.TrimSpace(in.Email), in.Password)
	if err != nil {
		jsonError(w, 409, "setup_completed", "平台已经完成初始化")
		return
	}
	view, secret, csrf, err := a.store.Login(user.Username, in.Password)
	if err != nil {
		storeError(w, err)
		return
	}
	setSessionCookie(w, r, secret)
	jsonOut(w, 201, map[string]any{"user": view, "csrfToken": csrf})
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Username, Password string }
	if !decodeJSON(w, r, &in) {
		return
	}
	user, secret, csrf, err := a.store.Login(strings.TrimSpace(in.Username), in.Password)
	if err != nil {
		jsonError(w, 401, "invalid_credentials", "用户名或密码错误")
		return
	}
	setSessionCookie(w, r, secret)
	jsonOut(w, 200, map[string]any{"user": user, "csrfToken": csrf})
}
func (a *App) logout(w http.ResponseWriter, r *http.Request, _ UserView, _ string) {
	cookie, _ := r.Cookie("pages_session")
	if cookie != nil {
		_ = a.store.Logout(cookie.Value)
	}
	clearSessionCookie(w)
	w.WriteHeader(204)
}
func (a *App) me(w http.ResponseWriter, _ *http.Request, user UserView, csrf string) {
	jsonOut(w, 200, map[string]any{"user": user, "csrfToken": csrf})
}
func (a *App) listUsers(w http.ResponseWriter, _ *http.Request, _ UserView) {
	jsonOut(w, 200, map[string]any{"users": a.store.ListUsers()})
}
func (a *App) createUser(w http.ResponseWriter, r *http.Request, _ UserView) {
	var in struct{ Username, Name, Email, Role, Password string }
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	in.Name = strings.TrimSpace(in.Name)
	if !validUserInput(w, in.Username, in.Name, in.Password) {
		return
	}
	if in.Role != "platform_admin" && in.Role != "user" {
		jsonError(w, 400, "invalid_role", "平台角色仅支持 platform_admin 或 user")
		return
	}
	result, err := a.store.CreateUser(in.Username, in.Name, strings.TrimSpace(in.Email), in.Role, in.Password)
	if err != nil {
		conflictOrServer(w, err)
		return
	}
	jsonOut(w, 201, result)
}
func (a *App) updateUser(w http.ResponseWriter, r *http.Request, current UserView) {
	var in struct{ Name, Email, Role, Status, Password string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Name == "" || len([]rune(in.Name)) > 80 {
		jsonError(w, 400, "invalid_name", "姓名不能为空且不能超过 80 个字符")
		return
	}
	if in.Role != "platform_admin" && in.Role != "user" {
		jsonError(w, 400, "invalid_role", "平台角色无效")
		return
	}
	if in.Status != "active" && in.Status != "disabled" {
		jsonError(w, 400, "invalid_status", "用户状态无效")
		return
	}
	if in.Password != "" && len(in.Password) < 10 {
		jsonError(w, 400, "weak_password", "密码至少需要 10 个字符")
		return
	}
	if r.PathValue("user") == current.ID && (in.Role != "platform_admin" || in.Status != "active") {
		jsonError(w, 400, "cannot_demote_self", "不能停用自己或移除自己的管理员角色")
		return
	}
	result, err := a.store.UpdateUser(r.PathValue("user"), strings.TrimSpace(in.Name), strings.TrimSpace(in.Email), in.Role, in.Status, in.Password)
	if err != nil {
		storeError(w, err)
		return
	}
	jsonOut(w, 200, result)
}
func validUserInput(w http.ResponseWriter, username, name, password string) bool {
	if !usernamePattern.MatchString(username) {
		jsonError(w, 400, "invalid_username", "用户名需要 3-40 位，只支持字母、数字、点、横线和下划线")
		return false
	}
	if name == "" || len([]rune(name)) > 80 {
		jsonError(w, 400, "invalid_name", "姓名不能为空且不能超过 80 个字符")
		return false
	}
	if len(password) < 10 {
		jsonError(w, 400, "weak_password", "密码至少需要 10 个字符")
		return false
	}
	return true
}
func (a *App) listWorkspaces(w http.ResponseWriter, _ *http.Request, auth authInfo) {
	if auth.user != nil {
		jsonOut(w, 200, map[string]any{"workspaces": a.store.ListWorkspacesForUser(auth.user.ID, auth.platform)})
		return
	}
	jsonOut(w, 200, map[string]any{"workspaces": a.store.ListWorkspaces(auth.workspace)})
}

func (a *App) createWorkspace(w http.ResponseWriter, r *http.Request, auth authInfo) {
	if auth.user == nil && !auth.platform {
		jsonError(w, 403, "user_required", "创建工作空间需要登录用户")
		return
	}
	var in struct{ Slug, Name string }
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Slug = cleanSlug(in.Slug)
	in.Name = strings.TrimSpace(in.Name)
	if !validNamedSlug(w, in.Slug, in.Name, "工作空间") {
		return
	}
	result, err := a.store.CreateWorkspace(in.Slug, in.Name)
	if err != nil {
		conflictOrServer(w, err)
		return
	}
	if auth.user != nil {
		_ = a.store.SetMember(result.Slug, auth.user.ID, "owner")
	}
	jsonOut(w, 201, result)
}
func (a *App) reorderWorkspaces(w http.ResponseWriter, r *http.Request, auth authInfo) {
	if !auth.platform {
		jsonError(w, 403, "platform_admin_required", "调整工作空间顺序需要平台管理员权限")
		return
	}
	var in struct{ Slugs []string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := a.store.ReorderWorkspaces(in.Slugs); err != nil {
		jsonError(w, 400, "invalid_order", err.Error())
		return
	}
	w.WriteHeader(204)
}
func (a *App) deleteWorkspace(w http.ResponseWriter, r *http.Request, auth authInfo) {
	if !auth.platform && (auth.user == nil || a.store.MemberRole(r.PathValue("workspace"), auth.user.ID) != "owner") {
		jsonError(w, 403, "workspace_owner_required", "只有工作空间所有者可以删除该空间")
		return
	}
	handleDelete(w, a.store.DeleteWorkspace(r.PathValue("workspace")))
}
func (a *App) updateWorkspace(w http.ResponseWriter, r *http.Request, _ authInfo) {
	var in struct{ Name string }
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len([]rune(in.Name)) > 120 {
		jsonError(w, 400, "invalid_name", "工作空间名称不能为空且不能超过 120 个字符")
		return
	}
	result, err := a.store.UpdateWorkspace(r.PathValue("workspace"), in.Name)
	if err != nil {
		storeError(w, err)
		return
	}
	jsonOut(w, 200, result)
}
func (a *App) listProjects(w http.ResponseWriter, r *http.Request, _ authInfo) {
	result, err := a.store.ListProjects(r.PathValue("workspace"))
	if err != nil {
		storeError(w, err)
		return
	}
	jsonOut(w, 200, map[string]any{"projects": result})
}
func (a *App) createProject(w http.ResponseWriter, r *http.Request, _ authInfo) {
	var in struct{ Slug, Name, Description string }
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Slug = cleanSlug(in.Slug)
	in.Name = strings.TrimSpace(in.Name)
	if !validNamedSlug(w, in.Slug, in.Name, "项目") {
		return
	}
	result, err := a.store.CreateProject(r.PathValue("workspace"), in.Slug, in.Name, strings.TrimSpace(in.Description))
	if err != nil {
		conflictOrServer(w, err)
		return
	}
	jsonOut(w, 201, result)
}
func (a *App) reorderProjects(w http.ResponseWriter, r *http.Request, _ authInfo) {
	var in struct{ Slugs []string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := a.store.ReorderProjects(r.PathValue("workspace"), in.Slugs); errors.Is(err, os.ErrNotExist) {
		storeError(w, err)
		return
	} else if err != nil {
		jsonError(w, 400, "invalid_order", err.Error())
		return
	}
	w.WriteHeader(204)
}
func (a *App) deleteProject(w http.ResponseWriter, r *http.Request, _ authInfo) {
	handleDelete(w, a.store.DeleteProject(r.PathValue("workspace"), r.PathValue("project")))
}
func (a *App) listTokens(w http.ResponseWriter, r *http.Request, _ authInfo) {
	jsonOut(w, 200, map[string]any{"tokens": a.store.ListTokens(r.PathValue("workspace"))})
}
func (a *App) createToken(w http.ResponseWriter, r *http.Request, _ authInfo) {
	var in struct {
		Name      string
		Scopes    []string
		ExpiresAt *time.Time
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		jsonError(w, 400, "invalid_name", "Token 名称不能为空")
		return
	}
	scopes, ok := validateScopes(in.Scopes)
	if !ok {
		jsonError(w, 400, "invalid_scopes", "权限仅支持 read、write、admin")
		return
	}
	token, secret, err := a.store.CreateToken(r.PathValue("workspace"), in.Name, scopes, in.ExpiresAt)
	if err != nil {
		storeError(w, err)
		return
	}
	jsonOut(w, 201, map[string]any{"token": token, "secret": secret, "notice": "密钥只显示一次，请立即保存"})
}
func (a *App) revokeToken(w http.ResponseWriter, r *http.Request, _ authInfo) {
	handleDelete(w, a.store.RevokeToken(r.PathValue("workspace"), r.PathValue("token")))
}
func (a *App) resetToken(w http.ResponseWriter, r *http.Request, _ authInfo) {
	token, secret, err := a.store.ResetToken(r.PathValue("workspace"), r.PathValue("token"))
	if errors.Is(err, os.ErrNotExist) {
		jsonError(w, 404, "not_found", "Token 不存在")
		return
	}
	if err != nil {
		jsonError(w, 409, "token_revoked", "已吊销的 Token 不能重置，请创建新 Token")
		return
	}
	jsonOut(w, 200, map[string]any{"token": token, "secret": secret, "notice": "旧密钥已立即失效，新密钥只显示一次"})
}
func (a *App) listMembers(w http.ResponseWriter, r *http.Request, _ authInfo) {
	result, err := a.store.ListMembers(r.PathValue("workspace"))
	if err != nil {
		storeError(w, err)
		return
	}
	jsonOut(w, 200, map[string]any{"members": result})
}
func (a *App) directoryUsers(w http.ResponseWriter, _ *http.Request, _ authInfo) {
	users := a.store.ListUsers()
	active := make([]UserView, 0, len(users))
	for _, user := range users {
		if user.Status == "active" {
			active = append(active, user)
		}
	}
	jsonOut(w, 200, map[string]any{"users": active})
}
func (a *App) setMember(w http.ResponseWriter, r *http.Request, _ authInfo) {
	var in struct{ Role string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validWorkspaceRole(in.Role) {
		jsonError(w, 400, "invalid_role", "空间角色仅支持 owner、admin、editor、viewer")
		return
	}
	if err := a.store.SetMember(r.PathValue("workspace"), r.PathValue("user"), in.Role); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a *App) removeMember(w http.ResponseWriter, r *http.Request, auth authInfo) {
	if auth.user != nil && r.PathValue("user") == auth.user.ID {
		jsonError(w, 400, "cannot_remove_self", "不能移除自己的工作空间权限")
		return
	}
	handleDelete(w, a.store.RemoveMember(r.PathValue("workspace"), r.PathValue("user")))
}
func validWorkspaceRole(role string) bool {
	return role == "owner" || role == "admin" || role == "editor" || role == "viewer"
}
func (a *App) listPages(w http.ResponseWriter, r *http.Request, _ authInfo) {
	result, err := a.store.ListPages(r.PathValue("workspace"), r.PathValue("project"))
	if err != nil {
		storeError(w, err)
		return
	}
	jsonOut(w, 200, map[string]any{"pages": result})
}
func (a *App) listVersions(w http.ResponseWriter, r *http.Request, _ authInfo) {
	page, versions, err := a.store.ListVersions(r.PathValue("workspace"), r.PathValue("project"), r.PathValue("slug"))
	if err != nil {
		storeError(w, err)
		return
	}
	jsonOut(w, 200, map[string]any{"page": page, "versions": versions})
}
func (a *App) publishVersion(w http.ResponseWriter, r *http.Request, _ authInfo) {
	var in struct{ Version string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Version == "" {
		jsonError(w, 400, "version_required", "请选择要发布的版本")
		return
	}
	page, err := a.store.PublishVersion(r.PathValue("workspace"), r.PathValue("project"), r.PathValue("slug"), in.Version)
	if err != nil {
		storeError(w, err)
		return
	}
	jsonOut(w, 200, page)
}

func readMultipartHTML(r *http.Request) ([]byte, string, string, error) {
	if err := r.ParseMultipartForm(maxHTMLSize); err != nil {
		return nil, "", "", err
	}
	f, h, err := r.FormFile("file")
	if err != nil {
		return nil, "", "", errors.New("multipart 字段 file 缺失")
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxHTMLSize+1))
	if err != nil {
		return nil, "", "", err
	}
	if len(b) > maxHTMLSize {
		return nil, "", "", errors.New("HTML 超过 5MB")
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = strings.TrimSuffix(h.Filename, filepath.Ext(h.Filename))
	}
	return b, r.FormValue("slug"), title, nil
}
func readJSONHTML(r *http.Request) ([]byte, string, string, error) {
	var in struct{ Slug, Title, HTML string }
	dec := json.NewDecoder(io.LimitReader(r.Body, maxHTMLSize+1024))
	if err := dec.Decode(&in); err != nil {
		return nil, "", "", err
	}
	return []byte(in.HTML), in.Slug, in.Title, nil
}
func readRawHTML(r *http.Request) ([]byte, string, string, error) {
	b, err := io.ReadAll(io.LimitReader(r.Body, maxHTMLSize+1))
	if err != nil {
		return nil, "", "", err
	}
	if len(b) > maxHTMLSize {
		return nil, "", "", errors.New("HTML 超过 5MB")
	}
	return b, r.URL.Query().Get("slug"), r.URL.Query().Get("title"), nil
}

func (a *App) uploadPage(w http.ResponseWriter, r *http.Request, _ authInfo) {
	var b []byte
	var slug, title string
	var err error
	ct := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ct, "multipart/form-data"):
		b, slug, title, err = readMultipartHTML(r)
	case strings.HasPrefix(ct, "application/json"):
		b, slug, title, err = readJSONHTML(r)
	default:
		b, slug, title, err = readRawHTML(r)
	}
	if err != nil {
		jsonError(w, 400, "invalid_upload", err.Error())
		return
	}
	slug = cleanSlug(slug)
	if slug == "" {
		slug = "page-" + randomHex(6)
	}
	if !slugPattern.MatchString(slug) {
		jsonError(w, 400, "invalid_slug", "slug 仅支持小写字母、数字和连字符，最长 63 位")
		return
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = slug
	}
	if len([]rune(title)) > 120 {
		jsonError(w, 400, "invalid_title", "标题不能超过 120 个字符")
		return
	}
	if len(b) == 0 {
		jsonError(w, 400, "empty_html", "HTML 内容不能为空")
		return
	}
	if len(b) > maxHTMLSize {
		jsonError(w, 413, "too_large", "HTML 不能超过 5MB")
		return
	}
	page, err := a.store.Put(r.PathValue("workspace"), r.PathValue("project"), slug, title, b)
	if err != nil {
		storeError(w, err)
		return
	}
	jsonOut(w, 201, page)
}
func (a *App) deletePage(w http.ResponseWriter, r *http.Request, _ authInfo) {
	handleDelete(w, a.store.DeletePage(r.PathValue("workspace"), r.PathValue("project"), r.PathValue("slug")))
}
func (a *App) showPage(w http.ResponseWriter, r *http.Request) {
	a.servePage(w, r, r.PathValue("workspace"), r.PathValue("project"), r.PathValue("slug"))
}
func (a *App) showLegacyPage(w http.ResponseWriter, r *http.Request) {
	a.servePage(w, r, "default", "legacy", r.PathValue("slug"))
}
func (a *App) showPreview(w http.ResponseWriter, r *http.Request) {
	workspace, project, slug, version := r.PathValue("workspace"), r.PathValue("project"), r.PathValue("slug"), r.PathValue("version")
	if !slugPattern.MatchString(workspace) || !slugPattern.MatchString(project) || !slugPattern.MatchString(slug) || !strings.HasPrefix(version, "ver_") {
		http.NotFound(w, r)
		return
	}
	_, path, ok := a.store.GetVersion(workspace, project, slug, version)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	serveHTMLFile(w, r, path)
}
func (a *App) servePage(w http.ResponseWriter, r *http.Request, workspace, project, slug string) {
	if !slugPattern.MatchString(workspace) || !slugPattern.MatchString(project) || !slugPattern.MatchString(slug) {
		http.NotFound(w, r)
		return
	}
	_, path, ok := a.store.GetPage(workspace, project, slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveHTMLFile(w, r, path)
}
func serveHTMLFile(w http.ResponseWriter, r *http.Request, path string) {
	w.Header().Set("Content-Security-Policy", "sandbox allow-scripts allow-forms allow-modals; default-src 'self' data: blob: https:; img-src 'self' data: blob: https:; style-src 'self' 'unsafe-inline' https:; script-src 'self' 'unsafe-inline' 'unsafe-eval' https:; connect-src https:")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, path)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return false
	}
	return true
}
func validNamedSlug(w http.ResponseWriter, slug, name, kind string) bool {
	if !slugPattern.MatchString(slug) {
		jsonError(w, 400, "invalid_slug", kind+"标识仅支持小写字母、数字和连字符，最长 63 位")
		return false
	}
	if name == "" || len([]rune(name)) > 120 {
		jsonError(w, 400, "invalid_name", kind+"名称不能为空且不能超过 120 个字符")
		return false
	}
	return true
}
func validateScopes(values []string) ([]string, bool) {
	if len(values) == 0 {
		return []string{"read", "write"}, true
	}
	seen := map[string]bool{}
	out := []string{}
	for _, v := range values {
		if v != "read" && v != "write" && v != "admin" {
			return nil, false
		}
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out, true
}
func handleDelete(w http.ResponseWriter, err error) {
	if errors.Is(err, os.ErrNotExist) {
		jsonError(w, 404, "not_found", "资源不存在")
		return
	}
	if err != nil {
		jsonError(w, 500, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(204)
}
func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, os.ErrNotExist) {
		jsonError(w, 404, "not_found", "资源不存在")
		return
	}
	jsonError(w, 500, "store_failed", err.Error())
}
func conflictOrServer(w http.ResponseWriter, err error) {
	if strings.Contains(err.Error(), "already exists") {
		jsonError(w, 409, "already_exists", "相同标识的资源已存在")
		return
	}
	storeError(w, err)
}
func cleanSlug(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
func pageURL(workspace, project, slug string) string {
	return "/p/" + workspace + "/" + project + "/" + slug
}
func previewURL(workspace, project, slug, version string) string {
	return "/preview/" + workspace + "/" + project + "/" + slug + "/" + version
}
func tokenHash(v string) string { sum := sha256.Sum256([]byte(v)); return hex.EncodeToString(sum[:]) }
func secureEqual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if !strings.HasPrefix(r.URL.Path, "/p/") {
			w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self' data:")
		}
		next.ServeHTTP(w, r)
	})
}
func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func jsonError(w http.ResponseWriter, status int, code, msg string) {
	jsonOut(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}

func main() {
	addr := env("PAGE_ADDR", ":8080")
	dir := env("PAGE_DATA", "./data")
	token := os.Getenv("PAGE_TOKEN")
	if token == "" {
		token = "dev-token"
		log.Printf("警告：PAGE_TOKEN 未设置，当前使用开发令牌 dev-token")
	}
	store, err := NewStore(dir)
	if err != nil {
		log.Fatal(err)
	}
	app := &App{store: store, token: token}
	server := &http.Server{Addr: addr, Handler: app.routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("Page Service 已启动：http://localhost%s（数据目录 %s）", addr, dir)
	log.Fatal(server.ListenAndServe())
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
