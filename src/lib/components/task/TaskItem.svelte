<script lang="ts">
	import type { Task } from '$lib/types/task';
	import { taskStore } from '$lib/stores/tasks.svelte';
	import { projectStore } from '$lib/stores/projects.svelte';
	import { areaStore } from '$lib/stores/areas.svelte';
	import { marked } from 'marked';
	import { tick } from 'svelte';
	import DateInput from '$lib/components/ui/DateInput.svelte';

	interface Props {
		task: Task;
		draggable?: boolean;
		showContext?: boolean;
	}

	let { task, draggable = false, showContext = false }: Props = $props();

	let contextParts = $derived.by(() => {
		if (!showContext) return [];
		const project = task.project_id ? projectStore.projects.find(p => p.id === task.project_id) : null;
		const areaId = project?.area_id ?? task.area_id;
		const area = areaId ? areaStore.areas.find(a => a.id === areaId) : null;
		const parts: { name: string; href: string }[] = [];
		if (area) parts.push({ name: area.name, href: `/areas/${area.id}` });
		if (project) parts.push({ name: project.name, href: `/projects/${project.id}` });
		return parts;
	});
	let notesExpanded = $state(false);
	let isEditing = $state(false);
	let editedTitle = $state('');
	let editedNotes = $state('');
	let editedTags = $state('');
	let editedDeadline = $state('');
	let editedReminderDate = $state('');
	let editedReminderTime = $state('');
	let titleInputElement: HTMLInputElement | undefined = $state();
	let notesTextareaElement: HTMLTextAreaElement | undefined = $state();

	async function handleToggle() {
		await taskStore.toggleComplete(task.id);
	}

	async function handleDelete() {
		if (confirm('Delete this task?')) {
			await taskStore.delete(task.id);
		}
	}

	function toggleNotes(e: Event) {
		e.stopPropagation();
		notesExpanded = !notesExpanded;
	}

	async function startEditing() {
		isEditing = true;
		notesExpanded = true;
		editedTitle = task.title;
		editedNotes = task.notes || '';
		editedTags = task.tags?.join(', ') || '';
		editedDeadline = task.deadline || '';

		// Parse reminder_time into date and time components
		if (task.reminder_time) {
			const reminderDate = new Date(task.reminder_time);
			editedReminderDate = reminderDate.toISOString().split('T')[0];
			editedReminderTime = reminderDate.toTimeString().slice(0, 5);
		} else {
			editedReminderDate = '';
			editedReminderTime = '';
		}

		await tick();
		titleInputElement?.focus();
		titleInputElement?.select();
	}

	function cancelEditing() {
		isEditing = false;
		editedTitle = '';
		editedNotes = '';
		editedTags = '';
		editedDeadline = '';
		editedReminderDate = '';
		editedReminderTime = '';
	}

	async function saveEditing() {
		const trimmedTitle = editedTitle.trim();
		const trimmedNotes = editedNotes.trim();
		const trimmedTags = editedTags.trim();

		if (!trimmedTitle) {
			// Don't allow empty titles
			cancelEditing();
			return;
		}

		const updates: any = {};

		if (trimmedTitle !== task.title) {
			updates.title = trimmedTitle;
		}

		if (trimmedNotes !== (task.notes || '')) {
			updates.notes = trimmedNotes || null;
		}

		// Handle tags
		const newTags = trimmedTags
			? trimmedTags.split(',').map(t => t.trim()).filter(t => t)
			: [];
		const currentTags = task.tags || [];
		if (JSON.stringify(newTags) !== JSON.stringify(currentTags)) {
			updates.tags = newTags;
		}

		// Handle deadline
		if (editedDeadline !== (task.deadline || '')) {
			updates.deadline = editedDeadline || null;
		}

		// Handle reminder_time
		const newReminderTime = editedReminderDate && editedReminderTime
			? `${editedReminderDate}T${editedReminderTime}:00Z`
			: null;
		const currentReminderTime = task.reminder_time || null;
		if (newReminderTime !== currentReminderTime) {
			updates.reminder_time = newReminderTime;
		}

		if (Object.keys(updates).length > 0) {
			try {
				await taskStore.update(task.id, updates);
			} catch (err) {
				console.error('Failed to update task:', err);
			}
		}

		isEditing = false;
		editedTitle = '';
		editedNotes = '';
		editedTags = '';
		editedDeadline = '';
		editedReminderDate = '';
		editedReminderTime = '';
	}

	function handleTitleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			notesTextareaElement?.focus();
		} else if (e.key === 'Escape') {
			e.preventDefault();
			cancelEditing();
		}
	}

	function handleNotesKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			cancelEditing();
		} else if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
			e.preventDefault();
			saveEditing();
		}
	}

	// Configure marked to open links in new window and handle checkboxes
	const renderer = new marked.Renderer();
	const originalLinkRenderer = renderer.link.bind(renderer);
	renderer.link = (href, title, text) => {
		const html = originalLinkRenderer(href, title, text);
		return html.replace('<a', '<a target="_blank" rel="noopener noreferrer"');
	};

	// Custom list item renderer
	let checkboxIndex = 0;
	const originalListItemRenderer = renderer.listitem.bind(renderer);
	renderer.listitem = (text, taskItem, checked) => {
		if (taskItem) {
			const index = checkboxIndex++;
			// Get the HTML from the original renderer
			let html = originalListItemRenderer(text, taskItem, checked);

			// Build our own checkbox without disabled attribute
			const checkbox = `<input type="checkbox" ${checked ? 'checked' : ''} data-checkbox-index="${index}" class="task-checkbox" />`;

			// More aggressive replacement - find and replace any input checkbox
			html = html.replace(/<input\s+([^>]*?)type=["']checkbox["']([^>]*?)>/gi, checkbox);
			html = html.replace(/<input\s+([^>]*?)disabled([^>]*?)>/gi, '<input $1 $2>');

			return html;
		}
		return originalListItemRenderer(text, taskItem, checked);
	};

	let renderedNotes = $derived.by(() => {
		checkboxIndex = 0; // Reset counter for each render
		let html = task.notes
			? marked.parse(task.notes, { async: false, renderer, gfm: true }) as string
			: '';

		// Post-process: remove any remaining disabled attributes from checkboxes
		html = html.replace(/(<input[^>]*type=["']checkbox["'][^>]*)disabled([^>]*>)/gi, '$1$2');
		html = html.replace(/(<input[^>]*)disabled([^>]*type=["']checkbox["'][^>]*>)/gi, '$1$2');

		return html;
	});

	// Svelte action to forcibly enable all checkboxes and add tracking attributes
	function enableCheckboxes(node: HTMLElement) {
		const enableAll = () => {
			const checkboxes = node.querySelectorAll('input[type="checkbox"]');
			checkboxes.forEach((checkbox, index) => {
				const input = checkbox as HTMLInputElement;
				// Remove disabled attribute
				input.removeAttribute('disabled');

				// Add our custom attributes if they don't exist
				if (!input.hasAttribute('data-checkbox-index')) {
					input.setAttribute('data-checkbox-index', index.toString());
				}
				if (!input.classList.contains('task-checkbox')) {
					input.classList.add('task-checkbox');
				}
			});
		};

		// Enable immediately
		enableAll();

		// Use MutationObserver to enable checkboxes whenever the content changes
		const observer = new MutationObserver(enableAll);
		observer.observe(node, { childList: true, subtree: true });

		return {
			destroy() {
				observer.disconnect();
			}
		};
	}

	async function handleCheckboxClick(e: Event) {
		const target = e.target as HTMLInputElement;

		// Check if this is a checkbox input
		if (target.tagName !== 'INPUT' || target.type !== 'checkbox') return;

		console.log('Checkbox clicked:', target);
		console.log('Has task-checkbox class:', target.classList.contains('task-checkbox'));
		console.log('data-checkbox-index:', target.getAttribute('data-checkbox-index'));

		// Get the current checked state after the click
		const isChecked = target.checked;
		const indexAttr = target.getAttribute('data-checkbox-index');

		if (!indexAttr) {
			console.error('No data-checkbox-index attribute found');
			return;
		}

		const index = parseInt(indexAttr);

		if (!task.notes) return;

		console.log('Looking for checkbox at index:', index);

		// Find and toggle the checkbox in the markdown
		const lines = task.notes.split('\n');
		let checkboxCount = 0;

		for (let i = 0; i < lines.length; i++) {
			const line = lines[i];
			const uncheckedMatch = line.match(/^(\s*[-*+]\s+)\[ \]/);
			const checkedMatch = line.match(/^(\s*[-*+]\s+)\[x\]/i);

			if (uncheckedMatch || checkedMatch) {
				console.log(`Found checkbox ${checkboxCount} in line ${i}: ${line}`);
				if (checkboxCount === index) {
					// Update based on the new checked state
					if (isChecked) {
						lines[i] = line.replace(/\[ \]/, '[x]');
					} else {
						lines[i] = line.replace(/\[x\]/i, '[ ]');
					}
					console.log(`Updated to: ${lines[i]}`);
					break;
				}
				checkboxCount++;
			}
		}

		const updatedNotes = lines.join('\n');
		console.log('Saving updated notes:', updatedNotes);

		try {
			await taskStore.update(task.id, { notes: updatedNotes });
			console.log('Notes saved successfully');
		} catch (err) {
			console.error('Failed to update checkbox:', err);
			// Revert the checkbox state on error
			target.checked = !isChecked;
		}
	}
</script>

<div class="flex items-center gap-3 py-2 px-1 hover:bg-gray-50 rounded group {draggable ? 'cursor-move' : ''}">
	{#if draggable}
		<button
			class="text-gray-400 hover:text-gray-600 opacity-0 group-hover:opacity-100 transition-opacity cursor-grab active:cursor-grabbing flex-shrink-0"
			aria-label="Drag to reorder"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8h16M4 16h16" />
			</svg>
		</button>
	{/if}

	<input
		type="checkbox"
		checked={task.is_completed}
		onchange={handleToggle}
		class="w-4 h-4 rounded border-gray-300 flex-shrink-0"
	/>

	<div class="flex-1 min-w-0">
		{#if isEditing}
			<div class="flex flex-col gap-2">
				<div class="flex items-start gap-1.5">
					<input
						bind:this={titleInputElement}
						bind:value={editedTitle}
						onkeydown={handleTitleKeyDown}
						class="flex-1 text-sm text-gray-900 border border-blue-500 rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500"
						type="text"
						placeholder="Task title"
					/>
				</div>

				<div class="flex flex-col gap-2">
					<textarea
						bind:this={notesTextareaElement}
						bind:value={editedNotes}
						onkeydown={handleNotesKeyDown}
						class="w-full text-xs text-gray-900 border border-blue-300 rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500 resize-y min-h-[60px]"
						placeholder="Notes (Markdown supported)"
					/>

					<div class="grid grid-cols-2 gap-2">
						<div class="flex flex-col gap-1">
							<label class="text-xs text-gray-600 font-medium">Tags</label>
							<input
								bind:value={editedTags}
								class="text-xs text-gray-900 border border-gray-300 rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500"
								type="text"
								placeholder="tag1, tag2, tag3"
							/>
						</div>

						<div class="flex flex-col gap-1">
							<label class="text-xs text-gray-600 font-medium">Due Date</label>
							<DateInput
								bind:value={editedDeadline}
								class="text-xs text-gray-900 border border-gray-300 rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500"
							/>
						</div>
					</div>

					<div class="flex flex-col gap-1">
						<label class="text-xs text-gray-600 font-medium">Reminder</label>
						<div class="grid grid-cols-2 gap-2">
							<DateInput
								bind:value={editedReminderDate}
								class="text-xs text-gray-900 border border-gray-300 rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500"
							/>
							<input
								bind:value={editedReminderTime}
								class="text-xs text-gray-900 border border-gray-300 rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500"
								type="time"
								placeholder="Time"
							/>
						</div>
					</div>

					<div class="flex gap-2 text-xs text-gray-500 pt-2 border-t border-gray-200 mt-1">
						<button
							onclick={saveEditing}
							class="px-3 py-1.5 bg-blue-500 text-white rounded hover:bg-blue-600 font-medium"
						>
							Save
						</button>
						<button
							onclick={cancelEditing}
							class="px-3 py-1.5 bg-gray-200 text-gray-700 rounded hover:bg-gray-300 font-medium"
						>
							Cancel
						</button>
						<span class="ml-auto self-center text-xs text-gray-400">
							Cmd/Ctrl+Enter to save, Esc to cancel
						</span>
					</div>
				</div>
			</div>
		{:else}
			<div class="flex items-center gap-1.5">
				{#if task.notes && !isEditing}
					<button
						onclick={toggleNotes}
						class="text-gray-500 hover:text-gray-700 transition-transform p-1 -ml-1 rounded hover:bg-gray-100 touch-manipulation flex-shrink-0"
						class:rotate-90={notesExpanded}
						aria-label={notesExpanded ? 'Hide notes' : 'Show notes'}
					>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
						</svg>
					</button>
				{/if}

				<p
					ondblclick={startEditing}
					class="text-sm text-gray-900 flex-1 cursor-text"
					class:line-through={task.is_completed}
					class:text-gray-500={task.is_completed}
				>
					{task.title}
				</p>
			</div>

			{#if task.notes && notesExpanded}
				<div
					ondblclick={(e) => {
						// Don't start editing if clicking on a checkbox
						if ((e.target as HTMLElement).classList.contains('task-checkbox')) {
							return;
						}
						startEditing();
					}}
					onclick={handleCheckboxClick}
					class="text-xs text-gray-600 mt-2 ml-5 prose prose-sm max-w-none"
					use:enableCheckboxes
				>
					{@html renderedNotes}
				</div>
			{/if}

			{#if contextParts.length > 0}
				<p class="text-xs mt-0.5 ml-5">
					{#each contextParts as part, i}
						{#if i > 0}<span class="text-blue-300"> › </span>{/if}
						<a
							href={part.href}
							class="text-blue-400 hover:text-blue-600 hover:underline"
							onclick={(e) => e.stopPropagation()}
						>{part.name}</a>
					{/each}
				</p>
			{/if}

			{#if task.tags && task.tags.length > 0}
				<div class="flex flex-wrap gap-1.5 mt-1 ml-5">
					{#each task.tags as tag}
						<span class="text-xs px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded">{tag}</span>
					{/each}
				</div>
			{/if}

			{#if task.deadline || task.reminder_time}
				<div class="flex gap-3 mt-1 ml-5 text-xs text-gray-500">
					{#if task.deadline}
						<div class="flex items-center gap-1">
							<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
							</svg>
							<span>Due: {task.deadline}</span>
						</div>
					{/if}
					{#if task.reminder_time}
						<div class="flex items-center gap-1">
							<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
							</svg>
							<span>Reminder: {new Date(task.reminder_time).toISOString().slice(0, 16).replace('T', ' ')}</span>
						</div>
					{/if}
				</div>
			{/if}
		{/if}
	</div>

	<button
		onclick={handleDelete}
		class="text-gray-400 hover:text-red-600 text-xs px-2 py-1 opacity-0 group-hover:opacity-100 transition-opacity"
	>
		Delete
	</button>
</div>
