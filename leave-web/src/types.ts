export type Role = "employee" | "manager" | "hr";

export interface User {
  id: string;
  email: string;
  name: string;
  role: Role;
  department?: string;
  team?: string;
  managerId?: string;
}

export interface LeaveType {
  id: string;
  name: string;
  defaultDays: number;
  carryOver?: boolean;
  accrualRate?: number;
  paid?: boolean;
}

export interface Balance {
  leaveTypeId: string;
  leaveTypeName: string;
  total: number;
  used: number;
  reserved: number;
  available: number;
}

export type LeaveStatus =
  | "pending"
  | "approved"
  | "rejected"
  | "cancelled";

export interface Attachment {
  id: string;
  filename: string;
  mimeType: string;
  size: number;
}

export interface LeaveRequest {
  id: string;
  userId: string;
  userName?: string;
  leaveTypeId: string;
  leaveTypeName?: string;
  startDate: string;
  endDate: string;
  days: number;
  reason?: string;
  status: LeaveStatus;
  comment?: string;
  submittedAt: string;
  decidedAt?: string;
  decidedBy?: string;
  attachments?: Attachment[];
}

export interface CalendarEvent {
  id: string;
  userId: string;
  userName: string;
  department?: string;
  team?: string;
  leaveTypeName: string;
  startDate: string;
  endDate: string;
  status: LeaveStatus;
}

export interface CalendarDayCapacity {
  date: string;
  total: number;
  onLeave: number;
  capacityPct: number;
}

export interface CalendarResponse {
  events: CalendarEvent[];
  capacities?: CalendarDayCapacity[];
  holidays?: string[];
}

export interface Blackout {
  id: string;
  name: string;
  startDate: string;
  endDate: string;
  department?: string;
  reason?: string;
}

export interface AuditEntry {
  id: string;
  timestamp: string;
  actorId: string;
  actorName?: string;
  action: string;
  target: string;
  details?: string;
  note?: string;
}

export interface InAppNotification {
  id: string;
  userId: string;
  kind: string;
  title: string;
  body?: string;
  read: boolean;
  createdAt: string;
}

export interface UtilizationReportRow {
  userId: string;
  userName: string;
  department?: string;
  team?: string;
  leaveTypeName: string;
  used: number;
  balance: number;
  utilizationPct: number;
}
