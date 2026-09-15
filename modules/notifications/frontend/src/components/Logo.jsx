// BUNG's mark - a raster logo provided directly by the user (see
// public/logo.png, already a rounded-square icon with its own padding/
// transparency baked in). Rendered directly via <img>; className controls
// sizing exactly like the previous inline-SVG monogram did (e.g. `h-8
// w-8`). The browser tab icon points at the same file - see index.html.
export default function Logo({ className = 'h-8 w-8' }) {
  return <img src="/logo.png" alt="BUNG" className={`${className} object-contain`} />
}
