import { ConversationTimeline } from '../components/ConversationTimeline';
import { LiveMetrics } from '../components/LiveMetrics';
import { StatusRail } from '../components/StatusRail';
import type { PageProps } from './types';

export function OverviewPage(props: PageProps) { const { locale, analysis, connections, selected } = props; return <div className="overview-grid"><div className="overview-main"><LiveMetrics analysis={analysis} locale={locale} /><div className="section-heading"><div><span>LIVE / 01</span><h2>{locale === 'zh' ? '正在发生的对话' : 'Conversation in progress'}</h2></div><button onClick={props.refresh}>{locale === 'zh' ? '立即刷新' : 'Refresh'} ↻</button></div><ConversationTimeline conversation={selected} locale={locale} /></div><StatusRail connections={connections} locale={locale} /></div>; }
