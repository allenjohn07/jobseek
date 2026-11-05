<script lang="ts">
  // A borderless input that looks like plain text until hovered/focused (Notion-style).
  // Saves on Enter or blur if the value changed; Escape reverts.
  let {
    value,
    onSave,
    type = 'text',
    placeholder = '',
    required = false,
    class: className = '',
    label,
    multiline = false,
    display,
  }: {
    value: string
    onSave: (value: string) => void
    type?: 'text' | 'date' | 'url'
    placeholder?: string
    required?: boolean
    class?: string
    label: string
    multiline?: boolean // textarea: Enter adds a new line, clicking away saves
    display?: (value: string) => string // how to show the value while NOT focused
  } = $props()

  let draft = $state('')
  let focused = $state(false)
  // Keep in sync when the parent value changes (e.g. after a rollback or reload).
  $effect(() => {
    draft = value
  })

  function commit() {
    focused = false
    const next = draft.trim()
    if (next === value) return
    if (required && !next) {
      draft = value // required fields can't be emptied
      return
    }
    onSave(next)
  }
</script>

{#if multiline}
  <textarea
    aria-label={label}
    {placeholder}
    rows="3"
    bind:value={draft}
    onblur={commit}
    onkeydown={(e) => {
      if (e.key === 'Escape') {
        draft = value
        ;(e.currentTarget as HTMLTextAreaElement).blur()
      }
    }}
    class="field-sizing-content min-h-12 w-full resize-none rounded-sm border-0 bg-transparent px-2 py-1.5 outline-none placeholder:text-muted-foreground/50 hover:bg-accent/60 focus:bg-accent {className}"
  ></textarea>
{:else}
  <input
    {type}
    {placeholder}
    aria-label={label}
    value={display && !focused ? display(draft) : draft}
    oninput={(e) => (draft = e.currentTarget.value)}
    onfocus={() => (focused = true)}
    onblur={commit}
    onkeydown={(e) => {
      if (e.key === 'Enter') (e.currentTarget as HTMLInputElement).blur()
      if (e.key === 'Escape') {
        draft = value
        ;(e.currentTarget as HTMLInputElement).blur()
      }
    }}
    class="w-full min-w-0 truncate rounded-sm border-0 bg-transparent px-2 py-1.5 outline-none placeholder:text-muted-foreground/50 hover:bg-accent/60 focus:bg-accent {className}"
  />
{/if}
