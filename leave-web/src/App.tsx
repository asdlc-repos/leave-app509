import { lazy, Suspense } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { ProtectedRoute } from "./auth/ProtectedRoute";
import { Layout } from "./components/Layout";
import { LoginPage } from "./pages/LoginPage";
import { useAuth } from "./auth/AuthContext";

const EmployeeDashboard = lazy(() => import("./pages/EmployeeDashboard"));
const ManagerQueue = lazy(() => import("./pages/ManagerQueue"));
const TeamCalendar = lazy(() => import("./pages/TeamCalendar"));
const CompanyCalendar = lazy(() => import("./pages/CompanyCalendar"));
const HRAdmin = lazy(() => import("./pages/HRAdmin"));
const AuditViewer = lazy(() => import("./pages/AuditViewer"));
const Notifications = lazy(() => import("./pages/Notifications"));

function HomeRedirect() {
  const { user } = useAuth();
  if (!user) return <Navigate to="/login" replace />;
  if (user.role === "manager") return <Navigate to="/manager/queue" replace />;
  if (user.role === "hr") return <Navigate to="/hr/admin" replace />;
  return <Navigate to="/dashboard" replace />;
}

export default function App() {
  return (
    <Suspense fallback={<div className="p-lg text-muted">Loading…</div>}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />

        <Route
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route index element={<HomeRedirect />} />

          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <EmployeeDashboard />
              </ProtectedRoute>
            }
          />
          <Route
            path="/calendar/company"
            element={
              <ProtectedRoute>
                <CompanyCalendar />
              </ProtectedRoute>
            }
          />
          <Route
            path="/notifications"
            element={
              <ProtectedRoute>
                <Notifications />
              </ProtectedRoute>
            }
          />

          <Route
            path="/manager/queue"
            element={
              <ProtectedRoute roles={["manager", "hr"]}>
                <ManagerQueue />
              </ProtectedRoute>
            }
          />
          <Route
            path="/manager/team-calendar"
            element={
              <ProtectedRoute roles={["manager", "hr"]}>
                <TeamCalendar />
              </ProtectedRoute>
            }
          />

          <Route
            path="/hr/admin"
            element={
              <ProtectedRoute roles={["hr"]}>
                <HRAdmin />
              </ProtectedRoute>
            }
          />
          <Route
            path="/hr/audit"
            element={
              <ProtectedRoute roles={["hr", "manager"]}>
                <AuditViewer />
              </ProtectedRoute>
            }
          />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  );
}
