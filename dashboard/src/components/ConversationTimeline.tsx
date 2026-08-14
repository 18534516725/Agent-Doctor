import type { Conversation } from '../api';
import type { Locale } from '../i18n';

export function ConversationTimeline({ conversation, locale }: { conversation?: Conversation; locale: Locale }) {
  if (!conversation) return <div className="listening-empty"><div className="radar" aria-hidden="true"><i /><i /><b /></div><strong>{locale === 'zh' ? '等待第一条完整对话' : 'Waiting for the first full conversation'}</strong><p>{locale === 'zh' ? '连接客户端并发起一次模型调用，消息会实时出现在这里。' : 'Connect a client and make a model call. Messages will appear here live.'}</p></div>;
  return <section className="conversation-timeline" aria-label={locale === 'zh' ? '完整对话' : 'Full conversation'}>{conversation.messages.map((message) => <article className={`message message-${message.role}`} key={message.id || `${message.sequence}-${message.role}`}><header><span>{roleName(message.role, locale)}</span><time>{formatTime(message.createdAt, locale)}</time></header>{message.content && <p>{message.content}</p>}{message.toolName && <div className="tool-call"><strong>{message.toolName}</strong><pre>{message.toolPayloadJson}</pre></div>}</article>)}</section>;
}

function roleName(role: string, locale: Locale) { const names = locale === 'zh' ? { system: '系统', user: '你', assistant: '模型', tool: '工具' } : { system: 'System', user: 'You', assistant: 'Assistant', tool: 'Tool' }; return names[role as keyof typeof names] ?? role; }
function formatTime(value: string, locale: Locale) { if (!value) return ''; const date = new Date(value); return Number.isNaN(date.valueOf()) ? value : date.toLocaleTimeString(locale === 'zh' ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit' }); }
