// BASE_URL is set by vite's `base` config (see vite.config.js), so this
// automatically points at /my-storage/api/ in this module, and at
// /<module>/api/ if this folder is duplicated for a new module.
const API_BASE = `${import.meta.env.BASE_URL}api/`

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, options)
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `Request failed: ${res.status}`)
  }
  if (res.status === 204) return null
  return res.json()
}

export const api = {
  listFiles: () => request('files'),

  uploadFile: (file) => {
    const form = new FormData()
    form.append('file', file)
    return request('files', { method: 'POST', body: form })
  },

  deleteFile: (id) => request(`files/${id}`, { method: 'DELETE' }),

  // Not fetched via JS - handed straight to the browser (<a href>) so it
  // streams and shows a native download, using the same-origin session
  // cookie automatically.
  downloadUrl: (id) => `${API_BASE}files/${id}/download`,
}
