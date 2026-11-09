<script lang="ts">
  import Segmented from '$lib/components/Segmented.svelte'
  import SunIcon from '@lucide/svelte/icons/sun'
  import MoonIcon from '@lucide/svelte/icons/moon'
  import MonitorIcon from '@lucide/svelte/icons/monitor'
  import { applyTheme, loadChoice, saveChoice, watchSystemTheme, type ThemeChoice } from '$lib/theme'

  // Light / Dark / System. Lives on the sky, so it uses the glass variant.
  let choice = $state<ThemeChoice>(loadChoice())

  const items = [
    { id: 'light', label: 'Light theme', icon: SunIcon },
    { id: 'dark', label: 'Dark theme', icon: MoonIcon },
    { id: 'system', label: 'Match system theme', icon: MonitorIcon },
  ]

  function pick(id: string) {
    choice = id as ThemeChoice
    saveChoice(choice)
    applyTheme(choice)
  }

  // While following the system, react when the OS flips between light and dark.
  $effect(() => {
    if (choice !== 'system') return
    return watchSystemTheme(() => applyTheme('system'))
  })
</script>

<Segmented variant="glass" iconOnly label="Theme" {items} value={choice} onChange={pick} />
