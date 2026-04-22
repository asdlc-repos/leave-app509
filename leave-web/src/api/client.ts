import type {
  AuditEntry,
  Balance,
  Blackout,
  CalendarResponse,
  InAppNotification,
  LeaveRequest,
  LeaveType,
  User,
  UtilizationReportRow,
} from "../types";

function resolveApiBaseUrl(): string {
  const runtime = typeof window !== "undefined" ? window.configs?.apiUrl : undefined;
  if (runtime && runtime.trim() !== "") return runtime.replace(/\/+$/, "");
  const buildTime = import.meta.env?.VITE_API_URL;
  if (buildTime && buildTime.trim() !== "") return buildTime.replace(/\/+$/, "");
  // Default: assume API served on the same host at port 9090 when no config is supplied
  if (typeof window !== "undefined" && window.location) {
    return `${window.location.protocol}//${window.location.hostname}:9090`;
  }
  return "http://localhost:9090";
}

export class ApiError extends Error {
  status: number;
  body: unknown;
  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

type TokenProvider = () => string | null;
type Unauthorized = () => void;

let tokenProvider: TokenProvider = () => null;
let onUnauthorized: Unauthorized = () => {};
let onActivity: () => void = () => {};

export function configureApiClient(opts: {
  getToken: TokenProvider;
  onUnauthorized?: Unauthorized;
  onActivity?: () => void;
}) {
  tokenProvider = opts.getToken;
  if (opts.onUnauthorized) onUnauthorized = opts.onUnauthorized;
  if (opts.onActivity) onActivity = opts.onActivity;
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  init?: { raw?: boolean; query?: Record<string, string | undefined> }
): Promise<T> {
  const base = resolveApiBaseUrl();
  let url = `${base}${path.startsWith("/") ? "" : "/"}${path}`;
  if (init?.query) {
    const qs = new URLSearchParams();
    for (const [k, v] of Object.entries(init.query)) {
      if (v !== undefined && v !== null && v !== "") qs.set(k, v);
    }
    const s = qs.toString();
    if (s) url += (url.includes("?") ? "&" : "?") + s;
  }
  const headers: Record<string, string> = {};
  if (!(body instanceof FormData) && body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  const token = tokenProvider();
  if (token) headers["Authorization"] = `Bearer ${token}`;

  let res: Response;
  try {
    res = await fetch(url, {
      method,
      headers,
      body:
        body === undefined
          ? undefined
          : body instanceof FormData
          ? body
          : JSON.stringify(body),
    });
  } catch (e) {
    throw new ApiError(0, `Network error: ${(e as Error).message}`);
  }

  // Any successful interaction counts as user activity for idle-logout refresh
  onActivity();

  if (res.status === 401) {
    onUnauthorized();
    throw new ApiError(401, "Unauthorized");
  }

  if (init?.raw) {
    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new ApiError(res.status, text || res.statusText);
    }
    return res as unknown as T;
  }

  if (res.status === 204) {
    return undefined as T;
  }

  const contentType = res.headers.get("Content-Type") || "";
  let parsed: unknown = null;
  if (contentType.includes("application/json")) {
    parsed = await res.json().catch(() => null);
  } else {
    parsed = await res.text().catch(() => "");
  }

  if (!res.ok) {
    const msg =
      (parsed && typeof parsed === "object" && (parsed as any).message) ||
      (typeof parsed === "string" && parsed) ||
      res.statusText ||
      "Request failed";
    throw new ApiError(res.status, String(msg), parsed);
  }
  return parsed as T;
}

export const api = {
  // Auth
  login: (email: string, password: string) =>
    request<{ token: string; user: User; expiresAt?: string }>("POST", "/auth/login", {
      email,
      password,
    }),
  logout: () => request<void>("POST", "/auth/logout"),
  me: () => request<User>("GET", "/auth/me"),

  // Users
  listUsers: () => request<User[]>("GET", "/users"),
  getBalances: (userId: string) =>
    request<Balance[]>("GET", `/users/${encodeURIComponent(userId)}/balances`),
  adjustBalance: (
    userId: string,
    body: { leaveTypeId: string; delta: number; note: string }
  ) =>
    request<Balance>(
      "POST",
      `/users/${encodeURIComponent(userId)}/balances/adjust`,
      body
    ),

  // Leave types
  listLeaveTypes: () => request<LeaveType[]>("GET", "/leave-types"),
  createLeaveType: (body: Partial<LeaveType>) =>
    request<LeaveType>("POST", "/leave-types", body),
  updateLeaveType: (id: string, body: Partial<LeaveType>) =>
    request<LeaveType>("PUT", `/leave-types/${encodeURIComponent(id)}`, body),

  // Leave requests
  listLeaveRequests: (params?: { userId?: string; status?: string }) =>
    request<LeaveRequest[]>("GET", "/leave-requests", undefined, { query: params }),
  getLeaveRequest: (id: string) =>
    request<LeaveRequest>("GET", `/leave-requests/${encodeURIComponent(id)}`),
  createLeaveRequest: (body: {
    leaveTypeId: string;
    startDate: string;
    endDate: string;
    reason?: string;
  }) => request<LeaveRequest>("POST", "/leave-requests", body),
  cancelLeaveRequest: (id: string) =>
    request<LeaveRequest>("POST", `/leave-requests/${encodeURIComponent(id)}/cancel`),
  approveLeaveRequest: (id: string, comment?: string) =>
    request<LeaveRequest>(
      "POST",
      `/leave-requests/${encodeURIComponent(id)}/approve`,
      { comment }
    ),
  rejectLeaveRequest: (id: string, comment: string) =>
    request<LeaveRequest>(
      "POST",
      `/leave-requests/${encodeURIComponent(id)}/reject`,
      { comment }
    ),
  uploadAttachment: (id: string, payload: { filename: string; mimeType: string; data: string }) =>
    request<{ id: string }>(
      "POST",
      `/leave-requests/${encodeURIComponent(id)}/attachments`,
      payload
    ),

  // Calendar
  getCalendar: (params?: {
    from?: string;
    to?: string;
    department?: string;
    team?: string;
    userId?: string;
  }) => request<CalendarResponse>("GET", "/calendar", undefined, { query: params }),

  // Manager
  managerQueue: () => request<LeaveRequest[]>("GET", "/manager/queue"),
  teamCalendar: (params?: { from?: string; to?: string }) =>
    request<CalendarResponse>("GET", "/manager/team-calendar", undefined, {
      query: params,
    }),

  // Policies
  listBlackouts: () => request<Blackout[]>("GET", "/policies/blackouts"),
  createBlackout: (body: Partial<Blackout>) =>
    request<Blackout>("POST", "/policies/blackouts", body),

  // Reports
  utilizationReport: (params?: {
    from?: string;
    to?: string;
    department?: string;
  }) =>
    request<{ rows: UtilizationReportRow[] }>(
      "GET",
      "/reports/utilization",
      undefined,
      { query: { ...params, format: "json" } }
    ),
  utilizationReportDownloadUrl: (
    format: "csv" | "pdf",
    params?: { from?: string; to?: string; department?: string }
  ) => {
    const base = resolveApiBaseUrl();
    const qs = new URLSearchParams();
    qs.set("format", format);
    if (params?.from) qs.set("from", params.from);
    if (params?.to) qs.set("to", params.to);
    if (params?.department) qs.set("department", params.department);
    return `${base}/reports/utilization?${qs.toString()}`;
  },
  downloadReport: async (
    format: "csv" | "pdf",
    params?: { from?: string; to?: string; department?: string }
  ) => {
    const token = tokenProvider();
    const url = api.utilizationReportDownloadUrl(format, params);
    const res = await fetch(url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (res.status === 401) {
      onUnauthorized();
      throw new ApiError(401, "Unauthorized");
    }
    if (!res.ok) {
      throw new ApiError(res.status, `Report download failed (${res.status})`);
    }
    onActivity();
    return res.blob();
  },

  // Audit
  audit: (employeeId: string) =>
    request<AuditEntry[]>("GET", `/audit/${encodeURIComponent(employeeId)}`),

  // In-app notifications
  inAppNotifications: () => request<InAppNotification[]>("GET", "/notifications/inapp"),
};

export { resolveApiBaseUrl };
