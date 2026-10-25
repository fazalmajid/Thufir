import type { Task } from '$lib/types/task';

class TagFilterStore {
	tag = $state<string | null>(null);

	toggle(tag: string) {
		this.tag = this.tag === tag ? null : tag;
	}

	clear() {
		this.tag = null;
	}

	apply(tasks: Task[]): Task[] {
		const tag = this.tag;
		return tag ? tasks.filter((t) => t.tags?.includes(tag)) : tasks;
	}
}

export const tagFilterStore = new TagFilterStore();
