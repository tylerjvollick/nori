<script lang="ts">
  import { page } from '$app/stores';
  import * as Card from '$lib/components/ui/card';
  import * as Alert from '$lib/components/ui/alert';
  import { Badge } from '$lib/components/ui/badge';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import { FileText, Target, BarChart3, Info, Plus } from '@lucide/svelte';

  let user = $derived($page.data.user);
</script>

<svelte:head>
  <title>Dashboard - Nori</title>
</svelte:head>

{#if user}
  <div class="min-h-screen bg-background">
    <main class="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
      <div class="px-4 py-6 sm:px-0">
        <div class="text-center mb-12">
          <h2 class="text-3xl font-bold text-foreground mb-4">
            Welcome back{user.firstName ? `, ${user.firstName}` : ''}!
          </h2>
          <p class="text-muted-foreground text-lg">
            A thin layer that holds everything together for your business processes.
          </p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
          <!-- SOPs Card (active) -->
          <a href="/sops" class="group">
            <Card.Root class="h-full p-8 border-2 border-transparent transition-all group-hover:border-primary group-hover:shadow-lg">
              <Card.Header class="px-0 py-0">
                <FileText class="size-10 text-primary mb-2" />
                <Card.Title class="text-xl group-hover:text-primary transition-colors">SOPs</Card.Title>
              </Card.Header>
              <Card.Content class="px-0 py-0">
                <Card.Description>
                  Document and manage your standard operating procedures with ease.
                </Card.Description>
              </Card.Content>
            </Card.Root>
          </a>

          <!-- Tasks Card (coming soon) -->
          <Card.Root class="h-full p-8 opacity-50 cursor-not-allowed">
            <Card.Header class="px-0 py-0">
              <Target class="size-10 text-muted-foreground mb-2" />
              <Card.Title class="text-xl">Tasks</Card.Title>
            </Card.Header>
            <Card.Content class="px-0 py-0">
              <Card.Description>
                Track tasks and workflows using Lean methodologies.
              </Card.Description>
              <Badge variant="secondary" class="mt-3">Coming Soon</Badge>
            </Card.Content>
          </Card.Root>

          <!-- Analytics Card (coming soon) -->
          <Card.Root class="h-full p-8 opacity-50 cursor-not-allowed">
            <Card.Header class="px-0 py-0">
              <BarChart3 class="size-10 text-muted-foreground mb-2" />
              <Card.Title class="text-xl">Analytics</Card.Title>
            </Card.Header>
            <Card.Content class="px-0 py-0">
              <Card.Description>
                Get insights into your process efficiency and bottlenecks.
              </Card.Description>
              <Badge variant="secondary" class="mt-3">Coming Soon</Badge>
            </Card.Content>
          </Card.Root>
        </div>

        <!-- Quick Start -->
        <div class="mt-12 max-w-3xl mx-auto">
          <Alert.Root>
            <Info class="size-4" />
            <Alert.Title>Quick Start</Alert.Title>
            <Alert.Description>
              <p class="mb-3">
                Get started by creating your first SOP template or task from the "+Create" button in the top navigation.
              </p>
              <div class="flex gap-3">
                <Badge variant="outline">
                  <Plus class="size-3" />
                  Create Task &rarr; From SOP template
                </Badge>
                <Badge variant="outline">
                  <Plus class="size-3" />
                  Create SOP &rarr; New template
                </Badge>
              </div>
            </Alert.Description>
          </Alert.Root>
        </div>
      </div>
    </main>
  </div>
{:else}
  <!-- Skeleton loading state -->
  <div class="min-h-screen bg-background">
    <main class="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
      <div class="px-4 py-6 sm:px-0">
        <div class="text-center mb-12">
          <Skeleton class="h-9 w-80 mx-auto mb-4" />
          <Skeleton class="h-6 w-96 mx-auto" />
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
          {#each [1, 2, 3] as _}
            <Card.Root class="h-full p-8">
              <Card.Header class="px-0 py-0">
                <Skeleton class="size-10 mb-2" />
                <Skeleton class="h-6 w-24" />
              </Card.Header>
              <Card.Content class="px-0 py-0">
                <Skeleton class="h-4 w-full mb-1" />
                <Skeleton class="h-4 w-3/4" />
              </Card.Content>
            </Card.Root>
          {/each}
        </div>

        <div class="mt-12 max-w-3xl mx-auto">
          <Skeleton class="h-24 w-full" />
        </div>
      </div>
    </main>
  </div>
{/if}
