import { getAccessToken } from "@/auth-state";

export interface ImmichAsset {
  id: string;
  filename: string;
  mimeType: string;
  size: number;
  type: string;
  thumbnailUrl: string;
  previewUrl: string;
}

export interface ListImmichAssetsResponse {
  assets: ImmichAsset[];
  nextPageToken?: string;
}

const fetchWithAuth = async (input: RequestInfo, init: RequestInit = {}) => {
  const token = getAccessToken();
  const headers = new Headers(init.headers || {});
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  return fetch(input, {
    ...init,
    headers,
    credentials: "include",
  });
};

export const listImmichAssets = async (params: { pageSize?: number; pageToken?: string } = {}): Promise<ListImmichAssetsResponse> => {
  const query = new URLSearchParams();
  if (params.pageSize) {
    query.set("pageSize", params.pageSize.toString());
  }
  if (params.pageToken) {
    query.set("pageToken", params.pageToken);
  }
  const url = `/api/immich/assets${query.toString() ? `?${query.toString()}` : ""}`;
  const response = await fetchWithAuth(url);
  if (!response.ok) {
    throw new Error(`Failed to fetch Immich assets (${response.status})`);
  }
  return response.json();
};
