<script>
	/** Local mochi controls: cycle auto/light/dark and pick one of four palettes. */
	import { onMount } from 'svelte';
	import {
		MODES, PALETTES, currentMode, currentPalette, currentPreference,
		onThemeChange, setPalette, toggleTheme
	} from '../design/theme.js';

	let mode = $state(currentPreference());
	let dark = $state(currentMode() === 'dark');
	let palette = $state(currentPalette());

	onMount(() => onThemeChange(() => {
		mode = currentPreference();
		dark = currentMode() === 'dark';
		palette = currentPalette();
	}));

	const modeLabel = { auto: 'อัตโนมัติ', light: 'สว่าง', dark: 'มืด' };
	const paletteLabel = { mochi: 'โมจิ', matcha: 'มัตฉะ', soda: 'โซดา', sakura: 'ซากุระ' };
</script>

<div class="theme-controls">
	<button
		type="button"
		class="chip mode-chip"
		onclick={toggleTheme}
		aria-label={`โหมดสี: ${modeLabel[mode]} (คลิกเพื่อเปลี่ยน)`}
		title={`โหมด ${modeLabel[mode]} · ${MODES.join(' → ')}`}
	>
		<span class="mode-icon" aria-hidden="true">{dark ? '☀' : '☾'}</span>
		<span class="mode-name">{modeLabel[mode]}</span>
	</button>
	<div class="palette-picks" role="group" aria-label="ชุดสี">
		{#each PALETTES as p}
			<button
				type="button"
				class="palette-pick"
				class:on={palette === p}
				data-palette-id={p}
				aria-pressed={palette === p}
				aria-label={`ชุดสี ${paletteLabel[p]}`}
				title={`ชุดสี ${paletteLabel[p]}`}
				onclick={() => setPalette(p)}
			>
				<span class="swatch" aria-hidden="true"></span>
				<span class="plabel">{p}</span>
			</button>
		{/each}
	</div>
</div>
