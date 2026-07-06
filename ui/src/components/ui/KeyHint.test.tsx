import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeAll, afterAll, beforeEach } from 'vitest'
import { KeyHint } from './KeyHint'

const writeText = vi.fn().mockResolvedValue(undefined)
const originalClipboard = navigator.clipboard

beforeAll(() => {
  Object.assign(navigator, { clipboard: { writeText } })
})

afterAll(() => {
  try {
    Object.assign(navigator, { clipboard: originalClipboard })
  } catch {
    /* jsdom may block restoring clipboard */
  }
})

beforeEach(() => {
  writeText.mockClear()
})

describe('KeyHint', () => {
  describe('Rendering', () => {
    it('renders the full hint text', () => {
      render(<KeyHint hint="sk-a3f2...2ad6" />)
      expect(screen.getByText('2ad6')).toBeInTheDocument()
    })

    it('renders prefix in tertiary color', () => {
      render(<KeyHint hint="sk-a3f2...2ad6" />)
      const prefix = screen.getByText('sk-a3f2...')
      expect(prefix.className).toContain('text-text-tertiary')
    })

    it('splits on ellipsis correctly for user key', () => {
      render(<KeyHint hint="sk-a3f2...e8b1" />)
      expect(screen.getByText('sk-a3f2...')).toBeInTheDocument()
      expect(screen.getByText('e8b1')).toBeInTheDocument()
    })

    it('splits on ellipsis correctly for session key', () => {
      render(<KeyHint hint="sk-abcd...0684" />)
      expect(screen.getByText('sk-abcd...')).toBeInTheDocument()
      expect(screen.getByText('0684')).toBeInTheDocument()
    })

    it('renders hint without ellipsis as plain text', () => {
      render(<KeyHint hint="somekey" />)
      expect(screen.getByText('somekey')).toBeInTheDocument()
    })
  })

  describe('Copy', () => {
    it('does not show copy button without copyValue', () => {
      render(<KeyHint hint="sk-a3f2...2ad6" />)
      expect(screen.queryByRole('button')).not.toBeInTheDocument()
    })

    it('copies copyValue when the button is clicked', async () => {
      render(<KeyHint hint="sk-a3f2...2ad6" copyValue="sk-a3f2fullkey2ad6" copyLabel="Copy key" />)
      fireEvent.click(screen.getByRole('button', { name: 'Copy key' }))
      await waitFor(() => expect(writeText).toHaveBeenCalledWith('sk-a3f2fullkey2ad6'))
    })
  })

  describe('Styling', () => {
    it('hint text has font-mono class', () => {
      render(<KeyHint hint="sk-a3f2...2ad6" />)
      const hint = screen.getByText('2ad6').closest('span.font-mono')
      expect(hint?.className).toContain('font-mono')
    })

    it('hint text has text-xs class', () => {
      render(<KeyHint hint="sk-a3f2...2ad6" />)
      const hint = screen.getByText('2ad6').closest('span.text-xs')
      expect(hint?.className).toContain('text-xs')
    })
  })

  describe('Native attributes', () => {
    it('passes data-testid', () => {
      render(<KeyHint hint="sk-a3f2...2ad6" data-testid="my-hint" />)
      expect(screen.getByTestId('my-hint')).toBeInTheDocument()
    })

    it('merges custom className', () => {
      render(<KeyHint hint="sk-a3f2...2ad6" className="extra" data-testid="kh" />)
      expect(screen.getByTestId('kh').className).toContain('extra')
    })
  })
})
