<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth';
  import { authApi } from '$lib/api/auth';
  import type { LoginRequest } from '$lib/api/auth';
  import { Button } from '$lib/components/ui/button';

  let email = '';
  let password = '';
  let isLoading = false;
  let error = '';

  async function handleLogin() {
    if (!email || !password) {
      error = 'Please fill in all fields';
      return;
    }

    isLoading = true;
    error = '';

    try {
      const loginData: LoginRequest = { email, password };
      const response = await authApi.login(loginData);
      await authStore.login(response);
      goto('/');
    } catch (err) {
      error = err instanceof Error ? err.message : 'Login failed';
    } finally {
      isLoading = false;
    }
  }

  function handleKeyPress(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      handleLogin();
    }
  }
</script>

<svelte:head>
  <title>Login - Nori</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-background py-12 px-4 sm:px-6 lg:px-8">
  <div class="max-w-md w-full space-y-8">
    <div>
      <div class="flex justify-center">
        <div class="w-16 h-16 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-lg flex items-center justify-center">
          <svg class="w-10 h-10 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13 10V3L4 14h7v7l9-11h-7z"
            />
          </svg>
        </div>
      </div>
      <h1 class="mt-4 text-center text-4xl font-bold text-foreground">
        Nori
      </h1>
      <h2 class="mt-6 text-center text-3xl font-extrabold text-foreground">
        Sign in to your account
      </h2>
      <p class="mt-2 text-center text-sm text-muted-foreground">
        Or
        <a href="/register" class="font-medium text-primary hover:text-primary/80">
          create a new account
        </a>
      </p>
    </div>

    <form class="mt-8 space-y-6" on:submit|preventDefault={handleLogin}>
      <div class="rounded-md shadow-sm -space-y-px">
        <div>
          <label for="email" class="sr-only">Email address</label>
          <input
            id="email"
            name="email"
            type="email"
            autocomplete="email"
            required
            class="appearance-none rounded-none relative block w-full px-3 py-2 border border-border placeholder-muted-foreground text-foreground bg-card rounded-t-md focus:outline-none focus:ring-ring focus:border-ring focus:z-10 sm:text-sm"
            placeholder="Email address"
            bind:value={email}
            on:keypress={handleKeyPress}
          />
        </div>
        <div>
          <label for="password" class="sr-only">Password</label>
          <input
            id="password"
            name="password"
            type="password"
            autocomplete="current-password"
            required
            class="appearance-none rounded-none relative block w-full px-3 py-2 border border-border placeholder-muted-foreground text-foreground bg-card rounded-b-md focus:outline-none focus:ring-ring focus:border-ring focus:z-10 sm:text-sm"
            placeholder="Password"
            bind:value={password}
            on:keypress={handleKeyPress}
          />
        </div>
      </div>

      {#if error}
        <div class="rounded-md bg-destructive/10 p-4">
          <div class="text-sm text-destructive">{error}</div>
        </div>
      {/if}

      <div>
        <Button
          type="submit"
          disabled={isLoading}
          class="w-full"
        >
          {#if isLoading}
            Signing in...
          {:else}
            Sign in
          {/if}
        </Button>
      </div>
    </form>
  </div>
</div>