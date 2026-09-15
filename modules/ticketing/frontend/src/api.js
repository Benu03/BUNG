// BASE_URL is set by vite's `base` config (see vite.config.js), so this
// automatically points at /ticketing/api/ in this module, and at
// /<module>/api/ if this folder is duplicated for a new module.
const API_BASE = `${import.meta.env.BASE_URL}api/`

async function request(path, options = {}) {
  // FormData (file uploads) must NOT get a manual Content-Type - the
  // browser sets its own with the correct multipart boundary. Only
  // JSON-body calls get the default header.
  const isFormData = options.body instanceof FormData
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: isFormData ? options.headers : { 'Content-Type': 'application/json', ...options.headers },
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

  uploadAttachment: (ticketId, file) => {
    const form = new FormData()
    form.append('file', file)
    return request(`tickets/${ticketId}/attachments`, { method: 'POST', body: form })
  },
  deleteAttachment: (ticketId, attachmentId) => request(`tickets/${ticketId}/attachments/${attachmentId}`, { method: 'DELETE' }),
  // Not fetched via JS - handed straight to the browser (<a href>) so it
  // streams and shows a native download, using the same-origin session
  // cookie automatically.
  attachmentDownloadUrl: (ticketId, attachmentId) => `${API_BASE}tickets/${ticketId}/attachments/${attachmentId}/download`,

  listUsers: () => request('users'),
}
