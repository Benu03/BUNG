import { useEffect, useRef, useState } from 'react'
import {
  Download, File as FileIcon, Folder as FolderIcon, FolderPlus, Home,
  LoaderCircle, Search, Trash2, TriangleAlert, Upload,
} from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Input } from './ui/input.jsx'
import { Card } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'
import { useTranslation } from '../lib/i18n.jsx'

function formatSize(bytes) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export default function FileList() {
  const { t } = useTranslation()
  const [folderId, setFolderId] = useState('')
  const [data, setData] = useState(null) // { folder, breadcrumb, folders, files }
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const [showNewFolder, setShowNewFolder] = useState(false)
  const [newFolderName, setNewFolderName] = useState('')
  const [filter, setFilter] = useState('')
  const inputRef = useRef(null)

  const load = (fid = folderId) => api.browse(fid).then(setData).catch((e) => setError(e.message))

  useEffect(() => { load(folderId) }, [folderId])

  const openFolder = (id) => setFolderId(id)

  const doUpload = async (fileList) => {
    if (!fileList || fileList.length === 0) return
    setError('')
    setUploading(true)
    try {
      for (const file of fileList) {
        await api.uploadFile(file, folderId)
      }
      load()
    } catch (e) {
      setError(e.message)
    } finally {
      setUploading(false)
    }
  }

  const createFolder = async (e) => {
    e.preventDefault()
    if (!newFolderName.trim()) return
    try {
      await api.createFolder(newFolderName.trim(), folderId)
      setNewFolderName('')
      setShowNewFolder(false)
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  const removeFolder = async (id, e) => {
    e.stopPropagation()
    if (!confirm(t('fileList.confirmDeleteFolder'))) return
    try {
      await api.deleteFolder(id)
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  const removeFile = async (id) => {
    if (!confirm(t('fileList.confirmDeleteFile'))) return
    try {
      await api.deleteFile(id)
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  const q = filter.trim().toLowerCase()
  const folders = (data?.folders || []).filter((f) => !q || f.name.toLowerCase().includes(q))
  const files = (data?.files || []).filter((f) => !q || f.filename.toLowerCase().includes(q))
  const isEmpty = data && folders.length === 0 && files.length === 0

  return (
    <div className="flex flex-col gap-6">
      {error && (
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Breadcrumb */}
      <div className="flex flex-wrap items-center gap-1 text-sm text-muted-foreground">
        <button onClick={() => setFolderId('')} className="flex items-center gap-1 rounded px-1.5 py-0.5 hover:bg-accent hover:text-foreground">
          <Home className="h-3.5 w-3.5" />
          {t('fileList.home')}
        </button>
        {data?.breadcrumb?.map((b) => (
          <span key={b.id} className="flex items-center gap-1">
            <span>/</span>
            <button onClick={() => setFolderId(b.id)} className="rounded px-1.5 py-0.5 hover:bg-accent hover:text-foreground">
              {b.name}
            </button>
          </span>
        ))}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" onClick={() => setShowNewFolder((v) => !v)}>
          <FolderPlus />
          {t('fileList.newFolder')}
        </Button>
        <Button size="sm" onClick={() => inputRef.current?.click()} disabled={uploading}>
          {uploading ? <LoaderCircle className="animate-spin" /> : <Upload />}
          {t('fileList.upload')}
        </Button>
        <input ref={inputRef} type="file" multiple className="hidden" onChange={(e) => doUpload(e.target.files)} />
      </div>

      {showNewFolder && (
        <form onSubmit={createFolder} className="flex max-w-sm gap-2">
          <Input autoFocus placeholder={t('fileList.folderNamePlaceholder')} value={newFolderName} onChange={(e) => setNewFolderName(e.target.value)} />
          <Button type="submit" size="sm">{t('fileList.create')}</Button>
          <Button type="button" variant="ghost" size="sm" onClick={() => setShowNewFolder(false)}>{t('common.cancel')}</Button>
        </form>
      )}

      {data && (data.folders.length > 0 || data.files.length > 0) && (
        <div className="flex items-center gap-2 rounded-lg border bg-card px-3 py-2">
          <Search className="h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t('common.filterPlaceholder')}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="h-8 max-w-xs border-0 shadow-none focus-visible:ring-0"
          />
        </div>
      )}

      <Card
        onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => { e.preventDefault(); setDragOver(false); doUpload(e.dataTransfer.files) }}
        className={`overflow-hidden transition-colors ${dragOver ? 'border-primary bg-primary/5' : ''}`}
      >
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-4 py-3 font-medium">{t('fileList.colName')}</th>
                <th className="px-4 py-3 font-medium">{t('fileList.colType')}</th>
                <th className="px-4 py-3 font-medium">{t('fileList.colSize')}</th>
                <th className="px-4 py-3 font-medium">{t('fileList.colCreated')}</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {folders.map((f) => (
                <tr key={f.id} onClick={() => openFolder(f.id)} className="cursor-pointer border-b last:border-0 hover:bg-muted/40">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2 font-medium">
                      <FolderIcon className="h-4 w-4 shrink-0 text-amber-500" />
                      {f.name}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{t('fileList.typeFolder')}</td>
                  <td className="px-4 py-3 text-muted-foreground">-</td>
                  <td className="px-4 py-3 text-muted-foreground">{new Date(f.createdAt).toLocaleString()}</td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end">
                      <Button variant="ghost" size="icon" title={t('common.delete')} onClick={(e) => removeFolder(f.id, e)}>
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}

              {files.map((f) => (
                <tr key={f.id} className="border-b last:border-0 hover:bg-muted/40">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2 font-medium">
                      <FileIcon className="h-4 w-4 shrink-0 text-muted-foreground" />
                      {f.filename}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{f.contentType}</td>
                  <td className="px-4 py-3 text-muted-foreground">{formatSize(f.size)}</td>
                  <td className="px-4 py-3 text-muted-foreground">{new Date(f.createdAt).toLocaleString()}</td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end gap-1">
                      <Button variant="ghost" size="icon" title="Download" asChild>
                        <a href={api.downloadUrl(f.id)}>
                          <Download className="h-4 w-4" />
                        </a>
                      </Button>
                      <Button variant="ghost" size="icon" title={t('common.delete')} onClick={() => removeFile(f.id)}>
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}

              {isEmpty && (
                <tr>
                  <td colSpan={5} className="px-4 py-10 text-center text-muted-foreground">
                    {q ? t('common.noMatches') : t('fileList.empty')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>

      <p className="text-center text-xs text-muted-foreground">{t('fileList.footerHint')}</p>
    </div>
  )
}
