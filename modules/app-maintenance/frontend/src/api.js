// BASE_URL is set by vite's `base` config (see vite.config.js), so this
// automatically points at /app-maintenance/api/ in this module, and at
// /<module>/api/ if this folder is duplicated for a new module.
const API_BASE = `${import.meta.env.BASE_URL}api/`

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `Request failed: ${res.status}`)
  }
  if (res.status === 204) return null
  return res.json()
}

export const api = {
  // users
  listUsers: () => request('users'),
  createUser: (data) => request('users', { method: 'POST', body: JSON.stringify(data) }),
  updateUser: (id, data) => request(`users/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteUser: (id) => request(`users/${id}`, { method: 'DELETE' }),
  setUserRoles: (id, roleIds) => request(`users/${id}/roles`, { method: 'PUT', body: JSON.stringify({ roleIds }) }),
  setUserPassword: (id, password) => request(`users/${id}/password`, { method: 'PUT', body: JSON.stringify({ password }) }),

  // roles
  listRoles: () => request('roles'),
  createRole: (data) => request('roles', { method: 'POST', body: JSON.stringify(data) }),
  updateRole: (id, data) => request(`roles/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteRole: (id) => request(`roles/${id}`, { method: 'DELETE' }),

  // modules
  listModules: () => request('modules'),
  createModule: (data) => request('modules', { method: 'POST', body: JSON.stringify(data) }),
  updateModule: (id, data) => request(`modules/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteModule: (id) => request(`modules/${id}`, { method: 'DELETE' }),

  // settings
  getSettings: () => request('settings'),
  updateSettings: (data) => request('settings', { method: 'PUT', body: JSON.stringify(data) }),

  // audit log
  listAuditLog: (limit, offset) => request(`audit-log?limit=${limit}&offset=${offset}`),
}
