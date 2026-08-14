import type { ClientConnection } from '../api';
import type { Locale } from '../i18n';

export function StatusRail({ connections, locale }: { connections: ClientConnection[]; locale: Locale }) {
  const visible = connections.filter((item) =>
    item.detected || item.state === 'connected' || item.state === 'active',
  );
  const active = visible.filter((item) => item.state === 'active' || item.state === 'connected');

  const statusLabel = (item: ClientConnection) => {
    if (item.state === 'active') return locale === 'zh' ? '活动' : 'Live';
    if (item.state === 'connected') return locale === 'zh' ? '已连接' : 'Connected';
    return locale === 'zh' ? '已发现' : 'Found';
  };

  return (
    <aside className="status-rail">
      <div className="rail-heading">
        <span>{locale === 'zh' ? '连接状态' : 'Connections'}</span>
        <strong>{active.length}/{visible.length}</strong>
      </div>
      <div className="rail-list">
        {visible.map((item) => (
          <article key={item.key} className={`connection-${item.state}`}>
            <i aria-hidden="true" />
            <div>
              <strong>{item.displayName}</strong>
              <small>{item.detail || item.capability || (locale === 'zh' ? '已发现本机客户端' : 'Client found on this device')}</small>
            </div>
            <span>{statusLabel(item)}</span>
          </article>
        ))}
        {visible.length === 0 && (
          <p className="rail-empty">
            {locale === 'zh' ? '暂未发现支持的客户端' : 'No supported clients found yet'}
          </p>
        )}
      </div>
    </aside>
  );
}
