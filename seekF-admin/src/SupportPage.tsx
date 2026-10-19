import { useEffect, useRef, useState } from "react";
import { api, errorMessage } from "./api";
import { ErrorNotice, clock } from "./components";
type Order = {
    order_no: string;
    user_id: string;
    creator_id: string;
    amount: string;
    status: number;
    refund_status: number;
    refund_reason: string;
    refund_review_note: string;
    refund_requested_at?: string;
    refund_reviewed_at?: string;
    refunded_at?: string;
    reviewer_id: string;
    last_error: string;
    paid_at?: string;
};
const statuses = ["未申请", "待审核", "已驳回", "处理中", "已全额退款", "退款失败"];
export default function SupportPage() {
    const [rows, setRows] = useState<Order[]>([]), [total, setTotal] = useState(0), [page, setPage] = useState(1);
    const [query, setQuery] = useState(""), [filter, setFilter] = useState("1"), [error, setError] = useState(""), [busy, setBusy] = useState(false);
    const [target, setTarget] = useState<Order | null>(null), [action, setAction] = useState("approve"), [note, setNote] = useState(""), [saving, setSaving] = useState(false);
    const [reviewError, setReviewError] = useState("");
    const savingRef = useRef(false);
    const [revision, setRevision] = useState(0);
    const pending = useRef<AbortController | null>(null), mutation = useRef<AbortController | null>(null), closeButton = useRef<HTMLButtonElement | null>(null);
    const activeElement = useRef<HTMLElement | null>(null);
    useEffect(() => {
        const control = new AbortController();
        pending.current = control;
        setBusy(true);
        setError("");
        setRows([]);
        api<{
            list: Order[];
            total: number;
        }>("/support/orders", { method: "POST", body: JSON.stringify({ order_no: query, page, refund_status: filter === "" ? null : Number(filter) }), signal: control.signal })
            .then(data => { if (!control.signal.aborted) {
            setRows(data.list);
            setTotal(data.total);
        } })
            .catch(e => { if (!control.signal.aborted)
            setError(errorMessage(e)); }).finally(() => { if (!control.signal.aborted)
            setBusy(false); });
        return () => control.abort();
    }, [query, filter, page, revision]);
    useEffect(() => () => mutation.current?.abort(), []);
    useEffect(() => {
        if (!target)
            return;
        activeElement.current = document.activeElement as HTMLElement;
        closeButton.current?.focus();
        const key = (e: KeyboardEvent) => {
            if (e.key === "Escape" && !savingRef.current)
                setTarget(null);
            if (e.key === "Tab") {
                const elements = Array.from(document.querySelectorAll<HTMLElement>('[role="dialog"] button:not(:disabled), [role="dialog"] textarea:not(:disabled)'));
                const first = elements[0], last = elements[elements.length - 1];
                if (!first) { e.preventDefault(); return; }
                if (e.shiftKey && document.activeElement === first) {
                    e.preventDefault();
                    last?.focus();
                }
                else if (!e.shiftKey && document.activeElement === last) {
                    e.preventDefault();
                    first?.focus();
                }
            }
        };
        window.addEventListener("keydown", key);
        return () => { window.removeEventListener("keydown", key); activeElement.current?.focus(); };
    }, [target]);
    async function save() {
        if (!target || !note.trim() || saving)
            return;
        setSaving(true);
        savingRef.current = true;
        setReviewError("");
        const control = new AbortController();
        mutation.current = control;
        try {
            await api("/support/refund/review", { method: "POST", signal: control.signal, body: JSON.stringify({ order_no: target.order_no, action, note: note.trim() }) });
            if (!control.signal.aborted) {
                setTarget(null);
                setRevision(v => v + 1);
            }
        }
        catch (e) {
            if (!control.signal.aborted) {
                setReviewError(errorMessage(e));
                setRevision(v => v + 1);
            }
        }
        finally {
            savingRef.current = false;
            if (!control.signal.aborted)
                setSaving(false);
        }
    }
    const open = (row: Order, kind: string) => { setTarget(row); setAction(kind); setNote(""); setReviewError(""); };
    const button = "rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm hover:bg-slate-50 focus-visible:outline-emerald-600 disabled:opacity-40";
    return <section className="space-y-5">
  <header><h1 className="text-2xl font-semibold">助力订单与退款</h1><p className="mt-2 text-sm text-slate-500">审核通过后由后台发起全额原路退款，确认成功前保留原有权益。</p></header>
  <div className="flex flex-wrap gap-3"><input aria-label="精确订单号" className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm" placeholder="精确订单号" value={query} maxLength={64} onChange={e => { setQuery(e.target.value); setPage(1); }}/><select aria-label="退款状态" className={button} value={filter} onChange={e => { setFilter(e.target.value); setPage(1); }}><option value="">全部订单</option>{statuses.map((s, i) => <option key={i} value={i}>{s}</option>)}</select><button className={button} disabled={busy} onClick={() => setRevision(v => v + 1)}>刷新</button></div>
  {error && <ErrorNotice message={error}/>}
  {busy ? <p role="status">正在查询…</p> : !rows.length ? <p className="p-8 text-center text-slate-500">暂无符合条件的订单</p> : rows.map(row => <article key={row.order_no} className="space-y-3 rounded-2xl border border-slate-200 bg-white p-5 text-sm">
   <div className="flex flex-wrap justify-between gap-3"><strong className="break-all">{row.order_no}</strong><span>¥{row.amount} · {["待付款", "已付款", "已关闭"][row.status]} · {statuses[row.refund_status]}</span></div>
   <p className="break-all text-slate-500">购买者：{row.user_id}　创作者：{row.creator_id}</p>
   <p className="text-slate-500">付款：{clock(row.paid_at)}　申请：{clock(row.refund_requested_at)}</p>
   {row.refund_reason && <p className="break-words">申请原因：{row.refund_reason}</p>}
   {row.refund_review_note && <p className="break-words">审核：{row.refund_review_note}（{row.reviewer_id}，{clock(row.refund_reviewed_at)}）</p>}
   {row.refunded_at && <p>退款确认：{clock(row.refunded_at)}</p>}
   {row.last_error && <p className="break-words text-amber-700">{row.last_error}</p>}
   <div className="flex gap-3">{row.refund_status === 1 && <><button className={button} onClick={() => open(row, "approve")}>批准全额退款</button><button className={button} onClick={() => open(row, "reject")}>驳回</button></>}{row.refund_status === 5 && !row.last_error.startsWith("需人工核对") && <button className={button} onClick={() => open(row, "retry")}>按原请求重试</button>}</div>
  </article>)}
  <div className="flex items-center justify-center gap-4 text-sm"><button className={button} disabled={busy || page === 1} onClick={() => setPage(v => v - 1)}>上一页</button><span>第{page}页 · 共{total}条</span><button className={button} disabled={busy || page * 20 >= total} onClick={() => setPage(v => v + 1)}>下一页</button></div>
  {target && <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"><div role="dialog" aria-modal="true" aria-labelledby="refund-review-title" className="w-full max-w-lg space-y-4 rounded-2xl bg-white p-6"><h2 id="refund-review-title" className="font-semibold">{action === "reject" ? "驳回退款申请" : action === "retry" ? "按原请求重试" : "批准全额退款"}</h2><p className="break-all text-sm text-slate-500">{target.order_no} · ¥{target.amount}</p><textarea aria-label="审核说明" placeholder="填写审核说明或驳回原因" className="w-full rounded-lg border border-slate-200 p-3 text-sm" rows={4} maxLength={500} value={note} disabled={saving} onChange={e => setNote(e.target.value)}/>{reviewError && <ErrorNotice message={reviewError}/>}<div className="flex justify-end gap-3"><button ref={closeButton} className={button} disabled={saving} onClick={() => setTarget(null)}>取消</button><button className={button} disabled={saving || !note.trim()} onClick={() => void save()}>{saving ? "正在保存…" : "确认提交"}</button></div></div></div>}
 </section>;
}
