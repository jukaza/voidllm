import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { AppTopBar } from './AppTopBar'

export function Shell() {
  return (
    <div className="min-h-screen bg-bg-primary">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:p-3 focus:bg-accent focus:text-white focus:rounded-md focus:m-2"
      >
        Skip to content
      </a>
      <Sidebar />
      <div className="ml-[13rem] max-w-[calc(100%-13rem)] flex min-h-screen flex-col">
        <AppTopBar />
        <main id="main-content" className="flex-1 p-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
