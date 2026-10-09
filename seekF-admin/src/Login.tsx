import { useState } from "react";
import type { FormEvent } from "react";
import { ArrowRight, Eye, EyeOff, ShieldCheck } from "lucide-react";
import { api, errorMessage } from "./api";
import { ErrorNotice, Spinner } from "./components";
import type { Identity } from "./types";

export default function Login({
  onLogin,
}: {
  onLogin: (user: Identity) => void;
}) {
  const [account, setAccount] = useState("");
  const [password, setPassword] = useState("");
  const [visible, setVisible] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setBusy(true);
    const identity = account.trim();
    try {
      const user = await api<Identity>("/login", {
        method: "POST",
        body: JSON.stringify({
          password,
          ...(identity.includes("@")
            ? { email: identity }
            : { telephone: identity }),
        }),
      });
      setPassword("");
      onLogin(user);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <main className="grid min-h-screen lg:grid-cols-2">
      <section className="relative hidden flex-col justify-between overflow-hidden bg-[#123c32] p-12 text-white lg:flex xl:p-16">
        <div className="flex items-center gap-3">
          <span className="grid size-11 place-items-center rounded-xl bg-[#b7edcc] text-2xl font-bold text-[#123c32]">
            s
          </span>
          <span className="text-2xl font-semibold tracking-tight">
            seekF
            <span className="ml-3 text-xs font-normal tracking-widest text-emerald-200/70">
              ADMIN
            </span>
          </span>
        </div>
        <div>
          <p className="mb-6 text-xs tracking-[.3em] text-emerald-200">
            管理控制台 / OPERATIONS
          </p>
          <h1 className="text-5xl font-medium leading-[1.3] tracking-tight">
            让每一次连接，
            <br />
            都有迹可循。
          </h1>
          <p className="mt-7 max-w-sm text-sm leading-7 text-emerald-100/70">
            从业务活动到服务状态，在一个清晰的工作空间中了解 seekF 的运行情况。
          </p>
          <div className="mt-10 grid max-w-sm grid-cols-3 gap-5 border-t border-white/15 pt-6 text-sm text-emerald-100">
            <span>业务概览</span>
            <span>运行状态</span>
            <span>日志与监控</span>
          </div>
        </div>
        <p className="text-xs text-emerald-100/50">seekF · 连接人与智能</p>
        <div
          aria-hidden
          className="pointer-events-none absolute -right-48 -bottom-52 size-[600px] rounded-full border-[80px] border-white/[.025]"
        />
      </section>
      <section className="flex items-center justify-center px-6 py-16">
        <div className="w-full max-w-sm">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1.5 text-xs text-emerald-800">
            <ShieldCheck className="size-3.5" />
            管理员访问
          </div>
          <h2 className="text-3xl font-semibold tracking-tight">欢迎回来</h2>
          <p className="mt-3 text-sm text-slate-500">
            使用你的 seekF 管理员账号登录。
          </p>
          <form onSubmit={submit} className="mt-9 space-y-5">
            {error && <ErrorNotice message={error} />}
            <div>
              <label
                htmlFor="account"
                className="mb-2 block text-sm font-medium"
              >
                邮箱或手机号
              </label>
              <input
                id="account"
                name="username"
                autoComplete="username"
                required
                maxLength={100}
                value={account}
                onChange={(e) => setAccount(e.target.value)}
                placeholder="请输入邮箱或手机号"
                className="w-full rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm transition-colors focus:border-emerald-600"
              />
            </div>
            <div>
              <label
                htmlFor="password"
                className="mb-2 block text-sm font-medium"
              >
                密码
              </label>
              <div className="relative">
                <input
                  id="password"
                  name="password"
                  autoComplete="current-password"
                  type={visible ? "text" : "password"}
                  required
                  maxLength={128}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="请输入登录密码"
                  className="w-full rounded-xl border border-slate-200 bg-white py-3 pl-4 pr-12 text-sm focus:border-emerald-600"
                />
                <button
                  type="button"
                  aria-label={visible ? "隐藏密码" : "显示密码"}
                  onClick={() => setVisible(!visible)}
                  className="absolute top-2 right-2 rounded-md p-2 text-slate-400 hover:text-emerald-700"
                >
                  {visible ? (
                    <EyeOff className="size-4" />
                  ) : (
                    <Eye className="size-4" />
                  )}
                </button>
              </div>
            </div>
            <button
              disabled={busy}
              className="flex w-full items-center justify-center gap-2 rounded-xl bg-[#123c32] px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-emerald-900 disabled:opacity-60"
            >
              {busy ? (
                <Spinner />
              ) : (
                <>
                  登录控制台
                  <ArrowRight className="size-4" />
                </>
              )}
            </button>
          </form>
          <p className="mt-7 text-center text-xs leading-6 text-slate-400">
            仅已授权且状态正常的管理员可访问。
          </p>
        </div>
      </section>
    </main>
  );
}
