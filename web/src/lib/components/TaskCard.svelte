<script lang="ts">
  import type { Task } from '$lib/types/task';
  import Card from '$lib/components/ui/Card.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  
  interface TaskCardProps {
    task: Task;
    onclick?: () => void;
    isDragging?: boolean;
  }
  
  let { task, onclick, isDragging = false }: TaskCardProps = $props();
  
  function timeAgo(timestamp: number): string {
    const minutes = Math.floor((Date.now() - timestamp) / 60000);
    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    return `${days}d ago`;
  }
  
  function getStatusColor(status: string): string {
    switch (status) {
      case 'todo':
        return 'bg-slate-100 text-slate-700 border-slate-200';
      case 'inProgress':
        return 'bg-blue-100 text-blue-700 border-blue-200';
      case 'done':
        return 'bg-emerald-100 text-emerald-700 border-emerald-200';
      default:
        return 'bg-slate-100 text-slate-700 border-slate-200';
    }
  }
  
  const totalSteps = task.sop.currentVersion?.steps?.length || 0;
  const completedCount = task.completedSteps.length;
  const progress = totalSteps > 0 ? (completedCount / totalSteps) * 100 : 0;
</script>

<Card 
  class="p-4 cursor-pointer hover:shadow-md transition-all border border-slate-200 dark:border-slate-700 {isDragging ? 'opacity-50' : 'opacity-100'}"
>
  <button 
    class="w-full text-left"
    onclick={onclick}
  >
    <div class="space-y-3">
      <div>
        <h3 class="text-slate-900 dark:text-slate-100 font-medium mb-1">{task.sop.name}</h3>
        {#if task.sop.currentVersion?.description}
          <p class="text-slate-500 dark:text-slate-400 text-sm line-clamp-2">
            {task.sop.currentVersion.description}
          </p>
        {/if}
      </div>

      <!-- Progress Bar -->
      {#if totalSteps > 0}
        <div class="space-y-1">
          <div class="flex justify-between text-xs text-slate-500 dark:text-slate-400">
            <span>Progress</span>
            <span>{completedCount}/{totalSteps} steps</span>
          </div>
          <div class="w-full bg-slate-100 dark:bg-slate-700 rounded-full h-2">
            <div 
              class="bg-emerald-500 h-2 rounded-full transition-all"
              style="width: {progress}%"
            ></div>
          </div>
        </div>
      {/if}

      <!-- Status Badge -->
      <div class="flex items-center gap-2 flex-wrap">
        <Badge variant="outline" class={getStatusColor(task.status)}>
          {task.status === 'todo' ? 'To Do' : task.status === 'inProgress' ? 'In Progress' : 'Done'}
        </Badge>
        {#if task.sop.currentVersion}
          <Badge variant="outline" class="bg-slate-100 text-slate-600 border-slate-200">
            v{task.sop.currentVersion.versionNumber}
          </Badge>
        {/if}
      </div>

      <!-- Footer -->
      <div class="flex items-center gap-4 text-sm text-slate-500 dark:text-slate-400">
        {#if task.assignedTo}
          <div class="flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
            </svg>
            <span>{task.assignedTo}</span>
          </div>
        {/if}
        <div class="flex items-center gap-1.5 ml-auto">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>{timeAgo(task.createdAt)}</span>
        </div>
      </div>
    </div>
  </button>
</Card>
