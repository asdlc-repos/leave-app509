import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  ReactNode,
} from "react";
import { api, ApiError, configureApiClient } from "../api/client";
import type { User } from "../types";

interface AuthContextValue {
  user: User | null;
  token: string | null;
  loading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: (reason?: "manual" | "idle" | "expired") => void;
  bumpActivity: () => void;
}

const IDLE_TIMEOUT_MS = 30 * 60 * 1000; // 30 minutes

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Keep latest token in a ref for the API client (avoids stale closures)
  const tokenRef = useRef<string | null>(null);
  tokenRef.current = token;

  const lastActivityRef = useRef<number>(Date.now());
  const idleTimerRef = useRef<number | null>(null);

  const doLogout = useCallback((_reason: "manual" | "idle" | "expired" = "manual") => {
    // Fire-and-forget server logout; ignore failures
    if (tokenRef.current) {
      api.logout().catch(() => undefined);
    }
    setUser(null);
    setToken(null);
    tokenRef.current = null;
    lastActivityRef.current = 0;
    if (idleTimerRef.current !== null) {
      window.clearTimeout(idleTimerRef.current);
      idleTimerRef.current = null;
    }
  }, []);

  const scheduleIdleTimer = useCallback(() => {
    if (idleTimerRef.current !== null) {
      window.clearTimeout(idleTimerRef.current);
    }
    idleTimerRef.current = window.setTimeout(() => {
      const elapsed = Date.now() - lastActivityRef.current;
      if (elapsed >= IDLE_TIMEOUT_MS) {
        doLogout("idle");
      } else {
        // Re-schedule for the remaining time
        const remaining = IDLE_TIMEOUT_MS - elapsed;
        idleTimerRef.current = window.setTimeout(() => doLogout("idle"), remaining);
      }
    }, IDLE_TIMEOUT_MS);
  }, [doLogout]);

  const bumpActivity = useCallback(() => {
    lastActivityRef.current = Date.now();
    if (tokenRef.current) {
      scheduleIdleTimer();
    }
  }, [scheduleIdleTimer]);

  // Configure shared API client once
  useEffect(() => {
    configureApiClient({
      getToken: () => tokenRef.current,
      onUnauthorized: () => doLogout("expired"),
      onActivity: () => {
        // Called from api/client after each successful request
        lastActivityRef.current = Date.now();
      },
    });
  }, [doLogout]);

  // Track real user interactions for idle-logout
  useEffect(() => {
    if (!token) return;
    const windowEvents: (keyof WindowEventMap)[] = [
      "mousemove",
      "mousedown",
      "keydown",
      "touchstart",
      "scroll",
      "wheel",
    ];
    const handler = () => bumpActivity();
    for (const ev of windowEvents) window.addEventListener(ev, handler, { passive: true });
    document.addEventListener("visibilitychange", handler);
    scheduleIdleTimer();
    return () => {
      for (const ev of windowEvents) window.removeEventListener(ev, handler);
      document.removeEventListener("visibilitychange", handler);
      if (idleTimerRef.current !== null) window.clearTimeout(idleTimerRef.current);
    };
  }, [token, bumpActivity, scheduleIdleTimer]);

  const login = useCallback(
    async (email: string, password: string) => {
      setLoading(true);
      setError(null);
      try {
        const res = await api.login(email, password);
        setToken(res.token);
        tokenRef.current = res.token;
        lastActivityRef.current = Date.now();
        let nextUser = res.user;
        if (!nextUser) {
          // Backend may not return the user embedded in login; fall back to /auth/me
          try {
            nextUser = await api.me();
          } catch {
            nextUser = null as unknown as User;
          }
        }
        setUser(nextUser);
      } catch (e) {
        const msg =
          e instanceof ApiError
            ? e.status === 401
              ? "Invalid email or password"
              : e.message
            : "Login failed";
        setError(msg);
        setToken(null);
        tokenRef.current = null;
        setUser(null);
        throw e;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      token,
      loading,
      error,
      login,
      logout: doLogout,
      bumpActivity,
    }),
    [user, token, loading, error, login, doLogout, bumpActivity]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside <AuthProvider>");
  return ctx;
}
