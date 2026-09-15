// BASE_URL is set by vite's `base` config (see vite.config.js), so this
// automatically points at /ticketing/api/ in this module, and at
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
  listTickets: () => request('tickets'),
  getTicket: (id) => request(`tickets/${id}`),
  createTicket: (data) => request('tickets', { method: 'POST', body: JSON.stringify(data) }),
  updateTicket: (id, data) => request(`tickets/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteTicket: (id) => request(`tickets/${id}`, { method: 'DELETE' }),

  addComment: (ticketId, body) => request(`tickets/${ticketId}/comments`, { method: 'POST', body: JSON.stringify({ body }) }),

  listUsers: () => request('users'),
}
