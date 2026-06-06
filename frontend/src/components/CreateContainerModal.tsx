import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { CalendarClock, X } from 'lucide-react'
import { batchCreate, getIPv6Status, getEnabledImages, getHostInfo, CreateContainerRequest, HostInfo, IPv6Status, Template } from '../services/api'
import { useDialog } from './Dialog'

interface CreateContainerModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: (containers: CreateContainerRequest[]) => void | Promise<void>
  existingNames?: string[]
}

const defaultForm: CreateContainerRequest = {
  name: '',
  template_id: '',
  vcpu: 1,
  cpu_percent: 100,
  ram_mb: 512,
  disk_gb: 10,
  network_bw_mbps: 0,
  monthly_traffic_gb: 0,
  traffic_mode: 'total',
  traffic_in_gb: 0,
  traffic_out_gb: 0,
  io_speed_mbps: 0,
  extra_ports: [],
  port_mapping_count: 2,
  snapshot_limit: 3,
  assign_ipv6: false,
  expires_at: '',
}

export default function CreateContainerModal({ isOpen, onClose, onSuccess, existingNames = [] }: CreateContainerModalProps) {
  const dialog = useDialog()
  const [templates, setTemplates] = useState<Template[]>([])
  const [loading, setLoading] = useState(false)
  const [batchCount, setBatchCount] = useState(1)
  const [form, setForm] = useState<CreateContainerRequest>(defaultForm)
  const [hostInfo, setHostInfo] = useState<HostInfo | null>(null)
  const [ipv6Status, setIPv6Status] = useState<IPv6Status | null>(null)
  const [nameError, setNameError] = useState('')

  useEffect(() => {
    if (!isOpen) return

    getEnabledImages()
      .then((res) => {
        const data = res.data.data || []
        setTemplates(data)
        if (data.length > 0) {
          setForm((prev) => ({ ...prev, template_id: prev.template_id || data[0].id }))
        }
      })
      .catch(console.error)

    getIPv6Status()
      .then((res) => {
        const status = res.data.data || null
        setIPv6Status(status)
        if (!status?.available) {
          setForm((prev) => ({ ...prev, assign_ipv6: false }))
        }
      })
      .catch(() => {
        setIPv6Status({ available: false, reachable: false, reason: 'IPv6 status check failed', prefixes: [] })
        setForm((prev) => ({ ...prev, assign_ipv6: false }))
      })

    getHostInfo()
      .then((res) => setHostInfo(res.data.data || null))
      .catch(() => setHostInfo(null))
  }, [isOpen])

  const ipv6Available = !!ipv6Status?.available
  const ipv6Prefix = ipv6Status?.prefixes?.[0]?.prefix || ''
  const maxVCPU = hostInfo?.cpu.cores || 64
  const maxRAMMB = hostInfo?.ram.total_mb ? Number(hostInfo.ram.total_mb) : undefined
  const maxDiskGB = hostInfo?.disk.total_gb ? Math.max(1, Math.floor(hostInfo.disk.total_gb)) : undefined

  const autoPorts = useMemo(() => {
    const count = Math.max(2, form.port_mapping_count)
    return Array.from({ length: count - 1 }, (_, index) => 22002 + index)
  }, [form.port_mapping_count])

  // SSH port preview (will be allocated sequentially, starting around 22000+)
  const sshPortPreview = 22000

  // Find next available batch index to avoid name conflicts
  const batchStartIndex = useMemo(() => {
    if (batchCount <= 1 || !form.name) return 1
    const prefix = `${form.name}-`
    let maxIdx = 0
    for (const existing of existingNames) {
      if (existing.startsWith(prefix)) {
        const suffix = existing.slice(prefix.length)
        const idx = parseInt(suffix, 10)
        if (!isNaN(idx) && idx > maxIdx) {
          maxIdx = idx
        }
      }
    }
    return maxIdx + 1
  }, [form.name, batchCount, existingNames])

  const handleNameChange = (value: string) => {
    setForm({ ...form, name: value })
    if (/\s/.test(value)) {
      setNameError('容器名称不能包含空格')
    } else if (value && existingNames.includes(value) && batchCount === 1) {
      setNameError('该容器名称已存在')
    } else {
      setNameError('')
    }
  }

  const handleSubmit = async () => {
    if (!form.name || !form.template_id) {
      dialog.alert('提示', '请填写容器名称并选择系统模板')
      return
    }

    const boundedForm = clampCreateForm(form, maxVCPU, maxRAMMB, maxDiskGB)

    // Build batch of containers
    const containers: CreateContainerRequest[] = []
    const startIndex = batchStartIndex
    for (let i = 0; i < batchCount; i++) {
      const name = batchCount > 1 ? `${boundedForm.name}-${startIndex + i}` : boundedForm.name
      containers.push({
        ...boundedForm,
        name,
        port_mapping_count: Math.max(2, boundedForm.port_mapping_count || 2),
        snapshot_limit: Math.max(1, boundedForm.snapshot_limit || 3),
        extra_ports: [],
      })
    }

    setLoading(true)
    try {
      await batchCreate(containers)
      await onSuccess(containers)
      onClose()
      setBatchCount(1)
      setForm({ ...defaultForm, template_id: templates[0]?.id || '' })
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } } }
      dialog.alert('创建失败', error.response?.data?.message || '请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg border border-gray-200 shadow-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-black">创建新容器</h2>
          <button onClick={onClose} className="p-1 hover:bg-gray-100 rounded text-gray-500" title="关闭">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="px-6 py-4 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <Field label="容器名称">
              <input
                type="text"
                value={form.name}
                onChange={(event) => handleNameChange(event.target.value)}
                className={`${inputClass} ${nameError ? 'border-red-400 focus:ring-red-400 focus:border-red-400' : ''}`}
                placeholder="my-container"
                required
              />
              {nameError && <p className="text-xs text-red-500 mt-1">{nameError}</p>}
            </Field>
            <Field label="批量创建数量">
              <NumberInput value={batchCount} min={1} max={50} onChange={(value) => setBatchCount(Math.max(1, value || 1))} />
            </Field>
          </div>
          {batchCount > 1 && <p className="text-xs text-gray-400">将创建 {batchCount} 个容器：{form.name}-{batchStartIndex} 至 {form.name}-{batchStartIndex + batchCount - 1}</p>}

          <Field label="系统模板">
            {templates.length === 0 ? (
              <div className="text-sm text-amber-600 bg-amber-50 border border-amber-200 rounded-md px-3 py-2">
                暂无可用的系统镜像，请先在「镜像管理」中下载镜像模板。
              </div>
            ) : (
            <select
              value={form.template_id}
              onChange={(event) => setForm({ ...form, template_id: event.target.value })}
              className={inputClass}
            >
              {templates.map((template) => (
                <option key={template.id} value={template.id}>
                  {template.name}
                </option>
              ))}
            </select>
            )}
          </Field>

          <label className={`flex items-start gap-3 rounded-md border px-3 py-2 text-sm ${ipv6Available ? 'border-gray-200 bg-white' : 'border-gray-200 bg-gray-50 text-gray-400'}`}>
            <input
              type="checkbox"
              checked={!!form.assign_ipv6}
              disabled={!ipv6Available}
              onChange={(event) => setForm({ ...form, assign_ipv6: event.target.checked })}
              className="mt-1"
            />
            <span className="min-w-0">
              <span className="block font-medium text-gray-800">Public IPv6</span>
              <span className="block text-xs text-gray-500 truncate">
                {ipv6Available ? `Use ${ipv6Prefix}` : (ipv6Status?.reason || 'Checking IPv6 prefix...')}
              </span>
            </span>
          </label>

          <div className="grid grid-cols-2 gap-4">
            <Field label="vCPU">
              <NumberInput value={form.vcpu} min={0.25} max={maxVCPU} step={0.25} onChange={(value) => setForm({ ...form, vcpu: clampVCPU(value, maxVCPU) })} />
            </Field>
            <Field label="内存 (MB)">
              <NumberInput value={form.ram_mb} min={128} max={maxRAMMB} step={128} onChange={(value) => setForm({ ...form, ram_mb: clampInt(value, 128, maxRAMMB, 512) })} />
            </Field>
          </div>

          <div className="grid grid-cols-3 gap-3">
            <Field label="磁盘 (GB)">
              <NumberInput value={form.disk_gb} min={1} max={maxDiskGB} onChange={(value) => setForm({ ...form, disk_gb: clampInt(value, 1, maxDiskGB, 10) })} />
            </Field>
            <Field label="带宽 (Mbps)">
              <NumberInput value={form.network_bw_mbps} min={0} onChange={(value) => setForm({ ...form, network_bw_mbps: value })} />
            </Field>
            <Field label="IO 速度 (MB/s)">
              <NumberInput value={form.io_speed_mbps} min={0} onChange={(value) => setForm({ ...form, io_speed_mbps: value })} />
            </Field>
          </div>

          {/* Traffic control */}
          <div>
            <div className="flex items-center gap-3 mb-2">
              <label className="text-sm font-medium text-gray-700">月流量</label>
              <select
                value={form.traffic_mode}
                onChange={(e) => setForm({ ...form, traffic_mode: e.target.value })}
                className="h-8 px-2 border border-gray-300 rounded text-xs text-gray-600 bg-white"
              >
                <option value="total">双向统计</option>
                <option value="in_out">入/出分离</option>
              </select>
            </div>
            {form.traffic_mode === 'total' ? (
              <div className="flex items-center gap-2">
                <NumberInput value={form.monthly_traffic_gb} min={0} onChange={(value) => setForm({ ...form, monthly_traffic_gb: value })} />
                <span className="text-xs text-gray-400">GB (0=不限制)</span>
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-3">
                <Field label="入站 (GB)">
                  <NumberInput value={form.traffic_in_gb} min={0} onChange={(value) => setForm({ ...form, traffic_in_gb: value || 0 })} />
                </Field>
                <Field label="出站 (GB)">
                  <NumberInput value={form.traffic_out_gb} min={0} onChange={(value) => setForm({ ...form, traffic_out_gb: value || 0 })} />
                </Field>
              </div>
            )}
          </div>

          <Field label="NAT 端口映射数量">
            <NumberInput
              value={form.port_mapping_count}
              min={2}
              max={64}
              onChange={(value) => setForm({ ...form, port_mapping_count: Math.max(2, value || 2) })}
            />
            <div className="mt-2 flex flex-wrap gap-1.5">
              <span className="inline-flex px-2 py-1 bg-emerald-50 text-emerald-700 rounded text-xs font-mono">
                SSH: {sshPortPreview} -&gt; 22
              </span>
              {autoPorts.map((port) => (
                <span key={port} className="inline-flex px-2 py-1 bg-gray-100 text-gray-700 rounded text-xs font-mono">
                  {port} -&gt; {port}
                </span>
              ))}
            </div>
          </Field>

          <Field label="子用户快照上限">
            <NumberInput
              value={form.snapshot_limit}
              min={1}
              max={999}
              onChange={(value) => setForm({ ...form, snapshot_limit: Math.max(1, Math.round(value || 1)) })}
            />
          </Field>

          <Field label="到期时间">
            <div className="relative">
              <CalendarClock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="date"
                value={form.expires_at}
                onChange={(event) => setForm({ ...form, expires_at: event.target.value })}
                min={new Date().toISOString().slice(0, 10)}
                className={`${inputClass} pl-10`}
              />
            </div>
            <p className="text-xs text-gray-400 mt-1.5">不选择则长期有效；选择日期后，到期会自动关机。</p>
          </Field>
        </div>

        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-200">
          <button onClick={onClose} className="px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-md transition-colors">
            取消
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading}
            className="px-4 py-2 text-sm bg-black text-white rounded-md hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? '创建中...' : '创建容器'}
          </button>
        </div>
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1.5">{label}</label>
      {children}
    </div>
  )
}

function NumberInput({
  value,
  min,
  max,
  step,
  onChange,
}: {
  value: number
  min?: number
  max?: number
  step?: number
  onChange: (value: number) => void
}) {
  return (
    <input
      type="number"
      value={value}
      min={min}
      max={max}
      step={step}
      onChange={(event) => {
        const raw = event.target.value
        const value = step && !Number.isInteger(step) ? parseFloat(raw) : parseInt(raw, 10)
        onChange(value)
      }}
      className={inputClass}
    />
  )
}

function clampCreateForm(form: CreateContainerRequest, maxVCPU: number, maxRAMMB?: number, maxDiskGB?: number): CreateContainerRequest {
  return {
    ...form,
    vcpu: clampVCPU(form.vcpu, maxVCPU),
    ram_mb: clampInt(form.ram_mb, 128, maxRAMMB, 512),
    disk_gb: clampInt(form.disk_gb, 1, maxDiskGB, 10),
    snapshot_limit: clampInt(form.snapshot_limit, 1, undefined, 3),
  }
}

function clampVCPU(value: number, max: number) {
  const rounded = Math.round((Number.isFinite(value) ? value : 1) * 4) / 4
  return Number(Math.min(Math.max(rounded, 0.25), max).toFixed(2))
}

function clampInt(value: number, min: number, max?: number, fallback = min) {
  const next = Math.round(Number.isFinite(value) ? value : fallback)
  return Math.min(Math.max(next, min), max ?? next)
}

const inputClass =
  'w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-black bg-white focus:outline-none focus:ring-2 focus:ring-black focus:border-black'
