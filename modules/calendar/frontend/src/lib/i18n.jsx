import { createContext, useContext, useEffect, useState } from 'react'
import en from '../locales/en.js'
import id from '../locales/id.js'

const dictionaries = { en, id }
const STORAGE_KEY = 'bung-lang'

// Default is Indonesian - flip with the toggle (see LanguageToggle.jsx),
// persisted the same way as the theme choice (localStorage, read once on
// mount). No OS/browser-language detection - explicit only.
const LanguageContext = createContext({ lang: 'id', setLang: () => {}, t: (key) => key })

export function LanguageProvider({ children }) {
  const [lang, setLang] = useState(() => localStorage.getItem(STORAGE_KEY) || 'id')

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, lang)
    } catch {
      // ignore (private browsing, storage disabled, etc.)
    }
  }, [lang])

  const t = (key) => {
    const dict = dictionaries[lang] || dictionaries.id
    const value = key.split('.').reduce((obj, part) => (obj == null ? undefined : obj[part]), dict)
    return value ?? key
  }

  return <LanguageContext.Provider value={{ lang, setLang, t }}>{children}</LanguageContext.Provider>
}

export function useTranslation() {
  return useContext(LanguageContext)
}
