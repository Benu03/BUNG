// BASE_URL is set by vite's `base` config (see vite.config.js), so this
// automatically points at /kanban/api/ in this module, and at
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
  listBoards: () => request('boards'),
  getBoard: (id) => request(`boards/${id}`),
  createBoard: (data) => request('boards', { method: 'POST', body: JSON.stringify(data) }),
  updateBoard: (id, data) => request(`boards/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteBoard: (id) => request(`boards/${id}`, { method: 'DELETE' }),

  createColumn: (boardId, data) => request(`boards/${boardId}/columns`, { method: 'POST', body: JSON.stringify(data) }),
  updateColumn: (id, data) => request(`columns/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteColumn: (id) => request(`columns/${id}`, { method: 'DELETE' }),

  createCard: (columnId, data) => request(`columns/${columnId}/cards`, { method: 'POST', body: JSON.stringify(data) }),
  updateCard: (id, data) => request(`cards/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  moveCard: (id, columnId, position) => request(`cards/${id}/move`, { method: 'PUT', body: JSON.stringify({ columnId, position }) }),
  deleteCard: (id) => request(`cards/${id}`, { method: 'DELETE' }),
}
