/** Stable portal mount outside #root — avoids insertBefore races when the app tree unmounts. */
export function getPortalRoot(): HTMLElement {
  let el = document.getElementById('portal-root')
  if (!el) {
    el = document.createElement('div')
    el.id = 'portal-root'
    document.body.appendChild(el)
  }
  return el
}