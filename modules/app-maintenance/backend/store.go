package main

import (
	"fmt"
	"sync"
	"time"
)

// Store is a simple in-memory, mutex-guarded data store.
//
// This is a starting point for the app-maintenance module so the whole
// stack (frontend <-> nginx <-> backend) can be scaffolded and run
// end-to-end without an external database. Swap this out for a real
// database (Postgres, SQLite, ...) once the shape of the data settles.
type Store struct {
	mu      sync.RWMutex
	users   map[string]*User
	roles   map[string]*Role
	modules map[string]*Module
	seq     int
}

func NewStore() *Store {
	s := &Store{
		users:   map[string]*User{},
		roles:   map[string]*Role{},
		modules: map[string]*Module{},
	}
	s.seed()
	return s
}

func (s *Store) nextID(prefix string) string {
	s.seq++
	return fmt.Sprintf("%s-%d", prefix, s.seq)
}

func (s *Store) seed() {
	now := time.Now()

	modAppMaint := &Module{ID: s.nextID("mod"), Code: "app-maintenance", Name: "App Maintenance", Description: "User, module and role administration", IsActive: true, CreatedAt: now, UpdatedAt: now}
	s.modules[modAppMaint.ID] = modAppMaint

	adminRole := &Role{ID: s.nextID("role"), Name: "Administrator", Description: "Full access to all modules", ModuleIDs: []string{modAppMaint.ID}, CreatedAt: now, UpdatedAt: now}
	s.roles[adminRole.ID] = adminRole

	admin := &User{ID: s.nextID("user"), Username: "admin", FullName: "System Administrator", Email: "admin@example.com", IsActive: true, RoleIDs: []string{adminRole.ID}, CreatedAt: now, UpdatedAt: now}
	s.users[admin.ID] = admin
}

// ---- Users ----

func (s *Store) ListUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out
}

func (s *Store) GetUser(id string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *Store) CreateUser(in *User) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	in.ID = s.nextID("user")
	in.CreatedAt = now
	in.UpdatedAt = now
	if in.RoleIDs == nil {
		in.RoleIDs = []string{}
	}
	s.users[in.ID] = in
	return in
}

func (s *Store) UpdateUser(id string, in *User) (*User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.users[id]
	if !ok {
		return nil, false
	}
	existing.Username = in.Username
	existing.FullName = in.FullName
	existing.Email = in.Email
	existing.IsActive = in.IsActive
	existing.UpdatedAt = time.Now()
	return existing, true
}

func (s *Store) DeleteUser(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return false
	}
	delete(s.users, id)
	return true
}

func (s *Store) SetUserRoles(id string, roleIDs []string) (*User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return nil, false
	}
	u.RoleIDs = roleIDs
	u.UpdatedAt = time.Now()
	return u, true
}

// ---- Roles ----

func (s *Store) ListRoles() []*Role {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Role, 0, len(s.roles))
	for _, r := range s.roles {
		out = append(out, r)
	}
	return out
}

func (s *Store) GetRole(id string) (*Role, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.roles[id]
	return r, ok
}

func (s *Store) CreateRole(in *Role) *Role {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	in.ID = s.nextID("role")
	in.CreatedAt = now
	in.UpdatedAt = now
	if in.ModuleIDs == nil {
		in.ModuleIDs = []string{}
	}
	s.roles[in.ID] = in
	return in
}

func (s *Store) UpdateRole(id string, in *Role) (*Role, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.roles[id]
	if !ok {
		return nil, false
	}
	existing.Name = in.Name
	existing.Description = in.Description
	existing.ModuleIDs = in.ModuleIDs
	existing.UpdatedAt = time.Now()
	return existing, true
}

func (s *Store) DeleteRole(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.roles[id]; !ok {
		return false
	}
	delete(s.roles, id)
	// unassign from any user that had it
	for _, u := range s.users {
		filtered := u.RoleIDs[:0]
		for _, rid := range u.RoleIDs {
			if rid != id {
				filtered = append(filtered, rid)
			}
		}
		u.RoleIDs = filtered
	}
	return true
}

// ---- Modules ----

func (s *Store) ListModules() []*Module {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Module, 0, len(s.modules))
	for _, m := range s.modules {
		out = append(out, m)
	}
	return out
}

func (s *Store) GetModule(id string) (*Module, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.modules[id]
	return m, ok
}

func (s *Store) CreateModule(in *Module) *Module {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	in.ID = s.nextID("mod")
	in.CreatedAt = now
	in.UpdatedAt = now
	s.modules[in.ID] = in
	return in
}

func (s *Store) UpdateModule(id string, in *Module) (*Module, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.modules[id]
	if !ok {
		return nil, false
	}
	existing.Code = in.Code
	existing.Name = in.Name
	existing.Description = in.Description
	existing.IsActive = in.IsActive
	existing.UpdatedAt = time.Now()
	return existing, true
}

func (s *Store) DeleteModule(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.modules[id]; !ok {
		return false
	}
	delete(s.modules, id)
	for _, r := range s.roles {
		filtered := r.ModuleIDs[:0]
		for _, mid := range r.ModuleIDs {
			if mid != id {
				filtered = append(filtered, mid)
			}
		}
		r.ModuleIDs = filtered
	}
	return true
}
