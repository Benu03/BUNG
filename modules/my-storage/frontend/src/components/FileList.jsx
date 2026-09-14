import { useEffect, useRef, useState } from 'react'
import { Download, File as FileIcon, LoaderCircle, Trash2, TriangleAlert, Upload } from 'lucide-react'
import { api } from '../api.js'
import { Button } from './ui/button.jsx'
import { Card } from './ui/card.jsx'
import { Alert, AlertDescription } from './ui/alert.jsx'

function formatSize(bytes) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export default function FileList() {
  const [files, setFiles] = useState(null)
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const inputRef = useRef(null)

  const load = () => api.listFiles().then(setFiles).catch((e) => setError(e.message))

  useEffect(() => { load() }, [])

  const doUpload = async (fileList) => {
    if (!fileList || fileList.length === 0) return
    setError('')
    setUploading(true)
    try {
      for (const file of fileList) {
        await api.uploadFile(file)
      }
      load()
    } catch (e) {
      setError(e.message)
    } finally {
      setUploading(false)
    }
  }

  const remove = async (id) => {
    if (!confirm('Delete this file?')) return
    try {
      await api.deleteFile(id)
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {error && (
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDragOver(false)
          doUpload(e.dataTransfer.files)
        }}
        className={`flex cursor-pointer flex-col items-center gap-2 rounded-2xl border-2 border-dashed py-10 text-center transition-colors ${
          dragOver ? 'border-primary bg-primary/5' : 'border-border'
        }`}
      >
        <input
          ref={inputRef}
          type="file"
          multiple
          className="hidden"
          onChange={(e) => doUpload(e.target.files)}
        />
        {uploading ? (
          <LoaderCircle className="h-6 w-6 animate-spin text-muted-foreground" />
        ) : (
          <Upload className="h-6 w-6 text-muted-foreground" />
        )}
        <p className="text-sm font-medium">Click to upload, or drag files here</p>
        <p className="text-xs text-muted-foreground">Only visible to you - private per account</p>
      </Card>

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Type</th>
                <th className="px-4 py-3 font-medium">Size</th>
                <th className="px-4 py-3 font-medium">Uploaded</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {(files || []).map((f) => (
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
                      <Button variant="ghost" size="icon" title="Delete" onClick={() => remove(f.id)}>
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {files && files.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">No files yet.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  )
}
