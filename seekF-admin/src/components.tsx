import {
  Activity,
  ArrowUpRight,
  CheckCircle2,
  CircleAlert,
  LoaderCircle,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import type { System } from "./types";

export const number = (value?: number) =>
  value == null ? "—" : new Intl.NumberFormat("zh-CN").format(value);
export const clock = (value?: string) =>
  value ? new Date(value).toLocaleTimeString("zh-CN", { hour12: false }) : "—";
export const duration = (seconds: number) =>
  seconds >= 86400
    ? `${Math.floor(seconds / 86400)} 天 ${Math.floor((seconds % 86400) / 3600)} 小时`
    : `${Math.floor(seconds / 3600)} 小时 ${Math.floor((seconds % 3600) / 60)} 分钟`;

export function Card({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <section
      className={`rounded-2xl border border-slate-200/80 bg-white shadow-[0_2px_8px_0_#182e2210] ${className}`}
    >
      {children}
    </section>
  );
}

export function Spinner() {
  return <LoaderCircle aria-label="加载中" className="size-5 animate-spin" />;
}

export function ErrorNotice({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900"
    >
      <CircleAlert className="size-5 shrink-0" />
      {message}
    </div>
  );
}

export function StatCard({
  title,
  value,
  caption,
  icon: Icon,
}: {
  title: string;
  value?: number;
  caption: string;
  icon: LucideIcon;
}) {
  return (
    <Card className="p-5">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-slate-500">{title}</span>
        <span className="rounded-xl bg-emerald-50 p-2.5 text-emerald-700">
          <Icon className="size-5" />
        </span>
      </div>
      <p className="mt-3 text-3xl font-semibold tracking-tight tabular-nums">
        {number(value)}
      </p>
      <p className="mt-3 text-xs text-slate-500">{caption}</p>
    </Card>
  );
}

export function ServiceStatus({ system }: { system: System | null }) {
  return (
    <Card className="p-6">
      <div className="mb-5 flex items-center gap-2">
        <Activity className="size-4 text-emerald-700" />
        <h2 className="font-semibold">依赖服务状态</h2>
      </div>
      {system ? (
        <div className="space-y-5">
          {system.services.map((service) => (
            <div
              key={service.name}
              className="flex items-start justify-between gap-4"
            >
              <div>
                <p className="text-sm font-semibold">{service.name}</p>
                <p className="mt-1 max-w-64 text-xs leading-5 text-slate-500">
                  {service.detail}
                </p>
              </div>
              <span
                className={`flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${service.status === "up" ? "bg-emerald-50 text-emerald-700" : "bg-amber-50 text-amber-700"}`}
              >
                {service.status === "up" ? (
                  <CheckCircle2 className="size-3" />
                ) : (
                  <CircleAlert className="size-3" />
                )}
                {service.status === "up" ? "正常" : "不可用"}
              </span>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-sm text-slate-400">等待状态数据</p>
      )}
      <p className="mt-5 border-t border-slate-100 pt-4 text-xs text-slate-400">
        状态来自即时探测，不代表全部业务可用性。
      </p>
    </Card>
  );
}

export function ToolLink({
  href,
  children,
  className = "",
}: {
  href?: string;
  children: ReactNode;
  className?: string;
}) {
  const valid = href && /^https?:\/\//.test(href);
  return valid ? (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={`inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors ${className}`}
    >
      {children}
      <ArrowUpRight className="size-4" />
    </a>
  ) : (
    <span className="inline-flex rounded-lg bg-slate-100 px-4 py-2.5 text-sm text-slate-400">
      入口未配置
    </span>
  );
}
