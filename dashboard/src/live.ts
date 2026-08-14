import { useEffect, useState } from 'react';
import { sessionToken } from './api';

export type LiveState = 'connecting' | 'connected' | 'error';
export type EventSourceLike = { close(): void; onopen: ((event: Event) => void) | null; onerror: ((event: Event) => void) | null; onmessage: ((event: MessageEvent) => void) | null; addEventListener(type: string, listener: EventListener): void };
export type EventSourceFactory = (url: string) => EventSourceLike;

export function useLiveUpdates(onChange: () => void, factory?: EventSourceFactory) {
  const [state, setState] = useState<LiveState>('connecting');
  useEffect(() => {
    const create = factory ?? ((url: string) => new EventSource(url));
    if (!factory && typeof EventSource === 'undefined') { setState('error'); return; }
    const source = create(`/api/v1/live?token=${encodeURIComponent(sessionToken())}`);
    source.onopen = () => setState('connected');
    source.onerror = () => setState('error');
    const notify = () => { setState('connected'); onChange(); };
    for (const kind of ['conversation.saved', 'conversation.deleted', 'connection.changed', 'analysis.changed']) source.addEventListener(kind, notify as EventListener);
    // 捕获代理可能运行在另一个本地进程中；SQLite 会同步写入，但它无法发布到
    // 当前进程的 SSE Hub。低频轮询作为跨进程兜底，保证无需手动刷新。
    const poll = window.setInterval(onChange, 2500);
    return () => { window.clearInterval(poll); source.close(); };
  }, [factory, onChange]);
  return state;
}
