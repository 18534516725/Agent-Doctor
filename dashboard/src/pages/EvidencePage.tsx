import type { Message } from '../api';
import { ConversationTimeline } from '../components/ConversationTimeline';
import type { PageProps } from './types';

export function EvidencePage({ locale, api, conversations, selected, selectConversation, refresh }: PageProps) {
  const save = async (message: Message) => { if (!selected) return; await api.createMemory(selected.projectId, { content: message.content, sourceKind: 'conversation-message', sourceId: message.id }); refresh(); };
  return <div className="split-workspace"><aside className="conversation-index"><h2>{locale === 'zh' ? '最近会话' : 'Recent conversations'}</h2>{conversations.length === 0 ? <p>{locale === 'zh' ? '暂无记录，监听器已待命。' : 'No records yet. Listener is ready.'}</p> : conversations.map((item) => <button className={selected?.id === item.id ? 'is-selected' : ''} key={item.id} onClick={() => selectConversation(item.id)}><strong>{item.client.name} · {item.model.displayName}</strong><span>{item.messages.find((message) => message.role === 'user')?.content || item.path}</span><small>{item.usage.inputTokens ?? '—'} in / {item.usage.outputTokens ?? '—'} out</small></button>)}</aside><ConversationTimeline conversation={selected} locale={locale} onSaveMemory={save} /></div>;
}
