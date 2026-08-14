import { useState } from 'react';

import { AnalysisCockpit } from '../components/AnalysisCockpit';
import { ConversationTimeline } from '../components/ConversationTimeline';
import { LiveMetrics } from '../components/LiveMetrics';
import { StatusRail } from '../components/StatusRail';
import { TaskGuardian } from '../components/TaskGuardian';
import type { PageProps } from './types';

export function OverviewPage(props: PageProps) {
  const { locale, analysis, connections, selected, guidance, guidanceStatus, controlLevel, setGuidanceControlLevel } = props;
  const [showRaw, setShowRaw] = useState(false);
  const current = guidance[0];
  const projectId = current?.projectId || analysis.projectId || selected?.projectId || '';
  const clientLabel = selected?.client.name || connections.find((item) => item.state === 'active')?.displayName || (locale === 'zh' ? '等待连接' : 'Awaiting client');
  const sessionLabel = current?.sessionId || selected?.sessionId || (locale === 'zh' ? '暂无活动任务' : 'No active task');
  const zh = locale === 'zh';
  const saveMemory = async (message: import('../api').Message) => { if (!selected) return; await props.api.createMemory(selected.projectId, { content: message.content, sourceKind: 'conversation-message', sourceId: message.id }); props.refresh(); };

  return <div className="overview-grid">
    <div className="overview-main">
      <TaskGuardian
        locale={locale}
        guidance={guidance}
        status={guidanceStatus}
        controlLevel={controlLevel}
        projectId={projectId}
        sessionLabel={sessionLabel}
        clientLabel={clientLabel}
        onControlLevelChange={setGuidanceControlLevel}
      />
      <details className="analysis-evidence">
        <summary>{zh ? '展开项目分析与运行指标' : 'Expand project analysis and runtime metrics'}</summary>
        <h2 className="analysis-evidence-title">{zh ? '项目分析驾驶舱' : 'Project analysis cockpit'}</h2>
        <AnalysisCockpit analysis={analysis} locale={locale} />
        <LiveMetrics analysis={analysis} locale={locale} />
      </details>
      <div className="evidence-disclosure">
        <div>
          <span>{zh ? '原始证据' : 'Raw evidence'}</span>
          <h2>{zh ? '需要时再核对完整对话' : 'Inspect the full transcript on demand'}</h2>
          <p>{zh ? '首页先给出干预结论；用户、模型和工具原文不会默认铺满页面。' : 'The overview leads with intervention; user, model and tool messages stay collapsed.'}</p>
        </div>
        <button aria-expanded={showRaw} onClick={() => setShowRaw((value) => !value)}>
          {showRaw ? (zh ? '收起原始对话' : 'Hide transcript') : (zh ? '查看原始对话' : 'View raw transcript')}
        </button>
      </div>
      {showRaw && <ConversationTimeline conversation={selected} locale={locale} onSaveMemory={saveMemory} />}
    </div>
    <StatusRail connections={connections} locale={locale} />
  </div>;
}
