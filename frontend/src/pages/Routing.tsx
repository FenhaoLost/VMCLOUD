import { useCallback, useEffect, useState } from 'react'
import { Network, RefreshCw, Route, Server } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { getRoutingInfo, RoutingInfo } from '../services/api'

export default function Routing() {
  const navigate = useNavigate()
  const [routing, setRouting] = useState<RoutingInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [nat4Page, setNat4Page] = useState(1)
  const [ipv6Page, setIPv6Page] = useState(1)

  const fetchData = useCallback(async () => {
    try {
      const res = await getRoutingInfo()
      setRouting(res.data.data || null)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [])

  useEffect(() => { fetchData() }, [fetchData])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-b-2 border-black" />
      </div>
    )
  }

  const nat4Mappings = routing?.nat4_mappings || []
  const ipv6Assignments = routing?.ipv6_assignments || []
  const ipv6Prefix = routing?.ipv6_prefixes?.[0]?.prefix || '-'
  const pageSize = 10
  const nat4TotalPages = Math.max(1, Math.ceil(nat4Mappings.length / pageSize))
  const ipv6TotalPages = Math.max(1, Math.ceil(ipv6Assignments.length / pageSize))
  const currentNat4Page = Math.min(nat4Page, nat4TotalPages)
  const currentIPv6Page = Math.min(ipv6Page, ipv6TotalPages)
  const pagedNat4Mappings = nat4Mappings.slice((currentNat4Page - 1) * pageSize, currentNat4Page * pageSize)
  const pagedIPv6Assignments = ipv6Assignments.slice((currentIPv6Page - 1) * pageSize, currentIPv6Page * pageSize)

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-black">路由管理</h1>
          <p className="mt-1 text-sm text-gray-500">宿主机分配给 LXC 的 NAT4 端口和 IPv6 地址</p>
        </div>
        <button
          onClick={() => { setRefreshing(true); fetchData() }}
          disabled={refreshing}
          className="inline-flex items-center gap-2 rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        >
          <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <CapacityCard
          title="NAT4 端口"
          icon={<Route className="h-5 w-5 text-gray-600" />}
          remaining={routing?.nat4.remaining || '0'}
          total={routing?.nat4.total || '0'}
          used={routing?.nat4.used || 0}
          label="剩余端口 / 端口总数"
        />
        <CapacityCard
          title="IPv6 地址"
          icon={<Network className="h-5 w-5 text-gray-600" />}
          remaining={formatCapacity(routing?.ipv6.remaining || '0')}
          total={formatCapacity(routing?.ipv6.total || '0')}
          used={routing?.ipv6.used || 0}
          label={`剩余地址 / 地址总数 · ${ipv6Prefix}`}
        />
      </div>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <div className="border-b border-gray-200 px-4 py-3">
          <div className="text-sm font-medium text-black">NAT4 端口分配</div>
          <div className="mt-1 text-xs text-gray-500">共 {nat4Mappings.length} 条映射</div>
        </div>
        {nat4Mappings.length === 0 ? (
          <EmptyState icon={<Route className="h-7 w-7 text-gray-400" />} text="暂无 NAT4 端口映射" />
        ) : (
          <>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[900px] text-sm">
              <thead className="border-b border-gray-200 bg-gray-50 text-xs text-gray-500">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">容器</th>
                  <th className="px-4 py-3 text-left font-medium">LXC 名称</th>
                  <th className="px-4 py-3 text-left font-medium">容器 IPv4</th>
                  <th className="px-4 py-3 text-left font-medium">宿主机端口</th>
                  <th className="px-4 py-3 text-left font-medium">容器端口</th>
                  <th className="px-4 py-3 text-left font-medium">协议</th>
                  <th className="px-4 py-3 text-left font-medium">说明</th>
                  <th className="px-4 py-3 text-left font-medium">状态</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {pagedNat4Mappings.map((mapping, index) => (
                  <tr key={`${mapping.container_id}-${mapping.host_port}-${mapping.protocol}-${index}`} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <button
                        onClick={() => navigate(`/container/${mapping.container_id}`)}
                        className="inline-flex items-center gap-2 text-left font-medium text-black hover:underline"
                      >
                        <Server className="h-4 w-4 text-gray-400" />
                        {mapping.container_name}
                      </button>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-600">{mapping.lxc_name}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-600">{mapping.ip || '-'}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-700">{mapping.host_port}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-700">{mapping.container_port}</td>
                    <td className="px-4 py-3 uppercase text-gray-600">{mapping.protocol || '-'}</td>
                    <td className="px-4 py-3 text-gray-600">{mapping.description || '-'}</td>
                    <td className="px-4 py-3"><StatusBadge status={mapping.status} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination
            page={currentNat4Page}
            totalPages={nat4TotalPages}
            totalItems={nat4Mappings.length}
            pageSize={pageSize}
            onPageChange={setNat4Page}
          />
          </>
        )}
      </div>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <div className="border-b border-gray-200 px-4 py-3">
          <div className="text-sm font-medium text-black">IPv6 地址分配</div>
          <div className="mt-1 text-xs text-gray-500">共 {ipv6Assignments.length} 个地址</div>
        </div>
        {ipv6Assignments.length === 0 ? (
          <EmptyState icon={<Network className="h-7 w-7 text-gray-400" />} text="暂无 IPv6 地址分配" />
        ) : (
          <>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[820px] text-sm">
              <thead className="border-b border-gray-200 bg-gray-50 text-xs text-gray-500">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">容器</th>
                  <th className="px-4 py-3 text-left font-medium">LXC 名称</th>
                  <th className="px-4 py-3 text-left font-medium">IPv6 地址</th>
                  <th className="px-4 py-3 text-left font-medium">前缀</th>
                  <th className="px-4 py-3 text-left font-medium">出口网卡</th>
                  <th className="px-4 py-3 text-left font-medium">状态</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {pagedIPv6Assignments.map((item) => (
                  <tr key={`${item.container_id}-${item.address}`} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <button
                        onClick={() => navigate(`/container/${item.container_id}`)}
                        className="inline-flex items-center gap-2 text-left font-medium text-black hover:underline"
                      >
                        <Server className="h-4 w-4 text-gray-400" />
                        {item.container_name}
                      </button>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-600">{item.lxc_name}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-700">{item.address}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-600">/{item.prefix_len || '-'}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-600">{item.interface || '-'}</td>
                    <td className="px-4 py-3"><StatusBadge status={item.status} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination
            page={currentIPv6Page}
            totalPages={ipv6TotalPages}
            totalItems={ipv6Assignments.length}
            pageSize={pageSize}
            onPageChange={setIPv6Page}
          />
          </>
        )}
      </div>
    </div>
  )
}

function Pagination({ page, totalPages, totalItems, pageSize, onPageChange }: {
  page: number
  totalPages: number
  totalItems: number
  pageSize: number
  onPageChange: (page: number) => void
}) {
  if (totalPages <= 1) return null

  const start = (page - 1) * pageSize + 1
  const end = Math.min(page * pageSize, totalItems)

  return (
    <div className="flex items-center justify-between gap-3 border-t border-gray-200 px-4 py-3 text-sm">
      <div className="text-xs text-gray-500">
        显示 {start}-{end}，共 {totalItems} 条
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={() => onPageChange(Math.max(1, page - 1))}
          disabled={page <= 1}
          className="rounded-md border border-gray-300 px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          上一页
        </button>
        <span className="min-w-16 text-center text-xs text-gray-500">
          {page} / {totalPages}
        </span>
        <button
          onClick={() => onPageChange(Math.min(totalPages, page + 1))}
          disabled={page >= totalPages}
          className="rounded-md border border-gray-300 px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          下一页
        </button>
      </div>
    </div>
  )
}

function CapacityCard({ title, icon, remaining, total, used, label }: {
  title: string
  icon: React.ReactNode
  remaining: string
  total: string
  used: number
  label: string
}) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-sm font-medium text-gray-700">{title}</div>
          <div className="mt-2 flex items-end gap-2">
            <span className="text-2xl font-semibold text-black">{remaining}</span>
            <span className="pb-1 text-sm text-gray-400">/ {total}</span>
          </div>
        </div>
        <div className="flex h-10 w-10 items-center justify-center rounded-md bg-gray-100">
          {icon}
        </div>
      </div>
      <div className="mt-3 text-xs text-gray-500">{label}</div>
      <div className="mt-1 text-xs text-gray-400">已分配 {used}</div>
    </div>
  )
}

function EmptyState({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
      <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-lg bg-gray-100">
        {icon}
      </div>
      <div className="text-sm font-medium text-gray-700">{text}</div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const running = status === 'running'
  return (
    <span className={`rounded px-2 py-1 text-xs ${running ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-700'}`}>
      {running ? '运行中' : (status || '未知')}
    </span>
  )
}

function formatCapacity(value: string): string {
  if (value === 'large') return '充足'
  return value
}
