<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth';
  import { authApi } from '$lib/api/auth';
  import type { LoginRequest } from '$lib/api/auth';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import * as Card from '$lib/components/ui/card';
  import * as Alert from '$lib/components/ui/alert';
  import { CircleAlert, LoaderCircle } from '@lucide/svelte';
  import Logo from '$lib/components/Logo.svelte';

  let email = $state('');
  let password = $state('');
  let isLoading = $state(false);
  let error = $state('');

  async function handleLogin(event: SubmitEvent) {
    event.preventDefault();

    if (!email || !password) {
      error = 'Please fill in all fields';
      return;
    }

    isLoading = true;
    error = '';

    try {
      const loginData: LoginRequest = { email, password };
      const response = await authApi.login(loginData);

      if (response.mustChangePassword) {
        await authStore.login(response);
        goto('/change-password');
        return;
      }

      await authStore.login(response);
      goto('/');
    } catch (err) {
      error = err instanceof Error ? err.message : 'Login failed';
    } finally {
      isLoading = false;
    }
  }
</script>

<svelte:head>
  <title>Login - Nori</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-background py-12 px-4 sm:px-6 lg:px-8">
  <Card.Root class="w-full max-w-md">
    <Card.Header class="text-center">
      <Logo />
      <Card.Description class="text-lg">Sign in to your account</Card.Description>
    </Card.Header>

    <Card.Content>
      <form class="space-y-4" onsubmit={handleLogin}>
        <div class="space-y-2">
          <Label for="email">Email address</Label>
          <Input
            id="email"
            name="email"
            type="email"
            autocomplete="email"
            required
            placeholder="Email address"
            bind:value={email}
          />
        </div>

        <div class="space-y-2">
          <Label for="password">Password</Label>
          <Input
            id="password"
            name="password"
            type="password"
            autocomplete="current-password"
            required
            placeholder="Password"
            bind:value={password}
          />
        </div>

        {#if error}
          <Alert.Root variant="destructive">
            <CircleAlert />
            <Alert.Description>{error}</Alert.Description>
          </Alert.Root>
        {/if}

        <Card.Footer class="px-0 pb-0">
          <Button
            type="submit"
            disabled={isLoading}
            class="w-full"
          >
            {#if isLoading}
              <LoaderCircle class="animate-spin" />
              Signing in...
            {:else}
              Sign in
            {/if}
          </Button>
        </Card.Footer>
      </form>
    </Card.Content>
  </Card.Root>
</div>
