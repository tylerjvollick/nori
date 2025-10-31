<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { sopStore } from '$lib/stores/sop';
  import { taskStore } from '$lib/stores/task';
  import { spaceStore } from '$lib/stores/space';
  import KanbanColumn from '$lib/components/KanbanColumn.svelte';
  import TaskDetailDialog from '$lib/components/TaskDetailDialog.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Dialog from '$lib/components/ui/Dialog.svelte';
  import type { Task, TaskStatus } from '$lib/types/task';
  import type { SOPTemplate } from '$lib/api/sop';
  
  let showCreateTaskDialog = $state(false);
  let selectedTask = $state<Task | null>(null);
  let selectedSOPForTask = $state<SOPTemplate | null>(null);
  let assignedToName = $state('');
  let showSpaceSettings = $state(false);
  let newSpaceName = $state('');

  // Get space ID from route params
  $effect(() => {
    const spaceId = $page.params.id;
    if (spaceId) {
      // Load the specific space and record visit
      spaceStore.loadSpace(spaceId);
      spaceStore.recordVisit(spaceId);
    }
  });

  onMount(() => {
    // Load SOPs for task creation
    sopStore.loadAllSOPs();
  });

  // Get current space
  const currentSpace = $derived($spaceStore.currentSpace);
  const loading = $derived($spaceStore.loading);
  const error = $derived($spaceStore.error);

  // Get tasks organized by status - these are read-only derived values
  function getTodoTasks() {
    return $taskStore.tasks.filter(task => task.status === 'todo');
  }
  
  function getInProgressTasks() {
    return $taskStore.tasks.filter(task => task.status === 'inProgress');
  }
  
  function getDoneTasks() {
    return $taskStore.tasks.filter(task => task.status === 'done');
  }

  function handleTaskClick(task: Task) {
    selectedTask = task;
  }

  function handleTaskMove(taskId: string, newStatus: TaskStatus) {
    taskStore.updateTaskStatus(taskId, newStatus);
  }

  function openCreateTaskDialog() {
    showCreateTaskDialog = true;
  }

  function handleCreateTask() {
    if (selectedSOPForTask && assignedToName.trim()) {
      taskStore.createTask(selectedSOPForTask, assignedToName.trim());
      showCreateTaskDialog = false;
      selectedSOPForTask = null;
      assignedToName = '';
    } else if (selectedSOPForTask) {
      taskStore.createTask(selectedSOPForTask);
      showCreateTaskDialog = false;
      selectedSOPForTask = null;
      assignedToName = '';
    }
  }

  function handleCancelCreateTask() {
    showCreateTaskDialog = false;
    selectedSOPForTask = null;
    assignedToName = '';
  }

  function openSpaceSettings() {
    if (currentSpace) {
      newSpaceName = currentSpace.name;
      showSpaceSettings = true;
    }
  }

  function handleCancelSettings() {
    showSpaceSettings = false;
    newSpaceName = '';
  }

  async function handleUpdateSpace() {
    if (currentSpace && newSpaceName.trim()) {
      await spaceStore.updateSpace(currentSpace.id, { name: newSpaceName.trim() });
      showSpaceSettings = false;
      newSpaceName = '';
    }
  }

  async function handleDeleteSpace() {
    if (currentSpace && confirm(`Are you sure you want to delete "${currentSpace.name}"? This action cannot be undone.`)) {
      const success = await spaceStore.deleteSpace(currentSpace.id);
      if (success) {
        // Navigate to home or first available space
        goto('/');
      }
    }
  }
</script>

<div class="container mx-auto px-4 py-8">
  {#if loading}
    <div class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600"></div>
    </div>
  {:else if error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg dark:bg-red-900/20 dark:border-red-800 dark:text-red-400">
      <p class="font-medium">Error loading space</p>
      <p class="text-sm">{error}</p>
    </div>
  {:else if currentSpace}
    <!-- Breadcrumb Navigation -->
    <nav class="flex items-center space-x-2 text-sm text-slate-600 dark:text-slate-400 mb-6">
      <a href="/" class="hover:text-emerald-600 dark:hover:text-emerald-400">Home</a>
      <span>/</span>
      <span class="text-slate-900 dark:text-white font-medium">{currentSpace.name}</span>
    </nav>

    <!-- Header -->
    <div class="flex justify-between items-center mb-8">
      <div class="flex items-center gap-4">
        <div>
          <h1 class="text-3xl font-bold text-slate-900 dark:text-white">{currentSpace.name}</h1>
          <p class="text-slate-500 dark:text-slate-400 mt-1">Manage tasks in this space</p>
        </div>
        <button
          onclick={openSpaceSettings}
          class="p-2 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
          title="Space settings"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </button>
      </div>
      <Button onclick={openCreateTaskDialog}>
        Create New Task
      </Button>
    </div>

    {#if $sopStore.loading}
      <div class="flex justify-center items-center py-12">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600"></div>
      </div>
    {:else if $sopStore.error}
      <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg dark:bg-red-900/20 dark:border-red-800 dark:text-red-400">
        <p class="font-medium">Error loading SOPs</p>
        <p class="text-sm">{$sopStore.error}</p>
      </div>
    {:else if !$sopStore.sops || $sopStore.sops.length === 0}
      <div class="text-center py-12">
        <p class="text-slate-500 dark:text-slate-400 mb-4">No SOPs created yet. Create an SOP first to start creating tasks.</p>
        <a href="/sops/create">
          <Button>Create Your First SOP</Button>
        </a>
      </div>
    {:else}
      <!-- Kanban Board -->
      <div class="flex gap-6 overflow-x-auto pb-4">
        <KanbanColumn
          title="To Do"
          status="todo"
          tasks={getTodoTasks()}
          onTaskClick={handleTaskClick}
          onTaskMove={handleTaskMove}
        />
        
        <KanbanColumn
          title="In Progress"
          status="inProgress"
          tasks={getInProgressTasks()}
          onTaskClick={handleTaskClick}
          onTaskMove={handleTaskMove}
        />
        
        <KanbanColumn
          title="Done"
          status="done"
          tasks={getDoneTasks()}
          onTaskClick={handleTaskClick}
          onTaskMove={handleTaskMove}
        />
      </div>

      {#if $taskStore.tasks.length === 0}
        <div class="text-center py-12 mt-8">
          <p class="text-slate-500 dark:text-slate-400 mb-4">No tasks yet. Create your first task from an SOP!</p>
          <Button onclick={openCreateTaskDialog}>Create Task</Button>
        </div>
      {/if}
    {/if}
  {:else}
    <div class="text-center py-12">
      <p class="text-slate-500 dark:text-slate-400">Space not found</p>
      <a href="/">
        <Button class="mt-4">Go Home</Button>
      </a>
    </div>
  {/if}
</div>

<!-- Create Task Dialog -->
{#if showCreateTaskDialog}
  <Dialog onClose={handleCancelCreateTask}>
    <div class="p-6">
      <h2 class="text-2xl font-bold text-slate-900 dark:text-white mb-6">Create New Task</h2>
      
      <div class="space-y-4">
        <!-- Select SOP -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Select SOP
          </label>
          <select
            bind:value={selectedSOPForTask}
            class="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
          >
            <option value={null}>Choose an SOP...</option>
            {#each $sopStore.sops as sop}
              <option value={sop}>{sop.name} (v{sop.currentVersion?.versionNumber || '1'})</option>
            {/each}
          </select>
        </div>

        <!-- Assigned To (Optional) -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Assign To (Optional)
          </label>
          <input
            type="text"
            bind:value={assignedToName}
            placeholder="Enter name..."
            class="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
          />
        </div>

        <!-- SOP Preview -->
        {#if selectedSOPForTask}
          <div class="mt-4 p-4 bg-slate-50 dark:bg-slate-800 rounded-lg">
            <p class="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">SOP Details:</p>
            <p class="text-sm text-slate-600 dark:text-slate-400">
              <strong>Name:</strong> {selectedSOPForTask.name}
            </p>
            {#if selectedSOPForTask.currentVersion?.description}
              <p class="text-sm text-slate-600 dark:text-slate-400 mt-1">
                <strong>Description:</strong> {selectedSOPForTask.currentVersion.description}
              </p>
            {/if}
            <p class="text-sm text-slate-600 dark:text-slate-400 mt-1">
              <strong>Steps:</strong> {selectedSOPForTask.currentVersion?.steps?.length || 0}
            </p>
          </div>
        {/if}
      </div>

      <div class="flex gap-4 mt-6">
        <Button variant="outline" onclick={handleCancelCreateTask} class="flex-1">
          Cancel
        </Button>
        <Button 
          onclick={handleCreateTask} 
          class="flex-1"
          disabled={!selectedSOPForTask}
        >
          Create Task
        </Button>
      </div>
    </div>
  </Dialog>
{/if}

<!-- Space Settings Dialog -->
{#if showSpaceSettings && currentSpace}
  <Dialog onClose={handleCancelSettings}>
    <div class="p-6">
      <h2 class="text-2xl font-bold text-slate-900 dark:text-white mb-6">Space Settings</h2>
      
      <div class="space-y-6">
        <!-- Rename Space -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Space Name
          </label>
          <input
            type="text"
            bind:value={newSpaceName}
            placeholder="Enter space name..."
            class="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
          />
        </div>

        <!-- Space Info -->
        <div class="p-4 bg-slate-50 dark:bg-slate-800 rounded-lg">
          <p class="text-sm text-slate-600 dark:text-slate-400">
            <strong>Created:</strong> {new Date(currentSpace.createdAt).toLocaleDateString()}
          </p>
          {#if currentSpace.isDefault}
            <p class="text-sm text-emerald-600 dark:text-emerald-400 mt-1">
              This is your default space
            </p>
          {/if}
        </div>

        <!-- Delete Space -->
        {#if !currentSpace.isDefault}
          <div class="pt-4 border-t border-slate-200 dark:border-slate-700">
            <p class="text-sm text-slate-600 dark:text-slate-400 mb-3">
              Deleting this space will remove all tasks associated with it. This action cannot be undone.
            </p>
            <Button variant="destructive" onclick={handleDeleteSpace} class="w-full">
              Delete Space
            </Button>
          </div>
        {:else}
          <div class="pt-4 border-t border-slate-200 dark:border-slate-700">
            <p class="text-sm text-slate-500 dark:text-slate-400 italic">
              Default spaces cannot be deleted
            </p>
          </div>
        {/if}
      </div>

      <div class="flex gap-4 mt-6">
        <Button variant="outline" onclick={handleCancelSettings} class="flex-1">
          Cancel
        </Button>
        <Button 
          onclick={handleUpdateSpace} 
          class="flex-1"
          disabled={!newSpaceName.trim() || newSpaceName === currentSpace.name}
        >
          Save Changes
        </Button>
      </div>
    </div>
  </Dialog>
{/if}

<!-- Task Detail Dialog -->
<TaskDetailDialog 
  task={selectedTask}
  onClose={() => selectedTask = null}
/>
