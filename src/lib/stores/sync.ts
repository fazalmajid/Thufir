import { writable } from 'svelte/store';

export const syncError = writable<string | null>(null);
export const lastSync = writable<Date | null>(null);
