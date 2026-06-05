import { useState, useEffect, useCallback, type ReactNode } from 'react'
import { Cpu, MemoryStick, HardDrive, RefreshCw, Save, RotateCcw } from 'lucide-react'
import {
  getOversell,
  updateOversell,
  getOversellStatus,
  getHostInfo,
  reclaimMemory,
  HostInfo,
  OversellConfig,
  OversellStatus,
} from '../services/api'
import { useDialog } from '../components/Dialog'
import { formatMB } from '../utils/labels'

export default function Oversell() {
  const dialog = useDialog()
  const [config, setConfig] = useState<OversellConfig | null>(null)
  const [status, setStatus] = useState<OversellStatus | null>(null)
  const [host, setHost] = useState<HostInfo | null>(null)
  const [estimateSpec, setEstimateSpec] = useState({ vcpu: 1, ramMb: 1024, diskGb: 10 })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [reclaiming, setReclaiming] = useState(false)

  const fetchData = useCallback(async () => {
    try {
      const [cfgRes, stRes, hostRes] = await Promise.all([
        getOversell(),
        getOversellStatus(),
        getHostInfo(),
      ])
      if (cfgRes.data.data) setConfig(cfgRes.data.data)
      if (stRes.data.data) setStatus(stRes.data.data)
      if (hostRes.data.data) setHost(hostRes.data.data)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchData() }, [fetchData])

  const handleSave = async () => {
    if (!config) return
    if (config.cpu_overcommit < 1 || config.ram_overcommit < 1 || config.disk_overcommit < 1) {
      await dialog.alert('参数错误', '超售倍数不能小于 1。')
      return
    }
    if (config.swappiness < 0 || config.swappiness > 100) {
      await dialog.alert('参数错误', 'Swap 倾向必须在 0 到 100 之间。')
      return
    }

    setSaving(true)
    try {
      await updateOversell(config)
      await fetchData()
      await dialog.alert('已应用', '宿主机控制参数已保存。')
    } catch (err) {
      console.error(err)
      await dialog.alert('保存失败', getErrorMessage(err, '请检查宿主机权限或稍后重试。'))
    } finally {
      setSaving(false)
    }
  }

  const handleReclaimMemory = async () => {
    setReclaiming(true)
    try {
      const res = await reclaimMemory()
      await fetchData()
      const result = res.data.data
      const errors = result?.errors?.length ? `\n失败: ${result.errors.join('; ')}` : ''
      await dialog.alert(
        '回收已触发',
        `已处理 ${result?.attempted || 0} 个运行中容器，成功 ${result?.reclaimed || 0} 个，不支持 ${result?.unsupported || 0} 个。${errors}`
      )
    } catch (err) {
      console.error(err)
      await dialog.alert('回收失败', getErrorMessage(err, '请检查宿主机是否支持 cgroup v2 memory.reclaim。'))
    } finally {
      setReclaiming(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-black"></div>
      </div>
    )
  }

  if (!config) return null

  const estimate = host ? buildCapacityEstimate(host, status, config, estimateSpec) : null
  const ksmSupported = status?.ksm_supported !== false
  const reclaimSupported = status?.reclaim_supported !== false

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-black">宿主机控制</h1>
          <p className="text-sm text-gray-500 mt-1">超售容量、KSM 与宿主机内存参数</p>
        </div>
        <button
          onClick={fetchData}
          className="inline-flex items-center gap-2 px-3 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 text-sm"
        >
          <RefreshCw className="w-4 h-4" />
          刷新
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <ResourceCard
          icon={<Cpu className="w-3.5 h-3.5" />}
          label="已分配 vCPU"
          value={String(status?.allocated_cpu || 0)}
          hint={`超售倍数: ${config.cpu_overcommit}x`}
        />
        <ResourceCard
          icon={<MemoryStick className="w-3.5 h-3.5" />}
          label="已分配内存"
          value={formatMB(status?.allocated_ram_mb || 0)}
          hint={`超售倍数: ${config.ram_overcommit}x`}
        />
        <ResourceCard
          icon={<HardDrive className="w-3.5 h-3.5" />}
          label="已分配磁盘"
          value={`${status?.allocated_disk_gb || 0} GB`}
          hint={`超售倍数: ${config.disk_overcommit}x`}
        />
      </div>

      <div className="bg-white border border-gray-200 rounded-lg p-5">
        <h2 className="text-sm font-semibold text-black mb-4">超售倍数</h2>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <SliderField
            label="CPU 超售"
            value={config.cpu_overcommit}
            min={1}
            max={32}
            suffix="x"
            onChange={(v) => setConfig({ ...config, cpu_overcommit: v })}
            hint="只用于容量估算，不改变单台容器限制"
          />
          <SliderField
            label="内存超售"
            value={config.ram_overcommit}
            min={1}
            max={16}
            suffix="x"
            onChange={(v) => setConfig({ ...config, ram_overcommit: v })}
            hint="只用于容量估算，不改变单台容器限制"
          />
          <SliderField
            label="磁盘超售"
            value={config.disk_overcommit}
            min={1}
            max={16}
            suffix="x"
            onChange={(v) => setConfig({ ...config, disk_overcommit: v })}
            hint="用于容量预估，实际写入仍受文件系统限制"
          />
        </div>
      </div>

      <div className="bg-white border border-gray-200 rounded-lg p-5">
        <div className="flex items-center justify-between gap-4 mb-4">
          <h2 className="text-sm font-semibold text-black">容量预估</h2>
          <span className="text-xs text-gray-500">按单台容器配置计算</span>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-[320px_1fr] gap-5">
          <div className="grid grid-cols-3 gap-3">
            <NumberField
              label="vCPU"
              value={estimateSpec.vcpu}
              min={0.25}
              step={0.25}
              onChange={(value) => setEstimateSpec({ ...estimateSpec, vcpu: value })}
            />
            <NumberField
              label="内存 MB"
              value={estimateSpec.ramMb}
              min={128}
              step={128}
              onChange={(value) => setEstimateSpec({ ...estimateSpec, ramMb: value })}
            />
            <NumberField
              label="磁盘 GB"
              value={estimateSpec.diskGb}
              min={1}
              onChange={(value) => setEstimateSpec({ ...estimateSpec, diskGb: value })}
            />
          </div>

          {estimate && (
            <div className="grid grid-cols-1 xl:grid-cols-[220px_1fr] gap-4">
              <div className="rounded-lg border border-gray-200 bg-gray-50 p-4">
                <div className="text-xs text-gray-500">预计最多可开</div>
                <div className="mt-1 text-3xl font-bold text-black">{estimate.remainingCount}</div>
                <div className="mt-1 text-xs text-gray-400">
                  理论上限 {estimate.totalCount} 台，当前受 {estimate.bottleneckLabel} 限制
                </div>
              </div>
              <div className="overflow-hidden rounded-lg border border-gray-200">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-xs text-gray-500">
                    <tr>
                      <th className="px-3 py-2 text-left font-medium">资源</th>
                      <th className="px-3 py-2 text-right font-medium">实际</th>
                      <th className="px-3 py-2 text-right font-medium">超售后</th>
                      <th className="px-3 py-2 text-right font-medium">已分配</th>
                      <th className="px-3 py-2 text-right font-medium">剩余可开</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {estimate.rows.map((row) => (
                      <tr key={row.label}>
                        <td className="px-3 py-2 text-gray-700">{row.label}</td>
                        <td className="px-3 py-2 text-right font-mono text-xs text-gray-600">{row.actual}</td>
                        <td className="px-3 py-2 text-right font-mono text-xs text-gray-600">{row.capacity}</td>
                        <td className="px-3 py-2 text-right font-mono text-xs text-gray-600">{row.allocated}</td>
                        <td className="px-3 py-2 text-right font-semibold text-black">{row.remainingCount}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="bg-white border border-gray-200 rounded-lg p-5">
        <h2 className="text-sm font-semibold text-black mb-4">内存优化</h2>
        <div className="space-y-4">
          <ToggleRow
            label="KSM 合并"
            desc="合并容器间相同内存页，减少实际内存占用"
            value={config.ksm_enabled && ksmSupported}
            disabled={!ksmSupported}
            onChange={(v) => setConfig({ ...config, ksm_enabled: v })}
            extra={ksmSupported ? `已合并 ${status?.ksm_pages || 0} 页` : '当前内核不支持 KSM'}
          />
          <SliderField
            label="Swap 倾向"
            value={config.swappiness}
            min={0}
            max={100}
            suffix=""
            onChange={(v) => setConfig({ ...config, swappiness: v })}
            hint="写入 /proc/sys/vm/swappiness，值越低越少使用 swap"
          />
          <ActionRow
            title="立即回收缓存"
            desc={reclaimSupported ? '对运行中容器触发一次 cgroup v2 memory.reclaim' : '当前环境未检测到 memory.reclaim'}
            disabled={!reclaimSupported || reclaiming}
            busy={reclaiming}
            onClick={handleReclaimMemory}
          />
        </div>
      </div>

      <div className="flex justify-end">
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-2 px-6 py-2.5 bg-black text-white rounded-md hover:bg-gray-800 transition-colors text-sm font-medium disabled:opacity-50"
        >
          <Save className="w-4 h-4" />
          {saving ? '保存中...' : '应用设置'}
        </button>
      </div>
    </div>
  )
}

function ResourceCard({ icon, label, value, hint }: {
  icon: ReactNode
  label: string
  value: string
  hint: string
}) {
  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4">
      <div className="flex items-center gap-2 text-xs text-gray-500 mb-1">
        {icon}{label}
      </div>
      <div className="text-2xl font-bold text-black">{value}</div>
      <div className="text-xs text-gray-400 mt-0.5">{hint}</div>
    </div>
  )
}

function SliderField({ label, value, min, max, suffix, onChange, hint }: {
  label: string
  value: number
  min: number
  max: number
  suffix: string
  onChange: (v: number) => void
  hint?: string
}) {
  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium text-gray-700">{label}</span>
        <span className="text-sm text-gray-500 font-mono">{value}{suffix}</span>
      </div>
      <input
        type="range"
        min={min}
        max={max}
        value={value}
        onChange={(e) => onChange(parseInt(e.target.value, 10) || min)}
        className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer accent-black"
      />
      <div className="flex justify-between text-[10px] text-gray-300 mt-0.5">
        <span>{min}{suffix}</span><span>{max}{suffix}</span>
      </div>
      {hint && <div className="text-[10px] text-gray-400 mt-1">{hint}</div>}
    </div>
  )
}

function NumberField({ label, value, min, step = 1, onChange }: {
  label: string
  value: number
  min: number
  step?: number
  onChange: (value: number) => void
}) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-xs font-medium text-gray-600">{label}</span>
      <input
        type="number"
        min={min}
        step={step}
        value={value}
        onChange={(e) => {
          const parsed = step % 1 === 0 ? parseInt(e.target.value, 10) : parseFloat(e.target.value)
          onChange(Math.max(min, Number.isFinite(parsed) ? parsed : min))
        }}
        className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-black focus:border-black focus:outline-none focus:ring-2 focus:ring-black"
      />
    </label>
  )
}

function ToggleRow({ label, desc, value, disabled = false, onChange, extra }: {
  label: string
  desc: string
  value: boolean
  disabled?: boolean
  onChange: (v: boolean) => void
  extra?: string
}) {
  return (
    <div className="flex items-center justify-between py-2">
      <div>
        <div className="text-sm font-medium text-gray-700">{label}</div>
        <div className="text-xs text-gray-400">{desc}</div>
        {extra && <div className="text-xs text-gray-500 mt-0.5">{extra}</div>}
      </div>
      <label className={`relative inline-flex items-center ${disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}`}>
        <input
          type="checkbox"
          checked={value}
          disabled={disabled}
          onChange={(e) => onChange(e.target.checked)}
          className="sr-only peer"
        />
        <div className="w-9 h-5 bg-gray-300 peer-checked:bg-black rounded-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-4"></div>
      </label>
    </div>
  )
}

function ActionRow({ title, desc, disabled, busy, onClick }: {
  title: string
  desc: string
  disabled: boolean
  busy: boolean
  onClick: () => void
}) {
  return (
    <div className="flex items-center justify-between py-2">
      <div>
        <div className="text-sm font-medium text-gray-700">{title}</div>
        <div className="text-xs text-gray-400">{desc}</div>
      </div>
      <button
        onClick={onClick}
        disabled={disabled}
        className="inline-flex items-center gap-2 px-3 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 text-sm disabled:cursor-not-allowed disabled:opacity-50"
      >
        <RotateCcw className={`w-4 h-4 ${busy ? 'animate-spin' : ''}`} />
        {busy ? '回收中...' : '执行'}
      </button>
    </div>
  )
}

type EstimateSpec = {
  vcpu: number
  ramMb: number
  diskGb: number
}

type EstimateRow = {
  label: string
  actual: string
  capacity: string
  allocated: string
  totalCount: number
  remainingCount: number
}

function buildCapacityEstimate(
  host: HostInfo,
  status: OversellStatus | null,
  config: OversellConfig,
  spec: EstimateSpec
) {
  const cpuCapacity = host.cpu.cores * config.cpu_overcommit
  const ramCapacity = host.ram.total_mb * config.ram_overcommit
  const diskCapacity = host.disk.total_gb * config.disk_overcommit

  const allocatedCPU = status?.allocated_cpu || 0
  const allocatedRAM = status?.allocated_ram_mb || 0
  const allocatedDisk = status?.allocated_disk_gb || 0

  const rows: EstimateRow[] = [
    {
      label: 'CPU',
      actual: `${host.cpu.cores} 核`,
      capacity: `${cpuCapacity} vCPU`,
      allocated: `${allocatedCPU} vCPU`,
      totalCount: safeFloor(cpuCapacity / spec.vcpu),
      remainingCount: safeFloor((cpuCapacity - allocatedCPU) / spec.vcpu),
    },
    {
      label: '内存',
      actual: formatMB(Number(host.ram.total_mb)),
      capacity: formatMB(ramCapacity),
      allocated: formatMB(allocatedRAM),
      totalCount: safeFloor(ramCapacity / spec.ramMb),
      remainingCount: safeFloor((ramCapacity - allocatedRAM) / spec.ramMb),
    },
    {
      label: '磁盘',
      actual: `${host.disk.total_gb} GB`,
      capacity: `${diskCapacity} GB`,
      allocated: `${allocatedDisk} GB`,
      totalCount: safeFloor(diskCapacity / spec.diskGb),
      remainingCount: safeFloor((diskCapacity - allocatedDisk) / spec.diskGb),
    },
  ]

  const totalCount = Math.min(...rows.map((row) => row.totalCount))
  const remainingCount = Math.min(...rows.map((row) => row.remainingCount))
  const bottleneck = rows.reduce((current, row) => row.remainingCount < current.remainingCount ? row : current, rows[0])

  return {
    rows,
    totalCount,
    remainingCount,
    bottleneckLabel: bottleneck.label,
  }
}

function safeFloor(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 0
  return Math.floor(value)
}

function getErrorMessage(err: unknown, fallback: string): string {
  if (typeof err === 'object' && err !== null && 'response' in err) {
    const response = (err as { response?: { data?: { message?: string } } }).response
    return response?.data?.message || fallback
  }
  return fallback
}
