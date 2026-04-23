import { apiClient } from './client';

export interface CustomerResponse {
	id: string;
	name: string;
	email?: string;
	phone?: string;
}

class CustomerApi {
	async listCustomers(): Promise<CustomerResponse[]> {
		return apiClient.get<CustomerResponse[]>('/api/v1/customers');
	}
}

export const customerApi = new CustomerApi();
