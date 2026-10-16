export class APIError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly errorCode?: string,
  ) {
    super(message);
    this.name = "APIError";
  }
}

type APIEnvelope<T> = { code: number; msg: string; data: T; error_code?: string };

export async function customFetch<T>(url: string, options: RequestInit = {}): Promise<T> {
  const multipart = typeof FormData !== "undefined" && options.body instanceof FormData;
  const response = await fetch(url, {
    credentials: "include",
    ...options,
    headers: {
      ...(options.body && !multipart ? { "Content-Type": "application/json" } : {}),
      ...options.headers,
    },
  });
  if (response.status === 401 && !url.includes("/api/v1/auth/")) {
    window.dispatchEvent(new Event("snakerule:unauthorized"));
  }
  if (response.status === 204) return undefined as T;

  const raw = await response.text();
  let envelope: APIEnvelope<T>;
  try {
    envelope = raw ? JSON.parse(raw) as APIEnvelope<T> : {} as APIEnvelope<T>;
  } catch {
    throw new APIError(`接口返回非 JSON 响应（HTTP ${response.status}）：${raw.slice(0, 160)}`, response.status);
  }
  if (!response.ok || envelope.code !== 0) {
    throw new APIError(envelope.msg || response.statusText, response.status, envelope.error_code);
  }
  return envelope.data as T;
}
