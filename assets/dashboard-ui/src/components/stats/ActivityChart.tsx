import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import type { DailyStats } from '../../api'
import { formatCompactNumber, formatCost, formatDateShort } from '../../lib/formatters'

export function ActivityChart({ data }: { data: DailyStats[] }) {
  const chartData = data.map((item) => ({
    ...item,
    label: formatDateShort(item.date),
  }))

  return (
    <div className="chart-wrap" aria-label="Graphique de l’activité quotidienne">
      <ResponsiveContainer width="100%" height={280}>
        <AreaChart data={chartData} margin={{ top: 8, right: 8, left: -20, bottom: 0 }}>
          <defs>
            <linearGradient id="tokensGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="var(--accent)" stopOpacity={0.28} />
              <stop offset="95%" stopColor="var(--accent)" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid stroke="var(--border)" strokeDasharray="4 4" vertical={false} />
          <XAxis dataKey="label" tick={{ fill: 'var(--muted)', fontSize: 11 }} tickLine={false} axisLine={false} />
          <YAxis tick={{ fill: 'var(--muted)', fontSize: 11 }} tickLine={false} axisLine={false} tickFormatter={formatCompactNumber} />
          <Tooltip
            contentStyle={{
              background: 'var(--surface-raised)',
              border: '1px solid var(--border)',
              borderRadius: 10,
              color: 'var(--text-strong)',
            }}
            labelStyle={{ color: 'var(--muted)' }}
            formatter={(value, name) => {
              const numeric = typeof value === 'number' ? value : Number(value)
              return [name === 'cost' ? formatCost(numeric) : formatCompactNumber(numeric), name === 'cost' ? 'Coût' : 'Tokens']
            }}
          />
          <Area type="monotone" dataKey="tokens" stroke="var(--accent)" fill="url(#tokensGradient)" strokeWidth={2.5} activeDot={{ r: 5 }} />
        </AreaChart>
      </ResponsiveContainer>
      <p className="chart-caption">Tokens traités par jour</p>
    </div>
  )
}
