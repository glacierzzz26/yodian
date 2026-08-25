// 菜品规格解析：后端 specs 为自由 JSONB（无强 schema，设计 810 仅随小票展示，后端不计加价）。
// 本仓库约定的可展示形态：[{"name":规格组, "options":[{"name":选项, "price_delta":分}]}]
// 解析失败/为空 → 返回 null，视为「无规格」直接加购。
export interface SpecGroup {
  name: string
  options: Array<{ name: string; price_delta?: number }>
}

export function parseSpecGroups(specs: unknown): SpecGroup[] | null {
  if (!Array.isArray(specs) || specs.length === 0) return null
  const groups: SpecGroup[] = []
  for (const g of specs) {
    if (!g || typeof g !== 'object') continue
    const name = (g as { name?: unknown }).name
    const opts = (g as { options?: unknown }).options
    if (typeof name !== 'string' || !name) continue
    if (!Array.isArray(opts) || opts.length === 0) continue
    const options: SpecGroup['options'] = []
    for (const o of opts) {
      if (!o || typeof o !== 'object') continue
      const oname = (o as { name?: unknown }).name
      if (typeof oname !== 'string' || !oname) continue
      options.push({ name: oname })
    }
    if (options.length === 0) continue
    groups.push({ name, options })
  }
  return groups.length > 0 ? groups : null
}

// 默认勾选：每组第一项。返回 {组名: 选项名}
export function defaultSelection(groups: SpecGroup[]): Record<string, string> {
  const sel: Record<string, string> = {}
  for (const g of groups) sel[g.name] = g.options[0].name
  return sel
}

// 展示标签：如「中辣 · 大份」
export function specLabel(sel: Record<string, string>): string {
  return Object.keys(sel)
    .map((k) => sel[k])
    .join(' · ')
}
