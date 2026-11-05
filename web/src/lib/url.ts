// URLs may be typed without a scheme ("google.com/careers"). Store them with
// https:// so links work, and show only the domain in the table.

const HAS_SCHEME = /^[a-z][a-z0-9+.-]*:\/\//i

/** "google.com/careers" -> "https://google.com/careers"; "" stays ""; existing schemes are kept. */
export function normalizeUrl(raw: string): string {
  const t = raw.trim()
  if (!t) return ''
  return HAS_SCHEME.test(t) ? t : 'https://' + t.replace(/^\/\//, '')
}

/** "https://www.google.com/careers/x" or "google.com/careers" -> "google.com". */
export function hostname(raw: string): string {
  try {
    return new URL(normalizeUrl(raw)).hostname.replace(/^www\./, '')
  } catch {
    return raw
  }
}
