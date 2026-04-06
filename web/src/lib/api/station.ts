import type {
	StationResponse,
	CreateStationRequest,
	UpdateStationRequest,
} from '$lib/types/station';
import { apiClient } from './client';

class StationApi {
	async listStations(): Promise<StationResponse[]> {
		return apiClient.get<StationResponse[]>('/api/v1/stations');
	}

	async getStation(id: string): Promise<StationResponse> {
		return apiClient.get<StationResponse>(`/api/v1/stations/${id}`);
	}

	async createStation(data: CreateStationRequest): Promise<StationResponse> {
		return apiClient.post<StationResponse>('/api/v1/stations', data);
	}

	async updateStation(id: string, data: UpdateStationRequest): Promise<StationResponse> {
		return apiClient.put<StationResponse>(`/api/v1/stations/${id}`, data);
	}

	async deleteStation(id: string): Promise<void> {
		return apiClient.delete<void>(`/api/v1/stations/${id}`);
	}
}

export const stationApi = new StationApi();
