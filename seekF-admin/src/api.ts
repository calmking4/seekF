export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

// 所有管理请求走同源代理，令牌只存放在HttpOnly Cookie中。
export async function api<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const timeout = AbortSignal.timeout(15000);
  const signal = options.signal
    ? AbortSignal.any([timeout, options.signal])
    : timeout;
  const response = await fetch(`/api/admin${path}`, {
    ...options,
    credentials: "same-origin",
    signal,
    headers: { "Content-Type": "application/json", ...options.headers },
  });
  let result: { code: number; message: string; data: T };
  try {
    result = await response.json();
  } catch {
    const message = response.status === 403
      ? "请求被访问策略拒绝（HTTP 403），请检查后端允许的页面来源"
      : response.status === 404
        ? "管理接口不存在（HTTP 404），请确认后端已加载管理端路由"
        : `管理接口返回异常响应（HTTP ${response.status}），请检查后端和代理连接`;
    throw new ApiError(
      response.status,
      message,
    );
  }
  if (!response.ok || result.code !== 200) {
    if (response.status === 401 || response.status === 403)
      window.dispatchEvent(new Event("admin:unauthorized"));
    throw new ApiError(
      response.status,
      result.message || "请求失败，请稍后重试",
    );
  }
  return result.data;
}

export function errorMessage(error: unknown): string {
  if (error instanceof Error && error.name === "TimeoutError")
    return "请求超时，请稍后重试";
  return error instanceof Error ? error.message : "请求失败，请稍后重试";
}
