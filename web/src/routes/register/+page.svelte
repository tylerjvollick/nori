<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  // Self-registration is not available. Users are created by admins.
  // Redirect to login page.
  onMount(() => {
    goto('/login');
  });
</script>

<svelte:head>
  <title>Register - Nori</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-background py-12 px-4 sm:px-6 lg:px-8">
  <div class="max-w-md w-full space-y-8 text-center">
    <h2 class="text-2xl font-bold text-foreground">Registration Not Available</h2>
    <p class="text-muted-foreground">
      User accounts are created by administrators. Please contact your admin for access.
    </p>
    <p class="text-sm text-muted-foreground">
      <a href="/login" class="font-medium text-primary hover:text-primary/80">
        Go to sign in
      </a>
    </p>
  </div>
</div>
