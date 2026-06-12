import type {
	StationResponse,
	CreateStationRequest,
	UpdateStationRequest,
} from '$lib/types/station';
import { apiClient } from './client';

class StationApi {
	async listStations(spaceId: string): Promise<StationResponse[]> {
		return apiClient.get<StationResponse[]>(`/api/v1/spaces/${spaceId}/stations`);
	}

	async getStation(spaceId: string, id: string): Promise<StationResponse> {
		return apiClient.get<StationResponse>(`/api/v1/spaces/${spaceId}/stations/${id}`);
	}

	async createStation(spaceId: string, data: CreateStationRequest): Promise<StationResponse> {
		return apiClient.post<StationResponse>(`/api/v1/spaces/${spaceId}/stations`, data);
	}

	async updateStation(spaceId: string, id: string, data: UpdateStationRequest): Promise<StationResponse> {
		return apiClient.put<StationResponse>(`/api/v1/spaces/${spaceId}/stations/${id}`, data);
	}

	async deleteStation(spaceId: string, id: string): Promise<void> {
		return apiClient.delete<void>(`/api/v1/spaces/${spaceId}/stations/${id}`);
	}
}

export const stationApi = new StationApi();
