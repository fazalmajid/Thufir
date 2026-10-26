import type { Task } from '#lib/types/task.js';

// Tags are arbitrary strings, so "untagged" needs a value no tag can equal.
export const UNTAGGED = Symbol('untagged');

class TagFilterStore {
	tag = $state<string | typeof UNTAGGED | null>(null);

	toggle(tag: string) {
		this.tag = this.tag === tag ? null : tag;
	}

	clear() {
		this.tag = null;
	}

	apply(tasks: Task[]): Task[] {
		const tag = this.tag;
		if (tag === null) return tasks;
		if (tag === UNTAGGED) return tasks.filter((t) => !t.tags?.length);
		return tasks.filter((t) => t.tags?.includes(tag));
	}
}

export const tagFilterStore = new TagFilterStore();
