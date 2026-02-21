import { getAccessToken } from "@/auth-state";

export type WritingAssistMode = "grammar" | "rewrite" | "explain";

interface WritingAssistResponse {
  mode: WritingAssistMode;
  output: string;
}

interface WritingAssistRequest {
  content: string;
  mode: WritingAssistMode;
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

export const assistWriting = async (request: WritingAssistRequest, signal?: AbortSignal): Promise<WritingAssistResponse> => {
  const response = await fetchWithAuth("/api/v1/ai/writing-assist", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    signal,
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    let message = `Failed to get writing suggestion (${response.status})`;
    try {
      const data = await response.json();
      if (data?.message && typeof data.message === "string") {
        message = data.message;
      }
    } catch {
      // Ignore parse failures and keep fallback message.
    }
    throw new Error(message);
  }

  return response.json();
};
