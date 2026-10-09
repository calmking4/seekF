import { useCallback, useEffect, useRef, useState } from "react";
import {
  Activity,
  ArrowUpRight,
  LayoutDashboard,
  LogOut,
  Menu,
  RefreshCw,
  ShieldCheck,
  X,
} from "lucide-react";
import {
  Navigate,
  NavLink,
  Route,
  Routes,
  useLocation,
} from "react-router-dom";
import { api, ApiError, errorMessage } from "./api";
import { clock, ErrorNotice, Spinner, ToolLink } from "./components";
import Login from "./Login";
import OverviewPage from "./OverviewPage";
import ObservabilityPage from "./ObservabilityPage";
import type { Identity, Overview, System } from "./types";

export default function App() {
  const [user, setUser] = useState<Identity | null>(null);
  const [initializing, setInitializing] = useState(true);
  const [sessionError, setSessionError] = useState("");
  const [sessionAttempt, setSessionAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setInitializing(true);
    setSessionError("");
    api<Identity>("/me", { signal: controller.signal })
      .then(setUser)
      .catch((e) => {
        if (
          !controller.signal.aborted &&
          !(e instanceof ApiError && [401, 403].includes(e.status))
        )
          setSessionError(errorMessage(e));
      })
      .finally(() => {
        if (!controller.signal.aborted) setInitializing(false);
      });
    const unauthorized = () => setUser(null);
    window.addEventListener("admin:unauthorized", unauthorized);
    return () => {
      controller.abort();
      window.removeEventListener("admin:unauthorized", unauthorized);
    };
  }, [sessionAttempt]);
  if (initializing)
    return (
      <div className="flex min-h-screen items-center justify-center gap-3 text-sm text-slate-500">
        <Spinner />
        正在连接管理控制台
      </div>
    );
  if (sessionError)
    return (
      <main className="mx-auto flex min-h-screen max-w-md flex-col justify-center gap-5 p-6">
        <h1 className="text-xl font-semibold">管理服务暂时无法连接</h1>
        <ErrorNotice message={sessionError} />
        <button
          onClick={() => setSessionAttempt((value) => value + 1)}
          className="rounded-xl bg-[#123c32] p-3 text-sm text-white"
        >
          重新连接
        </button>
      </main>
    );
  if (!user) return <Login onLogin={setUser} />;
  return <Console user={user} onLogout={() => setUser(null)} />;
}

function Console({ user, onLogout }: { user: Identity; onLogout: () => void }) {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [system, setSystem] = useState<System | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<string>();
  const pending = useRef<AbortController | null>(null);
  const closeMenuButton = useRef<HTMLButtonElement | null>(null);
  const openMenuButton = useRef<HTMLButtonElement | null>(null);
  const location = useLocation();
  const refresh = useCallback(async () => {
    if (pending.current) return;
    const controller = new AbortController();
    pending.current = controller;
    setBusy(true);
    setError("");
    try {
      const result = await Promise.allSettled([
        api<Overview>("/overview", { signal: controller.signal }),
        api<System>("/system", { signal: controller.signal }),
      ]);
      if (controller.signal.aborted) return;
      if (result[0].status === "fulfilled") setOverview(result[0].value);
      if (result[1].status === "fulfilled") setSystem(result[1].value);
      const errors = result.filter(
        (item): item is PromiseRejectedResult => item.status === "rejected",
      );
      if (errors.length)
        setError(
          errors.map((item) => errorMessage(item.reason)).join("；") +
            "。保留上次成功数据。",
        );
      else setLastUpdated(new Date().toISOString());
    } finally {
      if (pending.current === controller) pending.current = null;
      if (!controller.signal.aborted) setBusy(false);
    }
  }, []);
  useEffect(() => {
    void refresh();
    return () => {
      pending.current?.abort();
      pending.current = null;
    };
  }, [refresh]);
  useEffect(() => {
    if (!autoRefresh) return;
    const timer = window.setInterval(() => {
      if (document.visibilityState === "visible") void refresh();
    }, 60000);
    return () => window.clearInterval(timer);
  }, [autoRefresh, refresh]);
  useEffect(() => {
    setMenuOpen(false);
  }, [location.pathname]);
  useEffect(() => {
    if (!menuOpen) return;
    closeMenuButton.current?.focus();
    const close = (event: KeyboardEvent) => {
      if (event.key === "Escape") setMenuOpen(false);
    };
    window.addEventListener("keydown", close);
    return () => {
      window.removeEventListener("keydown", close);
      openMenuButton.current?.focus();
    };
  }, [menuOpen]);
  async function logout() {
    setLoggingOut(true);
    try {
      await api("/logout", { method: "POST" });
      onLogout();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoggingOut(false);
    }
  }
  return (
    <div className="min-h-screen">
      <a
        href="#main"
        className="sr-only rounded-lg bg-white p-3 focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50"
      >
        跳到主要内容
      </a>
      {menuOpen && (
        <button
          aria-label="关闭导航菜单"
          tabIndex={-1}
          onClick={() => setMenuOpen(false)}
          className="fixed inset-0 z-30 bg-slate-950/40 lg:hidden"
        />
      )}
      <aside
        aria-label="管理端导航"
        className={`fixed inset-y-0 left-0 z-40 w-60 flex-col bg-[#123c32] px-5 py-7 text-white lg:flex ${menuOpen ? "flex" : "hidden"}`}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="grid size-10 place-items-center rounded-xl bg-[#b7edcc] text-2xl font-bold text-[#123c32]">
              s
            </span>
            <div>
              <p className="text-xl font-semibold tracking-tight">seekF</p>
              <p className="text-[10px] tracking-[.2em] text-emerald-100/50">
                ADMIN CONSOLE
              </p>
            </div>
          </div>
          <button
            ref={closeMenuButton}
            onClick={() => setMenuOpen(false)}
            aria-label="收起导航菜单"
            className="rounded-lg p-1 lg:hidden"
          >
            <X className="size-5" />
          </button>
        </div>
        <p className="mt-12 mb-4 px-3 text-[10px] tracking-[.2em] text-emerald-100/40">
          工作空间
        </p>
        <nav className="space-y-2">
          {[
            { path: "/", title: "系统概览", icon: LayoutDashboard },
            { path: "/observability", title: "可观测性", icon: Activity },
          ].map((item) => (
            <NavLink
              end={item.path === "/"}
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-xl px-3 py-3 text-sm transition-colors ${isActive ? "bg-white/10 font-medium text-white" : "text-emerald-100/60 hover:bg-white/5 hover:text-white"}`
              }
            >
              <item.icon className="size-[18px]" />
              {item.title}
            </NavLink>
          ))}
        </nav>
        <div className="mt-10 border-t border-white/10 pt-5">
          <p className="mb-3 px-3 text-[10px] tracking-[.2em] text-emerald-100/40">
            运维工具
          </p>
          {system?.grafanaUrl && (
            <ToolLink
              href={system.grafanaUrl}
              className="w-full justify-between! px-3! text-emerald-100/65 hover:bg-white/5"
            >
              Grafana
            </ToolLink>
          )}
          {system?.kibanaUrl && (
            <ToolLink
              href={system.kibanaUrl}
              className="mt-1 w-full justify-between! px-3! text-emerald-100/65 hover:bg-white/5"
            >
              Kibana
            </ToolLink>
          )}
        </div>
        <div className="mt-auto pt-8">
          <div className="rounded-xl border border-emerald-200/10 bg-white/5 p-4">
            <ShieldCheck className="mb-2 size-5 text-[#b7edcc]" />
            <p className="text-xs font-medium text-emerald-50">
              管理员工作空间
            </p>
            <p className="mt-2 text-[11px] leading-5 text-emerald-100/45">
              业务概览与运行状态，
              <br />
              清晰呈现每一次变化。
            </p>
          </div>
          <div className="mt-5 flex items-center gap-3 border-t border-white/10 pt-5">
            <span className="grid size-9 shrink-0 place-items-center rounded-full bg-[#b7edcc]/15 text-sm text-[#b7edcc]">
              {user.nickname.slice(0, 1).toUpperCase()}
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-xs font-medium">{user.nickname}</p>
              <p className="mt-1 text-[10px] text-emerald-100/40">管理员</p>
            </div>
            <button
              disabled={loggingOut}
              onClick={() => void logout()}
              title="退出登录"
              aria-label="退出登录"
              className="rounded-lg p-2 text-emerald-100/50 hover:bg-white/10 hover:text-white"
            >
              {loggingOut ? <Spinner /> : <LogOut className="size-4" />}
            </button>
          </div>
        </div>
      </aside>
      <div className="lg:pl-60" inert={menuOpen}>
        <header className="flex min-h-20 flex-wrap items-center justify-between gap-3 border-b border-slate-200/70 bg-white px-5 py-4 sm:px-8">
          <div className="flex items-center gap-3">
            <button
              ref={openMenuButton}
              onClick={() => setMenuOpen(true)}
              aria-label="打开导航菜单"
              aria-expanded={menuOpen}
              className="rounded-lg p-2 text-slate-600 lg:hidden"
            >
              <Menu className="size-5" />
            </button>
            <div className="flex items-center gap-2 text-xs">
              <span className="text-slate-400">管理控制台</span>
              <span className="text-slate-300">/</span>
              <span className="font-medium text-slate-600">
                {location.pathname === "/observability"
                  ? "可观测性"
                  : "系统概览"}
              </span>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <label className="flex cursor-pointer items-center gap-2 text-xs text-slate-500">
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                className="size-3.5 accent-emerald-700"
              />
              每分钟刷新
            </label>
            <button
              disabled={busy}
              onClick={() => void refresh()}
              className="flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-600 transition-colors hover:bg-slate-50 disabled:opacity-50"
            >
              <RefreshCw className={`size-3.5 ${busy ? "animate-spin" : ""}`} />
              刷新数据
            </button>
          </div>
        </header>
        <main id="main" className="mx-auto max-w-[1500px] space-y-5 p-5 sm:p-8">
          {error && <ErrorNotice message={error} />}
          <Routes>
            <Route
              path="/"
              element={<OverviewPage overview={overview} system={system} />}
            />
            <Route
              path="/observability"
              element={<ObservabilityPage system={system} />}
            />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
          <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200/70 pt-6 text-[11px] text-slate-400">
            <span>seekF · 管理控制台</span>
            <span className="flex items-center gap-2">
              <ArrowUpRight className="size-3" />
              最近完整刷新：{clock(lastUpdated)}
            </span>
          </footer>
        </main>
      </div>
    </div>
  );
}
