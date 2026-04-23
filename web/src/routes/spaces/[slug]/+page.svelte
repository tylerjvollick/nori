<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	let slug = $derived($page.params.slug);

	$effect(() => {
		if (slug) {
			goto(`/spaces/${slug}/tasks`, { replaceState: true });
		}
	});
</script>
