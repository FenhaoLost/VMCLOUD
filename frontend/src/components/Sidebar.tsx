import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  ChevronLeft,
  ChevronRight,
  Code2,
  Camera,
  LayoutDashboard,
  LogOut,
  Moon,
  Package,
  Route,
  ScrollText,
  Server,
  Settings2,
  ShieldAlert,
  Sun,
  UserCog,
} from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useTheme } from '../contexts/ThemeContext'
import { getVersion } from '../services/api'
import AppIcon from './AppIcon'

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
}

export default function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { logout, isSubUser } = useAuth()
  const { theme, toggleTheme } = useTheme()
  const [version, setVersion] = useState('')

  useEffect(() => {
    getVersion()
      .then(res => {
        if (res.data?.data?.version) {
          setVersion(res.data.data.version)
        }
      })
      .catch(() => {})
  }, [])

  const isContainerPage =
    location.pathname.startsWith('/containers') ||
    location.pathname.startsWith('/container')

  const isImagesPage = location.pathname.startsWith('/images')
  const isOversellPage = location.pathname.startsWith('/oversell')
  const isSnapshotsPage = location.pathname.startsWith('/snapshots')
  const isRoutingPage = location.pathname.startsWith('/routing')
  const isAuditLogsPage = location.pathname.startsWith('/audit-logs')
  const isApiIntegrationPage = location.pathname.startsWith('/api-integration')
  const isSecurityPage = location.pathname.startsWith('/security')
  const isSettingsPage = location.pathname.startsWith('/settings')

  return (
    <aside
      className={`fixed left-0 top-0 h-full bg-white border-r border-gray-200 flex flex-col transition-all duration-300 z-30 dark:bg-gray-900 dark:border-gray-700 ${
        collapsed ? 'w-16' : 'w-60'
      }`}
    >
      <div className="flex items-center justify-between h-14 px-4 border-b border-gray-200 dark:border-gray-700">
        {!collapsed && (
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 bg-gray-100 rounded flex items-center justify-center dark:bg-gray-800">
              <AppIcon className="w-5 h-5" />
            </div>
            <span className="font-bold text-black text-sm dark:text-white">CLICD</span>
          </div>
        )}
        {collapsed && (
          <div className="w-7 h-7 bg-gray-100 rounded flex items-center justify-center mx-auto dark:bg-gray-800">
            <AppIcon className="w-5 h-5" />
          </div>
        )}
        <button
          onClick={onToggle}
          className="p-1 rounded hover:bg-gray-100 text-gray-500 dark:hover:bg-gray-800 dark:text-gray-400"
          title="切换侧边栏"
        >
          {collapsed ? (
            <ChevronRight className="w-4 h-4" />
          ) : (
            <ChevronLeft className="w-4 h-4" />
          )}
        </button>
      </div>

      <nav className="flex-1 py-4 px-2 space-y-1">
        {!isSubUser && (
          <button
            onClick={() => navigate('/')}
            className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
              location.pathname === '/'
                ? 'bg-black text-white dark:bg-white dark:text-black'
                : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
            }`}
          >
            <LayoutDashboard className="w-4 h-4" />
            {!collapsed && <span>控制面板</span>}
          </button>
        )}

        <button
          onClick={() => navigate('/containers')}
          className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
            isContainerPage
              ? 'bg-black text-white dark:bg-white dark:text-black'
              : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
          }`}
        >
          <Server className="w-4 h-4" />
          {!collapsed && <span>容器管理</span>}
        </button>

        {!isSubUser && (
          <button
            onClick={() => navigate('/images')}
            className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
              isImagesPage
                ? 'bg-black text-white dark:bg-white dark:text-black'
                : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
            }`}
          >
            <Package className="w-4 h-4" />
            {!collapsed && <span>镜像管理</span>}
          </button>
        )}

        {!isSubUser && (
          <>
            <button
              onClick={() => navigate('/oversell')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
                isOversellPage
                  ? 'bg-black text-white dark:bg-white dark:text-black'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
              }`}
            >
              <Settings2 className="w-4 h-4" />
              {!collapsed && <span>宿主机控制</span>}
            </button>

            <button
              onClick={() => navigate('/security')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
                isSecurityPage
                  ? 'bg-black text-white dark:bg-white dark:text-black'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
              }`}
            >
              <ShieldAlert className="w-4 h-4" />
              {!collapsed && <span>安全告警</span>}
            </button>

            <button
              onClick={() => navigate('/snapshots')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
                isSnapshotsPage
                  ? 'bg-black text-white dark:bg-white dark:text-black'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
              }`}
            >
              <Camera className="w-4 h-4" />
              {!collapsed && <span>快照管理</span>}
            </button>

            <button
              onClick={() => navigate('/routing')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
                isRoutingPage
                  ? 'bg-black text-white dark:bg-white dark:text-black'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
              }`}
            >
              <Route className="w-4 h-4" />
              {!collapsed && <span>路由管理</span>}
            </button>

            <button
              onClick={() => navigate('/audit-logs')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
                isAuditLogsPage
                  ? 'bg-black text-white dark:bg-white dark:text-black'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
              }`}
            >
              <ScrollText className="w-4 h-4" />
              {!collapsed && <span>操作日志</span>}
            </button>

            <button
              onClick={() => navigate('/sub-users')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
                location.pathname.startsWith('/sub-users')
                  ? 'bg-black text-white dark:bg-white dark:text-black'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
              }`}
            >
              <UserCog className="w-4 h-4" />
              {!collapsed && <span>子用户管理</span>}
            </button>

            <button
              onClick={() => navigate('/api-integration')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
                isApiIntegrationPage
                  ? 'bg-black text-white dark:bg-white dark:text-black'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
              }`}
            >
              <Code2 className="w-4 h-4" />
              {!collapsed && <span>API 集成</span>}
            </button>

            <button
              onClick={() => navigate('/settings')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors ${
                isSettingsPage
                  ? 'bg-black text-white dark:bg-white dark:text-black'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
              }`}
            >
              <UserCog className="w-4 h-4" />
              {!collapsed && <span>面板设置</span>}
            </button>
          </>
        )}
      </nav>

      <div className="border-t border-gray-200 dark:border-gray-700 p-2 space-y-1">
        {/* Theme Toggle */}
        <button
          onClick={toggleTheme}
          className="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm text-gray-600 hover:bg-gray-100 transition-colors dark:text-gray-400 dark:hover:bg-gray-800"
          title={theme === 'dark' ? '切换亮色模式' : '切换暗黑模式'}
        >
          {theme === 'dark' ? (
            <Sun className="w-4 h-4" />
          ) : (
            <Moon className="w-4 h-4" />
          )}
          {!collapsed && <span>{theme === 'dark' ? '亮色模式' : '暗黑模式'}</span>}
        </button>

        {/* Version */}
        {version && (
          <div className={`px-3 py-2 text-xs text-gray-400 dark:text-gray-500 ${collapsed ? 'text-center' : ''}`}>
            {collapsed ? (
              <span title={`v${version}`}>v{version.split('.').slice(0, 2).join('.')}</span>
            ) : (
              <span>v{version}</span>
            )}
          </div>
        )}

        {/* Logout */}
        <button
          onClick={logout}
          className="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm text-gray-600 hover:bg-gray-100 transition-colors dark:text-gray-400 dark:hover:bg-gray-800"
        >
          <LogOut className="w-4 h-4" />
          {!collapsed && <span>退出登录</span>}
        </button>
      </div>
    </aside>
  )
}
