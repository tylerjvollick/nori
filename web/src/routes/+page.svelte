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

  function handleLogout() {
    authStore.logout();
    goto('/login');
  }
</script>

<svelte:head>
  <title>Dashboard - Nori</title>
</svelte:head>

{#if isLoading}
  <div class="min-h-screen flex items-center justify-center">
    <div class="animate-spin rounded-full h-32 w-32 border-b-2 border-indigo-600"></div>
  </div>
{:else if user}
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <nav class="bg-white shadow">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex items-center">
            <h1 class="text-xl font-bold text-gray-900">Nori</h1>
          </div>
          <div class="flex items-center space-x-4">
            <span class="text-sm text-gray-700">
              Welcome, {user.firstName} {user.lastName}
            </span>
            <button
              on:click={handleLogout}
              class="bg-indigo-600 hover:bg-indigo-700 text-white px-3 py-2 rounded-md text-sm font-medium"
            >
              Logout
            </button>
          </div>
        </div>
      </div>
    </nav>

    <!-- Main content -->
    <main class="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
      <div class="px-4 py-6 sm:px-0">
        <div class="border-4 border-dashed border-gray-200 rounded-lg h-96 flex items-center justify-center">
          <div class="text-center">
            <h2 class="text-2xl font-bold text-gray-900 mb-4">Welcome to Nori</h2>
            <p class="text-gray-600 mb-6">
              A thin layer that holds everything together for your business processes.
            </p>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-4xl mx-auto">
              <div class="bg-white p-6 rounded-lg shadow">
                <h3 class="text-lg font-semibold text-gray-900 mb-2">📋 Process Management</h3>
                <p class="text-gray-600 text-sm">
                  Document and manage your standard operating procedures with ease.
                </p>
              </div>
              <div class="bg-white p-6 rounded-lg shadow">
                <h3 class="text-lg font-semibold text-gray-900 mb-2">🎯 Task Tracking</h3>
                <p class="text-gray-600 text-sm">
                  Track tasks and workflows using Lean methodologies.
                </p>
              </div>
              <div class="bg-white p-6 rounded-lg shadow">
                <h3 class="text-lg font-semibold text-gray-900 mb-2">📊 Analytics</h3>
                <p class="text-gray-600 text-sm">
                  Get insights into your process efficiency and bottlenecks.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
{/if}
