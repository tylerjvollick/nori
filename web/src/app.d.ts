// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
import type { User } from '$lib/api/auth';

declare global {
	namespace App {
		// interface Error {}
		interface Locals {
			/** The authenticated user, or null if unauthenticated. */
			user: User | null;
		}
		interface PageData {
			/** The authenticated user, passed from the root layout server load. */
			user: User | null;
			/** The current space, resolved in /spaces/[slug]/+layout.server.ts. */
			space?: import('$lib/api/space').Space;
		}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
