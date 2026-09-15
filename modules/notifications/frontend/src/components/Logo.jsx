// BUNG's mark: a monogram "B" in a rounded square, using the same primary
// gradient as the rest of the UI (see index.css --primary/--primary-dark).
// Self-contained (own background) - render it directly, no wrapper div
// needed. Also mirrored as a static file at public/favicon.svg for the
// browser tab icon.
export default function Logo({ className = 'h-8 w-8' }) {
  return (
    <svg viewBox="0 0 32 32" className={className} xmlns="http://www.w3.org/2000/svg" role="img" aria-label="BUNG">
      <rect width="32" height="32" rx="9" fill="url(#bung-logo-gradient)" />
      <path
        d="M11 8.5h6.2c2.1 0 3.6 1.2 3.6 3.1 0 1.3-.7 2.2-1.8 2.7 1.5.4 2.4 1.5 2.4 3.1 0 2.1-1.7 3.4-4 3.4H11V8.5Zm5.7 5c1 0 1.6-.5 1.6-1.4s-.6-1.4-1.6-1.4h-3v2.8h3Zm.3 5.2c1.1 0 1.8-.6 1.8-1.5s-.7-1.5-1.8-1.5h-3.3v3h3.3Z"
        fill="white"
      />
      <defs>
        <linearGradient id="bung-logo-gradient" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
          <stop stopColor="#8a5f7f" />
          <stop offset="1" stopColor="#4f3348" />
        </linearGradient>
      </defs>
    </svg>
  )
}
