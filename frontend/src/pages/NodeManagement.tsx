import { useCallback, useEffect, useState } from 'react'
import {
  Copy,
  Cpu,
  Eye,
  HardDrive,
  Loader2,
  MemoryStick,
  Monitor,
  Plus,
  Power,
  RefreshCw,
  RotateCw,
  Server,
  Square,
  Trash2,
  X,
} from 'lucide-react'
import {
  createNode,
  deleteNode,
  getNodeContainers,
  getNodeInstallScript,
  getNodes,
  nodeContainerAction,
  type Container,
  type ManagedNode,
} from '../services/api'
import { useDialog } from '../components/Dialog'
import { useLanguage } from '../contexts/LanguageContext'

function formatMB(mb?: number) {
  if (!mb || mb <= 0) return '-'
  if (mb >= 1024 * 1024) return `${(mb / 1024 / 1024).toFixed(1)} TB`
  if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`
  return `${Math.round(mb)} MB`
}

function formatGB(gb?: number) {
  if (gb === undefined || gb === null || gb <= 0) return '-'
  if (gb >= 1024) return `${(gb / 1024).toFixed(1)} TB`
  return `${gb.toFixed(1)} GB`
}

export default function NodeManagement() {
  const { t } = useLanguage()
  const { confirm, alert } = useDialog()
  const [nodes, setNodes] = useState<ManagedNode[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [newAddress, setNewAddress] = useState('')
  const [creating, setCreating] = useState(false)
  const [scriptNode, setScriptNode] = useState<ManagedNode | null>(null)
  const [scriptText, setScriptText] = useState('')
  const [scriptLoading, setScriptLoading] = useState(false)
  const [detailNode, setDetailNode] = useState<ManagedNode | null>(null)
  const [nodeContainers, setNodeContainers] = useState<Container[]>([])
  const [detailLoading, setDetailLoading] = useState(false)
  const [busyId, setBusyId] = useState<string>('')

  const refresh = useCallback(async () => {
    try {
      const res = await getNodes()
      setNodes(res.data.data || [])
    } catch {
      // 保留上次数据
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
    const timer = window.setInterval(refresh, 8000)
    return () => window.clearInterval(timer)
  }, [refresh])

  const handleCreate = async () => {
    const name = newName.trim()
    const address = newAddress.trim()
    if (!name && !address) {
      await alert(t('提示'), t('请填写节点名称或地址'))
      return
    }
    setCreating(true)
    try {
      await createNode(name || undefined, address)
      await alert(t('完成'), t('节点已创建，可查看一键安装脚本并在被控服务器上执行'))
      setShowCreate(false)
      setNewName('')
      setNewAddress('')
      refresh()
    } catch (e: any) {
      await alert(t('创建失败'), e?.response?.data?.message || String(e))
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (node: ManagedNode) => {
    const ok = await confirm(t('删除节点'), `${t('确定删除节点')} ${node.name}？`)
    if (!ok) return
    try {
      await deleteNode(node.id)
      await alert(t('完成'), t('节点已删除'))
      refresh()
    } catch (e: any) {
      await alert(t('删除失败'), e?.response?.data?.message || String(e))
    }
  }

  const loadScript = async (node: ManagedNode) => {
    setScriptNode(node)
    setScriptText('')
    setScriptLoading(true)
    try {
      const res = await getNodeInstallScript(node.id)
      setScriptText(res.data as unknown as string)
    } catch (e: any) {
      await alert(t('获取脚本失败'), e?.response?.data?.message || String(e))
    } finally {
      setScriptLoading(false)
    }
  }

  const copyScript = async () => {
    try {
      await navigator.clipboard.writeText(scriptText)
      await alert(t('已复制'), t('安装脚本已复制到剪贴板'))
    } catch {
      const ta = document.createElement('textarea')
      ta.value = scriptText
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      await alert(t('已复制'), t('安装脚本已复制到剪贴板'))
    }
  }

  const loadDetail = async (node: ManagedNode) => {
    setDetailNode(node)
    setNodeContainers([])
    setDetailLoading(true)
    try {
      const res = await getNodeContainers(node.id)
      setNodeContainers(res.data.data || [])
    } catch (e: any) {
      await alert(t('加载失败'), e?.response?.data?.message || String(e))
    } finally {
      setDetailLoading(false)
    }
  }

  const runAction = async (container: Container, action: 'start' | 'stop' | 'restart') => {
    if (!detailNode) return
    setBusyId(`${container.id}:${action}`)
    try {
      await nodeContainerAction(detailNode.id, container.id, action)
      await loadDetail(detailNode)
    } catch (e: any) {
      await alert(t('操作失败'), e?.response?.data?.message || String(e))
    } finally {
      setBusyId('')
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold text-black dark:text-white">{t('节点管理')}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t('主控节点：统一管理多台被控服务器（一键安装 Agent 后自动接入）')}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            onClick={refresh}
            className="flex items-center gap-1.5 rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            {t('刷新')}
          </button>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 rounded-md bg-black px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-white dark:text-black dark:hover:bg-gray-200"
          >
            <Plus className="h-3.5 w-3.5" />
            {t('添加节点')}
          </button>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-20">
          <div className="h-8 w-8 animate-spin rounded-full border-b-2 border-black dark:border-white"></div>
        </div>
      ) : nodes.length === 0 ? (
        <div className="rounded-lg border border-gray-200 bg-white py-16 text-center dark:border-gray-700 dark:bg-gray-900">
          <Server className="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600" />
          <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">{t('暂无被控节点')}</p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-4 inline-flex items-center gap-1.5 rounded-md bg-black px-4 py-2 text-sm text-white dark:bg-white dark:text-black"
          >
            <Plus className="h-4 w-4" />
            {t('添加第一个节点')}
          </button>
        </div>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
          <table className="w-full min-w-[820px] text-left text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-400">
                <th className="px-4 py-3 font-medium">{t('节点')}</th>
                <th className="px-4 py-3 font-medium">{t('地址')}</th>
                <th className="px-4 py-3 font-medium">{t('状态')}</th>
                <th className="px-4 py-3 font-medium">{t('资源')}</th>
                <th className="px-4 py-3 font-medium">{t('容器')}</th>
                <th className="px-4 py-3 font-medium">{t('最后心跳')}</th>
                <th className="px-4 py-3 text-right font-medium">{t('操作')}</th>
              </tr>
            </thead>
            <tbody>
              {nodes.map((node) => {
                const online = node.status === 'online'
                return (
                  <tr key={node.id} className="border-b border-gray-100 last:border-0 dark:border-gray-800">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Monitor className="h-4 w-4 shrink-0 text-gray-400" />
                        <div className="min-w-0">
                          <div className="truncate font-medium text-black dark:text-white">{node.name}</div>
                          <div className="text-xs text-gray-400">{node.version || node.id}</div>
                        </div>
                      </div>
                    </td>
                    <td className="max-w-[200px] truncate px-4 py-3 text-gray-600 dark:text-gray-300" title={node.address}>
                      {node.address || '-'}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${online ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300' : 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'}`}>
                        <span className={`h-1.5 w-1.5 rounded-full ${online ? 'bg-emerald-500' : 'bg-gray-400'}`} />
                        {online ? t('在线') : node.status === 'pending' ? t('待接入') : t('离线')}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-gray-600 dark:text-gray-300">
                      {node.cpu_count ? (
                        <div className="space-y-0.5">
                          <div className="flex items-center gap-1"><Cpu className="h-3 w-3 text-gray-400" /> {node.cpu_count} {t('核')}</div>
                          <div className="flex items-center gap-1"><MemoryStick className="h-3 w-3 text-gray-400" /> {formatMB(node.ram_used_mb)} / {formatMB(node.ram_total_mb)}</div>
                          <div className="flex items-center gap-1"><HardDrive className="h-3 w-3 text-gray-400" /> {formatGB(node.disk_used_gb)} / {formatGB(node.disk_total_gb)}</div>
                        </div>
                      ) : (
                        '-'
                      )}
                    </td>
                    <td className="px-4 py-3 text-gray-600 dark:text-gray-300">{node.container_count ?? '-'}</td>
                    <td className="px-4 py-3 text-xs text-gray-400">{node.last_seen || '-'}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => loadScript(node)}
                          className="rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
                          title={t('一键安装脚本')}
                        >
                          {t('安装脚本')}
                        </button>
                        <button
                          onClick={() => loadDetail(node)}
                          disabled={!online}
                          className="rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 disabled:opacity-40 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
                          title={t('查看被控节点')}
                        >
                          <Eye className="h-3.5 w-3.5" />
                        </button>
                        <button
                          onClick={() => handleDelete(node)}
                          className="rounded-md border border-gray-200 px-2 py-1 text-xs text-red-600 hover:bg-red-50 disabled:opacity-40 dark:border-gray-700 dark:hover:bg-red-950"
                          title={t('删除节点')}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* 添加节点 */}
      {showCreate && (
        <div className="fixed inset-0 z-[110] flex items-center justify-center bg-black/50 p-4 dark:bg-black/70">
          <div className="w-full max-w-md overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-900">
            <div className="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-gray-700">
              <h3 className="text-sm font-semibold text-black dark:text-white">{t('添加被控节点')}</h3>
              <button onClick={() => setShowCreate(false)} className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-black dark:hover:bg-gray-800 dark:hover:text-white">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="space-y-4 px-5 py-4">
              <div>
                <label className="mb-1.5 block text-xs text-gray-500">{t('节点名称')}</label>
                <input
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="node-1"
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-black outline-none focus:border-black dark:border-gray-700 dark:bg-gray-950 dark:text-white"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-xs text-gray-500">{t('被控面板地址（可选）')}</label>
                <input
                  value={newAddress}
                  onChange={(e) => setNewAddress(e.target.value)}
                  placeholder="http://1.2.3.4:8999"
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-black outline-none focus:border-black dark:border-gray-700 dark:bg-gray-950 dark:text-white"
                />
              </div>
              <p className="text-xs text-gray-400">{t('地址留空时，可在 Agent 安装脚本中通过第二个参数指定')}</p>
            </div>
            <div className="flex justify-end gap-2 border-t border-gray-100 bg-gray-50 px-5 py-3 dark:border-gray-700 dark:bg-gray-800">
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-md px-4 py-2 text-sm text-gray-700 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                {t('取消')}
              </button>
              <button
                onClick={handleCreate}
                disabled={creating}
                className="inline-flex items-center gap-2 rounded-md bg-black px-4 py-2 text-sm text-white hover:bg-gray-800 disabled:opacity-50 dark:bg-white dark:text-black dark:hover:bg-gray-200"
              >
                {creating && <Loader2 className="h-4 w-4 animate-spin" />}
                {t('创建')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 一键安装脚本 */}
      {scriptNode && (
        <div className="fixed inset-0 z-[110] flex items-center justify-center bg-black/50 p-4 dark:bg-black/70">
          <div className="flex max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-900">
            <div className="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-gray-700">
              <h3 className="text-sm font-semibold text-black dark:text-white">
                {t('一键安装脚本')} · {scriptNode.name}
              </h3>
              <button onClick={() => setScriptNode(null)} className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-black dark:hover:bg-gray-800 dark:hover:text-white">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="min-h-0 flex-1 overflow-auto px-5 py-4">
              <p className="mb-3 text-xs text-gray-500 dark:text-gray-400">
                {t('在被控服务器（root 权限）上执行以下脚本即可接入主控。可选参数：脚本名称 [节点名称] [被控面板地址]。')}
              </p>
              {scriptLoading ? (
                <div className="flex items-center justify-center py-10">
                  <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
                </div>
              ) : (
                <pre className="overflow-x-auto rounded-md bg-gray-950 p-4 text-xs leading-relaxed text-gray-100">
                  <code>{scriptText}</code>
                </pre>
              )}
            </div>
            <div className="flex justify-end gap-2 border-t border-gray-100 bg-gray-50 px-5 py-3 dark:border-gray-700 dark:bg-gray-800">
              <button
                onClick={copyScript}
                className="inline-flex items-center gap-2 rounded-md bg-black px-4 py-2 text-sm text-white hover:bg-gray-800 dark:bg-white dark:text-black dark:hover:bg-gray-200"
              >
                <Copy className="h-4 w-4" />
                {t('复制脚本')}
              </button>
              <button
                onClick={() => setScriptNode(null)}
                className="rounded-md px-4 py-2 text-sm text-gray-700 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                {t('关闭')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 节点容器详情 */}
      {detailNode && (
        <div className="fixed inset-0 z-[110] flex items-center justify-center bg-black/50 p-4 dark:bg-black/70">
          <div className="flex max-h-[85vh] w-full max-w-3xl flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-900">
            <div className="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-gray-700">
              <h3 className="text-sm font-semibold text-black dark:text-white">
                {t('被控节点')} · {detailNode.name}
                <span className="ml-2 text-xs font-normal text-gray-400">{detailNode.address}</span>
              </h3>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => loadDetail(detailNode)}
                  className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-black dark:hover:bg-gray-800 dark:hover:text-white"
                  title={t('刷新')}
                >
                  <RefreshCw className="h-4 w-4" />
                </button>
                <button onClick={() => setDetailNode(null)} className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-black dark:hover:bg-gray-800 dark:hover:text-white">
                  <X className="h-4 w-4" />
                </button>
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-auto px-5 py-4">
              {detailLoading ? (
                <div className="flex items-center justify-center py-10">
                  <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
                </div>
              ) : nodeContainers.length === 0 ? (
                <p className="py-10 text-center text-sm text-gray-400">{t('被控节点暂无容器')}</p>
              ) : (
                <div className="space-y-2">
                  {nodeContainers.map((container) => (
                    <div key={container.id} className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-gray-200 px-3 py-2.5 dark:border-gray-700">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="truncate text-sm font-medium text-black dark:text-white">{container.name}</span>
                          <span className={`rounded-full px-2 py-0.5 text-xs ${container.status === 'running' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300' : 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'}`}>
                            {container.status === 'running' ? t('运行中') : t('已停止')}
                          </span>
                        </div>
                        <div className="mt-0.5 text-xs text-gray-400">
                          {container.template} · {container.vcpu} vCPU · {container.ram_mb}MB · {container.disk_gb}GB
                          {container.ipv6 ? ` · ${container.ipv6}` : ''}
                        </div>
                      </div>
                      <div className="flex shrink-0 items-center gap-1">
                        <button
                          onClick={() => runAction(container, 'start')}
                          disabled={container.status === 'running' || busyId === `${container.id}:start`}
                          className="rounded-md border border-gray-200 p-1.5 text-gray-600 hover:bg-emerald-50 hover:text-emerald-700 disabled:opacity-40 dark:border-gray-700 dark:text-gray-300"
                          title={t('开机')}
                        >
                          <Power className="h-3.5 w-3.5" />
                        </button>
                        <button
                          onClick={() => runAction(container, 'stop')}
                          disabled={container.status !== 'running' || busyId === `${container.id}:stop`}
                          className="rounded-md border border-gray-200 p-1.5 text-gray-600 hover:bg-red-50 hover:text-red-700 disabled:opacity-40 dark:border-gray-700 dark:text-gray-300"
                          title={t('关机')}
                        >
                          <Square className="h-3.5 w-3.5" />
                        </button>
                        <button
                          onClick={() => runAction(container, 'restart')}
                          disabled={busyId === `${container.id}:restart`}
                          className="rounded-md border border-gray-200 p-1.5 text-gray-600 hover:bg-amber-50 hover:text-amber-700 disabled:opacity-40 dark:border-gray-700 dark:text-gray-300"
                          title={t('重启')}
                        >
                          <RotateCw className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="flex justify-end border-t border-gray-100 bg-gray-50 px-5 py-3 dark:border-gray-700 dark:bg-gray-800">
              <button
                onClick={() => setDetailNode(null)}
                className="rounded-md px-4 py-2 text-sm text-gray-700 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                {t('关闭')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
