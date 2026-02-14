export type ApiOk<T> = { data: T };
export type ApiErr = { errors: Record<string, string> };

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
    ...init,
  });

  const text = await res.text();
  const json = text ? JSON.parse(text) : {};

  if (!res.ok) {
    const msg = json?.errors ? JSON.stringify(json.errors, null, 2) : `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return (json as ApiOk<T>).data;
}

export function pretty(v: any) {
  if (v === null || v === undefined) return "";
  if (typeof v === "object") return JSON.stringify(v);
  return String(v);
}
