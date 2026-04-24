<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';

	let slug = $derived($page.params.slug);

	$effect(() => {
		if (slug) {
			const lastTab = browser ? localStorage.getItem(`lastVisitedTab:${slug}`) : null;
			goto(`/spaces/${slug}/${lastTab ?? 'jobs'}`, { replaceState: true });
		}
	});
</script>
