import {
  Activity,
  BellRing,
  ChartNoAxesCombined,
  FileSearch,
  ArrowRight,
  ShieldCheck,
} from "lucide-react";
import { Card, ServiceStatus, ToolLink } from "./components";
import type { System } from "./types";

export default function ObservabilityPage({
  system,
}: {
  system: System | null;
}) {
  return (
    <div className="space-y-6">
      <div>
        <p className="mb-2 text-xs font-medium tracking-widest text-emerald-700">
          OBSERVABILITY
        </p>
        <h1 className="text-2xl font-semibold tracking-tight">
          可观测性工作台
        </h1>
        <p className="mt-2 text-sm text-slate-500">
          业务概览在这里，深入排障交给专业工具。
        </p>
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        <Card className="relative overflow-hidden p-7">
          <span className="inline-flex rounded-xl bg-orange-50 p-3 text-orange-600">
            <ChartNoAxesCombined className="size-7" />
          </span>
          <span className="absolute top-8 right-7 rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-500">
            指标与告警
          </span>
          <h2 className="mt-6 text-2xl font-semibold">Grafana</h2>
          <p className="mt-3 max-w-md text-sm leading-7 text-slate-500">
            查看接口耗时、错误率和资源趋势，在 Grafana
            中管理告警规则与通知渠道。
          </p>
          <div className="mt-5 flex flex-wrap gap-2">
            {["指标仪表盘", "性能趋势", "告警通知"].map((text) => (
              <span
                key={text}
                className="rounded-md border border-slate-200 px-2 py-1 text-xs text-slate-500"
              >
                {text}
              </span>
            ))}
          </div>
          <div className="mt-8">
            <ToolLink
              href={system?.grafanaUrl}
              className="bg-[#123c32] text-white hover:bg-emerald-900"
            >
              打开 Grafana
            </ToolLink>
          </div>
        </Card>
        <Card className="relative overflow-hidden p-7">
          <span className="inline-flex rounded-xl bg-indigo-50 p-3 text-indigo-600">
            <FileSearch className="size-7" />
          </span>
          <span className="absolute top-8 right-7 rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-500">
            日志检索
          </span>
          <h2 className="mt-6 text-2xl font-semibold">Kibana</h2>
          <p className="mt-3 max-w-md text-sm leading-7 text-slate-500">
            检索 Filebeat 收集到 Elasticsearch
            的日志，按时间范围和业务标识定位问题。
          </p>
          <div className="mt-5 flex flex-wrap gap-2">
            {["应用日志", "慢 SQL", "错误上下文"].map((text) => (
              <span
                key={text}
                className="rounded-md border border-slate-200 px-2 py-1 text-xs text-slate-500"
              >
                {text}
              </span>
            ))}
          </div>
          <div className="mt-8">
            <ToolLink
              href={system?.kibanaUrl}
              className="border border-slate-200 text-slate-700 hover:bg-slate-50"
            >
              打开 Kibana
            </ToolLink>
          </div>
        </Card>
      </div>
      <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
        <Card className="p-6">
          <h2 className="font-semibold">排障路径</h2>
          <div className="mt-6 space-y-6">
            {[
              {
                icon: Activity,
                title: "先确认服务状态",
                detail: "查看当前依赖探测和业务变化，确认异常影响范围。",
              },
              {
                icon: ChartNoAxesCombined,
                title: "再定位异常时段",
                detail:
                  "在 Grafana 的 seekF 后端看板中查看请求量、错误率和耗时趋势。",
              },
              {
                icon: FileSearch,
                title: "最后查看日志上下文",
                detail: "在 Kibana 中按相同时间范围和业务标识查找相关日志。",
              },
            ].map((step, i) => (
              <div key={step.title} className="flex gap-4">
                <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-slate-50 text-slate-500">
                  <step.icon className="size-4" />
                </span>
                <div>
                  <h3 className="text-sm font-medium">
                    <span className="mr-2 text-slate-400">0{i + 1}</span>
                    {step.title}
                  </h3>
                  <p className="mt-1 text-xs leading-6 text-slate-500">
                    {step.detail}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </Card>
        <ServiceStatus system={system} />
      </div>
      <Card className="flex flex-col gap-5 p-6 sm:flex-row">
        <span className="self-start rounded-lg bg-emerald-50 p-2 text-emerald-700">
          <BellRing className="size-5" />
        </span>
        <div className="flex-1">
          <h2 className="font-semibold">告警由 Grafana 统一管理</h2>
          <p className="mt-2 text-sm leading-6 text-slate-500">
            在 Grafana
            配置规则、联系点和通知策略。本页不读取告警记录，当前服务状态也不等同于告警状态。
          </p>
        </div>
        <ToolLink
          href={system?.grafanaUrl}
          className="self-start text-emerald-700 hover:bg-emerald-50"
        >
          进入告警配置
        </ToolLink>
      </Card>
      <p className="flex items-center gap-2 text-xs text-slate-400">
        <ShieldCheck className="size-4 shrink-0" />
        工具入口沿用各自的登录权限。
        <ArrowRight className="size-3" />
        管理端不保存监控工具密码。
      </p>
    </div>
  );
}
