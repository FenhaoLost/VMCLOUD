import { useState, useEffect, useCallback } from 'react'
import { UserCog, Key, LogIn, Monitor, Clock, Globe } from 'lucide-react'
import {
  changePassword,
  changeUsername,
  getLoginLogs,
  LoginLog,
} from '../services/api'
import { useDialog } from '../components/Dialog'
import { useAuth } from '../contexts/AuthContext'

export default function Settings() {
  const dialog = useDialog()
  const { username } = useAuth()
  const [logs, setLogs] = useState<LoginLog[]>([])
  const [loading, setLoading] = useState(true)
  const [logPage, setLogPage] = useState(1)
  const pageSize = 10

  const [oldPwd, setOldPwd] = useState('')
  const [newPwd, setNewPwd] = useState('')
  const [newUsername, setNewUsername] = useState('')
  const [pwdForUser, setPwdForUser] = useState('')

  const fetchLogs = useCallback(async () => {
    try {
      const res = await getLoginLogs()
      if (res.data.data) setLogs(res.data.data)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchLogs(); const t = setInterval(fetchLogs, 15000); return () => clearInterval(t) }, [fetchLogs])

  const handleSaveAccount = async () => {
    if (!oldPwd) { dialog.alert('提示', '请输入当前密码以确认修改'); return }
    if (!newPwd && !newUsername) { dialog.alert('提示', '至少填写新密码或新用户名中的一项'); return }
    if (newPwd && newPwd.length < 6) { dialog.alert('提示', '新密码至少 6 位'); return }
    if (newUsername && newUsername.length < 3) { dialog.alert('提示', '用户名至少 3 位'); return }

    let results: string[] = []
    try {
      // 先改用户名（用旧密码验证），再改密码，否则改完密码后旧密码就失效了
      if (newUsername) {
        const res = await changeUsername(newUsername, oldPwd)
        if (res.data.success) results.push('用户名已修改')
        else results.push('用户名修改失败')
      }
      if (newPwd) {
        const res = await changePassword(oldPwd, newPwd)
        if (res.data.success) results.push('密码已修改')
        else results.push('密码修改失败')
      }
      if (results.length > 0) {
        dialog.alert('完成', results.join('，') + '。下次登录生效')
        setOldPwd(''); setNewPwd(''); setNewUsername('')
      }
    } catch (err: unknown) {
      const e = err as { response?: { data?: { message?: string } } }
      dialog.alert('失败', e.response?.data?.message || '修改失败')
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-black"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-black">面板设置</h1>
        <p className="text-sm text-gray-500 mt-1">账号管理与登录日志</p>
      </div>

      {/* Account Settings */}
      <div className="bg-white border border-gray-200 rounded-lg p-5">
        <h2 className="text-sm font-semibold text-black mb-4 flex items-center gap-2">
          <UserCog className="w-4 h-4" />账号设置
        </h2>
        <div className="space-y-4">
          <div>
            <label className="block text-xs text-gray-500 mb-1">当前用户名</label>
            <input type="text" value={username || ''} disabled className="w-full px-3 py-2 border border-gray-200 rounded-md text-sm text-gray-400 bg-gray-50" />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">新用户名（留空则不修改）</label>
            <input type="text" value={newUsername} onChange={(e) => setNewUsername(e.target.value)} className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-black bg-white" placeholder="至少 3 位" />
          </div>
          <div className="border-t border-gray-100 pt-3">
            <label className="block text-xs text-gray-500 mb-1">新密码（留空则不修改）</label>
            <input type="password" value={newPwd} onChange={(e) => setNewPwd(e.target.value)} className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-black bg-white" placeholder="至少 6 位" />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">当前密码（验证身份）</label>
            <input type="password" value={oldPwd} onChange={(e) => setOldPwd(e.target.value)} className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-black bg-white" placeholder="输入当前密码以确认修改" />
          </div>
          <button onClick={handleSaveAccount} className="w-full px-4 py-2 bg-black text-white rounded-md text-sm hover:bg-gray-800">保存修改</button>
        </div>
      </div>

      {/* Login Logs */}
      <div className="bg-white border border-gray-200 rounded-lg p-5">
        <h2 className="text-sm font-semibold text-black mb-4 flex items-center gap-2">
          <LogIn className="w-4 h-4" />登录日志
        </h2>
        {logs.length === 0 ? (
          <p className="text-sm text-gray-400">暂无登录记录</p>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-gray-400 border-b border-gray-100">
                    <th className="text-left py-2 font-medium w-40"><span className="inline-flex items-center gap-1"><Clock className="w-3 h-3" />时间</span></th>
                    <th className="text-left py-2 font-medium">用户名</th>
                    <th className="text-left py-2 font-medium"><span className="inline-flex items-center gap-1"><Globe className="w-3 h-3" />IP</span></th>
                    <th className="text-left py-2 font-medium"><span className="inline-flex items-center gap-1"><Monitor className="w-3 h-3" />设备</span></th>
                    <th className="text-left py-2 font-medium">结果</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {logs.slice((logPage - 1) * pageSize, logPage * pageSize).map((log, i) => (
                    <tr key={i}>
                      <td className="py-1.5 text-gray-500 font-mono whitespace-nowrap">{log.time}</td>
                      <td className="py-1.5 text-gray-700">{log.username}</td>
                      <td className="py-1.5 text-gray-500 font-mono">{log.ip}</td>
                      <td className="py-1.5 text-gray-500 max-w-[180px] truncate" title={log.user_agent}>{formatUA(log.user_agent)}</td>
                      <td className="py-1.5">
                        <span className={`px-1.5 py-0.5 rounded text-xs ${log.success ? 'bg-gray-100 text-gray-700' : 'bg-red-50 text-red-600'}`}>
                          {log.success ? '成功' : '失败'}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {logs.length > pageSize && (
              <div className="flex items-center justify-between mt-3 pt-3 border-t border-gray-100">
                <span className="text-xs text-gray-400">共 {logs.length} 条，第 {logPage}/{Math.ceil(logs.length / pageSize)} 页</span>
                <div className="flex items-center gap-1">
                  <button onClick={() => setLogPage(1)} disabled={logPage === 1} className="px-2 py-1 text-xs border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-30">首页</button>
                  <button onClick={() => setLogPage(p => Math.max(1, p - 1))} disabled={logPage === 1} className="px-2 py-1 text-xs border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-30">上一页</button>
                  {Array.from({length: Math.min(5, Math.ceil(logs.length / pageSize))}, (_, i) => {
                    const totalPages = Math.ceil(logs.length / pageSize)
                    let start = Math.max(1, logPage - 2)
                    if (start + 4 > totalPages) start = Math.max(1, totalPages - 4)
                    const page = start + i
                    if (page > totalPages) return null
                    return (
                      <button key={page} onClick={() => setLogPage(page)} className={`w-7 h-7 text-xs rounded ${page === logPage ? 'bg-black text-white' : 'border border-gray-200 hover:bg-gray-50'}`}>{page}</button>
                    )
                  })}
                  <button onClick={() => setLogPage(p => Math.min(Math.ceil(logs.length / pageSize), p + 1))} disabled={logPage >= Math.ceil(logs.length / pageSize)} className="px-2 py-1 text-xs border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-30">下一页</button>
                  <button onClick={() => setLogPage(Math.ceil(logs.length / pageSize))} disabled={logPage >= Math.ceil(logs.length / pageSize)} className="px-2 py-1 text-xs border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-30">末页</button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function formatUA(ua: string): string {
  // Extract browser/OS info from UA string
  const parts: string[] = []
  if (ua.includes('Windows NT')) parts.push('Windows')
  else if (ua.includes('Mac OS X')) parts.push('macOS')
  else if (ua.includes('Linux')) parts.push('Linux')
  else if (ua.includes('Android')) parts.push('Android')
  else if (ua.includes('iPhone') || ua.includes('iPad')) parts.push('iOS')

  if (ua.includes('Chrome') && !ua.includes('Edg')) parts.push('Chrome')
  else if (ua.includes('Firefox')) parts.push('Firefox')
  else if (ua.includes('Edg')) parts.push('Edge')
  else if (ua.includes('Safari') && !ua.includes('Chrome')) parts.push('Safari')

  return parts.join(' / ') || ua.substring(0, 40)
}
