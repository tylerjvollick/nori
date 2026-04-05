<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth';
	import { authApi } from '$lib/api/auth';
	import type { ChangePasswordRequest } from '$lib/api/auth';
	import { Button } from '$lib/components/ui/button';

	const MIN_PASSWORD_LENGTH = 8;

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let isLoading = $state(false);
	let error = $state('');

	function validate(): string | null {
		if (!currentPassword || !newPassword || !confirmPassword) {
			return 'All fields are required';
		}
		if (newPassword.length < MIN_PASSWORD_LENGTH) {
			return `New password must be at least ${MIN_PASSWORD_LENGTH} characters`;
		}
		if (newPassword !== confirmPassword) {
			return 'New passwords do not match';
		}
		return null;
	}

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		const validationError = validate();
		if (validationError) {
			error = validationError;
			return;
		}

		isLoading = true;
		error = '';

		try {
			const data: ChangePasswordRequest = {
				currentPassword,
				newPassword,
			};
			const response = await authApi.changePassword(data);

			// Update auth store with new token/user
			await authStore.login(response);

			// Redirect to home (authStore.login fetches the user profile;
			// if the user has no spaces, the home page / onboarding will handle it)
			goto('/');
		} catch (err) {
			if (err instanceof Error) {
				// Map backend error messages to user-friendly text
				if (err.message.includes('current password is incorrect')) {
					error = 'Current password is incorrect';
				} else {
					error = err.message;
				}
			} else {
				error = 'Failed to change password';
			}
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>Change Password - Nori</title>
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
							d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
						/>
					</svg>
				</div>
			</div>
			<h1 class="mt-4 text-center text-4xl font-bold text-foreground">
				Nori
			</h1>
			<h2 class="mt-6 text-center text-3xl font-extrabold text-foreground">
				Change your password
			</h2>
			<p class="mt-2 text-center text-sm text-muted-foreground">
				You must set a new password before continuing.
			</p>
		</div>

		<form class="mt-8 space-y-6" onsubmit={handleSubmit}>
			<div class="rounded-md shadow-sm -space-y-px">
				<div>
					<label for="currentPassword" class="sr-only">Current password</label>
					<input
						id="currentPassword"
						name="currentPassword"
						type="password"
						autocomplete="current-password"
						required
						class="appearance-none rounded-none relative block w-full px-3 py-2 border border-border placeholder-muted-foreground text-foreground bg-card rounded-t-md focus:outline-none focus:ring-ring focus:border-ring focus:z-10 sm:text-sm"
						placeholder="Current password"
						bind:value={currentPassword}
					/>
				</div>
				<div>
					<label for="newPassword" class="sr-only">New password</label>
					<input
						id="newPassword"
						name="newPassword"
						type="password"
						autocomplete="new-password"
						required
						class="appearance-none rounded-none relative block w-full px-3 py-2 border border-border placeholder-muted-foreground text-foreground bg-card focus:outline-none focus:ring-ring focus:border-ring focus:z-10 sm:text-sm"
						placeholder="New password"
						bind:value={newPassword}
					/>
				</div>
				<div>
					<label for="confirmPassword" class="sr-only">Confirm new password</label>
					<input
						id="confirmPassword"
						name="confirmPassword"
						type="password"
						autocomplete="new-password"
						required
						class="appearance-none rounded-none relative block w-full px-3 py-2 border border-border placeholder-muted-foreground text-foreground bg-card rounded-b-md focus:outline-none focus:ring-ring focus:border-ring focus:z-10 sm:text-sm"
						placeholder="Confirm new password"
						bind:value={confirmPassword}
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
						Changing password...
					{:else}
						Change password
					{/if}
				</Button>
			</div>
		</form>
	</div>
</div>
