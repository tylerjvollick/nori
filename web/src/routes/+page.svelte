<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';

  let user = $derived($page.data.user);

  // Forward an authenticated user straight into a space rather than showing a
  // static landing page. Prefer the last-visited space (persisted in
  // spaces/[slug]/+layout.svelte), falling back to the user's default/first
  // accessible space. The /spaces/{slug} route then forwards to the last
  // visited tab. Users with no spaces never reach here — hooks.server.ts
  // already redirects them to /onboarding.
  $effect(() => {
    if (!browser || !user) return;

    const spaces = user.accessibleSpaces ?? [];
    if (spaces.length === 0) return;

    const stored = localStorage.getItem('lastVisitedSpace');
    const target =
      (stored && spaces.find((s) => s.slug === stored)?.slug) ??
      spaces.find((s) => s.isDefault)?.slug ??
      spaces[0].slug;

    goto(`/spaces/${target}`, { replaceState: true });
  });
</script>

<svelte:head>
  <title>Dashboard - Nori</title>
</svelte:head>

<!-- Authenticated users are redirected away by the effect above; this is a
     brief loading placeholder shown while that navigation resolves. -->
<div class="flex min-h-screen items-center justify-center bg-background">
  <div class="size-6 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" aria-label="Loading" role="status"></div>
</div>
