import type { ClientConnection } from '../api';
import type { Locale } from '../i18n';

export function StatusRail({ connections, locale }: { connections: ClientConnection[]; locale: Locale }) {
  const active = connections.filter((item) => item.state === 'active' || item.state === 'connected');
  return <aside className="status-rail"><div className="rail-heading"><span>{locale === 'zh' ? '连接状态' : 'Connections'}</span><strong>{active.length}/{connections.length || 11}</strong></div><div className="rail-list">{connections.map((item) => <article key={item.key} className={`connection-${item.state}`}><i aria-hidden="true" /><div><strong>{item.displayName}</strong><small>{item.detail || item.capability || (locale === 'zh' ? '等待配置' : 'Awaiting setup')}</small></div><span>{item.state === 'active' ? (locale === 'zh' ? '活动' : 'Live') : item.detected ? (locale === 'zh' ? '已发现' : 'Found') : (locale === 'zh' ? '未连接' : 'Offline')}</span></article>)}</div></aside>;
}
