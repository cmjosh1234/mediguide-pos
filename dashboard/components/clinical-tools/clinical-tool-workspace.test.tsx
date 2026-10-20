import * as React from 'react'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { ClinicalToolWorkspace } from './clinical-tool-workspace'
import { BackendRequestError } from '@/lib/backend-client'
import { clinicalToolService } from '@/services/clinical-tool.service'

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

describe('authoring workflow and outline', () => {
  const bmi = envelope('bmi-calculator').definition
  const savedVersion = (overrides: Record<string, unknown>) => ({
    id: 'v1', calculator_id: 'tool', semantic_version: '1.0.0', schema_version: '1.0', definition: bmi,
    definition_checksum: 'x', status: 'draft', change_summary: '', validation_passed: false, tests_passed: false,
    lock_version: 1, created_at: '', updated_at: '', ...overrides,
  }) as unknown as React.ComponentProps<typeof ClinicalToolWorkspace>['initialVersion']

  it('asks for a saved draft before any review step', () => {
    render(<ClinicalToolWorkspace toolId="tool" />)
    expect(screen.getByText('No saved version yet')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Submit for review/ })).toBeNull()
  })

  it('only allows submitting a draft once validation and tests pass', () => {
    const { unmount } = render(<ClinicalToolWorkspace toolId="tool" initialVersion={savedVersion({})} versions={[savedVersion({})!]} />)
    expect(screen.getByRole('button', { name: /Submit for review/ })).toBeDisabled()
    expect(screen.getByText('Validate and run tests to submit for review.')).toBeInTheDocument()
    unmount()
    render(<ClinicalToolWorkspace toolId="tool" initialVersion={savedVersion({ validation_passed: true, tests_passed: true })} />)
    expect(screen.getByRole('button', { name: /Submit for review/ })).toBeEnabled()
  })

  it('offers only the next step for the version status', () => {
    render(<ClinicalToolWorkspace toolId="tool" initialVersion={savedVersion({ status: 'approved', validation_passed: true, tests_passed: true })} />)
    expect(screen.getByRole('button', { name: /^Publish$/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Approve/ })).toBeNull()
    expect(screen.getByText('Approved').closest('[aria-current]')).toHaveAttribute('aria-current', 'step')
  })

  it('summarises the definition in the outline tab', () => {
    render(<ClinicalToolWorkspace toolId="tool" initialVersion={savedVersion({})} />)
    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Outline' }))
    fireEvent.click(screen.getByRole('tab', { name: 'Outline' }))
    expect(screen.getByText('Inputs')).toBeInTheDocument()
    for (const input of bmi.inputs) expect(screen.getAllByText(input.label).length).toBeGreaterThan(0)
  })

  it('flags a definition whose tool_type does not match the tool before saving', () => {
    const create = vi.spyOn(clinicalToolService, 'create')
    render(<ClinicalToolWorkspace toolId="tool" toolType="decision_tool" initialVersion={savedVersion({})} />)
    expect(screen.getByRole('alert')).toHaveTextContent('This tool is a decision tool')
    fireEvent.click(screen.getByRole('button', { name: /Save draft/ }))
    expect(create).not.toHaveBeenCalled()
    create.mockRestore()
  })

  it('lists the backend validation issues when a save is rejected', async () => {
    const create = vi.spyOn(clinicalToolService, 'create').mockRejectedValue(
      new BackendRequestError('calculator definition validation failed', 422, undefined, [{ path: '$.inputs[0].maximum', code: 'range', message: 'maximum must be greater than minimum' }]),
    )
    const { container } = render(<ClinicalToolWorkspace toolId="tool" toolType="calculator" />)
    await importFile(container, envelope('bmi-calculator'))
    fireEvent.click(screen.getByRole('button', { name: /Save draft/ }))
    expect(await screen.findByText('maximum must be greater than minimum', { exact: false })).toBeInTheDocument()
    expect(screen.getByText('$.inputs[0].maximum')).toBeInTheDocument()
    create.mockRestore()
  })
})
