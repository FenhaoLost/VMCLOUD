import { useState, useEffect, useCallback } from 'react'
import { RefreshCw } from 'lucide-react'
import { getSecurityAlerts, SecurityAlert } from '../services/api'

const typeLabels: Record<string, string> = {
  port_scan: '端口扫描',
  horizontal_scan: '横向扫描',
  brute_force: '暴力破解',
  ddos: 'DDoS/大规模扫描',
  spam: '垃圾邮件',
  malware: '恶意软件',
  mining: '挖矿连接',
  proxy: '代理/VPN/Tor',
  reflection: 'UDP反射放大',
}

const severityLabels: Record<string, string> = {
  critical: '严重',
  high: '高危',
  medium: '中危',
  low: '低危',
}

export default function Security() {
  const [alerts, setAlerts] = useState<SecurityAlert[]>([])
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    try {
      const alertRes = await getSecurityAlerts()
      if (alertRes.data.data) setAlerts(alertRes.data.data)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 10000)
    return () => clearInterval(interval)
  }, [fetchData])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-black"></div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-black">安全告警</h1>
        <button
          onClick={fetchData}
          className="inline-flex items-center gap-2 px-3 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 text-sm"
        >
          <RefreshCw className="w-4 h-4" />
          刷新
        </button>
      </div>

      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
          <h2 className="text-sm font-semibold text-black">告警列表 ({alerts.length})</h2>
        </div>
        {alerts.length === 0 ? (
          <div className="p-8 text-center text-sm text-gray-500">暂无安全告警</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-100 text-left text-xs font-medium text-gray-500">
                  <th className="px-4 py-2.5 whitespace-nowrap">时间</th>
                  <th className="px-4 py-2.5 whitespace-nowrap">等级</th>
                  <th className="px-4 py-2.5 whitespace-nowrap">类型</th>
                  <th className="px-4 py-2.5 whitespace-nowrap">容器</th>
                  <th className="px-4 py-2.5 whitespace-nowrap">源IP</th>
                  <th className="px-4 py-2.5 whitespace-nowrap">目标</th>
                  <th className="px-4 py-2.5 whitespace-nowrap">次数</th>
                  <th className="px-4 py-2.5">详情</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {alerts.map((alert) => (
                  <tr key={alert.id} className="hover:bg-gray-50">
                    <td className="px-4 py-2.5 font-mono text-xs text-gray-500 whitespace-nowrap">{alert.timestamp}</td>
                    <td className="px-4 py-2.5 whitespace-nowrap">
                      <SeverityBadge severity={alert.severity} />
                    </td>
                    <td className="px-4 py-2.5 text-gray-800 whitespace-nowrap">{typeLabels[alert.type] || alert.type}</td>
                    <td className="px-4 py-2.5 font-mono text-xs text-gray-700 whitespace-nowrap">{alert.container_name}</td>
                    <td className="px-4 py-2.5 font-mono text-xs text-gray-600 whitespace-nowrap">{alert.source_ip}</td>
                    <td className="px-4 py-2.5 font-mono text-xs text-gray-600 whitespace-nowrap">
                      {formatTarget(alert)}
                    </td>
                    <td className="px-4 py-2.5 text-gray-600 whitespace-nowrap">{alert.count}</td>
                    <td className="px-4 py-2.5 text-gray-600 min-w-[260px]">{alert.detail}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

function SeverityBadge({ severity }: { severity: string }) {
  const colors: Record<string, string> = {
    critical: 'bg-red-100 text-red-700',
    high: 'bg-amber-100 text-amber-700',
    medium: 'bg-gray-100 text-gray-700',
    low: 'bg-gray-50 text-gray-500',
  }
  return (
    <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${colors[severity] || 'bg-gray-100 text-gray-700'}`}>
      {severityLabels[severity] || severity}
    </span>
  )
}

function formatTarget(alert: SecurityAlert): string {
  if (alert.target_ip === '*') return '*'
  if (!alert.target_ip) return '-'
  return alert.target_port > 0 ? `${alert.target_ip}:${alert.target_port}` : alert.target_ip
}
