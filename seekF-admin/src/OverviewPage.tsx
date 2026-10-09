import { useState } from "react";
import {
  ArrowRight,
  Database,
  MessageSquare,
  Radio,
  Users,
  Files,
  Layers3,
} from "lucide-react";
import { Link } from "react-router-dom";
import {
  Card,
  clock,
  duration,
  number,
  ServiceStatus,
  StatCard,
} from "./components";
import type { DayActivity, Overview, System } from "./types";

function ActivityChart({ days }: { days: DayActivity[] }) {
  const [metric, setMetric] = useState<"messages" | "users" | "posts">(
    "messages",
  );
  const maximum = Math.max(1, ...days.map((day) => day[metric]));
  const points = days
    .map(
      (day, index) =>
        `${48 + index * 100},${180 - (day[metric] / maximum) * 140}`,
    )
    .join(" ");
  return (
    <Card className="p-5 sm:p-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="font-semibold">业务活动趋势</h2>
          <p className="mt-1 text-xs text-slate-400">
            最近 7 个自然日 · 按后端本地时区
          </p>
        </div>
        <div className="flex rounded-lg bg-slate-100 p-1">
          {(["messages", "users", "posts"] as const).map((key) => (
            <button
              key={key}
              onClick={() => setMetric(key)}
              aria-pressed={metric === key}
              className={`rounded-md px-3 py-1.5 text-xs transition-colors ${metric === key ? "bg-white font-medium text-emerald-800 shadow-sm" : "text-slate-500 hover:text-slate-800"}`}
            >
              {{ messages: "消息", users: "新用户", posts: "帖子" }[key]}
            </button>
          ))}
        </div>
      </div>
      {days.length ? (
        <>
          <svg
            viewBox="0 0 696 226"
            role="img"
            aria-label={`最近七天${{ messages: "消息", users: "新增用户", posts: "帖子" }[metric]}数量趋势`}
            className="mt-5 w-full overflow-visible"
          >
            <defs>
              <linearGradient id="activity-fill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#199875" stopOpacity=".14" />
                <stop offset="100%" stopColor="#199875" stopOpacity="0" />
              </linearGradient>
            </defs>
            {[0, 1, 2, 3].map((i) => (
              <g key={i}>
                <line
                  x1="48"
                  y1={40 + (i * 140) / 3}
                  x2="648"
                  y2={40 + (i * 140) / 3}
                  stroke="#edf1ef"
                  strokeDasharray="4 4"
                />
                <text x="0" y={44 + (i * 140) / 3} fill="#94a3b8" fontSize="10">
                  {number(Math.round((maximum * (3 - i)) / 3))}
                </text>
              </g>
            ))}
            <polygon
              points={`48,180 ${points} 648,180`}
              fill="url(#activity-fill)"
            />
            <polyline
              points={points}
              fill="none"
              stroke="#199875"
              strokeWidth="2.5"
              strokeLinejoin="round"
            />
            {days.map((day, i) => (
              <g key={day.date}>
                <circle
                  cx={48 + i * 100}
                  cy={180 - (day[metric] / maximum) * 140}
                  r="4"
                  fill="white"
                  stroke="#199875"
                  strokeWidth="2"
                >
                  <title>
                    {day.date}：{day[metric]}
                  </title>
                </circle>
                <text
                  x={48 + i * 100}
                  y="213"
                  textAnchor="middle"
                  fill="#94a3b8"
                  fontSize="11"
                >
                  {day.date.slice(5).replace("-", "/")}
                </text>
              </g>
            ))}
          </svg>
          <div className="mt-1 grid grid-cols-7 gap-1 border-t border-slate-100 pt-3 text-center text-[10px] text-slate-500">
            {days.map((day) => (
              <span key={day.date}>
                {day.date.slice(5)}
                <b className="mt-1 block text-xs font-medium text-slate-700">
                  {number(day[metric])}
                </b>
              </span>
            ))}
          </div>
        </>
      ) : (
        <div className="grid min-h-64 place-items-center text-sm text-slate-400">
          等待业务数据
        </div>
      )}
    </Card>
  );
}

export default function OverviewPage({
  overview,
  system,
}: {
  overview: Overview | null;
  system: System | null;
}) {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">系统概览</h1>
          <p className="mt-2 text-sm text-slate-500">
            从这里，了解 seekF 的今天。
          </p>
        </div>
        <span className="text-xs text-slate-400">
          业务统计更新于 {clock(overview?.generatedAt)}
        </span>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          title="用户总数"
          value={overview?.users}
          caption={
            overview
              ? `今日新增 ${number(overview.todayUsers)} 位用户`
              : "等待业务统计"
          }
          icon={Users}
        />
        <StatCard
          title="今日消息"
          value={overview?.todayMessages}
          caption="已持久化消息，包含 AI 消息"
          icon={MessageSquare}
        />
        <StatCard
          title="在线用户"
          value={system?.onlineUsers}
          caption="当前 WebSocket 已连接用户"
          icon={Radio}
        />
        <StatCard
          title="帖子总数"
          value={overview?.posts}
          caption="当前未删除的发现页帖子"
          icon={Files}
        />
      </div>
      <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
        <ActivityChart days={overview?.activity ?? []} />
        <div className="space-y-6">
          <ServiceStatus system={system} />
          <Card className="p-6">
            <h2 className="font-semibold">业务资源</h2>
            <div className="mt-5 flex items-center justify-between text-sm">
              <span className="flex items-center gap-2 text-slate-500">
                <Layers3 className="size-4" />
                群组数量
              </span>
              <b className="font-semibold tabular-nums">
                {number(overview?.groups)}
              </b>
            </div>
            <div className="mt-4 flex items-center justify-between text-sm">
              <span className="flex items-center gap-2 text-slate-500">
                <Database className="size-4" />
                知识库文档
              </span>
              <b className="font-semibold tabular-nums">
                {number(overview?.knowledge)}
              </b>
            </div>
          </Card>
        </div>
      </div>
      <Card className="p-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="font-semibold">Go 服务运行状态</h2>
            <p className="mt-1 text-xs text-slate-400">
              当前后端进程快照，不代表整机或 Docker 容器资源占用。
            </p>
          </div>
          <Link
            to="/observability"
            className="flex items-center gap-2 rounded-lg p-2 text-xs font-medium text-emerald-700 hover:bg-emerald-50"
          >
            查看监控入口
            <ArrowRight className="size-4" />
          </Link>
        </div>
        <div className="mt-6 grid gap-6 sm:grid-cols-3">
          <div>
            <p className="text-xs text-slate-500">服务运行时长</p>
            <p className="mt-2 text-xl font-semibold">
              {system ? duration(system.uptimeSeconds) : "—"}
            </p>
          </div>
          <div>
            <p className="text-xs text-slate-500">Go 堆内存</p>
            <p className="mt-2 text-xl font-semibold tabular-nums">
              {system ? `${(system.heapBytes / 1048576).toFixed(1)} MiB` : "—"}
            </p>
          </div>
          <div>
            <p className="text-xs text-slate-500">Goroutine 数量</p>
            <p className="mt-2 text-xl font-semibold tabular-nums">
              {number(system?.goroutines)}
            </p>
          </div>
        </div>
      </Card>
    </div>
  );
}
