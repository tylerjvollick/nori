<script lang="ts">
  import type { Task } from '$lib/types/task';
  import Card from '$lib/components/ui/Card.svelte';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  
  interface TaskCardProps {
    task: Task;
    onclick?: (e?: MouseEvent) => void;
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
        return 'bg-secondary text-secondary-foreground border-border';
      case 'inProgress':
        return 'bg-blue-50 dark:bg-blue-950 text-blue-700 dark:text-blue-300 border-blue-200 dark:border-blue-800';
      case 'done':
        return 'bg-primary/10 text-primary border-primary/20';
      default:
        return 'bg-secondary text-secondary-foreground border-border';
    }
  }
  
  const totalSteps = task.sop.currentVersion?.steps?.length || 0;
  const completedCount = task.completedSteps.length;
  const progress = totalSteps > 0 ? (completedCount / totalSteps) * 100 : 0;
</script>

<Button
  variant="ghost"
  class="w-full text-left h-auto p-0 hover:bg-transparent"
  onclick={(e) => onclick?.(e)}
  type="button"
>
  <Card 
    class="p-4 cursor-pointer hover:shadow-md transition-all border border-border {isDragging ? 'opacity-50' : 'opacity-100'}"
  >
    <div class="space-y-3">
      <div>
        <h3 class="text-foreground font-medium mb-1">{task.sop.name}</h3>
        {#if task.sop.currentVersion?.description}
          <p class="text-muted-foreground text-sm line-clamp-2">
            {task.sop.currentVersion.description}
          </p>
        {/if}
      </div>

      <!-- Progress Bar -->
      {#if totalSteps > 0}
        <div class="space-y-1">
          <div class="flex justify-between text-xs text-muted-foreground">
            <span>Progress</span>
            <span>{completedCount}/{totalSteps} steps</span>
          </div>
          <div class="w-full bg-secondary rounded-full h-2">
            <div 
              class="bg-primary h-2 rounded-full transition-all"
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
          <Badge variant="outline" class="bg-secondary text-secondary-foreground border-border">
            v{task.sop.currentVersion.versionNumber}
          </Badge>
        {/if}
      </div>

      <!-- Footer -->
      <div class="flex items-center gap-4 text-sm text-muted-foreground">
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
  </Card>
</Button>
