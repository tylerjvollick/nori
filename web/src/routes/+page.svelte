<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth';
  import type { User } from '$lib/api/auth';

  let user: User | null = null;
  let isLoading = true;

  onMount(() => {
    authStore.initialize();
  });

  // Subscribe to auth store changes
  authStore.subscribe((state) => {
    user = state.user;
    isLoading = state.isLoading;

    // Redirect to login if not authenticated
    if (!isLoading && !state.isAuthenticated) {
      goto('/login');
    }
  });
</script>

<svelte:head>
  <title>Dashboard - Nori</title>
</svelte:head>

{#if isLoading}
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
    <div class="animate-spin rounded-full h-32 w-32 border-b-2 border-emerald-600 dark:border-emerald-400"></div>
  </div>
{:else if user}
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- Main content -->
    <main class="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
      <div class="px-4 py-6 sm:px-0">
        <div class="text-center mb-12">
          <h2 class="text-3xl font-bold text-gray-900 dark:text-white mb-4">
            Welcome back, {user.firstName}!
          </h2>
          <p class="text-gray-600 dark:text-gray-400 text-lg">
            A thin layer that holds everything together for your business processes.
          </p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
          <a 
            href="/sops"
            class="bg-white dark:bg-gray-800 p-8 rounded-lg shadow-md border-2 border-transparent hover:border-emerald-500 dark:hover:border-emerald-500 transition-all hover:shadow-lg"
          >
            <div class="text-4xl mb-4">📋</div>
            <h3 class="text-xl font-semibold text-gray-900 dark:text-white mb-3">SOPs</h3>
            <p class="text-gray-600 dark:text-gray-400 text-sm">
              Document and manage your standard operating procedures with ease.
            </p>
          </a>

          <button
            class="bg-white dark:bg-gray-800 p-8 rounded-lg shadow-md border-2 border-transparent hover:border-emerald-500 dark:hover:border-emerald-500 transition-all hover:shadow-lg text-left opacity-50 cursor-not-allowed"
            disabled
          >
            <div class="text-4xl mb-4">🎯</div>
            <h3 class="text-xl font-semibold text-gray-900 dark:text-white mb-3">Tasks</h3>
            <p class="text-gray-600 dark:text-gray-400 text-sm">
              Track tasks and workflows using Lean methodologies.
            </p>
            <span class="text-xs text-emerald-600 dark:text-emerald-400 font-medium mt-2 block">Coming Soon</span>
          </button>

          <button
            class="bg-white dark:bg-gray-800 p-8 rounded-lg shadow-md border-2 border-transparent hover:border-emerald-500 dark:hover:border-emerald-500 transition-all hover:shadow-lg text-left opacity-50 cursor-not-allowed"
            disabled
          >
            <div class="text-4xl mb-4">📊</div>
            <h3 class="text-xl font-semibold text-gray-900 dark:text-white mb-3">Analytics</h3>
            <p class="text-gray-600 dark:text-gray-400 text-sm">
              Get insights into your process efficiency and bottlenecks.
            </p>
            <span class="text-xs text-emerald-600 dark:text-emerald-400 font-medium mt-2 block">Coming Soon</span>
          </button>
        </div>

        <div class="mt-12 bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800 rounded-lg p-6 max-w-3xl mx-auto">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">Quick Start</h3>
          <p class="text-gray-700 dark:text-gray-300 text-sm mb-4">
            Get started by creating your first SOP template or task from the "+Create" button in the top navigation.
          </p>
          <div class="flex gap-3 text-sm">
            <span class="px-3 py-1 bg-white dark:bg-gray-800 rounded border border-emerald-200 dark:border-emerald-700 text-gray-700 dark:text-gray-300">
              ➕ Create Task → From SOP template
            </span>
            <span class="px-3 py-1 bg-white dark:bg-gray-800 rounded border border-emerald-200 dark:border-emerald-700 text-gray-700 dark:text-gray-300">
              ➕ Create SOP → New template
            </span>
          </div>
        </div>
      </div>
    </main>
  </div>
{/if}
