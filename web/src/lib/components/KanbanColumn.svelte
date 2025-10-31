<script lang="ts">
  import type { Task, TaskStatus } from '$lib/types/task';
  import TaskCard from './TaskCard.svelte';
  import { dndzone } from 'svelte-dnd-action';
  
  interface KanbanColumnProps {
    title: string;
    status: TaskStatus;
    tasks: Task[];
    onTaskClick: (task: Task) => void;
    onTaskMove?: (taskId: string, newStatus: TaskStatus) => void;
  }
  
  let { title, status, tasks = [], onTaskClick, onTaskMove }: KanbanColumnProps = $props();
  
  const columnIcons = {
    todo: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
    inProgress: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
    done: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z'
  };
  
  const columnColors = {
    todo: 'text-slate-500',
    inProgress: 'text-blue-500',
    done: 'text-emerald-500'
  };
  
  // Transform tasks to include an `id` property required by dndzone
  // Use state instead of derived so we can modify it during drag operations
  let items = $state(tasks.map(task => ({ id: task.id, task })));
  
  // Update items when tasks prop changes
  $effect(() => {
    items = tasks.map(task => ({ id: task.id, task }));
  });
  
  function handleDndConsider(e: CustomEvent<{ items: typeof items; info: any }>) {
    items = e.detail.items;
  }
  
  function handleDndFinalize(e: CustomEvent<{ items: typeof items; info: any }>) {
    items = e.detail.items;
    
    // Check if a task was moved to this column from another column
    const movedItem = e.detail.info?.id;
    if (movedItem && onTaskMove) {
      const task = items.find(t => t.id === movedItem);
      if (task && task.task.status !== status) {
        onTaskMove(movedItem, status);
      }
    }
  }
</script>

<div class="flex flex-col flex-1 min-w-[320px]">
  <div class="flex items-center gap-2 mb-4 px-1">
    <svg class="w-5 h-5 {columnColors[status]}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={columnIcons[status]} />
    </svg>
    <h2 class="text-slate-700 dark:text-slate-300 font-medium">{title}</h2>
    <span class="ml-auto text-slate-400 dark:text-slate-500 text-sm">
      {tasks.length}
    </span>
  </div>
  
  <div
    class="flex-1 p-4 rounded-lg bg-slate-50 dark:bg-slate-800/50 min-h-[200px]"
    use:dndzone={{ items, dropTargetStyle: {} }}
    onconsider={handleDndConsider}
    onfinalize={handleDndFinalize}
  >
    <div class="space-y-3">
      {#each items as item (item.id)}
        <div>
          <TaskCard 
            task={item.task}
            onclick={(e?: MouseEvent) => {
              e?.stopPropagation();
              onTaskClick(item.task);
            }}
          />
        </div>
      {/each}
      {#if tasks.length === 0}
        <div class="text-center py-12 text-slate-400 dark:text-slate-500">
          No tasks
        </div>
      {/if}
    </div>
  </div>
</div>
