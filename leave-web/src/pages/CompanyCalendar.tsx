import { useCallback, useEffect, useMemo, useState } from "react";
import { api, ApiError } from "../api/client";
import type { CalendarResponse, User } from "../types";
import { addDays, isoDate } from "../utils/date";
import { CalendarGrid } from "../components/CalendarGrid";

export default function CompanyCalendar() {
  const [data, setData] = useState<CalendarResponse | null>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [department, setDepartment] = useState("");
  const [team, setTeam] = useState("");
  const [userId, setUserId] = useState("");

  const today = isoDate(new Date());
  const end = isoDate(addDays(new Date(), 89));

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [cal, us] = await Promise.all([
        api.getCalendar({
          from: today,
          to: end,
          department: department || undefined,
          team: team || undefined,
          userId: userId || undefined,
        }),
        users.length === 0 ? api.listUsers() : Promise.resolve(users),
      ]);
      setData(cal);
      if (users.length === 0) setUsers(us as User[]);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [today, end, department, team, userId, users]);

  useEffect(() => {
    load();
  }, [load]);

  const departments = useMemo(() => {
    const s = new Set<string>();
    users.forEach((u) => u.department && s.add(u.department));
    return Array.from(s).sort();
  }, [users]);

  const teams = useMemo(() => {
    const s = new Set<string>();
    users.forEach((u) => u.team && (!department || u.department === department) && s.add(u.team));
    return Array.from(s).sort();
  }, [users, department]);

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>Company Calendar</h1>
          <p className="text-muted">Next 90 days. Filter to narrow the view.</p>
        </div>
        <button className="btn btn-ghost" onClick={load}>Refresh</button>
      </div>

      <div className="filter-bar">
        <label className="field">
          <span>Department</span>
          <select value={department} onChange={(e) => { setDepartment(e.target.value); setTeam(""); }}>
            <option value="">All departments</option>
            {departments.map((d) => <option key={d} value={d}>{d}</option>)}
          </select>
        </label>
        <label className="field">
          <span>Team</span>
          <select value={team} onChange={(e) => setTeam(e.target.value)}>
            <option value="">All teams</option>
            {teams.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
        </label>
        <label className="field">
          <span>Employee</span>
          <select value={userId} onChange={(e) => setUserId(e.target.value)}>
            <option value="">All employees</option>
            {users.map((u) => (
              <option key={u.id} value={u.id}>
                {u.name || u.email}
              </option>
            ))}
          </select>
        </label>
        <button
          className="btn btn-ghost"
          onClick={() => { setDepartment(""); setTeam(""); setUserId(""); }}
        >
          Clear filters
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="text-muted">Loading…</div>}
      {data && (
        <CalendarGrid
          from={today}
          days={90}
          events={data.events || []}
          capacities={data.capacities}
          holidays={data.holidays}
        />
      )}
    </div>
  );
}
