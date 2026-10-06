import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
// @ts-expect-error -- jsdom types are optional in this repository.
import { JSDOM } from 'jsdom'
import type { ClinicalToolDefinition } from '@/services/clinical-tool.service'
import { clinicalToolInitialInputs, clinicalToolInputVisible, previewClinicalTool } from './clinical-tool-evaluator'
const directory = join(process.cwd(), '..', 'clinical-tools', 'migrations', 'v1')
const load = (name: string): ClinicalToolDefinition => JSON.parse(readFileSync(join(directory, 'definitions', `${name}.json`), 'utf8')).definition
const bindings = JSON.parse(readFileSync(join(directory, 'html-control-bindings.json'), 'utf8')) as Record<string, Array<{html_id: string; input_key: string; value?: unknown}>>
const normalize = (value: string) => value.replace(/\s+/g, ' ').trim()
describe('HTML controls preserved in schema definitions', () => {
  for (const [name, controls] of Object.entries(bindings)) it(`${name}: named checkboxes and scored choices`, () => {
    const definition = load(name)
    const dom = new JSDOM(readFileSync(join(process.cwd(), 'samples', `${name}.html`), 'utf8'))
    try {
      for (const binding of controls) {
        const control = dom.window.document.getElementById(binding.html_id)
        expect(control, binding.html_id).not.toBeNull()
        const input = definition.inputs.find((item) => item.key === binding.input_key)!
        expect(input, binding.input_key).toBeDefined()
        const label = dom.window.document.querySelector(`label[for="${binding.html_id}"]`) ?? control.closest('label')
        if ('value' in binding) {
          expect(input.type).toBe('single_selection')
          const option = input.options!.find((item) => item.value === binding.value)!
          expect(String(option.value)).toBe(control.value)
          expect(normalize(option.label)).toBe(normalize(label.textContent))
        } else { expect(input.type).toBe('boolean'); expect(input.default).toBe(false) }
      }
      for (const checkbox of dom.window.document.querySelectorAll('input[type=checkbox]')) expect(controls.some((item) => item.html_id === checkbox.id), checkbox.id).toBe(true)
    } finally { dom.window.close() }
  })
})
describe('counts and priorities are derived', () => {
  const aggregates: Record<string, string[]> = {
    'dehydration-assessment': ['additional_symptoms'], 'pediatric-fever-management': ['danger_sign_count'],
    'cardiac-risk-assessment': ['risk_factor_count'],
    'emergency-triage-assessment': ['has_priority_1_sign', 'has_priority_2_sign', 'has_priority_3_sign', 'has_priority_4_sign'],
    'wound-assessment-tool': ['infection_sign_count', 'risk_factor_count'],
    'immunization-schedule-checker': ['has_pneumococcal_risk_condition'],
  }
  for (const [name, keys] of Object.entries(aggregates)) for (const key of keys) it(`${name}/${key}`, () => {
    const definition = load(name)
    expect(definition.inputs.some((item) => item.key === key)).toBe(false)
    const calculation = definition.calculation.find((item) => item.key === key)!
    const isolated = { ...definition, inputs: definition.inputs.filter((item) => item.type === 'boolean'), calculation: [calculation], rules: [], interpretations: [], warnings: [], outputs: [{key:'aggregate',label:'Aggregate',value:{op:'field',field:key}}] }
    const refs = calculation.expression.args!.map((arg) => arg.op === 'field' ? arg.field! : arg.args![0].field!)
    const any = calculation.expression.op === 'or'
    expect(previewClinicalTool(isolated, {}).values.aggregate).toBe(any ? false : 0)
    for (const selected of refs) expect(previewClinicalTool(isolated, {[selected]:true}).values.aggregate).toBe(any ? true : 1)
    expect(previewClinicalTool(isolated, Object.fromEntries(refs.map((ref) => [ref,true]))).values.aggregate).toBe(any ? true : refs.length)
    expect(() => previewClinicalTool(definition, {[key]:1})).toThrow('Unknown input')
  })
})
describe('dynamic form contract', () => {
  it('initializes defaults and switches pregnancy fields', () => {
    const definition = load('pregnancy-due-date-calculator'), inputs = clinicalToolInitialInputs(definition)
    expect(inputs.method).toBe('lmp'); expect(inputs.cycle_length).toBe(28)
    expect(clinicalToolInputVisible(definition,'lmp_date',inputs)).toBe(true)
    expect(clinicalToolInputVisible(definition,'ultrasound_weeks',inputs)).toBe(false)
    expect(clinicalToolInputVisible(definition,'ultrasound_weeks',{...inputs,method:'ultrasound'})).toBe(true)
  })
  it('requires custom dose fields only for custom medication', () => {
    const definition = load('medication-dosage-calculator'), fixture = definition.test_cases[0]
    expect(clinicalToolInputVisible(definition,'custom_mg_per_kg',fixture.inputs)).toBe(false)
    expect(() => previewClinicalTool(definition,fixture.inputs)).not.toThrow()
    expect(() => previewClinicalTool(definition,{...fixture.inputs,medication:'custom'})).toThrow('Required input')
  })
  it.each([NaN,Infinity,'70',-1,{value:-1,unit:'kg'},{value:70,unit:'cm'},{value:70,unit:'stone'}])('rejects invalid BMI measurement %j', (weight) => {
    expect(() => previewClinicalTool(load('bmi-calculator'),{weight,height:{value:170,unit:'cm'}})).toThrow()
  })
  it('checks measurement bounds after conversion', () => {
    const definition = load('bmi-calculator'); definition.inputs.find((input) => input.key === 'weight')!.maximum = 50
    expect(() => previewClinicalTool(definition,{weight:{value:100,unit:'lb'},height:{value:170,unit:'cm'}})).not.toThrow()
    expect(() => previewClinicalTool(definition,{weight:{value:120,unit:'lb'},height:{value:170,unit:'cm'}})).toThrow('maximum')
  })
  it('rejects fractional integers, unknown options and impossible dates', () => {
    expect(() => previewClinicalTool(load('blood-pressure-assessment'),{systolic:120.5,diastolic:80})).toThrow('numeric')
    expect(() => previewClinicalTool(load('apgar-score-calculator'),{activity:99})).toThrow('option')
    expect(() => previewClinicalTool(load('pregnancy-due-date-calculator'),{lmp_date:'2026-02-30'})).toThrow('date')
  })
})

const fieldBindings = JSON.parse(readFileSync(join(directory, 'html-field-bindings.json'), 'utf8')) as Record<string, Array<{
  html_id: string; input_key: string; value?: unknown;
  option_values?: Record<string, unknown>; unit_values?: Record<string, string>;
}>>

describe('complete HTML field coverage', () => {
  it('covers every sample, including tools with no checkbox controls', () => {
    const catalog = JSON.parse(readFileSync(join(directory, 'catalog.json'), 'utf8'))
    expect(Object.keys(fieldBindings).sort()).toEqual(catalog.tools.map((tool: { legacy_id: string }) => tool.legacy_id).sort())
  })
  for (const [name, controls] of Object.entries(fieldBindings)) it(`${name}: every HTML field and option has a typed binding`, () => {
    const definition = load(name)
    const dom = new JSDOM(readFileSync(join(process.cwd(), 'samples', `${name}.html`), 'utf8'))
    try {
      const elements = [...dom.window.document.querySelectorAll('input, select, textarea')]
      expect(controls.map((binding) => binding.html_id).sort()).toEqual(elements.map((element) => element.id).sort())
      for (const element of elements) {
        const binding = controls.find((item) => item.html_id === element.id)!
        const field = definition.inputs.find((item) => item.key === binding.input_key)!
        expect(field, element.id).toBeDefined()
        expect(definition.sections.some((section) => section.key === field.section_key), field.key).toBe(true)
        if (element.type === 'checkbox') expect(field.type).toBe('boolean')
        else if (element.type === 'radio') {
          expect(field.type).toBe('single_selection')
          expect(field.control).toBe('radio')
          expect(field.options!.some((option) => option.value === binding.value), element.id).toBe(true)
          expect(String(binding.value)).toBe(element.value)
        } else if (element.tagName === 'TEXTAREA') {
          expect(field.type).toBe('text'); expect(field.control).toBe('textarea')
        } else if (element.tagName === 'SELECT') {
          const options = [...element.options].filter((option) => option.value !== '')
          if (binding.unit_values) {
            expect(Object.keys(binding.unit_values)).toEqual(options.map((option) => option.value))
            for (const option of options) expect(field.allowed_units).toContain(binding.unit_values[option.value])
          } else {
            expect(field.type).toBe('single_selection'); expect(field.control).toBe('select')
            expect(Object.keys(binding.option_values!)).toEqual(options.map((option) => option.value))
            for (const option of options) {
              const mapped = field.options!.find((item) => item.value === binding.option_values![option.value])!
              expect(mapped, option.value).toBeDefined()
              expect(normalize(mapped.label)).toBe(normalize(option.textContent))
            }
          }
        } else if (element.type === 'number') {
          expect(['integer', 'number', 'measurement']).toContain(field.type)
          if (element.step) expect(field.step).toBe(Number(element.step))
        } else expect(field.type).toBe(element.type)
      }
    } finally { dom.window.close() }
  })
  it('records optional context without changing calculated results', () => {
    const contexts: Record<string, Record<string, unknown>> = {
      'apgar-score-calculator': {score_1min: 3, score_5min: 8},
      'cardiac-risk-assessment': {ldl_cholesterol: 110, triglycerides: 150, statin_therapy: 'yes'},
      'emergency-triage-assessment': {chief_complaint: 'Headache'},
      'fluid-balance-calculator': {hour_1_4: 100, hour_5_8: 200, hour_9_12: 100, hour_13_16: 200, hour_17_20: 100, hour_21_24: 200},
      'pain-assessment-scale': {pqrst_provocation: 'Movement', pqrst_quality: 'sharp', pqrst_region: 'Lower back', pqrst_timing: 'Intermittent'},
    }
    for (const [name, context] of Object.entries(contexts)) {
      const definition = load(name), fixture = definition.test_cases[0]
      const inputs = name === 'pain-assessment-scale' ? {...fixture.inputs, scale: 'pqrst', direct_score: 3} : fixture.inputs
      expect(previewClinicalTool(definition, {...inputs, ...context}, {fixedNow: fixture.fixed_now}).values).toEqual(previewClinicalTool(definition, inputs, {fixedNow: fixture.fixed_now}).values)
    }
  })
})
