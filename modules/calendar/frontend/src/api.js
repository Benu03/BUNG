// BASE_URL is set by vite's `base` config (see vite.config.js), so this
// automatically points at /calendar/api/ in this module, and at
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
  me: () => request('me'),

  // from/to are optional Date objects (RFC3339 on the wire) - omit both to
  // get every event.
  listEvents: (from, to) => {
    const params = new URLSearchParams()
    if (from) params.set('from', from.toISOString())
    if (to) params.set('to', to.toISOString())
    const qs = params.toString()
    return request(`events${qs ? `?${qs}` : ''}`)
  },
  // Full detail (includes attendees) - fetched when opening the edit modal.
  getEvent: (id) => request(`events/${id}`),
  createEvent: (data) => request('events', { method: 'POST', body: JSON.stringify(data) }),
  updateEvent: (id, data) => request(`events/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteEvent: (id) => request(`events/${id}`, { method: 'DELETE' }),

  inviteAttendee: (id, username) => request(`events/${id}/invite`, { method: 'POST', body: JSON.stringify({ username }) }),
  removeAttendee: (id, userId) => request(`events/${id}/invite/${userId}`, { method: 'DELETE' }),
}
