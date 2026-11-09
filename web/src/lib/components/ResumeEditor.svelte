<script lang="ts">
  import EditableCell from '$lib/components/EditableCell.svelte'
  import IconButton from '$lib/components/IconButton.svelte'
  import PlusIcon from '@lucide/svelte/icons/plus'
  import Trash2Icon from '@lucide/svelte/icons/trash-2'
  import { clone, newEducation, newExperience, newItem, newProject } from '$lib/resume'
  import type { Basics, Item, ResumeContent } from '$lib/types'

  // Edits a resume. It never mutates `content`: each change is applied to a copy and
  // handed to `onChange`, and the parent decides whether it saved.
  let { content, onChange }: { content: ResumeContent; onChange: (next: ResumeContent) => void } = $props()

  function edit(fn: (c: ResumeContent) => void) {
    const next = clone(content)
    fn(next)
    onChange(next)
  }

  const basicsFields: { key: keyof Basics; label: string }[] = [
    { key: 'name', label: 'Name' },
    { key: 'headline', label: 'Headline' },
    { key: 'email', label: 'Email' },
    { key: 'phone', label: 'Phone' },
    { key: 'location', label: 'Location' },
    { key: 'links', label: 'Links' },
  ]

  const heading = 'mt-9 mb-4 flex flex-wrap items-center justify-between gap-2 border-b pb-3 text-sm font-semibold tracking-tight'
  const addBtn =
    'flex items-center gap-1 rounded-md border bg-background px-2.5 py-1.5 text-xs font-medium tracking-normal text-muted-foreground transition-colors hover:bg-accent hover:text-foreground'
</script>

{#snippet add(label: string, onclick: () => void)}
  <button type="button" {onclick} class={addBtn}><PlusIcon class="size-3.5" />{label}</button>
{/snippet}

{#snippet bullets(list: Item[], noun: string, update: (fn: (l: Item[]) => void) => void)}
  {#each list as b (b.id)}
    <div class="flex items-start gap-1">
      <span class="pt-2 pl-2 text-muted-foreground" aria-hidden="true">•</span>
      <div class="min-w-0 flex-1">
        <EditableCell
          multiline
          label="{noun} bullet"
          placeholder="Describe what you did…"
          value={b.text}
          onSave={(v) =>
            update((l) => {
              l.find((x) => x.id === b.id)!.text = v
            })}
        />
      </div>
      <IconButton
        icon={Trash2Icon}
        label="Delete bullet"
        danger
        onclick={() =>
          update((l) => {
            l.splice(l.findIndex((x) => x.id === b.id), 1)
          })}
      />
    </div>
  {/each}
  {@render add('Add bullet', () => update((l) => void l.push(newItem())))}
{/snippet}

<div class="resume-editor text-sm">
  <div class="mb-6 rounded-lg bg-muted/40 px-4 py-3"><p class="font-medium">Make it yours</p><p class="mt-1 text-xs leading-5 text-muted-foreground">Refine the details below, or ask the resume assistant to help with any section.</p></div>
  <h2 class={heading.replace('mt-9', 'mt-0')}>Basics</h2>
  <div class="grid gap-4 sm:grid-cols-2">
    {#each basicsFields as f (f.key)}
      <div class="grid min-w-0 gap-1.5">
        <span class="px-1 text-xs font-medium text-muted-foreground">{f.label}</span>
        <EditableCell
          label={f.label}
          placeholder="Empty"
          value={content.basics[f.key]}
          onSave={(v) => edit((c) => (c.basics[f.key] = v))}
        />
      </div>
    {/each}
  </div>

  <h2 class={heading}>Summary</h2>
  <EditableCell
    multiline
    label="Summary"
    placeholder="Two or three lines about you…"
    value={content.summary.text}
    onSave={(v) => edit((c) => (c.summary.text = v))}
  />

  <h2 class={heading}>Experience {@render add('Add role', () => edit((c) => void c.experience.push(newExperience())))}</h2>
  {#each content.experience as exp, i (exp.id)}
    <div class="mb-4 rounded-xl border bg-muted/10 p-4">
      <div class="flex items-center gap-1">
        <div class="grid min-w-0 flex-1 grid-cols-1 gap-3 sm:grid-cols-2">
          <EditableCell label="Role" placeholder="Role" class="font-medium" value={exp.role} onSave={(v) => edit((c) => (c.experience[i].role = v))} />
          <EditableCell label="Company" placeholder="Company" value={exp.company} onSave={(v) => edit((c) => (c.experience[i].company = v))} />
          <EditableCell label="Start" placeholder="Start (e.g. Jan 2023)" value={exp.start} onSave={(v) => edit((c) => (c.experience[i].start = v))} />
          <EditableCell label="End" placeholder="End (or Present)" value={exp.end} onSave={(v) => edit((c) => (c.experience[i].end = v))} />
        </div>
        <IconButton icon={Trash2Icon} label="Delete role" danger onclick={() => edit((c) => void c.experience.splice(i, 1))} />
      </div>
      <div class="mt-4 space-y-2">
        {@render bullets(exp.bullets, `${exp.role || 'Role'}`, (fn) => edit((c) => fn(c.experience[i].bullets)))}
      </div>
    </div>
  {/each}

  <h2 class={heading}>Projects {@render add('Add project', () => edit((c) => void c.projects.push(newProject())))}</h2>
  {#each content.projects as proj, i (proj.id)}
    <div class="mb-4 rounded-xl border bg-muted/10 p-4">
      <div class="flex items-center gap-1">
        <div class="grid min-w-0 flex-1 grid-cols-1 gap-3 sm:grid-cols-2">
          <EditableCell label="Project name" placeholder="Project name" class="font-medium" value={proj.name} onSave={(v) => edit((c) => (c.projects[i].name = v))} />
          <EditableCell label="Project link" placeholder="Link" value={proj.link} onSave={(v) => edit((c) => (c.projects[i].link = v))} />
        </div>
        <IconButton icon={Trash2Icon} label="Delete project" danger onclick={() => edit((c) => void c.projects.splice(i, 1))} />
      </div>
      <div class="mt-4 space-y-2">
        {@render bullets(proj.bullets, `${proj.name || 'Project'}`, (fn) => edit((c) => fn(c.projects[i].bullets)))}
      </div>
    </div>
  {/each}

  <h2 class={heading}>Education {@render add('Add education', () => edit((c) => void c.education.push(newEducation())))}</h2>
  {#each content.education as ed, i (ed.id)}
    <div class="mb-3 flex items-center gap-3 rounded-xl border bg-muted/10 p-4">
      <div class="grid min-w-0 flex-1 grid-cols-1 gap-3 sm:grid-cols-2">
        <EditableCell label="School" placeholder="School" class="font-medium" value={ed.school} onSave={(v) => edit((c) => (c.education[i].school = v))} />
        <EditableCell label="Degree" placeholder="Degree" value={ed.degree} onSave={(v) => edit((c) => (c.education[i].degree = v))} />
        <EditableCell label="Education start" placeholder="Start" value={ed.start} onSave={(v) => edit((c) => (c.education[i].start = v))} />
        <EditableCell label="Education end" placeholder="End" value={ed.end} onSave={(v) => edit((c) => (c.education[i].end = v))} />
      </div>
      <IconButton icon={Trash2Icon} label="Delete education" danger onclick={() => edit((c) => void c.education.splice(i, 1))} />
    </div>
  {/each}

  <h2 class={heading}>Skills {@render add('Add skills line', () => edit((c) => void c.skills.push(newItem())))}</h2>
  {#each content.skills as s (s.id)}
    <div class="flex items-center gap-1">
      <div class="min-w-0 flex-1">
        <EditableCell
          label="Skills line"
          placeholder="e.g. Languages: Go, TypeScript, SQL"
          value={s.text}
          onSave={(v) => edit((c) => (c.skills.find((x) => x.id === s.id)!.text = v))}
        />
      </div>
      <IconButton icon={Trash2Icon} label="Delete skills line" danger onclick={() => edit((c) => void c.skills.splice(c.skills.findIndex((x) => x.id === s.id), 1))} />
    </div>
  {/each}
</div>

<style>
  .resume-editor :global(input), .resume-editor :global(textarea) {
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    background: var(--background);
    padding: 0.625rem 0.75rem;
    transition: border-color 150ms, box-shadow 150ms;
  }
  .resume-editor :global(input:focus), .resume-editor :global(textarea:focus) {
    border-color: var(--ring);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--ring) 12%, transparent);
  }
  .resume-editor :global(textarea) { line-height: 1.7; min-height: 6rem; }
</style>
