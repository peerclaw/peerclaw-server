export function exportToCSV(
  data: Record<string, unknown>[],
  columns: { key: string; label: string }[],
  filename: string
): void {
  const header = columns.map((c) => c.label).join(",")
  const rows = data.map((row) =>
    columns
      .map((c) => {
        const val = row[c.key]
        const str = val == null ? "" : String(val)
        return str.includes(",") || str.includes('"') || str.includes("\n")
          ? `"${str.replace(/"/g, '""')}"`
          : str
      })
      .join(",")
  )
  const csv = "\uFEFF" + [header, ...rows].join("\n")
  download(csv, `${filename}.csv`, "text/csv;charset=utf-8;")
}

export function exportToJSON(data: unknown, filename: string): void {
  const json = JSON.stringify(data, null, 2)
  download(json, `${filename}.json`, "application/json;charset=utf-8;")
}

function download(content: string, filename: string, mime: string): void {
  const blob = new Blob([content], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
