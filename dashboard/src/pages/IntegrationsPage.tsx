import { StatusRail } from '../components/StatusRail';
import type { PageProps } from './types';
export function IntegrationsPage({ locale, connections }: PageProps) { return <div className="integration-page"><div><span>AUTO CONNECT</span><h2>{locale === 'zh' ? '编辑器连接不是猜测' : 'Editor connections are never guessed'}</h2><p>{locale === 'zh' ? '只显示真实检测和连接状态。支持稳定配置格式的客户端可以自动接入，需要重启或手动设置的客户端会明确说明。' : 'Only real detection and connection state are shown. Restart and manual steps remain explicit.'}</p></div><StatusRail connections={connections} locale={locale} /></div>; }
