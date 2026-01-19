import { Users, Play, Crown, CheckCircle, Clock, ArrowRightLeft } from 'lucide-react'
import type { TeamMemberStatus, Role } from '../../types'
import clsx from 'clsx'
import { useState } from 'react'

interface WaitingRoomViewProps {
  teamMembers: TeamMemberStatus[]
  facilitatorId: string
  currentUserId: string
  isFacilitator: boolean
  send: (type: string, payload: Record<string, unknown>) => void
}

// Check if role can claim facilitator (only admins can)
const canRoleClaimFacilitator = (role: Role): boolean => {
  return role === 'admin'
}

// Get initials from name
const getInitials = (name: string): string => {
  if (!name) return '??'
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return name.slice(0, 2).toUpperCase()
}

// Role labels in French
const roleLabels: Record<Role, string> = {
  admin: 'Admin',
  member: 'Membre',
}

export default function WaitingRoomView({
  teamMembers,
  facilitatorId,
  currentUserId,
  isFacilitator,
  send,
}: WaitingRoomViewProps) {
  const [showTransferSelect, setShowTransferSelect] = useState(false)
  const connectedCount = teamMembers.filter(m => m.isConnected).length
  const totalCount = teamMembers.length

  // Find current user's role
  const currentUserMember = teamMembers.find(m => m.userId === currentUserId)
  const currentUserRole = currentUserMember?.role || 'member'
  const canClaimFacilitator = !isFacilitator && canRoleClaimFacilitator(currentUserRole)

  // Get connected members for transfer (excluding current user)
  const transferableMembers = teamMembers.filter(
    m => m.isConnected && m.userId !== currentUserId
  )

  const handleStartRetro = () => {
    send('phase_next', {})
  }

  const handleClaimFacilitator = () => {
    send('facilitator_claim', {})
  }

  const handleTransferFacilitator = (targetUserId: string) => {
    send('facilitator_transfer', { userId: targetUserId })
    setShowTransferSelect(false)
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] p-8">
      <div className="max-w-3xl w-full">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-primary-100 rounded-full mb-4">
            <Users className="w-8 h-8 text-primary-600" />
          </div>
          <h2 className="text-2xl font-bold text-gray-900 mb-2">
            Salle d'attente
          </h2>
          <p className="text-gray-600">
            En attente que tous les participants rejoignent la rétrospective
          </p>
        </div>

        {/* Progress indicator */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-3 px-6 py-3 bg-white rounded-full border border-gray-200 shadow-sm">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 bg-green-500 rounded-full animate-pulse"></div>
              <span className="text-lg font-semibold text-gray-900">
                {connectedCount}
              </span>
            </div>
            <span className="text-gray-400">/</span>
            <span className="text-lg font-semibold text-gray-900">
              {totalCount}
            </span>
            <span className="text-gray-600">participants connectés</span>
          </div>
        </div>

        {/* Team members grid */}
        <div className="bg-white rounded-xl border border-gray-200 p-6 mb-8">
          <h3 className="text-sm font-medium text-gray-500 uppercase tracking-wide mb-4">
            Membres de l'équipe
          </h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {teamMembers.map((member) => {
              const isMemberFacilitator = member.userId === facilitatorId
              const isCurrentUser = member.userId === currentUserId

              return (
                <div
                  key={member.userId}
                  className={clsx(
                    'flex items-center gap-3 p-4 rounded-lg border-2 transition-all',
                    member.isConnected
                      ? 'bg-green-50 border-green-200'
                      : 'bg-gray-50 border-gray-200 opacity-60'
                  )}
                >
                  {/* Avatar */}
                  <div className="relative">
                    <div className={clsx(
                      'w-10 h-10 rounded-full flex items-center justify-center text-sm font-medium shadow-sm',
                      member.isConnected
                        ? 'bg-white text-gray-700'
                        : 'bg-gray-200 text-gray-500'
                    )}>
                      {getInitials(member.displayName)}
                    </div>
                    {/* Connection status indicator */}
                    <div className={clsx(
                      'absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full border-2 border-white',
                      member.isConnected ? 'bg-green-500' : 'bg-gray-400'
                    )} />
                  </div>

                  {/* Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-1.5">
                      <p className={clsx(
                        'text-sm font-medium truncate',
                        member.isConnected ? 'text-gray-900' : 'text-gray-500'
                      )}>
                        {member.displayName}
                        {isCurrentUser && ' (vous)'}
                      </p>
                      {isMemberFacilitator && (
                        <Crown className="w-4 h-4 text-yellow-500 flex-shrink-0" />
                      )}
                    </div>
                    <p className="text-xs text-gray-500">
                      {roleLabels[member.role]}
                    </p>
                  </div>

                  {/* Status icon */}
                  <div className="flex-shrink-0">
                    {member.isConnected ? (
                      <CheckCircle className="w-5 h-5 text-green-500" />
                    ) : (
                      <Clock className="w-5 h-5 text-gray-400" />
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        {/* Facilitator controls */}
        {isFacilitator ? (
          <div className="flex flex-col items-center gap-4">
            <button
              onClick={handleStartRetro}
              className="flex items-center gap-2 px-8 py-4 bg-primary-600 text-white rounded-xl hover:bg-primary-700 transition-colors font-semibold text-lg shadow-lg hover:shadow-xl"
            >
              <Play className="w-6 h-6" />
              Démarrer la rétrospective
            </button>

            {/* Transfer facilitator option */}
            {transferableMembers.length > 0 && (
              <div className="flex flex-col items-center gap-2">
                {!showTransferSelect ? (
                  <button
                    onClick={() => setShowTransferSelect(true)}
                    className="flex items-center gap-2 px-4 py-2 text-sm text-gray-600 hover:text-gray-800 hover:bg-gray-100 rounded-lg transition-colors"
                  >
                    <ArrowRightLeft className="w-4 h-4" />
                    Transférer le rôle de facilitateur
                  </button>
                ) : (
                  <div className="flex items-center gap-2">
                    <select
                      className="px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                      onChange={(e) => {
                        if (e.target.value) {
                          handleTransferFacilitator(e.target.value)
                        }
                      }}
                      defaultValue=""
                    >
                      <option value="">Sélectionner un participant...</option>
                      {transferableMembers.map(m => (
                        <option key={m.userId} value={m.userId}>
                          {m.displayName}
                        </option>
                      ))}
                    </select>
                    <button
                      onClick={() => setShowTransferSelect(false)}
                      className="px-3 py-2 text-sm text-gray-500 hover:text-gray-700"
                    >
                      Annuler
                    </button>
                  </div>
                )}
              </div>
            )}

            <p className="text-sm text-gray-500">
              {connectedCount < totalCount
                ? `${totalCount - connectedCount} membre${totalCount - connectedCount > 1 ? 's' : ''} non connecté${totalCount - connectedCount > 1 ? 's' : ''}`
                : 'Tous les membres sont connectés !'}
            </p>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-4">
            {/* Claim facilitator button for admins/facilitators */}
            {canClaimFacilitator && (
              <button
                onClick={handleClaimFacilitator}
                className="flex items-center gap-2 px-6 py-3 bg-yellow-500 text-white rounded-xl hover:bg-yellow-600 transition-colors font-semibold shadow-lg hover:shadow-xl"
              >
                <Crown className="w-5 h-5" />
                Devenir facilitateur
              </button>
            )}

            <div className="inline-flex items-center gap-2 px-4 py-2 bg-gray-100 rounded-full">
              <div className="w-2 h-2 bg-yellow-500 rounded-full animate-pulse"></div>
              <span className="text-sm text-gray-600">
                En attente que le facilitateur démarre la rétrospective...
              </span>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
