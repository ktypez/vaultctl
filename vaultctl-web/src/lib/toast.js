import { writable } from 'svelte/store'

export const toastStore = writable({ msg: '', kind: '', visible: false })

let timer = null
export function toast(msg, kind = '') {
  toastStore.set({ msg, kind, visible: true })
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => {
    toastStore.update((t) => ({ ...t, visible: false }))
  }, 2600)
}