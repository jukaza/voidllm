import { LOCAL_STORAGE_KEY } from './constants'

type UnauthorizedHandler = () => void

let unauthorizedHandler: UnauthorizedHandler | null = null

export function setUnauthorizedHandler(handler: UnauthorizedHandler | null) {
  unauthorizedHandler = handler
}

/** Clear session and navigate to login without a full page reload when possible. */
export function handleUnauthorized() {
  localStorage.removeItem(LOCAL_STORAGE_KEY)
  // Defer navigation so open dialogs/toasts can unmount from the portal root first.
  const run = () => {
    if (unauthorizedHandler) {
      unauthorizedHandler()
      return
    }
    const path = window.location.pathname
    if (path !== '/login' && path !== '/register') {
      window.location.replace('/login')
    }
  }
  queueMicrotask(run)
}