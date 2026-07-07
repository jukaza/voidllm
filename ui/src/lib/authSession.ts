import { LOCAL_STORAGE_KEY } from './constants'

/** Full page logout — avoids SPA navigation DOM races entirely. */
export function logoutToLogin() {
  localStorage.removeItem(LOCAL_STORAGE_KEY)
  window.location.replace('/login')
}