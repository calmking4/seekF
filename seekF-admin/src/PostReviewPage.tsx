import { useEffect, useRef, useState } from "react";
import { api, errorMessage } from "./api";
import { ErrorNotice } from "./components";
import { ImageIcon, Play } from "lucide-react";

type Post = {
  uuid: string; user_id: string; author_name: string; title: string; content?: string;
  media_type: number; visibility: number; support_only: number; status: number;
  version: number; review_note: string; cover_url: string; tags?: string[]; urls?: string[]; created_at: string; updated_at: string;
};
type Result = { list: Post[]; total: number; counts: Record<number, number> };
const states = ["已发布", "已下架", "审核中", "未通过"];
const button = "rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm hover:bg-slate-50 focus-visible:outline-emerald-600 disabled:opacity-40";
const date = (value: string) => new Date(value).toLocaleString("zh-CN", { hour12: false });
const stateStyles = ["bg-emerald-50 text-emerald-700", "bg-slate-100 text-slate-500", "bg-amber-50 text-amber-700", "bg-red-50 text-red-600"];

function PostCover({ post, disabled, onOpen }: { post: Post; disabled: boolean; onOpen: () => void }) {
  const [failed, setFailed] = useState(false);
  return (
    <button type="button" disabled={disabled} onClick={onOpen} aria-label={`查看帖子：${post.title}`} className="relative h-32 w-24 shrink-0 overflow-hidden rounded-xl bg-slate-100 focus-visible:outline-emerald-600 disabled:opacity-60 sm:h-36 sm:w-28">
      {post.cover_url && !failed
        ? <img src={post.cover_url} alt={post.title} loading="lazy" onError={() => setFailed(true)} className="h-full w-full object-cover" />
        : <span className="flex h-full flex-col items-center justify-center gap-2 text-slate-400"><ImageIcon className="size-7" /><span className="text-[10px]">{failed ? "封面加载失败" : "暂无封面"}</span></span>}
      {post.media_type === 1 && <span className="absolute bottom-2 right-2 rounded-full bg-black/50 p-1.5 text-white"><Play className="size-3" aria-hidden="true" /></span>}
    </button>
  );
}

export default function PostReviewPage() {
  const [status, setStatus] = useState<number | null>(2);
  const [keyword, setKeyword] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [revision, setRevision] = useState(0);
  const [result, setResult] = useState<Result>({ list: [], total: 0, counts: {} });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<Post | null>(null);
  const [opening, setOpening] = useState(false);
  const [saving, setSaving] = useState(false);
  const [note, setNote] = useState("");
  const [reviewError, setReviewError] = useState("");
  const dialog = useRef<HTMLDialogElement>(null);
  const backdropPointerDown = useRef(false);
  const detailRequest = useRef<AbortController | null>(null);
  const actionRequest = useRef<AbortController | null>(null);

  // 只有从遮罩按下并在遮罩松开的点击才关闭，避免正文拖选误触。
  const outsideDialog = (x: number, y: number) => {
    const bounds = dialog.current?.getBoundingClientRect();
    return !!bounds && (x < bounds.left || x > bounds.right || y < bounds.top || y > bounds.bottom);
  };

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true); setError("");
    api<Result>("/discover/posts", {
      method: "POST", body: JSON.stringify({ status, keyword: query, page }), signal: controller.signal,
    }).then(data => {
      if (!controller.signal.aborted) {
        setResult(data);
        if (!data.list.length && page > 1 && data.total > 0) setPage(Math.ceil(data.total / 20));
      }
    }).catch(cause => {
      if (!controller.signal.aborted) setError(errorMessage(cause));
    }).finally(() => {
      if (!controller.signal.aborted) setLoading(false);
    });
    return () => controller.abort();
  }, [status, query, page, revision]);

  useEffect(() => {
    if (detail && !dialog.current?.open) dialog.current?.showModal();
    if (!detail && dialog.current?.open) dialog.current.close();
  }, [detail]);
  useEffect(() => () => { detailRequest.current?.abort(); actionRequest.current?.abort(); }, []);

  const open = async (post: Post) => {
    detailRequest.current?.abort();
    const controller = new AbortController(); detailRequest.current = controller;
    setOpening(true); setError(""); setReviewError(""); setNote("");
    try {
      const data = await api<Post>("/discover/detail", {
        method: "POST", body: JSON.stringify({ post_uuid: post.uuid }), signal: controller.signal,
      });
      if (!controller.signal.aborted) setDetail(data);
    } catch (cause) { if (!controller.signal.aborted) setError(errorMessage(cause)); }
    finally { if (!controller.signal.aborted) setOpening(false); }
  };

  const review = async (target: number) => {
    if (!detail || saving) return;
    if (target !== 0 && !note.trim()) { setReviewError("请填写驳回或下架原因"); return; }
    const controller = new AbortController(); actionRequest.current = controller;
    setSaving(true); setReviewError("");
    try {
      await api("/discover/review", {
        method: "POST", signal: controller.signal,
        body: JSON.stringify({ post_uuid: detail.uuid, version: detail.version, status: target, note: target === 0 ? "" : note.trim() }),
      });
      if (!controller.signal.aborted) { setDetail(null); setRevision(value => value + 1); }
    } catch (cause) { if (!controller.signal.aborted) setReviewError(errorMessage(cause)); }
    finally { if (!controller.signal.aborted) setSaving(false); }
  };

  return (
    <section className="min-w-0 space-y-5">
      <header><h1 className="text-2xl font-semibold">帖子审核</h1><p className="mt-2 text-sm text-slate-500">新发布及编辑后的作品审核通过后展示，驳回或下架需填写原因。</p></header>
      <div className="flex flex-wrap gap-3">
        <button className={`${button} ${status === null ? "!border-emerald-200 !bg-emerald-50 font-medium !text-emerald-800" : ""}`} aria-pressed={status === null} onClick={() => { setStatus(null); setPage(1); }}>全部</button>
        {[2, 0, 3, 1].map(value => <button key={value} className={`${button} ${status === value ? "!border-emerald-200 !bg-emerald-50 font-medium !text-emerald-800" : ""}`} aria-pressed={status === value} onClick={() => { setStatus(value); setPage(1); }}>{states[value]} <span className="ml-1 tabular-nums">{result.counts[value] || 0}</span></button>)}
        <button className={button} disabled={loading} onClick={() => setRevision(value => value + 1)}>刷新</button>
      </div>
      <form className="flex flex-wrap gap-3" onSubmit={event => { event.preventDefault(); setQuery(keyword.trim()); setPage(1); setRevision(value => value + 1); }}>
        <input className={`${button} min-w-0 flex-1 sm:max-w-md`} aria-label="搜索帖子标题、作者或编号" placeholder="搜索帖子标题、作者或编号" maxLength={100} value={keyword} onChange={event => setKeyword(event.target.value)} />
        <button className={button} type="submit">搜索</button>
      </form>
      {error && <ErrorNotice message={error} />}
      {loading ? <p role="status">正在查询…</p> : !result.list.length ? <p className="p-8 text-center text-slate-500">暂无对应帖子</p> : <div className="grid min-w-0 gap-4 xl:grid-cols-2">{result.list.map(post => (
        <article key={post.uuid} className="flex min-w-0 items-start gap-4 rounded-2xl border border-slate-200 bg-white p-4 text-sm sm:p-5">
          <PostCover key={`${post.uuid}:${post.cover_url}`} post={post} disabled={opening || saving} onOpen={() => void open(post)} />
          <div className="flex min-w-0 flex-1 flex-col items-start gap-2.5">
            <div className="flex w-full min-w-0 items-start justify-between gap-2"><h2 className="line-clamp-2 min-w-0 break-words font-semibold leading-6">{post.title}</h2><span className={`shrink-0 rounded-md px-2 py-1 text-[11px] ${stateStyles[post.status] || "bg-slate-100 text-slate-500"}`}>{states[post.status]}</span></div>
            <p className="max-w-full truncate text-xs text-slate-500" title={`${post.author_name} · ${post.user_id}`}>{post.author_name} · {post.user_id}</p>
            <div className="flex flex-wrap gap-1.5 text-[10px] text-slate-500"><span className="rounded bg-slate-50 px-1.5 py-0.5">{post.media_type === 1 ? "视频" : "图文"}</span><span className="rounded bg-slate-50 px-1.5 py-0.5">{post.visibility === 1 ? "仅作者可见" : "公开"}</span><span className="rounded bg-slate-50 px-1.5 py-0.5">{post.support_only === 1 ? "助力专属" : "免费"}</span></div>
            <p className="max-w-full truncate text-[11px] text-slate-400" title={post.uuid}>{post.uuid} · {date(post.updated_at)}</p>
            {post.review_note && <p className="line-clamp-2 break-words text-xs text-amber-700" title={post.review_note}>处理原因：{post.review_note}</p>}
            <button className={`${button} mt-0.5 !py-1.5 !text-xs`} disabled={opening || saving} onClick={() => void open(post)}>{opening ? "读取中…" : "查看与审核"}</button>
          </div>
        </article>
      ))}</div>}
      <div className="flex items-center justify-center gap-3 text-sm text-slate-500">
        <button className={button} disabled={loading || page <= 1} onClick={() => setPage(value => value - 1)}>上一页</button><span>第 {page} 页 · 共 {result.total} 条</span><button className={button} disabled={loading || page * 20 >= result.total} onClick={() => setPage(value => value + 1)}>下一页</button>
      </div>
      <dialog ref={dialog} aria-labelledby="post-review-title" onPointerDown={event => { backdropPointerDown.current = event.target === event.currentTarget && outsideDialog(event.clientX, event.clientY); }} onClick={event => { if (!saving && backdropPointerDown.current && event.target === event.currentTarget && outsideDialog(event.clientX, event.clientY)) setDetail(null); backdropPointerDown.current = false; }} onCancel={event => { if (saving) event.preventDefault(); }} onClose={() => setDetail(null)} className="fixed inset-0 m-auto max-h-[90dvh] w-[calc(100%-32px)] max-w-3xl overflow-y-auto rounded-2xl bg-white p-5 text-slate-800 shadow-xl backdrop:bg-black/40 sm:p-7">
        {detail && <div className="space-y-5">
          <div className="flex items-start justify-between gap-4"><h2 id="post-review-title" className="break-all text-xl font-semibold">{detail.title}</h2><button autoFocus className={button} disabled={saving} onClick={() => setDetail(null)}>关闭</button></div>
          <p className="break-all text-sm text-slate-500">{detail.author_name} · {states[detail.status]} · {detail.visibility === 1 ? "仅作者可见" : "公开"} · {detail.support_only === 1 ? "助力专属" : "免费"}</p>
          <p className="whitespace-pre-wrap break-words text-sm leading-7">{detail.content || "无正文"}</p>
          {!!detail.tags?.length && <p className="break-words text-sm text-slate-500">标签：{detail.tags.join(" · ")}</p>}
          {detail.media_type === 1 && detail.cover_url && <figure className="space-y-2"><figcaption className="text-sm text-slate-500">视频封面</figcaption><img src={detail.cover_url} alt="视频封面" className="max-h-80 w-full rounded-xl object-contain" /></figure>}
          <div className="space-y-3">{detail.urls?.map((url, index) => detail.media_type === 1 ? <video key={url} src={url} controls preload="metadata" className="max-h-[60dvh] w-full rounded-xl bg-black" aria-label={`审核视频 ${index + 1}`} /> : <img key={url} src={url} alt={`帖子图片 ${index + 1}`} loading="lazy" className="max-h-[70dvh] w-full rounded-xl object-contain" />)}</div>
          {detail.review_note && <p className="text-sm text-amber-700">最近处理原因：{detail.review_note}</p>}
          {(detail.status === 2 || detail.status === 0) && <label className="block space-y-2 text-sm"><span>驳回 / 下架原因</span><textarea value={note} maxLength={300} disabled={saving} onChange={event => setNote(event.target.value)} rows={3} className="block w-full rounded-lg border border-slate-200 p-3 focus-visible:outline-emerald-600" placeholder="最多300字；审核通过无需填写" /></label>}
          {reviewError && <ErrorNotice message={reviewError} />}
          <div className="flex flex-wrap justify-end gap-3">
            {detail.status === 2 && <><button className={button} disabled={saving} onClick={() => void review(3)}>驳回</button><button className={button} disabled={saving} onClick={() => void review(0)}>审核通过</button></>}
            {detail.status === 0 && <button className={button} disabled={saving} onClick={() => void review(1)}>下架帖子</button>}
            {detail.status === 1 && <button className={button} disabled={saving} onClick={() => void review(0)}>恢复发布</button>}
            {detail.status === 3 && <p className="text-sm text-slate-500">等待作者修改后重新提交审核。</p>}
            {saving && <span role="status" className="text-sm text-slate-500">正在提交…</span>}
          </div>
        </div>}
      </dialog>
    </section>
  );
}
