import { useEffect, useState } from 'react';

import type { ProjectMemory } from '../api';
import type { PageProps } from './types';

type Filter = '' | ProjectMemory['state'];

export function MemoryPage({ locale, api, analysis, selected }: PageProps) {
  const projectId = analysis.projectId || selected?.projectId || '';
  const [items, setItems] = useState<ProjectMemory[]>([]);
  const [filter, setFilter] = useState<Filter>('');
  const [content, setContent] = useState('');
  const zh = locale === 'zh';
  const refresh = () => projectId ? api.loadMemories(projectId, filter).then(setItems) : Promise.resolve();
  useEffect(() => { void refresh(); }, [api, projectId, filter]);
  const create = async () => { if (!content.trim() || !projectId) return; await api.createMemory(projectId, { content: content.trim(), sourceKind: 'manual' }); setContent(''); await refresh(); };
  const update = async (item: ProjectMemory, input: { content?: string; state?: string }) => { await api.updateMemory(projectId, item.id, input); await refresh(); };
  if (!projectId) return <section className="diagnosis-empty"><h2>{zh ? '等待第一个项目' : 'Waiting for a project'}</h2><p>{zh ? '完成一次真实 AI 调用后，就能在这里建立可复用的项目记忆。' : 'Complete one real AI call to create reusable project memory.'}</p></section>;
  return <div className="memory-workspace">
    <section className="memory-compose"><span>{zh ? '明确保存，不自动猜测' : 'EXPLICIT, NEVER INFERRED'}</span><h2>{zh ? '添加一条项目记忆' : 'Add project memory'}</h2><p>{zh ? '新内容先进入“待确认”，确认后才会成为可复用上下文。' : 'New items remain candidates until you confirm them.'}</p><textarea aria-label={zh ? '记忆内容' : 'Memory content'} maxLength={16384} value={content} onChange={(event) => setContent(event.target.value)} placeholder={zh ? '例如：完成前必须运行 go test ./...' : 'Example: run tests before claiming completion'} /><button className="primary-action" disabled={!content.trim()} onClick={() => void create()}>{zh ? '保存为待确认记忆' : 'Save as candidate'}</button></section>
    <nav className="memory-filters" aria-label={zh ? '记忆状态' : 'Memory state'}>{([['', zh ? '全部' : 'All'], ['candidate', zh ? '待确认' : 'Candidate'], ['active', zh ? '已启用' : 'Active'], ['disabled', zh ? '已停用' : 'Disabled']] as [Filter,string][]).map(([value,label]) => <button key={value} className={filter === value ? 'is-active' : ''} onClick={() => setFilter(value)}>{label}</button>)}</nav>
    <section className="memory-list">{items.length === 0 ? <div className="listening-empty"><h2>{zh ? '这个状态下还没有记忆' : 'No memories in this state'}</h2><p>{zh ? '你可以手动添加，或在完整对话中选择值得保留的消息。' : 'Add one manually or save a useful conversation message.'}</p></div> : items.map((item) => <MemoryRow key={item.id} item={item} zh={zh} update={update} remove={async () => { await api.deleteMemory(projectId, item.id); await refresh(); }} />)}</section>
  </div>;
}

function MemoryRow({ item, zh, update, remove }: { item: ProjectMemory; zh: boolean; update(item: ProjectMemory, input: { content?: string; state?: string }): Promise<void>; remove(): Promise<void> }) {
  const [draft, setDraft] = useState(item.content);
  return <article className={`memory-row memory-${item.state}`}><div><span>{stateLabel(item.state, zh)} · {item.sourceKind}</span><textarea aria-label={zh ? '编辑记忆' : 'Edit memory'} value={draft} onChange={(event) => setDraft(event.target.value)} /><small>{item.sourceId ? `${zh ? '来源' : 'Source'} ${item.sourceId}` : (zh ? '手动创建' : 'Created manually')}</small></div><div className="memory-actions"><button disabled={draft.trim() === item.content} onClick={() => void update(item, { content: draft.trim() })}>{zh ? '保存修改' : 'Save edit'}</button>{item.state === 'candidate' && <button onClick={() => void update(item, { state: 'active' })}>{zh ? '确认并启用' : 'Confirm'}</button>}{item.state === 'active' && <button onClick={() => void update(item, { state: 'disabled' })}>{zh ? '停用' : 'Disable'}</button>}{item.state === 'disabled' && <button onClick={() => void update(item, { state: 'active' })}>{zh ? '重新启用' : 'Enable'}</button>}<button onClick={() => void remove()}>{zh ? '删除' : 'Delete'}</button></div></article>;
}
function stateLabel(state: ProjectMemory['state'], zh: boolean) { return zh ? ({ candidate: '待确认', active: '已启用', disabled: '已停用' }[state]) : state; }
