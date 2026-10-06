import * as React from 'react'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { ClinicalToolWorkspace } from './clinical-tool-workspace'

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }))
afterEach(cleanup)

const envelope = (name: string) => JSON.parse(readFileSync(join(process.cwd(), '..', 'clinical-tools', 'migrations', 'v1', 'definitions', `${name}.json`), 'utf8'))
const importFile = async (container: HTMLElement, contents: unknown) => {
  const file = new File([], 'clinical-tool.json', { type: 'application/json' })
  Object.defineProperty(file, 'text', { value: async () => JSON.stringify(contents) })
  fireEvent.change(container.querySelector('input[type=file]')!, { target: { files: [file] } })
  await waitFor(() => expect(screen.getByTestId('native-clinical-tool')).toBeInTheDocument())
}

describe('clinical authoring preview', () => {
  it('imports repository envelopes and previews the actual radio and numeric controls', async () => {
    const { container } = render(<ClinicalToolWorkspace toolId="test-tool" />)
    await importFile(container, envelope('apgar-score-calculator'))
    expect(screen.getByText('unsaved · 1.0.0-review.3')).toBeInTheDocument()
    expect(screen.getAllByRole('radio')).toHaveLength(15)
    expect(screen.getByLabelText('1 Minute Score').getAttribute('type')).toBe('number')
    expect(screen.getByLabelText('5 Minutes Score').getAttribute('type')).toBe('number')
  })
  it('also imports plain definitions and previews conditional PQRST fields', async () => {
    const { container } = render(<ClinicalToolWorkspace toolId="test-tool" />)
    await importFile(container, envelope('pain-assessment-scale').definition)
    expect(screen.queryByLabelText(/P - Provocation/)).toBeNull()
    fireEvent.change(screen.getByLabelText('Pain scale *'), { target: { value: 'pqrst' } })
    expect(screen.getByLabelText(/P - Provocation/).tagName).toBe('TEXTAREA')
    expect(screen.getByLabelText(/Q - Quality/).tagName).toBe('SELECT')
  })
})
