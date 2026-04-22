import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { NotificationBell } from "./NotificationBell";

export function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const onLogout = () => {
    logout("manual");
    navigate("/login", { replace: true });
  };

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="brand">Leave Management</div>
        <nav className="nav-links">
          <NavLink to="/dashboard">Dashboard</NavLink>
          <NavLink to="/calendar/company">Company Calendar</NavLink>
          {(user?.role === "manager" || user?.role === "hr") && (
            <>
              <NavLink to="/manager/queue">Approvals</NavLink>
              <NavLink to="/manager/team-calendar">Team Calendar</NavLink>
            </>
          )}
          {user?.role === "hr" && <NavLink to="/hr/admin">HR Admin</NavLink>}
          {(user?.role === "hr" || user?.role === "manager") && (
            <NavLink to="/hr/audit">Audit</NavLink>
          )}
        </nav>
        <div className="header-right">
          <NotificationBell />
          <div className="user-chip" title={user?.email}>
            <span className="user-name">{user?.name || user?.email}</span>
            <span className={`role-tag role-${user?.role}`}>{user?.role}</span>
          </div>
          <button className="btn btn-ghost" onClick={onLogout}>Logout</button>
        </div>
      </header>
      <main className="app-main">
        <Outlet />
      </main>
      <footer className="app-footer">
        <span>Auto-logout after 30 min of inactivity</span>
      </footer>
    </div>
  );
}
