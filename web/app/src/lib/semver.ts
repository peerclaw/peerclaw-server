/**
 * Returns true if `current` is older than `latest`.
 * Both must be semver strings (with optional "v" prefix).
 * Returns false if either cannot be parsed.
 */
export function isOutdated(current: string, latest: string): boolean {
  const a = parseSemver(current)
  const b = parseSemver(latest)
  if (!a || !b) return false
  for (let i = 0; i < 3; i++) {
    if (a[i] < b[i]) return true
    if (a[i] > b[i]) return false
  }
  return false
}

function parseSemver(v: string): [number, number, number] | null {
  const s = v.replace(/^v/, "")
  const parts = s.split(".")
  if (parts.length !== 3) return null
  const nums = parts.map((p) => parseInt(p.split("-")[0], 10))
  if (nums.some(isNaN)) return null
  return nums as [number, number, number]
}
