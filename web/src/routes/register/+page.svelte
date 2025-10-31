<script lang="ts">
  import { goto } from '$app/navigation';
  import { authApi } from '$lib/api/auth';
  import type { RegisterRequest } from '$lib/api/auth';

  let firstName = '';
  let lastName = '';
  let email = '';
  let password = '';
  let confirmPassword = '';
  let isLoading = false;
  let error = '';
  let success = '';

  async function handleRegister() {
    if (!firstName || !lastName || !email || !password || !confirmPassword) {
      error = 'Please fill in all fields';
      return;
    }

    if (password !== confirmPassword) {
      error = 'Passwords do not match';
      return;
    }

    if (password.length < 6) {
      error = 'Password must be at least 6 characters long';
      return;
    }

    isLoading = true;
    error = '';
    success = '';

    try {
      const registerData: RegisterRequest = {
        firstName,
        lastName,
        email,
        password,
      };

      await authApi.register(registerData);
      success = 'Account created successfully! You can now sign in.';
      // Clear form
      firstName = '';
      lastName = '';
      email = '';
      password = '';
      confirmPassword = '';
    } catch (err) {
      error = err instanceof Error ? err.message : 'Registration failed';
    } finally {
      isLoading = false;
    }
  }

  function handleKeyPress(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      handleRegister();
    }
  }
</script>

<svelte:head>
  <title>Register - Nori</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 py-12 px-4 sm:px-6 lg:px-8">
  <div class="max-w-md w-full space-y-8">
    <div>
      <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900 dark:text-white">
        Create your account
      </h2>
      <p class="mt-2 text-center text-sm text-gray-600 dark:text-gray-400">
        Or
        <a href="/login" class="font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-500 dark:hover:text-indigo-300">
          sign in to existing account
        </a>
      </p>
    </div>

    <form class="mt-8 space-y-6" on:submit|preventDefault={handleRegister}>
      <div class="rounded-md shadow-sm -space-y-px">
        <div>
          <label for="firstName" class="sr-only">First Name</label>
          <input
            id="firstName"
            name="firstName"
            type="text"
            required
            class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
            placeholder="First Name"
            bind:value={firstName}
          />
        </div>
        <div>
          <label for="lastName" class="sr-only">Last Name</label>
          <input
            id="lastName"
            name="lastName"
            type="text"
            required
            class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
            placeholder="Last Name"
            bind:value={lastName}
          />
        </div>
        <div>
          <label for="email" class="sr-only">Email address</label>
          <input
            id="email"
            name="email"
            type="email"
            autocomplete="email"
            required
            class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
            placeholder="Email address"
            bind:value={email}
          />
        </div>
        <div>
          <label for="password" class="sr-only">Password</label>
          <input
            id="password"
            name="password"
            type="password"
            autocomplete="new-password"
            required
            class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
            placeholder="Password"
            bind:value={password}
          />
        </div>
        <div>
          <label for="confirmPassword" class="sr-only">Confirm Password</label>
          <input
            id="confirmPassword"
            name="confirmPassword"
            type="password"
            autocomplete="new-password"
            required
            class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
            placeholder="Confirm Password"
            bind:value={confirmPassword}
            on:keypress={handleKeyPress}
          />
        </div>
      </div>

      {#if error}
        <div class="rounded-md bg-red-50 dark:bg-red-900/30 p-4">
          <div class="text-sm text-red-700 dark:text-red-400">{error}</div>
        </div>
      {/if}

      {#if success}
        <div class="rounded-md bg-green-50 dark:bg-green-900/30 p-4">
          <div class="text-sm text-green-700 dark:text-green-400">{success}</div>
        </div>
      {/if}

      <div>
        <button
          type="submit"
          disabled={isLoading}
          class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {#if isLoading}
            <span class="absolute left-1/2 transform -translate-x-1/2">
              Creating account...
            </span>
          {:else}
            Create account
          {/if}
        </button>
      </div>
    </form>
  </div>
</div>