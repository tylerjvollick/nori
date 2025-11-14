<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import type { Task } from '$lib/types/task';
  import { taskStore } from '$lib/stores/task';
  import Dialog from '$lib/components/ui/Dialog.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import { Badge } from '$lib/components/ui/badge';
  import Card from '$lib/components/ui/Card.svelte';
  
  interface TaskDetailDialogProps {
    task: Task | null;
    onClose: () => void;
  }
  
  let { task, onClose }: TaskDetailDialogProps = $props();
  
  let currentTime = $state(Date.now());
  let timerInterval: number | undefined;
  
  onMount(() => {
    // Update current time every second for active timers
    timerInterval = window.setInterval(() => {
      currentTime = Date.now();
    }, 1000);
  });
  
  onDestroy(() => {
    if (timerInterval) {
      clearInterval(timerInterval);
    }
  });
  
  function getSteps() {
    return task?.sop.currentVersion?.steps || [];
  }
  
  function getProgress() {
    if (!task) return 0;
    const steps = getSteps();
    if (steps.length === 0) return 0;
    return (task.completedSteps.length / steps.length) * 100;
  }
  
  function toggleStepComplete(stepIndex: number) {
    if (!task) return;
    
    if (task.completedSteps.includes(stepIndex)) {
      // Uncomplete and stop timer if running
      if (task.stepTimers[stepIndex]?.startTime) {
        taskStore.stopStepTimer(task.id, stepIndex);
      }
      taskStore.uncompleteStep(task.id, stepIndex);
    } else {
      // Complete and stop timer if running
      if (task.stepTimers[stepIndex]?.startTime) {
        taskStore.stopStepTimer(task.id, stepIndex);
      }
      taskStore.completeStep(task.id, stepIndex);
    }
  }
  
  function toggleStepTimer(stepIndex: number) {
    if (!task) return;
    
    const timer = task.stepTimers[stepIndex];
    if (timer?.startTime) {
      // Stop timer
      taskStore.stopStepTimer(task.id, stepIndex);
    } else {
      // Start timer
      taskStore.startStepTimer(task.id, stepIndex);
    }
  }
  
  function formatTime(ms: number): string {
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) {
      return `${hours}h ${minutes % 60}m`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  }
  
  function getStepTimeDisplay(stepIndex: number): string {
    if (!task) return '0s';
    
    const timer = task.stepTimers[stepIndex];
    if (!timer) return '0s';
    
    let totalTime = timer.timeSpent;
    if (timer.startTime) {
      totalTime += (currentTime - timer.startTime);
    }
    
    return formatTime(totalTime);
  }
  
  function isStepTimerActive(stepIndex: number): boolean {
    if (!task) return false;
    return !!task.stepTimers[stepIndex]?.startTime;
  }
  
  function handleStatusChange(newStatus: 'todo' | 'inProgress' | 'done') {
    if (!task) return;
    taskStore.updateTaskStatus(task.id, newStatus);
  }
  
  function toggleMaterial(material: string) {
    if (!task) return;
    taskStore.toggleMaterial(task.id, material);
  }
  
  function toggleEquipment(equipment: string) {
    if (!task) return;
    taskStore.toggleEquipment(task.id, equipment);
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
  
  const materials = $derived(task?.sop.currentVersion?.materials || []);
  const equipment = $derived(task?.sop.currentVersion?.equipment || []);
</script>

<Dialog open={!!task} onClose={onClose} class="w-[95vw] max-w-[1400px]">
  {#if task}
    <div class="flex flex-col max-h-[90vh]">
      <!-- Header with Title and Badges -->
      <div class="p-6 border-b border-border">
        <div class="space-y-3">
          <h2 class="text-2xl font-bold text-foreground">
            {task.sop.name}
          </h2>
          <div class="flex items-center gap-2 flex-wrap">
            <Badge variant="outline" class={getStatusColor(task.status)}>
              {task.status === 'todo' ? 'To Do' : task.status === 'inProgress' ? 'In Progress' : 'Done'}
            </Badge>
            {#if task.sop.currentVersion}
              <Badge variant="outline" class="bg-secondary text-secondary-foreground border-border">
                v{task.sop.currentVersion.versionNumber}
              </Badge>
            {/if}
            {#if task.assignedTo}
              <Badge variant="outline" class="bg-blue-50 dark:bg-blue-950 text-blue-700 dark:text-blue-300 border-blue-200 dark:border-blue-800">
                <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                </svg>
                {task.assignedTo}
              </Badge>
            {/if}
          </div>
        </div>
      </div>

      <!-- Two Column Layout -->
      <div class="grid grid-cols-1 lg:grid-cols-[1fr_350px] gap-6 p-6 flex-1 min-h-0 overflow-hidden">
        <!-- Left Column - Steps -->
        <div class="space-y-4 overflow-y-auto pr-2">
          <!-- Progress -->
          <div class="space-y-2">
            <div class="flex items-center justify-between text-sm">
              <span class="text-muted-foreground">Progress</span>
              <span class="text-foreground font-medium">
                {task.completedSteps.length} / {getSteps().length} steps
              </span>
            </div>
            <div class="w-full bg-secondary rounded-full h-2">
              <div
                class="bg-primary h-2 rounded-full transition-all"
                style="width: {getProgress()}%"
              ></div>
            </div>
          </div>

          <div class="border-t border-border"></div>

          <!-- Steps -->
          <div class="space-y-4">
            <h3 class="text-lg font-semibold text-foreground">Procedure Steps</h3>
            
            {#each getSteps() as step, index}
              <Card class="p-4 {task.completedSteps.includes(index) ? 'bg-primary/5' : ''}">
                <div class="flex items-start gap-4">
                  <!-- Checkbox -->
                  <Button
                    variant="ghost"
                    size="sm"
                    onclick={() => toggleStepComplete(index)}
                    class="flex-shrink-0 mt-1 p-0 h-auto"
                  >
                    {#if task.completedSteps.includes(index)}
                      <svg class="w-6 h-6 text-primary" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    {:else}
                      <svg class="w-6 h-6 text-muted-foreground hover:text-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    {/if}
                  </Button>
                  
                  <!-- Step Content -->
                  <div class="flex-1 min-w-0">
                    <div class="flex items-start justify-between gap-4">
                      <div class="flex-1">
                        <h4 class="font-medium text-foreground mb-1">
                          Step {index + 1}: {step.title}
                        </h4>
                        {#if step.instructions}
                          <p class="text-sm text-muted-foreground">
                            {step.instructions}
                          </p>
                        {/if}
                      </div>
                      
                      <!-- Timer Controls -->
                      <div class="flex items-center gap-2">
                        <div class="text-sm font-mono text-foreground min-w-[80px] text-right">
                          {getStepTimeDisplay(index)}
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onclick={() => toggleStepTimer(index)}
                          class="p-2 {isStepTimerActive(index) ? 'text-destructive hover:text-destructive' : 'text-primary hover:text-primary'}"
                          title={isStepTimerActive(index) ? 'Pause timer' : 'Start timer'}
                        >
                          {#if isStepTimerActive(index)}
                            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                              <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                            </svg>
                          {:else}
                            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                              <path d="M8 5v14l11-7z" />
                            </svg>
                          {/if}
                        </Button>
                      </div>
                    </div>
                    
                    {#if step.estimatedTimeMinutes}
                      <div class="mt-2 text-xs text-muted-foreground flex items-center gap-1">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        Est. {step.estimatedTimeMinutes}m
                      </div>
                    {/if}
                  </div>
                </div>
              </Card>
            {/each}
            
            {#if getSteps().length === 0}
              <Card class="p-8 text-center">
                <p class="text-muted-foreground">No steps defined for this SOP</p>
              </Card>
            {/if}
          </div>
        </div>

        <!-- Right Column - Sidebar -->
        <div class="space-y-4 overflow-y-auto">
          <!-- Status Card -->
          <Card class="p-4">
            <h3 class="font-semibold text-foreground mb-3">Status</h3>
            <div class="space-y-2">
              <Button
                variant={task.status === 'todo' ? 'secondary' : 'outline'}
                onclick={() => handleStatusChange('todo')}
                class="w-full justify-start"
              >
                <span class="text-sm text-foreground">To Do</span>
              </Button>
              <Button
                variant={task.status === 'inProgress' ? 'secondary' : 'outline'}
                onclick={() => handleStatusChange('inProgress')}
                class="w-full justify-start"
              >
                <span class="text-sm text-foreground">In Progress</span>
              </Button>
              <Button
                variant={task.status === 'done' ? 'secondary' : 'outline'}
                onclick={() => handleStatusChange('done')}
                class="w-full justify-start"
              >
                <span class="text-sm text-foreground">Done</span>
              </Button>
            </div>
          </Card>

          <!-- Materials -->
          {#if materials.length > 0}
            <Card class="p-4">
              <h3 class="font-semibold text-foreground mb-3 flex items-center gap-2">
                <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                </svg>
                Materials
              </h3>
              <div class="space-y-2">
                {#each materials as material}
                  <label class="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={task.materialsChecked.includes(material)}
                      onchange={() => toggleMaterial(material)}
                      class="w-4 h-4 text-primary border-input rounded focus:ring-ring"
                    />
                    <span class="text-sm text-foreground">{material}</span>
                  </label>
                {/each}
              </div>
            </Card>
          {/if}
          
          <!-- Equipment -->
          {#if equipment.length > 0}
            <Card class="p-4">
              <h3 class="font-semibold text-foreground mb-3 flex items-center gap-2">
                <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
                </svg>
                Equipment
              </h3>
              <div class="space-y-2">
                {#each equipment as item}
                  <label class="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={task.equipmentChecked.includes(item)}
                      onchange={() => toggleEquipment(item)}
                      class="w-4 h-4 text-primary border-input rounded focus:ring-ring"
                    />
                    <span class="text-sm text-foreground">{item}</span>
                  </label>
                {/each}
              </div>
            </Card>
          {/if}
        </div>
      </div>

      <!-- Footer -->
      <div class="border-t border-border p-6">
        <div class="flex justify-end gap-4">
          <Button variant="outline" onclick={onClose}>
            Close
          </Button>
          {#if task.status !== 'done'}
            <Button onclick={() => handleStatusChange('done')}>
              Mark Complete
            </Button>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</Dialog>
