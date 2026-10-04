import { formatCompactNumber, formatCost, formatNumber } from '../../lib/formatters'
import type { ModelStats, ToolStats } from '../../api'

export function ModelBreakdown({ models }: { models: ModelStats[] }) {
  const maximum = Math.max(...models.map((model) => model.tokens), 1)

  return (
    <div className="breakdown-list">
      {models.map((model) => (
        <div className="breakdown-row" key={model.model}>
          <div className="breakdown-row-head">
            <span className="breakdown-name" title={model.model}>{model.model}</span>
            <span>{formatCompactNumber(model.tokens)} tokens</span>
          </div>
          <div className="progress-track" aria-hidden="true">
            <span style={{ width: `${Math.max((model.tokens / maximum) * 100, 3)}%` }} />
          </div>
          <div className="breakdown-meta">
            <span>{formatNumber(model.sessions)} sessions</span>
            <span>{formatCost(model.cost)}</span>
          </div>
        </div>
      ))}
    </div>
  )
}

export function ToolBreakdown({ tools }: { tools: ToolStats[] }) {
  const maximum = Math.max(...tools.map((tool) => tool.calls), 1)

  return (
    <div className="breakdown-list">
      {tools.map((tool) => (
        <div className="breakdown-row" key={tool.tool}>
          <div className="breakdown-row-head">
            <span className="breakdown-name" title={tool.tool}>{tool.tool}</span>
            <span>{formatNumber(tool.calls)} appels</span>
          </div>
          <div className="progress-track progress-track-blue" aria-hidden="true">
            <span style={{ width: `${Math.max((tool.calls / maximum) * 100, 3)}%` }} />
          </div>
          {tool.cost !== undefined ? <div className="breakdown-meta"><span>{formatCost(tool.cost)}</span></div> : null}
        </div>
      ))}
    </div>
  )
}
