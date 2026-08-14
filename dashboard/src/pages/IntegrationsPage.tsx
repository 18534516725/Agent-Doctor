import type { ClientConnection } from '../api';
import type { PageProps } from './types';

export function IntegrationsPage({ locale, connections }: PageProps) {
  const zh = locale === 'zh';
  return <div className="operations-page"><section className="operations-intro"><span>{zh ? '能力边界，而不只是在线灯' : 'CAPABILITIES, NOT JUST DOTS'}</span><h2>{zh ? '每个客户端到底能做什么' : 'What each client can actually do'}</h2><p>{zh ? '“已发现”不等于“正在监听”。只有出现心跳或真实证据才标记活动。' : 'Detected does not mean observed; active requires heartbeat or real evidence.'}</p></section><div className="capability-list">{connections.map((item) => <Capability key={item.key} item={item} zh={zh} />)}</div></div>;
}

function Capability({ item, zh }: { item: ClientConnection; zh: boolean }) {
  const observed = item.state === 'active' || item.state === 'connected';
  const advice = item.key === 'codex' || item.key === 'claude-code';
  const enforcement = item.key === 'claude-code';
  const repair = item.key === 'codex' ? (zh ? '运行 agent-doctor setup --all --yes 后重启 Codex' : 'Run setup --all --yes, then restart Codex') : item.key === 'claude-code' ? (zh ? '安装 Hook 后重启 Claude Code' : 'Install the Hook, then restart Claude Code') : (zh ? '当前仅支持检测；尚无实时引导适配器' : 'Detection only; no live guidance adapter yet');
  return <article className={`capability-card state-${item.state}`}><header><i /><div><h3>{item.displayName}</h3><p>{state(item.state, zh)}</p></div><time>{item.lastHeartbeatAt ? new Date(item.lastHeartbeatAt).toLocaleString(zh ? 'zh-CN' : 'en-US') : (zh ? '暂无心跳' : 'No heartbeat')}</time></header><dl><Flag label={zh ? '观察调用' : 'Observe'} value={observed} zh={zh} /><Flag label={zh ? '指导 AI' : 'Advise'} value={advice && observed} zh={zh} /><Flag label={zh ? '有限拦截' : 'Enforce'} value={enforcement && observed} zh={zh} /></dl><p className="repair-copy"><strong>{zh ? '接入方法：' : 'Repair: '}</strong>{repair}</p>{item.detail && <small>{item.detail}</small>}</article>;
}
function Flag({ label, value, zh }: { label: string; value: boolean; zh: boolean }) { return <div><dt>{label}</dt><dd>{value ? (zh ? '可用' : 'Available') : (zh ? '不可用' : 'Unavailable')}</dd></div>; }
function state(value: ClientConnection['state'], zh: boolean) { return zh ? ({ unavailable: '未安装或未发现', detected: '已发现，尚未连接', connected: '已连接，等待证据', active: '正在监听真实调用', error: '连接异常' }[value]) : value; }
