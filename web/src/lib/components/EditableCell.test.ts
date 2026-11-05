import { render, screen } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import EditableCell from './EditableCell.svelte'

function setup(props: { value?: string; required?: boolean } = {}) {
  const onSave = vi.fn()
  render(EditableCell, { value: 'Acme', label: 'Company', onSave, ...props })
  return { onSave, input: screen.getByLabelText('Company') as HTMLInputElement, user: userEvent.setup() }
}

describe('EditableCell', () => {
  it('saves a changed value on Enter', async () => {
    const { onSave, input, user } = setup()
    await user.clear(input)
    await user.type(input, 'Beta{Enter}')
    expect(onSave).toHaveBeenCalledExactlyOnceWith('Beta')
  })

  it('saves on blur and trims whitespace', async () => {
    const { onSave, input, user } = setup()
    await user.clear(input)
    await user.type(input, '  Beta  ')
    await user.tab()
    expect(onSave).toHaveBeenCalledExactlyOnceWith('Beta')
  })

  it('does not save when unchanged', async () => {
    const { onSave, input, user } = setup()
    await user.click(input)
    await user.tab()
    expect(onSave).not.toHaveBeenCalled()
  })

  it('reverts on Escape without saving', async () => {
    const { onSave, input, user } = setup()
    await user.clear(input)
    await user.type(input, 'Beta{Escape}')
    expect(onSave).not.toHaveBeenCalled()
    expect(input.value).toBe('Acme')
  })

  it('cannot empty a required field', async () => {
    const { onSave, input, user } = setup({ required: true })
    await user.clear(input)
    await user.tab()
    expect(onSave).not.toHaveBeenCalled()
    expect(input.value).toBe('Acme')
  })
})
