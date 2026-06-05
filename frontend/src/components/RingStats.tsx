import type { ReactNode } from 'react'

interface RingStatProps {
  value: number
  max?: number
  label: string
  subLabel?: ReactNode
  size?: number
  strokeWidth?: number
}

export function RingStat({ value, max = 100, label, subLabel, size = 120, strokeWidth = 8 }: RingStatProps) {
  const radius = (size - strokeWidth) / 2
  const circumference = radius * 2 * Math.PI
  const percentage = Math.min(Math.max(value / max * 100, 0), 100)
  const strokeDashoffset = circumference - (percentage / 100) * circumference

  return (
    <div className="flex flex-col items-center">
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size} className="transform -rotate-90">
          {/* Background ring */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke="#f3f4f6"
            strokeWidth={strokeWidth}
          />
          {/* Progress ring */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke="#000000"
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={strokeDashoffset}
            style={{ transition: 'stroke-dashoffset 0.5s ease' }}
          />
        </svg>
        {/* Center value */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-2xl font-bold text-black">{value.toFixed(percentage < 1 ? 2 : 1)}%</span>
        </div>
      </div>
      <div className="mt-2 text-center">
        <div className="text-sm font-medium text-gray-800">{label}</div>
        {subLabel && <div className="text-xs text-gray-400 mt-0.5">{subLabel}</div>}
      </div>
    </div>
  )
}

interface RingStatsProps {
  cpuPercent: number
  cpuCores: number
  cpuUsed: number
  ramPercent: number
  ramUsed: number
  ramTotal: number
  swapPercent?: number
  swapUsed?: number
  swapTotal?: number
  loadPercent: number
  loadStatus: string
  diskPercent: number
  diskUsed: number
  diskTotal: number
}

export default function RingStats({
  cpuPercent,
  cpuCores,
  cpuUsed,
  ramPercent,
  ramUsed,
  ramTotal,
  swapPercent = 0,
  swapUsed = 0,
  swapTotal = 0,
  loadPercent,
  loadStatus,
  diskPercent,
  diskUsed,
  diskTotal,
}: RingStatsProps) {
  const formatGB = (mb: number) => {
    if (mb >= 1024) return `${(mb / 1024).toFixed(2)} GB`
    return `${mb} MB`
  }

  const hasSwap = swapTotal > 0

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-5">
      <h2 className="text-sm font-semibold text-black mb-4">状态</h2>
      <div className={`grid ${hasSwap ? 'grid-cols-5' : 'grid-cols-4'} gap-3`}>
        <RingStat
          value={cpuPercent}
          label="CPU"
          subLabel={`(${cpuUsed.toFixed(1)} / ${cpuCores} 核)`}
        />
        <RingStat
          value={ramPercent}
          label="内存"
          subLabel={`${formatGB(ramUsed)} / ${formatGB(ramTotal)}`}
        />
        {hasSwap && (
          <RingStat
            value={swapPercent}
            label="SWAP"
            subLabel={`${formatGB(swapUsed)} / ${formatGB(swapTotal)}`}
          />
        )}
        <RingStat
          value={loadPercent}
          label="负载"
          subLabel={loadStatus}
        />
        <RingStat
          value={diskPercent}
          label="/"
          subLabel={`${formatGB(diskUsed)} / ${formatGB(diskTotal)}`}
        />
      </div>
    </div>
  )
}
