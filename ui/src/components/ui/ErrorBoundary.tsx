import React from 'react'
import { Button } from './Button'

interface ErrorBoundaryProps {
  children: React.ReactNode
}

interface ErrorBoundaryState {
  error: Error | null
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error('UI error:', error, info.componentStack)
  }

  render() {
    if (this.state.error) {
      return (
        <div className="min-h-screen flex items-center justify-center bg-bg-primary px-4 text-text-primary">
          <div className="w-full max-w-md rounded-xl border border-border bg-bg-secondary p-8 text-center space-y-4">
            <h1 className="text-lg font-semibold">Something went wrong</h1>
            <p className="text-sm text-text-secondary">
              The page failed to load. This often happens after an update — reload to fetch the
              latest UI.
            </p>
            <p className="text-xs text-text-tertiary break-all">{this.state.error.message}</p>
            <div className="flex justify-center gap-3">
              <Button onClick={() => window.location.reload()}>Reload page</Button>
              <Button variant="secondary" onClick={() => window.location.replace('/login')}>
                Back to login
              </Button>
            </div>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}