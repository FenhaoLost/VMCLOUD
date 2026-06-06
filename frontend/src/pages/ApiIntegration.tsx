import { useState, useEffect, useCallback } from 'react'
import { Key, Plus, Trash2, Copy, RefreshCw, Code, X } from 'lucide-react'
import api, { APIResponse } from '../services/api'
import { copyToClipboard } from '../utils/clipboard'

interface ApiKeyItem {
  id: string
  name: string
  key?: string
  prefix: string
  ip_whitelist: string
  created_at: string
  last_used: string
}

const BASE_URL = window.location.origin

export default function ApiIntegration() {
  const [keys, setKeys] = useState<ApiKeyItem[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [newIPs, setNewIPs] = useState('')
  const [creating, setCreating] = useState(false)
  const [newKey, setNewKey] = useState('')
  const [showDocs, setShowDocs] = useState(true)
  const [copiedKey, setCopiedKey] = useState(false)

  const fetchKeys = useCallback(async () => {
    try {
      const res = await api.get<APIResponse<ApiKeyItem[]>>('/api-keys')
      setKeys(res.data.data || [])
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchKeys() }, [fetchKeys])

  const createKey = async () => {
    if (!newName.trim()) return
    setCreating(true)
    try {
      const res = await api.post<APIResponse<ApiKeyItem>>('/api-keys', {
        name: newName.trim(),
        ip_whitelist: newIPs.trim(),
      })
      if (res.data.data?.key) {
        setNewKey(res.data.data.key)
        setKeys(prev => [res.data.data!, ...prev])
      }
      setNewName('')
      setNewIPs('')
      setShowCreate(false)
    } catch { /* ignore */ }
    finally { setCreating(false) }
  }

  const deleteKey = async (id: string) => {
    if (!window.confirm('确定要删除此 API Key 吗？')) return
    try {
      await api.delete(`/api-keys/${id}`)
      setKeys(prev => prev.filter(k => k.id !== id))
    } catch { /* ignore */ }
  }

  const copyKey = async () => {
    const copied = await copyToClipboard(newKey)
    if (copied) {
      setCopiedKey(true)
      setTimeout(() => setCopiedKey(false), 2000)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-black">API 集成</h1>
        <p className="text-sm text-gray-500 mt-1">管理 API Key 与查看接口文档</p>
      </div>

      {/* API Keys */}
      <div className="bg-white border border-gray-200 rounded-lg p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-black flex items-center gap-2">
            <Key className="w-4 h-4" />API Keys
          </h2>
          <div className="flex items-center gap-2">
            <button onClick={fetchKeys} className="p-1.5 text-gray-400 hover:text-black rounded" title="刷新"><RefreshCw className="w-3.5 h-3.5" /></button>
            <button onClick={() => setShowCreate(true)} className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-black text-white rounded-md text-xs hover:bg-gray-800">
              <Plus className="w-3.5 h-3.5" />创建 Key
            </button>
          </div>
        </div>

        {newKey && (
          <div className="mb-4 p-4 bg-amber-50 border border-amber-200 rounded-lg">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-semibold text-amber-800">新 API Key 已生成</span>
              <button onClick={() => setNewKey('')} className="text-amber-600 hover:text-amber-800 text-xs">关闭</button>
            </div>
            <p className="text-xs text-amber-700 mb-2">此 Key 仅显示一次，请立即复制保存。</p>
            <div className="flex items-center gap-2">
              <code className="flex-1 px-3 py-2 bg-white border border-amber-300 rounded text-xs font-mono text-gray-800 break-all">{newKey}</code>
              <button onClick={copyKey} className="px-3 py-2 bg-amber-600 text-white rounded-md text-xs hover:bg-amber-700 whitespace-nowrap">
                {copiedKey ? '已复制' : '复制'}
              </button>
            </div>
          </div>
        )}

        {loading ? (
          <div className="py-8 text-center text-sm text-gray-400">加载中...</div>
        ) : keys.length === 0 ? (
          <div className="py-8 text-center text-sm text-gray-400">暂无 API Key，点击"创建 Key"开始</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-100 text-left text-xs font-medium text-gray-500">
                  <th className="px-3 py-2">名称</th>
                  <th className="px-3 py-2">Key 前缀</th>
                  <th className="px-3 py-2">IP 白名单</th>
                  <th className="px-3 py-2">创建时间</th>
                  <th className="px-3 py-2">最后使用</th>
                  <th className="px-3 py-2 text-right">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {keys.map(k => (
                  <tr key={k.id} className="hover:bg-gray-50">
                    <td className="px-3 py-2.5 font-medium text-gray-800">{k.name}</td>
                    <td className="px-3 py-2.5 font-mono text-xs text-gray-500">{k.prefix}</td>
                    <td className="px-3 py-2.5 text-xs text-gray-500">{k.ip_whitelist || '不限制'}</td>
                    <td className="px-3 py-2.5 text-xs text-gray-500">{k.created_at}</td>
                    <td className="px-3 py-2.5 text-xs text-gray-500">{k.last_used || '未使用'}</td>
                    <td className="px-3 py-2.5 text-right">
                      <button onClick={() => deleteKey(k.id)} className="p-1 text-gray-400 hover:text-red-600 rounded" title="删除">
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create Key Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/30" onClick={() => setShowCreate(false)} />
          <div className="relative bg-white rounded-lg shadow-xl w-full max-w-md mx-4 p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-base font-semibold text-black">创建 API Key</h3>
              <button onClick={() => setShowCreate(false)} className="p-1 text-gray-400 hover:text-black rounded"><X className="w-4 h-4" /></button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-xs text-gray-500 mb-1">名称</label>
                <input
                  value={newName}
                  onChange={e => setNewName(e.target.value)}
                  placeholder="例如：自动化脚本、CI/CD"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
                  onKeyDown={e => e.key === 'Enter' && createKey()}
                  autoFocus
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">IP 白名单（每行一个，留空不限制）</label>
                <textarea
                  value={newIPs}
                  onChange={e => setNewIPs(e.target.value)}
                  placeholder={`1.2.3.4\n10.0.0.0/24`}
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm font-mono resize-none"
                />
                <p className="text-[10px] text-gray-400 mt-1">支持单个 IP 或 CIDR 网段。留空表示允许所有 IP。</p>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button onClick={() => setShowCreate(false)} className="px-4 py-2 text-sm text-gray-600 border border-gray-200 rounded-md hover:bg-gray-50">取消</button>
                <button onClick={createKey} disabled={creating || !newName.trim()} className="px-4 py-2 text-sm bg-black text-white rounded-md hover:bg-gray-800 disabled:opacity-50">
                  {creating ? '创建中...' : '创建'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
      {/* API Documentation */}
      <div className="bg-white border border-gray-200 rounded-lg p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-black flex items-center gap-2">
            <Code className="w-4 h-4" />API 文档
          </h2>
          <button onClick={() => setShowDocs(!showDocs)} className="text-xs text-gray-500 hover:text-black">
            {showDocs ? '收起' : '展开'}
          </button>
        </div>

        {showDocs && (
          <div className="space-y-6 text-sm">
            <section>
              <h3 className="font-semibold text-black mb-2">认证方式</h3>
              <p className="text-gray-600 mb-3">所有 API 使用 <strong>POST</strong> 方法，在请求头中携带 API Key：</p>
              <div className="bg-gray-900 text-gray-100 rounded-lg p-4 font-mono text-xs space-y-2">
                <div><span className="text-blue-400">curl</span> -X POST -H <span className="text-green-400">"X-API-Key: clicd_sk_xxxx"</span> {BASE_URL}/api/containers/list</div>
                <div className="text-gray-500"># 或 Bearer 方式</div>
                <div><span className="text-blue-400">curl</span> -X POST -H <span className="text-green-400">"Authorization: Bearer clicd_sk_xxxx"</span> {BASE_URL}/api/containers/list</div>
              </div>
            </section>

            <section>
              <h3 className="font-semibold text-black mb-2">容器管理</h3>
              <Endpoint method="POST" path="/api/containers/list" desc="获取容器列表" />
              <Endpoint method="POST" path="/api/containers/detail" desc="获取容器详情" body='{"id": 1}' />
              <Endpoint method="POST" path="/api/containers/create" desc="创建容器" body={`{\n  "name": "my-container",\n  "template_id": "ubuntu-noble",\n  "vcpu": 2,\n  "ram_mb": 1024,\n  "disk_gb": 20,\n  "network_bw_mbps": 100,\n  "monthly_traffic_gb": 1000,\n  "io_speed_mbps": 500\n}`} />
              <Endpoint method="POST" path="/api/containers/start" desc="启动容器" body='{"id": 1}' />
              <Endpoint method="POST" path="/api/containers/stop" desc="停止容器" body='{"id": 1}' />
              <Endpoint method="POST" path="/api/containers/restart" desc="重启容器" body='{"id": 1}' />
              <Endpoint method="POST" path="/api/containers/delete" desc="删除容器" body='{"id": 1}' />
              <Endpoint method="POST" path="/api/containers/reinstall" desc="重装系统" body='{"id": 1, "template_id": "debian-bookworm"}' />
              <Endpoint method="POST" path="/api/containers/usage" desc="获取资源用量" body='{"id": 1}' />
              <Endpoint method="POST" path="/api/containers/traffic" desc="获取流量统计" body='{"id": 1}' />
              <Endpoint method="POST" path="/api/containers/traffic-reset" desc="重置流量" body='{"id": 1}' />
              <Endpoint method="POST" path="/api/containers/traffic-limit" desc="修改流量限制" body='{"id": 1, "traffic_mode": "total", "monthly_traffic_gb": 1000}' />
              <Endpoint method="POST" path="/api/containers/resource-limit" desc="修改资源限制" body='{"id": 1, "vcpu": 2, "ram_mb": 2048, "io_speed_mbps": 500, "network_bw_mbps": 100}' />
              <Endpoint method="POST" path="/api/containers/expiry" desc="修改到期时间" body='{"id": 1, "expires_at": "2026-12-31 23:59:59"}' />
              <Endpoint method="POST" path="/api/containers/reset-password" desc="重置 SSH 密码" body='{"id": 1}' />
            </section>

            <section>
              <h3 className="font-semibold text-black mb-2">端口映射</h3>
              <Endpoint method="POST" path="/api/containers/port-mappings/add" desc="添加映射" body='{"id": 1, "container_port": 8080, "host_port": 8080, "protocol": "tcp", "description": "Web"}' />
              <Endpoint method="POST" path="/api/containers/port-mappings/update" desc="更新映射" body='{"id": 1, "index": 0, "container_port": 8080, "host_port": 9090, "protocol": "tcp", "description": "API"}' />
              <Endpoint method="POST" path="/api/containers/port-mappings/delete" desc="删除映射" body='{"id": 1, "index": 0}' />
              <Endpoint method="POST" path="/api/containers/random-port" desc="获取随机空闲端口" body='{"id": 1}' />
            </section>

            <section>
              <h3 className="font-semibold text-black mb-2">仪表盘 & 系统</h3>
              <Endpoint method="POST" path="/api/dashboard" desc="容器统计概览" />
              <Endpoint method="POST" path="/api/host-info" desc="宿主机资源信息" />
              <Endpoint method="POST" path="/api/templates" desc="可用系统模板列表" />
              <Endpoint method="POST" path="/api/tasks" desc="任务队列" />
              <Endpoint method="POST" path="/api/tasks/delete" desc="删除任务" body='{"id": "task-1"}' />
            </section>

            <section>
              <h3 className="font-semibold text-black mb-2">超售 & 批量</h3>
              <Endpoint method="POST" path="/api/oversell" desc="获取/更新超售配置" body='{"cpu_overcommit": 4, "ram_overcommit": 2, "disk_overcommit": 1, "ksm_enabled": true, "swappiness": 10}' />
              <Endpoint method="POST" path="/api/oversell/reclaim" desc="触发一次内存回收" />
              <Endpoint method="POST" path="/api/oversell/status" desc="超售状态" />
              <Endpoint method="POST" path="/api/batch-create" desc="批量创建" body='{"containers": [{...}]}' />
              <Endpoint method="POST" path="/api/batch-action" desc="批量操作" body='{"action": "start", "containers": [1, 2, 3]}' />
            </section>

            <section>
              <h3 className="font-semibold text-black mb-2">子用户 & 日志</h3>
              <Endpoint method="POST" path="/api/sub-user/create" desc="创建管理链接" body='{"container_name": "my-container"}' />
              <Endpoint method="POST" path="/api/audit-logs" desc="操作日志" />
              <Endpoint method="POST" path="/api/login-logs" desc="登录日志" />
              <Endpoint method="POST" path="/api/security/alerts" desc="安全告警" />
            </section>

            <section>
              <h3 className="font-semibold text-black mb-2">响应格式</h3>
              <div className="bg-gray-50 border border-gray-200 rounded-lg p-4 font-mono text-xs text-gray-700">
{`{
  "success": true,
  "message": "操作成功",
  "data": { ... }
}`}
              </div>
            </section>
          </div>
        )}
      </div>
    </div>
  )
}

function Endpoint({ method, path, desc, body }: { method: string; path: string; desc: string; body?: string }) {
  return (
    <div className="flex items-start gap-3 py-2 border-b border-gray-50">
      <span className="shrink-0 px-1.5 py-0.5 rounded border text-[10px] font-mono font-bold bg-blue-50 text-blue-700 border-blue-200">{method}</span>
      <code className="shrink-0 text-xs text-gray-800 font-mono">{path}</code>
      <span className="text-xs text-gray-500 min-w-0">{desc}</span>
      {body && (
        <details className="text-xs">
          <summary className="text-gray-400 cursor-pointer hover:text-gray-600">Body</summary>
          <pre className="mt-1 p-2 bg-gray-50 rounded text-xs text-gray-600 overflow-x-auto">{body}</pre>
        </details>
      )}
    </div>
  )
}
