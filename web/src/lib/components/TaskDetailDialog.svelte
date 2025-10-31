<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import type { Task } from '$lib/types/task';
  import { taskStore } from '$lib/stores/task';
  import Dialog from '$lib/components/ui/Dialog.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
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
        return 'bg-slate-100 text-slate-700 border-slate-200';
      case 'inProgress':
        return 'bg-blue-100 text-blue-700 border-blue-200';
      case 'done':
        return 'bg-emerald-100 text-emerald-700 border-emerald-200';
      default:
        return 'bg-slate-100 text-slate-700 border-slate-200';
    }
  }
  
  const materials = $derived(task?.sop.currentVersion?.materials || []);
  const equipment = $derived(task?.sop.currentVersion?.equipment || []);
</script>

<Dialog open={!!task} onClose={onClose} class="w-[95vw] max-w-[1400px]">
  {#if task}
    <div class="flex flex-col max-h-[90vh]">
      <!-- Header with Title and Badges -->
      <div class="p-6 border-b border-slate-200 dark:border-slate-700">
        <div class="space-y-3">
          <h2 class="text-2xl font-bold text-slate-900 dark:text-white">
            {task.sop.name}
          </h2>
          <div class="flex items-center gap-2 flex-wrap">
            <Badge variant="outline" class={getStatusColor(task.status)}>
              {task.status === 'todo' ? 'To Do' : task.status === 'inProgress' ? 'In Progress' : 'Done'}
            </Badge>
            {#if task.sop.currentVersion}
              <Badge variant="outline" class="bg-slate-100 text-slate-600 border-slate-200 dark:bg-slate-700 dark:text-slate-300 dark:border-slate-600">
                v{task.sop.currentVersion.versionNumber}
              </Badge>
            {/if}
            {#if task.assignedTo}
              <Badge variant="outline" class="bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-800">
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
              <span class="text-slate-600 dark:text-slate-400">Progress</span>
              <span class="text-slate-900 dark:text-white font-medium">
                {task.completedSteps.length} / {getSteps().length} steps
              </span>
            </div>
            <div class="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-2">
              <div
                class="bg-emerald-500 h-2 rounded-full transition-all"
                style="width: {getProgress()}%"
              ></div>
            </div>
          </div>

          <div class="border-t border-slate-200 dark:border-slate-700"></div>

          <!-- Steps -->
          <div class="space-y-4">
            <h3 class="text-lg font-semibold text-slate-900 dark:text-white">Procedure Steps</h3>
            
            {#each getSteps() as step, index}
              <Card class="p-4 {task.completedSteps.includes(index) ? 'bg-emerald-50/50 dark:bg-emerald-900/10' : ''}">
                <div class="flex items-start gap-4">
                  <!-- Checkbox -->
                  <button
                    onclick={() => toggleStepComplete(index)}
                    class="flex-shrink-0 mt-1"
                  >
                    {#if task.completedSteps.includes(index)}
                      <svg class="w-6 h-6 text-emerald-600" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    {:else}
                      <svg class="w-6 h-6 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    {/if}
                  </button>
                  
                  <!-- Step Content -->
                  <div class="flex-1 min-w-0">
                    <div class="flex items-start justify-between gap-4">
                      <div class="flex-1">
                        <h4 class="font-medium text-slate-900 dark:text-white mb-1">
                          Step {index + 1}: {step.title}
                        </h4>
                        {#if step.instructions}
                          <p class="text-sm text-slate-600 dark:text-slate-400">
                            {step.instructions}
                          </p>
                        {/if}
                      </div>
                      
                      <!-- Timer Controls -->
                      <div class="flex items-center gap-2">
                        <div class="text-sm font-mono text-slate-700 dark:text-slate-300 min-w-[80px] text-right">
                          {getStepTimeDisplay(index)}
                        </div>
                        <button
                          onclick={() => toggleStepTimer(index)}
                          class="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors {isStepTimerActive(index) ? 'text-red-600' : 'text-emerald-600'}"
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
                        </button>
                      </div>
                    </div>
                    
                    {#if step.estimatedTimeMinutes}
                      <div class="mt-2 text-xs text-slate-500 dark:text-slate-400 flex items-center gap-1">
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
                <p class="text-slate-500 dark:text-slate-400">No steps defined for this SOP</p>
              </Card>
            {/if}
          </div>
        </div>

        <!-- Right Column - Sidebar -->
        <div class="space-y-4 overflow-y-auto">
          <!-- Status Card -->
          <Card class="p-4">
            <h3 class="font-semibold text-slate-900 dark:text-white mb-3">Status</h3>
            <div class="space-y-2">
              <button
                onclick={() => handleStatusChange('todo')}
                class="w-full p-2 text-left rounded-lg border transition-colors {task.status === 'todo' ? 'bg-slate-100 border-slate-300 dark:bg-slate-700 dark:border-slate-600' : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800'}"
              >
                <span class="text-sm text-slate-700 dark:text-slate-300">To Do</span>
              </button>
              <button
                onclick={() => handleStatusChange('inProgress')}
                class="w-full p-2 text-left rounded-lg border transition-colors {task.status === 'inProgress' ? 'bg-blue-100 border-blue-300 dark:bg-blue-900/20 dark:border-blue-800' : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800'}"
              >
                <span class="text-sm text-slate-700 dark:text-slate-300">In Progress</span>
              </button>
              <button
                onclick={() => handleStatusChange('done')}
                class="w-full p-2 text-left rounded-lg border transition-colors {task.status === 'done' ? 'bg-emerald-100 border-emerald-300 dark:bg-emerald-900/20 dark:border-emerald-800' : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800'}"
              >
                <span class="text-sm text-slate-700 dark:text-slate-300">Done</span>
              </button>
            </div>
          </Card>

          <!-- Materials -->
          {#if materials.length > 0}
            <Card class="p-4">
              <h3 class="font-semibold text-slate-900 dark:text-white mb-3 flex items-center gap-2">
                <svg class="w-5 h-5 text-emerald-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
                      class="w-4 h-4 text-emerald-600 border-slate-300 rounded focus:ring-emerald-500"
                    />
                    <span class="text-sm text-slate-700 dark:text-slate-300">{material}</span>
                  </label>
                {/each}
              </div>
            </Card>
          {/if}
          
          <!-- Equipment -->
          {#if equipment.length > 0}
            <Card class="p-4">
              <h3 class="font-semibold text-slate-900 dark:text-white mb-3 flex items-center gap-2">
                <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
                      class="w-4 h-4 text-blue-600 border-slate-300 rounded focus:ring-blue-500"
                    />
                    <span class="text-sm text-slate-700 dark:text-slate-300">{item}</span>
                  </label>
                {/each}
              </div>
            </Card>
          {/if}
        </div>
      </div>

      <!-- Footer -->
      <div class="border-t border-slate-200 dark:border-slate-700 p-6">
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
