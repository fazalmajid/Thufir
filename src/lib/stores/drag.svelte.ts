import type { Task } from '$lib/types/task';

type Zone = { el: HTMLElement; onDrop: (t: Task) => void | Promise<void> };

class DragStore {
	task = $state<Task | null>(null);
	activeZone = $state<Zone | null>(null);
	dropped = $state(false);
	private zones: Zone[] = [];

	registerZone(el: HTMLElement, onDrop: (t: Task) => void | Promise<void>): () => void {
		const entry: Zone = { el, onDrop };
		this.zones.push(entry);
		return () => {
			const i = this.zones.indexOf(entry);
			if (i >= 0) this.zones.splice(i, 1);
		};
	}

	updateActiveZone(x: number, y: number): void {
		let found: Zone | undefined;
		for (const zone of this.zones) {
			const rect = zone.el.getBoundingClientRect();
			if (x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom) {
				found = zone;
				break;
			}
		}
		this.activeZone = found ?? null;
	}

	drop(): void {
		if (this.activeZone && this.task) {
			this.activeZone.onDrop(this.task);
			this.dropped = true;
		}
		this.activeZone = null;
	}

	clear(): void {
		this.task = null;
		this.activeZone = null;
		this.dropped = false;
	}
}

export const dragStore = new DragStore();
